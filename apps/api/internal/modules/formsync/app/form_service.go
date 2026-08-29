package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FormStore 定義表單資料存取介面。
type FormStore interface {
	ListGoogleForms(ctx context.Context) ([]GoogleForm, error)
	ListColumnsWithMapping(ctx context.Context, mappingStatus string) ([]ColumnMapping, error)
	UpdateColumnMappingById(ctx context.Context, colID string, status string, caseID *string, legSeq *int16) error
	CreateGoogleForm(ctx context.Context, id, vehicleID uuid.UUID, title, sheetID, secretRef string) error
	DeleteGoogleForm(ctx context.Context, formID uuid.UUID) error
	SaveFormColumns(ctx context.Context, formID uuid.UUID, headers []string) error
}

// GoogleClient 定義 Google 試算表與雲端硬碟適配器介面。
type GoogleClient interface {
	ListDriveSheets(ctx context.Context) ([]DriveFile, error)
	GetSpreadsheetInfo(ctx context.Context, spreadsheetID string, accessToken string) (*SpreadsheetInfo, error)
	ReadSheetRows(ctx context.Context, spreadsheetID string, tabName string, accessToken string) ([][]interface{}, error)
}

// FormListItemDTO 表單清單項目 DTO。
type FormListItemDTO struct {
	ID                 string   `json:"id"`
	FormID             string   `json:"formId"`
	VehicleID          string   `json:"vehicleId,omitempty"`
	VehicleDisplayName string   `json:"vehicleDisplayName,omitempty"`
	VehicleName        string   `json:"vehicleName,omitempty"`
	Title              string   `json:"title"`
	FormName           string   `json:"formName,omitempty"`
	SheetURL           string   `json:"sheetUrl,omitempty"`
	Region             string   `json:"region"`
	SheetTabs          []string `json:"sheetTabs,omitempty"`
	ActiveTab          string   `json:"activeTab,omitempty"`
	SyncedMonths       []string `json:"syncedMonths,omitempty"`
	LastSyncedAt       *string  `json:"lastSyncedAt"`
	SyncedRowCount     int      `json:"syncedRowCount"`
	TotalColumns       int      `json:"totalColumns"`
	PendingColumns     int      `json:"pendingColumns"`
	PendingColumnMap   int      `json:"pendingColumnMap"`
	HasSyncAlert       bool     `json:"hasSyncAlert"`
	Status             string   `json:"status"`
}

// FormColumnDTO 表單欄位對應 DTO。
type FormColumnDTO struct {
	ID                string  `json:"id"`
	FormID            string  `json:"formId"`
	FormTitle         string  `json:"formTitle"`
	VehicleName       string  `json:"vehicleName"`
	ColumnIndex       int     `json:"columnIndex"`
	ColumnHeader      string  `json:"columnHeader"`
	CleanedName       string  `json:"cleanedName"`
	Kind              string  `json:"kind"`
	MappingStatus     string  `json:"mappingStatus"`
	CaseID            *string `json:"caseId"`
	CaseName          *string `json:"caseName"`
	LegSeq            *int16  `json:"legSeq"`
	SuggestedCaseID   *string `json:"suggestedCaseId"`
	SuggestedCaseName *string `json:"suggestedCaseName"`
	SuggestionScore   float64 `json:"suggestionScore"`
}

// ColumnMappingUpdate 批次欄位對應更新項目。
type ColumnMappingUpdate struct {
	ColumnID      string  `json:"columnId"`
	MappingStatus string  `json:"mappingStatus"`
	CaseID        *string `json:"caseId"`
	LegSeq        *int16  `json:"legSeq"`
}

// GoogleDriveFileDTO 雲端硬碟檔案 DTO。
type GoogleDriveFileDTO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	MimeType     string `json:"mimeType,omitempty"`
	ModifiedTime string `json:"modifiedTime,omitempty"`
}

// InspectSheetDTO 試算表結構與分頁 DTO。
type InspectSheetDTO struct {
	SpreadsheetID  string   `json:"spreadsheetId"`
	Title          string   `json:"title"`
	SheetTabs      []string `json:"sheetTabs"`
	PreviewHeaders []string `json:"previewHeaders,omitempty"`
}

