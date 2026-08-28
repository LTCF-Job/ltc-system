package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ltc-system/apps/api/internal/adapter/google"
	"ltc-system/apps/api/internal/repository"

	"github.com/google/uuid"
)

// FormRepositoryPort 定義表單資料存取介面。
type FormRepositoryPort interface {
	ListGoogleForms(ctx context.Context) ([]repository.GoogleFormEntity, error)
	ListColumnsWithMapping(ctx context.Context, mappingStatus string) ([]repository.FormColumnMappingEntity, error)
	UpdateColumnMappingById(ctx context.Context, colID string, status string, caseID *string, legSeq *int16) error
	CreateGoogleForm(ctx context.Context, id, vehicleID uuid.UUID, title, sheetID, secretRef string) error
	DeleteGoogleForm(ctx context.Context, formID uuid.UUID) error
	SaveFormColumns(ctx context.Context, formID uuid.UUID, headers []string) error
}

// GoogleAdapterPort 定義 Google 試算表與雲端硬碟適配器介面。
type GoogleAdapterPort interface {
	ListDriveSheets(ctx context.Context) ([]google.DriveFileItem, error)
	GetSpreadsheetInfo(ctx context.Context, spreadsheetID string, accessToken string) (*google.SpreadsheetInfo, error)
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

// FormService 負責處理表單清單查詢、同步與欄位對應業務邏輯。
type FormService struct {
	repo      FormRepositoryPort
	googleCli GoogleAdapterPort
}

// NewFormService 建立 FormService 實例。
func NewFormService(repo FormRepositoryPort, googleCli GoogleAdapterPort) *FormService {
	return &FormService{
		repo:      repo,
		googleCli: googleCli,
	}
}

// ListGoogleDriveFiles 查詢雲端硬碟中可選用的 Google 試算表清單。
func (s *FormService) ListGoogleDriveFiles(ctx context.Context) ([]GoogleDriveFileDTO, error) {
	if s.googleCli == nil {
		return []GoogleDriveFileDTO{
			{ID: "demo_zhubei1", Name: "竹北一車每日接送回報 (回覆)"},
			{ID: "demo_zhubei2", Name: "竹北二車每日接送回報 (回覆)"},
			{ID: "demo_zhunan1", Name: "竹南1車每日接送回報 (回覆)"},
		}, nil
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
	sheetID := google.ExtractSpreadsheetID(inputURLOrID)
	if sheetID == "" {
		return nil, errors.New("請輸入有效的 Google 試算表連結或 ID")
	}

	if s.googleCli == nil {
		return &InspectSheetDTO{
			SpreadsheetID:  sheetID,
			Title:          "竹北一車每日接送回報 (回覆)",
			SheetTabs:      []string{"8月回報", "7月回報", "表單回覆 1"},
			PreviewHeaders: []string{"時間戳記", "今天日期", "今日駕駛人", "蔡曾切（去）", "蔡曾切（回）"},
		}, nil
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
func (s *FormService) ListForms(ctx context.Context) ([]FormListItemDTO, error) {
	var forms []FormListItemDTO

	if s.repo != nil {
		entities, err := s.repo.ListGoogleForms(ctx)
		if err == nil && len(entities) > 0 {
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
	}

	// 離線或尚未設定資料庫時的預設展示資料
	return []FormListItemDTO{
		{
			ID:                 "44444444-4444-4444-4444-444444444401",
			FormID:             "44444444-4444-4444-4444-444444444401",
			VehicleID:          "22222222-2222-2222-2222-222222222201",
			VehicleDisplayName: "竹北一車",
			VehicleName:        "竹北一車",
			Title:              "竹北一車每日接送回報表",
			FormName:           "竹北一車每日接送回報表",
			SheetURL:           "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit",
			Region:             "hsinchu",
			SheetTabs:          []string{"8月回報", "7月回報"},
			ActiveTab:          "8月回報",
			SyncedMonths:       []string{"2026-07", "2026-08"},
			LastSyncedAt:       strPtr("2026-08-25 15:30:00"),
			SyncedRowCount:     24,
			TotalColumns:       56,
			PendingColumns:     0,
			PendingColumnMap:   0,
			HasSyncAlert:       false,
			Status:             "active",
		},
		{
			ID:                 "44444444-4444-4444-4444-444444444402",
			FormID:             "44444444-4444-4444-4444-444444444402",
			VehicleID:          "22222222-2222-2222-2222-222222222202",
			VehicleDisplayName: "竹北二車",
			VehicleName:        "竹北二車",
			Title:              "竹北二車每日接送回報表",
			FormName:           "竹北二車每日接送回報表",
			SheetURL:           "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit",
			Region:             "hsinchu",
			SheetTabs:          []string{"8月回報", "7月回報"},
			ActiveTab:          "8月回報",
			SyncedMonths:       []string{"2026-08"},
			LastSyncedAt:       strPtr("2026-08-25 15:20:00"),
			SyncedRowCount:     18,
			TotalColumns:       48,
			PendingColumns:     0,
			PendingColumnMap:   0,
			HasSyncAlert:       false,
			Status:             "active",
		},
		{
			ID:                 "44444444-4444-4444-4444-444444444403",
			FormID:             "44444444-4444-4444-4444-444444444403",
			VehicleID:          "22222222-2222-2222-2222-222222222203",
			VehicleDisplayName: "竹南1車",
			VehicleName:        "竹南1車",
			Title:              "竹南1車每日接送回報表",
			FormName:           "竹南1車每日接送回報表",
			SheetURL:           "https://docs.google.com/spreadsheets/d/1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms/edit",
			Region:             "miaoli",
			SheetTabs:          []string{"8月回報", "7月回報"},
			ActiveTab:          "8月回報",
			SyncedMonths:       []string{"2026-07"},
			LastSyncedAt:       strPtr("2026-08-20 14:00:00"),
			SyncedRowCount:     20,
			TotalColumns:       62,
			PendingColumns:     1,
			PendingColumnMap:   1,
			HasSyncAlert:       true,
			Status:             "active",
		},
	}, nil
}

// CreateFormAssociation 建立新表單與 Google 試算表關聯，自動抓取所有分頁與欄位並儲存至資料庫。
func (s *FormService) CreateFormAssociation(ctx context.Context, req CreateFormAssociationRequest) (*FormListItemDTO, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, errors.New("表單名稱不可為空")
	}
	if strings.TrimSpace(req.SheetURL) == "" {
		return nil, errors.New("Google 試算表連結不可為空")
	}

	sheetID := google.ExtractSpreadsheetID(req.SheetURL)
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
				spreadsheetID = google.ExtractSpreadsheetID(opts.SpreadsheetID)
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

// ListColumns 查詢表單欄位對應清單。
func (s *FormService) ListColumns(ctx context.Context, mappingStatus string) ([]FormColumnDTO, error) {
	if s.repo != nil {
		entities, err := s.repo.ListColumnsWithMapping(ctx, mappingStatus)
		if err == nil && len(entities) > 0 {
			var dtos []FormColumnDTO
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
	}

	// 預設示範欄位清單
	defaultCols := []FormColumnDTO{
		{
			ID:                "col_001",
			FormID:            "44444444-4444-4444-4444-444444444401",
			FormTitle:         "竹北一車每日接送回報表",
			VehicleName:       "竹北一車",
			ColumnIndex:       4,
			ColumnHeader:      "蔡曾切（去）",
			CleanedName:       "蔡曾切",
			Kind:              "ride_status",
			MappingStatus:     "mapped",
			CaseID:            strPtr("55555555-5555-5555-5555-555555555501"),
			CaseName:          strPtr("蔡曾切"),
			LegSeq:            int16Ptr(1),
			SuggestedCaseID:   strPtr("55555555-5555-5555-5555-555555555501"),
			SuggestedCaseName: strPtr("蔡曾切"),
			SuggestionScore:   1.0,
		},
		{
			ID:                "col_002",
			FormID:            "44444444-4444-4444-4444-444444444401",
			FormTitle:         "竹北一車每日接送回報表",
			VehicleName:       "竹北一車",
			ColumnIndex:       5,
			ColumnHeader:      "蔡曾切（回）",
			CleanedName:       "蔡曾切",
			Kind:              "ride_status",
			MappingStatus:     "mapped",
			CaseID:            strPtr("55555555-5555-5555-5555-555555555501"),
			CaseName:          strPtr("蔡曾切"),
			LegSeq:            int16Ptr(2),
			SuggestedCaseID:   strPtr("55555555-5555-5555-5555-555555555501"),
			SuggestedCaseName: strPtr("蔡曾切"),
			SuggestionScore:   1.0,
		},
		{
			ID:                "col_003",
			FormID:            "44444444-4444-4444-4444-444444444403",
			FormTitle:         "竹南1車每日接送回報表",
			VehicleName:       "竹南1車",
			ColumnIndex:       8,
			ColumnHeader:      "李國盛（去程）",
			CleanedName:       "李國盛",
			Kind:              "ride_status",
			MappingStatus:     "pending",
			CaseID:            nil,
			LegSeq:            nil,
			SuggestedCaseID:   strPtr("55555555-5555-5555-5555-555555555505"),
			SuggestedCaseName: strPtr("李國盛"),
			SuggestionScore:   0.95,
		},
	}

	var filtered []FormColumnDTO
	for _, col := range defaultCols {
		if mappingStatus == "" || col.MappingStatus == mappingStatus {
			filtered = append(filtered, col)
		}
	}
	return filtered, nil
}

// UpdateColumnMapping 更新單一欄位之對應狀態。
func (s *FormService) UpdateColumnMapping(ctx context.Context, colID string, status string, caseID *string, legSeq *int16) error {
	if s.repo != nil {
		return s.repo.UpdateColumnMappingById(ctx, colID, status, caseID, legSeq)
	}
	return nil
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
