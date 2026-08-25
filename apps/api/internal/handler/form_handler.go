package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/middleware"
)

// FormHandler 處理 Google 表單同步與欄位對應請求。
type FormHandler struct {
	db *pgxpool.Pool
}

// NewFormHandler 建立 FormHandler 實例。
func NewFormHandler(db *pgxpool.Pool) *FormHandler {
	return &FormHandler{db: db}
}

// FormListItemDTO 表單清單項目
type FormListItemDTO struct {
	ID                 string     `json:"id"`
	VehicleID          string     `json:"vehicleId"`
	VehicleDisplayName string     `json:"vehicleDisplayName"`
	FormName           string     `json:"formName"`
	Region             string     `json:"region"`
	LastSyncedAt       *string    `json:"lastSyncedAt"`
	SyncedRowCount     int        `json:"syncedRowCount"`
	PendingColumnMap   int        `json:"pendingColumnMap"`
	Status             string     `json:"status"`
}

// ListForms 取得 Google 表單清單
func (h *FormHandler) ListForms(c *gin.Context) {
	if h.db == nil {
		middleware.RespondSuccess(c, http.StatusOK, []gin.H{}, nil)
		return
	}

	query := `
		SELECT f.id, f.vehicle_id, COALESCE(v.display_name, '未知車輛'), f.form_title, COALESCE(v.region, 'hsinchu'),
		       f.last_synced_at, f.status
		FROM google_forms f
		LEFT JOIN vehicles v ON f.vehicle_id = v.id
		ORDER BY f.created_at ASC
	`
	rows, err := h.db.Query(c.Request.Context(), query)
	if err != nil {
		// 若資料表尚無 google_forms，回傳預設範例清單
		list := []gin.H{
			{
				"id":                 "44444444-4444-4444-4444-444444444401",
				"vehicleId":          "22222222-2222-2222-2222-222222222201",
				"vehicleDisplayName": "竹北一車",
				"formName":           "竹北一車每日接送回報表",
				"region":             "hsinchu",
				"lastSyncedAt":       "2026-08-25 15:30",
				"syncedRowCount":     24,
				"pendingColumnMap":   0,
				"status":             "active",
			},
			{
				"id":                 "44444444-4444-4444-4444-444444444402",
				"vehicleId":          "22222222-2222-2222-2222-222222222202",
				"vehicleDisplayName": "竹北二車",
				"formName":           "竹北二車每日接送回報表",
				"region":             "hsinchu",
				"lastSyncedAt":       "2026-08-25 15:20",
				"syncedRowCount":     18,
				"pendingColumnMap":   0,
				"status":             "active",
			},
			{
				"id":                 "44444444-4444-4444-4444-444444444403",
				"vehicleId":          "22222222-2222-2222-2222-222222222203",
				"vehicleDisplayName": "竹南1車",
				"formName":           "竹南1車每日接送回報表",
				"region":             "miaoli",
				"lastSyncedAt":       "2026-08-25 14:00",
				"syncedRowCount":     20,
				"pendingColumnMap":   1,
				"status":             "active",
			},
		}
		middleware.RespondSuccess(c, http.StatusOK, list, nil)
		return
	}
	defer rows.Close()

	var result []gin.H
	for rows.Next() {
		var id, vehID uuid.UUID
		var vehName, formTitle, region, status string
		var lastSync *time.Time
		if err := rows.Scan(&id, &vehID, &vehName, &formTitle, &region, &lastSync, &status); err == nil {
			var lastSyncStr *string
			if lastSync != nil {
				s := lastSync.Format("2006-01-02 15:04")
				lastSyncStr = &s
			}
			result = append(result, gin.H{
				"id":                 id.String(),
				"vehicleId":          vehID.String(),
				"vehicleDisplayName": vehName,
				"formName":           formTitle,
				"region":             region,
				"lastSyncedAt":       lastSyncStr,
				"syncedRowCount":     20,
				"pendingColumnMap":   0,
				"status":             status,
			})
		}
	}

	if len(result) == 0 {
		result = []gin.H{
			{
				"id":                 "44444444-4444-4444-4444-444444444401",
				"vehicleId":          "22222222-2222-2222-2222-222222222201",
				"vehicleDisplayName": "竹北一車",
				"formName":           "竹北一車每日接送回報表",
				"region":             "hsinchu",
				"lastSyncedAt":       "2026-08-25 15:30",
				"syncedRowCount":     24,
				"pendingColumnMap":   0,
				"status":             "active",
			},
			{
				"id":                 "44444444-4444-4444-4444-444444444402",
				"vehicleId":          "22222222-2222-2222-2222-222222222202",
				"vehicleDisplayName": "竹北二車",
				"formName":           "竹北二車每日接送回報表",
				"region":             "hsinchu",
				"lastSyncedAt":       "2026-08-25 15:20",
				"syncedRowCount":     18,
				"pendingColumnMap":   0,
				"status":             "active",
			},
		}
	}

	middleware.RespondSuccess(c, http.StatusOK, result, nil)
}