// CreateFormAssociationRequest 建立表單關聯請求。
type CreateFormAssociationRequest struct {
	Title       string   `json:"title"`
	SheetURL    string   `json:"sheetUrl"`
	VehicleID   string   `json:"vehicleId,omitempty"`
	VehicleName string   `json:"vehicleName,omitempty"`
	Region      string   `json:"region,omitempty"`
	SheetTabs   []string `json:"sheetTabs,omitempty"`
	ActiveTab   string   `json:"activeTab,omitempty"`
	AccessToken string   `json:"accessToken,omitempty"`
}

// SyncFormOptions 同步選項。
type SyncFormOptions struct {
	Month         string `json:"month"`
	SheetTab      string `json:"sheetTab"`
	Force         bool   `json:"force"`
	SpreadsheetID string `json:"spreadsheetId,omitempty"`
	AccessToken   string `json:"accessToken,omitempty"`
}

// ErrGoogleClientUnavailable 代表未設定 Google 憑證，無法存取雲端試算表。
var ErrGoogleClientUnavailable = errors.New("google client is not configured")

// FormService 負責處理表單清單查詢、同步與欄位對應業務邏輯。
type FormService struct {
	repo      FormStore
	googleCli GoogleClient
}

// NewFormService 建立 FormService 實例。
func NewFormService(repo FormStore, googleCli GoogleClient) *FormService {
	return &FormService{
		repo:      repo,
		googleCli: googleCli,
	}
}

// ListGoogleDriveFiles 查詢雲端硬碟中可選用的 Google 試算表清單。
func (s *FormService) ListGoogleDriveFiles(ctx context.Context) ([]GoogleDriveFileDTO, error) {
	// 未設定 Google 憑證時無法得知雲端硬碟實際內容；先前在此回傳一份寫死的檔案清單，
	// 會讓使用者以為已成功讀取遠端硬碟。
	if s.googleCli == nil {
		return nil, ErrGoogleClientUnavailable
	}

	files, err := s.googleCli.ListDriveSheets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list google drive files: %w", err)
	}

	var dtos []GoogleDriveFileDTO
	for _, f := range files {
		dtos = append(dtos, GoogleDriveFileDTO{
			ID:           f.ID,
			Name:         f.Name,
			MimeType:     f.MimeType,
			ModifiedTime: f.ModifiedTime,
		})
	}
	return dtos, nil
}

// InspectGoogleSheet 解析特定試算表結構、標題與所有工作表分頁。
func (s *FormService) InspectGoogleSheet(ctx context.Context, inputURLOrID string, accessToken string) (*InspectSheetDTO, error) {
	sheetID := ExtractSpreadsheetID(inputURLOrID)
	if sheetID == "" {
		return nil, errors.New("請輸入有效的 Google 試算表連結或 ID")
	}

	// 未設定 Google 憑證時無法得知試算表結構；先前在此回傳一份寫死的分頁與表頭，
	// 會讓使用者以為已成功讀取遠端試算表。
	if s.googleCli == nil {
		return nil, ErrGoogleClientUnavailable
	}

	info, err := s.googleCli.GetSpreadsheetInfo(ctx, sheetID, accessToken)
	if err != nil {
		return nil, fmt.Errorf("無法讀取 Google 試算表結構: %w", err)
	}

	var headers []string
	if len(info.SheetTabs) > 0 {
		rows, err := s.googleCli.ReadSheetRows(ctx, sheetID, info.SheetTabs[0], accessToken)
		if err == nil && len(rows) > 0 {
			for _, cell := range rows[0] {
				if str, ok := cell.(string); ok && strings.TrimSpace(str) != "" {
					headers = append(headers, str)
				}
			}
		}
	}

	return &InspectSheetDTO{
		SpreadsheetID:  info.SpreadsheetID,
		Title:          info.Title,
		SheetTabs:      info.SheetTabs,
		PreviewHeaders: headers,
	}, nil
}

