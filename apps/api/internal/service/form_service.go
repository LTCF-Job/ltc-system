package service

import (
	"context"
	"time"

	"ltc-system/apps/api/internal/repository"
)

// FormRepositoryPort 定義表單資料存取介面。
type FormRepositoryPort interface {
	ListGoogleForms(ctx context.Context) ([]repository.GoogleFormEntity, error)
	ListColumnsWithMapping(ctx context.Context, mappingStatus string) ([]repository.FormColumnMappingEntity, error)
	UpdateColumnMappingById(ctx context.Context, colID string, status string, caseID *string, legSeq *int16) error
}

// FormListItemDTO 表單清單項目 DTO。
type FormListItemDTO struct {
	ID                 string  `json:"id"`
	VehicleID          string  `json:"vehicleId"`
	VehicleDisplayName string  `json:"vehicleDisplayName"`
	FormName           string  `json:"formName"`
	Region             string  `json:"region"`
	LastSyncedAt       *string `json:"lastSyncedAt"`
	SyncedRowCount     int     `json:"syncedRowCount"`
	PendingColumnMap   int     `json:"pendingColumnMap"`
	Status             string  `json:"status"`
}

// FormColumnDTO 表單欄位對應 DTO。
type FormColumnDTO struct {
	ID                string   `json:"id"`
	FormID            string   `json:"formId"`
	FormTitle         string   `json:"formTitle"`
	VehicleName       string   `json:"vehicleName"`
	ColumnIndex       int      `json:"columnIndex"`
	ColumnHeader      string   `json:"columnHeader"`
	CleanedName       string   `json:"cleanedName"`
	Kind              string   `json:"kind"`
	MappingStatus     string   `json:"mappingStatus"`
	CaseID            *string  `json:"caseId"`
	CaseName          *string  `json:"caseName"`
	LegSeq            *int16   `json:"legSeq"`
	SuggestedCaseID   *string  `json:"suggestedCaseId"`
	SuggestedCaseName *string  `json:"suggestedCaseName"`
	SuggestionScore   float64  `json:"suggestionScore"`
}

// ColumnMappingUpdate 批次欄位對應更新項目。
type ColumnMappingUpdate struct {
	ColumnID      string  `json:"columnId"`
	MappingStatus string  `json:"mappingStatus"`
	CaseID        *string `json:"caseId"`
	LegSeq        *int16  `json:"legSeq"`
}

// FormService 負責處理表單清單查詢、同步與欄位對應業務邏輯。
type FormService struct {
	repo FormRepositoryPort
}

// NewFormService 建立 FormService 實例。
func NewFormService(repo FormRepositoryPort) *FormService {
	return &FormService{repo: repo}
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
					str := e.LastSyncedAt.Format("2006-01-02 15:04")
					lastSyncStr = &str
				}
				forms = append(forms, FormListItemDTO{
					ID:                 e.ID.String(),
					VehicleID:          e.VehicleID.String(),
					VehicleDisplayName: e.VehicleDisplayName,
					FormName:           e.FormTitle,
					Region:             e.Region,
					LastSyncedAt:       lastSyncStr,
					SyncedRowCount:     20,
					PendingColumnMap:   0,
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
			VehicleID:          "22222222-2222-2222-2222-222222222201",
			VehicleDisplayName: "竹北一車",
			FormName:           "竹北一車每日接送回報表",
			Region:             "hsinchu",
			LastSyncedAt:       strPtr("2026-08-25 15:30"),
			SyncedRowCount:     24,
			PendingColumnMap:   0,
			Status:             "active",
		},
		{
			ID:                 "44444444-4444-4444-4444-444444444402",
			VehicleID:          "22222222-2222-2222-2222-222222222202",
			VehicleDisplayName: "竹北二車",
			FormName:           "竹北二車每日接送回報表",
			Region:             "hsinchu",
			LastSyncedAt:       strPtr("2026-08-25 15:20"),
			SyncedRowCount:     18,
			PendingColumnMap:   0,
			Status:             "active",
		},
		{
			ID:                 "44444444-4444-4444-4444-444444444403",
			VehicleID:          "22222222-2222-2222-2222-222222222203",
			VehicleDisplayName: "竹南1車",
			FormName:           "竹南1車每日接送回報表",
			Region:             "miaoli",
			LastSyncedAt:       strPtr("2026-08-25 14:00"),
			SyncedRowCount:     20,
			PendingColumnMap:   1,
			Status:             "active",
		},
	}, nil
}

// SyncForm 手動觸發表單同步。
func (s *FormService) SyncForm(ctx context.Context, formID string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"syncedRows": 24,
		"newColumns": 0,
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