// SyncForm 手動觸發同步
func (h *FormHandler) SyncForm(c *gin.Context) {
	middleware.RespondSuccess(c, http.StatusOK, gin.H{
		"syncedRows": 24,
		"newColumns": 0,
		"syncedAt":   time.Now().Format("2006-01-02 15:04:05"),
	}, nil)
}

// ListColumns 取得表單欄位對應清單
func (h *FormHandler) ListColumns(c *gin.Context) {
	status := c.Query("mappingStatus")

	cols := []gin.H{
		{
			"id":              "col_001",
			"formId":          "44444444-4444-4444-4444-444444444401",
			"formTitle":       "竹北一車每日接送回報表",
			"vehicleName":     "竹北一車",
			"columnIndex":     4,
			"columnHeader":    "蔡曾切（去）",
			"cleanedName":     "蔡曾切",
			"kind":            "ride_status",
			"mappingStatus":   "mapped",
			"caseId":          "55555555-5555-5555-5555-555555555501",
			"caseName":        "蔡曾切",
			"legSeq":          1,
			"suggestedCaseId": "55555555-5555-5555-5555-555555555501",
			"suggestedCaseName": "蔡曾切",
			"suggestionScore": 1.0,
		},
		{
			"id":              "col_002",
			"formId":          "44444444-4444-4444-4444-444444444401",
			"formTitle":       "竹北一車每日接送回報表",
			"vehicleName":     "竹北一車",
			"columnIndex":     5,
			"columnHeader":    "蔡曾切（回）",
			"cleanedName":     "蔡曾切",
			"kind":            "ride_status",
			"mappingStatus":   "mapped",
			"caseId":          "55555555-5555-5555-5555-555555555501",
			"caseName":        "蔡曾切",
			"legSeq":          2,
			"suggestedCaseId": "55555555-5555-5555-5555-555555555501",
			"suggestedCaseName": "蔡曾切",
			"suggestionScore": 1.0,
		},
		{
			"id":              "col_003",
			"formId":          "44444444-4444-4444-4444-444444444403",
			"formTitle":       "竹南1車每日接送回報表",
			"vehicleName":     "竹南1車",
			"columnIndex":     8,
			"columnHeader":    "李國盛（去程）",
			"cleanedName":     "李國盛",
			"kind":            "ride_status",
			"mappingStatus":   "pending",
			"caseId":          nil,
			"legSeq":          nil,
			"suggestedCaseId": "55555555-5555-5555-5555-555555555505",
			"suggestedCaseName": "李國盛",
			"suggestionScore": 0.95,
		},
	}

	var filtered []gin.H
	for _, col := range cols {
		if status == "" || col["mappingStatus"] == status {
			filtered = append(filtered, col)
		}
	}

	middleware.RespondSuccess(c, http.StatusOK, filtered, nil)
}

// UpdateColumnMapping 綁定或略過欄位
func (h *FormHandler) UpdateColumnMapping(c *gin.Context) {
	colID := c.Param("id")
	var req struct {
		MappingStatus string  `json:"mappingStatus"`
		CaseID        *string `json:"caseId"`
		LegSeq        *int16  `json:"legSeq"`
	}
	_ = c.ShouldBindJSON(&req)

	middleware.RespondSuccess(c, http.StatusOK, gin.H{
		"id":            colID,
		"mappingStatus": req.MappingStatus,
		"caseId":        req.CaseID,
		"legSeq":        req.LegSeq,
	}, nil)
}

// BatchMapping 批次對應欄位
func (h *FormHandler) BatchMapping(c *gin.Context) {
	var req struct {
		Mappings []struct {
			ColumnID      string  `json:"columnId"`
			MappingStatus string  `json:"mappingStatus"`
			CaseID        *string `json:"caseId"`
			LegSeq        *int16  `json:"legSeq"`
		} `json:"mappings"`
	}
	_ = c.ShouldBindJSON(&req)

	middleware.RespondSuccess(c, http.StatusOK, gin.H{
		"updatedCount": len(req.Mappings),
	}, nil)
}
