package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ltc-system/apps/api/internal/domain/namenorm"

	"github.com/google/uuid"
)

// ErrVehicleRequired 代表建立匯報表時未指定車輛。
var ErrVehicleRequired = errors.New("vehicle is required")

// DriverReportService 負責司機接送匯報表的登錄、範本產生、匯入與欄位對應。
type DriverReportService struct {
	repo         FormStore
	excel        SpreadsheetReader
	template     TemplateRenderer
	caseRepo     CaseLookup
	driverRepo   DriverResolver
	rideIngestor RideIngestor
	auditRepo    AuditWriter
}

// NewDriverReportService 建立 DriverReportService 實例。
func NewDriverReportService(
	repo FormStore,
	excel SpreadsheetReader,
	template TemplateRenderer,
	caseRepo CaseLookup,
	driverRepo DriverResolver,
	rideIngestor RideIngestor,
	auditRepo AuditWriter,
) *DriverReportService {
	return &DriverReportService{
		repo:         repo,
		excel:        excel,
		template:     template,
		caseRepo:     caseRepo,
		driverRepo:   driverRepo,
		rideIngestor: rideIngestor,
		auditRepo:    auditRepo,
	}
}

// ListForms 查詢所有車輛的匯報表與其對應進度。
func (s *DriverReportService) ListForms(ctx context.Context) ([]ReportForm, error) {
	forms, err := s.repo.ListForms(ctx)
	if err != nil {
		return nil, err
	}
	if forms == nil {
		return []ReportForm{}, nil
	}
	return forms, nil
}

// CreateForm 為一台車建立匯報表。
func (s *DriverReportService) CreateForm(ctx context.Context, vehicleID, title string) (*ReportForm, error) {
	vehUUID, err := uuid.Parse(strings.TrimSpace(vehicleID))
	if err != nil {
		return nil, ErrVehicleRequired
	}
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("匯報表名稱不可為空")
	}

	formID := uuid.New()
	if err := s.repo.CreateForm(ctx, formID, vehUUID, strings.TrimSpace(title)); err != nil {
		return nil, err
	}
	return s.repo.GetForm(ctx, formID)
}

// DeleteForm 刪除匯報表；其欄位對應與匯報紀錄由資料庫的 ON DELETE CASCADE 一併清除。
func (s *DriverReportService) DeleteForm(ctx context.Context, formID string) error {
	parsed, err := uuid.Parse(formID)
	if err != nil {
		return ErrFormNotFound
	}
	return s.repo.DeleteForm(ctx, parsed)
}

// ListColumns 查詢欄位對應清單，可依匯報表與對應狀態篩選。
//
// 推薦趟次不入庫：它完全由表頭的 [去程]／[回程] 標記決定，改由查詢時即時推導，
// 避免前端各自再解析一次表頭而產生兩套規則。
func (s *DriverReportService) ListColumns(ctx context.Context, formID, mappingStatus string) ([]ColumnMapping, error) {
	cols, err := s.repo.ListColumnsWithMapping(ctx, formID, mappingStatus)
	if err != nil {
		return nil, err
	}

	out := make([]ColumnMapping, 0, len(cols))
	for _, c := range cols {
		c.SuggestedLegSeq = legSeqForDirection(namenorm.ParseColumnHeader(c.ColumnHeader).Direction)
		out = append(out, c)
	}
	return out, nil
}

// UpdateColumnMapping 更新單一欄位之對應狀態。
func (s *DriverReportService) UpdateColumnMapping(ctx context.Context, colID, status string, caseID *string, legSeq *int16) error {
	if status == "mapped" && (caseID == nil || legSeq == nil) {
		return errors.New("標記為已對應時必須同時指定個案與趟次")
	}
	return s.repo.UpdateColumnMappingByID(ctx, colID, status, caseID, legSeq)
}

// BatchMapping 批次更新欄位對應狀態，回傳成功更新筆數。
func (s *DriverReportService) BatchMapping(ctx context.Context, updates []ColumnMappingUpdate) (int, error) {
	count := 0
	for _, u := range updates {
		if err := s.UpdateColumnMapping(ctx, u.ColumnID, u.MappingStatus, u.CaseID, u.LegSeq); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// TemplateExcel 產生指定匯報表的空白範本；欄位由該車已對應的個案趟次組成，
// 尚未對應過任何欄位時只給出固定欄位，讓使用者先貼上實際表頭再匯入。
func (s *DriverReportService) TemplateExcel(ctx context.Context, formID uuid.UUID) ([]byte, string, error) {
	form, err := s.repo.GetForm(ctx, formID)
	if err != nil {
		return nil, "", err
	}
	if form == nil {
		return nil, "", ErrFormNotFound
	}

	cols, err := s.repo.ListColumnsWithMapping(ctx, formID.String(), "mapped")
	if err != nil {
		return nil, "", err
	}

	headers := make([]string, 0, len(cols))
	for _, c := range cols {
		headers = append(headers, c.ColumnHeader)
	}

	bytesOut, err := s.template.RenderDriverReportTemplate(form.VehicleDisplayName, headers)
	if err != nil {
		return nil, "", fmt.Errorf("產生匯報表範本失敗: %w", err)
	}
	return bytesOut, form.VehicleDisplayName, nil
}