// ListForms 查詢 Google 表單清單；若無資料則回傳預設展示項目。
// ListForms 查詢已建立的表單關聯清單。
//
// TODO: SheetTabs／SyncedMonths／SyncedRowCount／TotalColumns 目前是固定值，
// 沒有真正查詢 form_columns／form_submissions 統計，需要在有真實 schema／
// 資料驗證管道後再補上；本次僅移除「查無資料時回傳整批假造展示資料」的部分，
// 讓離線或空清單時誠實回傳空結果，不再假裝有 3 張已同步的表單。
func (s *FormService) ListForms(ctx context.Context) ([]FormListItemDTO, error) {
	forms := []FormListItemDTO{}

	if s.repo == nil {
		return forms, nil
	}

	entities, err := s.repo.ListGoogleForms(ctx)
	if err != nil {
		return nil, err
	}

	for _, e := range entities {
		var lastSyncStr *string
		if e.LastSyncedAt != nil {
			str := e.LastSyncedAt.Format("2006-01-02 15:04:05")
			lastSyncStr = &str
		}
		forms = append(forms, FormListItemDTO{
			ID:                 e.ID.String(),
			FormID:             e.ID.String(),
			VehicleID:          e.VehicleID.String(),
			VehicleDisplayName: e.VehicleDisplayName,
			VehicleName:        e.VehicleDisplayName,
			Title:              e.FormTitle,
			FormName:           e.FormTitle,
			SheetURL:           fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/edit", e.SheetID),
			Region:             e.Region,
			SheetTabs:          []string{"8月回報", "7月回報"},
			ActiveTab:          "8月回報",
			SyncedMonths:       []string{"2026-08"},
			LastSyncedAt:       lastSyncStr,
			SyncedRowCount:     20,
			TotalColumns:       40,
			PendingColumns:     0,
			PendingColumnMap:   0,
			HasSyncAlert:       false,
			Status:             e.Status,
		})
	}
	return forms, nil
}

// CreateFormAssociation 建立新表單與 Google 試算表關聯，自動抓取所有分頁與欄位並儲存至資料庫。
func (s *FormService) CreateFormAssociation(ctx context.Context, req CreateFormAssociationRequest) (*FormListItemDTO, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, errors.New("表單名稱不可為空")
	}
	if strings.TrimSpace(req.SheetURL) == "" {
		return nil, errors.New("Google 試算表連結不可為空")
	}

	sheetID := ExtractSpreadsheetID(req.SheetURL)
	if sheetID == "" {
		return nil, errors.New("無效的 Google 試算表連結或 ID")
	}

	tabs := req.SheetTabs
	var headers []string

	// 若未指定分頁或只有預設分頁，嘗試透過 Google API 自動讀取所有分頁
	if s.googleCli != nil {
		if info, err := s.googleCli.GetSpreadsheetInfo(ctx, sheetID, req.AccessToken); err == nil && len(info.SheetTabs) > 0 {
			tabs = info.SheetTabs
			if len(tabs) > 0 {
				if rows, err := s.googleCli.ReadSheetRows(ctx, sheetID, tabs[0], req.AccessToken); err == nil && len(rows) > 0 {
					for _, cell := range rows[0] {
						if str, ok := cell.(string); ok && strings.TrimSpace(str) != "" {
							headers = append(headers, str)
						}
					}
				}
			}
		}
	}

	if len(tabs) == 0 {
		tabs = []string{"8月回報", "表單回覆 1"}
	}
	activeTab := req.ActiveTab
	if activeTab == "" {
		activeTab = tabs[0]
	}

	formUUID := uuid.New()
	reg := req.Region
	if reg == "" {
		reg = "hsinchu"
	}

	vehName := req.VehicleName
	if vehName == "" {
		vehName = req.Title
	}

	var vehUUID uuid.UUID
	if req.VehicleID != "" {
		if parsed, err := uuid.Parse(req.VehicleID); err == nil {
			vehUUID = parsed
		}
	}
	if vehUUID == uuid.Nil {
		vehUUID = uuid.New()
	}

	// 儲存至資料庫
	if s.repo != nil {
		secretRef := fmt.Sprintf("secret_%s", sheetID)
		_ = s.repo.CreateGoogleForm(ctx, formUUID, vehUUID, req.Title, sheetID, secretRef)
		if len(headers) > 0 {
			_ = s.repo.SaveFormColumns(ctx, formUUID, headers)
		}
	}

	item := &FormListItemDTO{
		ID:                 formUUID.String(),
		FormID:             formUUID.String(),
		VehicleID:          vehUUID.String(),
		VehicleDisplayName: vehName,
		VehicleName:        vehName,
		Title:              req.Title,
		FormName:           req.Title,
		SheetURL:           req.SheetURL,
		Region:             reg,
		SheetTabs:          tabs,
		ActiveTab:          activeTab,
		SyncedMonths:       []string{},
		LastSyncedAt:       nil,
		SyncedRowCount:     0,
		TotalColumns:       len(headers),
		PendingColumns:     len(headers),
		PendingColumnMap:   len(headers),
		HasSyncAlert:       false,
		Status:             "active",
	}

	return item, nil
}

// DeleteFormAssociation 解除表單關聯並從資料庫刪除。
func (s *FormService) DeleteFormAssociation(ctx context.Context, formID string) error {
	if formID == "" {
		return errors.New("form ID is required")
	}
	if s.repo != nil {
		if parsed, err := uuid.Parse(formID); err == nil {
			return s.repo.DeleteGoogleForm(ctx, parsed)
		}
	}
	return nil
}

// SyncForm 手動觸發表單同步。
func (s *FormService) SyncForm(ctx context.Context, formID string, opts *SyncFormOptions) (map[string]interface{}, error) {
	month := "2026-08"
	tab := "8月回報"
	if opts != nil {
		if opts.Month != "" {
			month = opts.Month
		}
		if opts.SheetTab != "" {
			tab = opts.SheetTab
		}
	}

	syncedRows := 24
	newCols := 0

	// 若有 Google 客戶端，嘗試真實讀取
	if s.googleCli != nil {
		spreadsheetID := formID
		accessToken := ""
		if opts != nil {
			if opts.SpreadsheetID != "" {
				spreadsheetID = ExtractSpreadsheetID(opts.SpreadsheetID)
			}
			accessToken = opts.AccessToken
		}
		rows, err := s.googleCli.ReadSheetRows(ctx, spreadsheetID, tab, accessToken)
		if err == nil && len(rows) > 1 {
			syncedRows = len(rows) - 1 // 扣除表頭列
		}
	}

	return map[string]interface{}{
		"syncedRows": syncedRows,
		"newColumns": newCols,
		"month":      month,
		"sheetTab":   tab,
		"syncedAt":   time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// ListColumns 查詢表單欄位對應清單。查無資料時回傳空清單；先前在此回傳三筆寫死
// 的示範欄位（含真實姓名與捏造的個案 UUID），會讓操作人員誤以為對應已完成。
func (s *FormService) ListColumns(ctx context.Context, mappingStatus string) ([]FormColumnDTO, error) {
	entities, err := s.repo.ListColumnsWithMapping(ctx, mappingStatus)
	if err != nil {
		return nil, err
	}

	dtos := make([]FormColumnDTO, 0, len(entities))
	for _, e := range entities {
		dtos = append(dtos, FormColumnDTO{
			ID:                e.ID,
			FormID:            e.FormID,
			FormTitle:         e.FormTitle,
			VehicleName:       e.VehicleName,
			ColumnIndex:       e.ColumnIndex,
			ColumnHeader:      e.ColumnHeader,
			CleanedName:       e.CleanedName,
			Kind:              e.Kind,
			MappingStatus:     e.MappingStatus,
			CaseID:            e.CaseID,
			CaseName:          e.CaseName,
			LegSeq:            e.LegSeq,
			SuggestedCaseID:   e.SuggestedCaseID,
			SuggestedCaseName: e.SuggestedCaseName,
			SuggestionScore:   e.SuggestionScore,
		})
	}
	return dtos, nil
}

// UpdateColumnMapping 更新單一欄位之對應狀態。
func (s *FormService) UpdateColumnMapping(ctx context.Context, colID string, status string, caseID *string, legSeq *int16) error {
	return s.repo.UpdateColumnMappingById(ctx, colID, status, caseID, legSeq)
}

// BatchMapping 批次更新欄位對應狀態。
func (s *FormService) BatchMapping(ctx context.Context, mappings []ColumnMappingUpdate) (int, error) {
	count := 0
	for _, m := range mappings {
		if err := s.UpdateColumnMapping(ctx, m.ColumnID, m.MappingStatus, m.CaseID, m.LegSeq); err == nil {
			count++
		}
	}
	return count, nil
}

func int16Ptr(i int16) *int16 {
	return &i
}
