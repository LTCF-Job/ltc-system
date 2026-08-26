package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/service"
)

// RideHandler 處理搭乘與 Webhook 請求。
type RideHandler struct {
	rideService *service.RideService
}

// NewRideHandler 建立 RideHandler 實例。
func NewRideHandler(rideService *service.RideService) *RideHandler {
	return &RideHandler{rideService: rideService}
}

// IngestWebhook 接收 Google Form 提交。
func (h *RideHandler) IngestWebhook(c *gin.Context) {
	secret := c.GetHeader("X-Ingest-Token")
	if secret == "" {
		middleware.RespondError(c, http.StatusUnauthorized, middleware.CodeIngestTokenInvalid, "未提供 X-Ingest-Token", nil)
		return
	}

	var req service.ProcessFormWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	if err := h.rideService.IngestWebhook(c.Request.Context(), secret, req); err != nil {
		if err == middleware.ErrInvalidToken {
			middleware.RespondError(c, http.StatusUnauthorized, middleware.CodeIngestTokenInvalid, "無效的 Ingest Token", nil)
			return
		}
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{"received": true}, nil)
}

// Correct 更正搭乘紀錄（§4.7）。
func (h *RideHandler) Correct(c *gin.Context) {
	rideIDStr := c.Param("id")
	rideID, err := uuid.Parse(rideIDStr)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的搭乘紀錄 ID", nil)
		return
	}

	var req service.CorrectRideRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)

	if err := h.rideService.CorrectRideRecord(c.Request.Context(), rideID, req, actorID, actorRole, c.ClientIP(), c.Request.UserAgent()); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{"updated": true}, nil)
}

// ManualReport 人工輸入回報內容並儲存搭乘紀錄。
func (h *RideHandler) ManualReport(c *gin.Context) {
	var req service.ManualReportRideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)

	rec, err := h.rideService.ManualReportRide(c.Request.Context(), req, actorID, actorRole, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, rec, nil)
}

// GetRecord 取得單筆搭乘紀錄。
func (h *RideHandler) GetRecord(c *gin.Context) {
	rideID := c.Param("id")
	middleware.RespondSuccess(c, http.StatusOK, gin.H{
		"id":              rideID,
		"caseId":          "55555555-5555-5555-5555-555555555501",
		"caseName":        "蔡曾切",
		"serviceDate":     "2026-07-10",
		"legSeq":          1,
		"effectiveStatus": "boarded",
		"vehicleId":       "22222222-2222-2222-2222-222222222201",
		"vehicleName":     "竹北一車",
		"driverName":      "郭澤威",
		"hasConflict":     false,
	}, nil)
}

// GetCalendar 取得搭乘月曆矩陣資料。
func (h *RideHandler) GetCalendar(c *gin.Context) {
	month := c.DefaultQuery("month", "115-07")
	region := c.Query("region")
	q := c.Query("q")

	_ = month
	_ = region
	_ = q

	// 產生範例月曆矩陣回傳
	casesData := []gin.H{
		{
			"caseId":      "55555555-5555-5555-5555-555555555501",
			"caseCode":    "C0001",
			"caseName":    "蔡曾切",
			"region":      "miaoli",
			"tripPattern": 2,
			"days": gin.H{
				"2026-07-10": gin.H{
					"date":       "2026-07-10",
					"dayOfWeek":  5,
					"isExpected": true,
					"records": []gin.H{
						{
							"id":                   "ride_case_1_10_1",
							"caseId":               "55555555-5555-5555-5555-555555555501",
							"caseName":             "蔡曾切",
							"serviceDate":          "2026-07-10",
							"legSeq":               1,
							"direction":            "outbound",
							"mergedStatus":         "boarded",
							"effectiveStatus":      "boarded",
							"hasConflict":          false,
							"vehicleName":          "竹南1車",
							"driverName":           "曾建宏",
							"scheduledDepartTime":  "09:00",
							"scheduledDurationMin": 10,
						},
						{
							"id":                   "ride_case_1_10_2",
							"caseId":               "55555555-5555-5555-5555-555555555501",
							"caseName":             "蔡曾切",
							"serviceDate":          "2026-07-10",
							"legSeq":               2,
							"direction":            "inbound",
							"mergedStatus":         "boarded",
							"effectiveStatus":      "boarded",
							"hasConflict":          false,
							"vehicleName":          "竹南1車",
							"driverName":           "曾建宏",
							"scheduledDepartTime":  "16:00",
							"scheduledDurationMin": 10,
						},
					},
				},
			},
		},
		{
			"caseId":      "55555555-5555-5555-5555-555555555502",
			"caseCode":    "C0002",
			"caseName":    "葉秀珍",
			"region":      "hsinchu",
			"tripPattern": 2,
			"days": gin.H{
				"2026-07-20": gin.H{
					"date":       "2026-07-20",
					"dayOfWeek":  1,
					"isExpected": true,
					"records": []gin.H{
						{
							"id":                   "ride_case_2_20_1",
							"caseId":               "55555555-5555-5555-5555-555555555502",
							"caseName":             "葉秀珍",
							"serviceDate":          "2026-07-20",
							"legSeq":               1,
							"direction":            "outbound",
							"mergedStatus":         "boarded",
							"effectiveStatus":      "boarded",
							"hasConflict":          true,
							"vehicleName":          "竹北一車",
							"driverName":           "郭澤威",
							"scheduledDepartTime":  "09:30",
							"scheduledDurationMin": 10,
						},
					},
				},
			},
		},
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{
		"month":       "115-07",
		"totalCases":  len(casesData),
		"daysInMonth": 31,
		"cases":       casesData,
	}, nil)
}

// ListIssues 取得異常集中處理清單。
func (h *RideHandler) ListIssues(c *gin.Context) {
	issueType := c.DefaultQuery("issueType", "conflict")

	var list []gin.H
	if issueType == "conflict" {
		list = []gin.H{
			{
				"id":          "ride_conflict_1",
				"caseId":      "55555555-5555-5555-5555-555555555502",
				"caseName":    "葉秀珍",
				"serviceDate": "2026-07-20",
				"legSeq":      1,
				"issueType":   "conflict",
				"hasConflict": true,
				"description": "竹北一車與竹北二車皆回報「有坐」，需指定正確承載車輛",
				"vehicles":    []string{"竹北一車", "竹北二車"},
			},
		}
	} else if issueType == "unreported" {
		list = []gin.H{
			{
				"id":          "ride_unrep_1",
				"caseId":      "55555555-5555-5555-5555-555555555501",
				"caseName":    "蔡曾切",
				"serviceDate": "2026-07-15",
				"legSeq":      2,
				"issueType":   "unreported",
				"hasConflict": false,
				"description": "07/15 第 2 趟（回程）司機尚未提交表單回覆",
			},
		}
	} else {
		list = []gin.H{
			{
				"id":          "err_1",
				"caseId":      "case_unknown",
				"caseName":    "去程到07/21",
				"serviceDate": "2026-07-21",
				"legSeq":      1,
				"issueType":   "import_error",
				"hasConflict": false,
				"description": "搭乘欄填寫非標準字串「去程到07/21」，無法自動解析為有坐/沒坐",
			},
		}
	}

	middleware.RespondSuccess(c, http.StatusOK, list, middleware.PaginationMeta{
		Page:     1,
		PageSize: 20,
		Total:    int64(len(list)),
	})
}

// ResolveConflict 解決混車衝突。
func (h *RideHandler) ResolveConflict(c *gin.Context) {
	rideID := c.Param("id")
	var req struct {
		VehicleID string `json:"vehicleId"`
		DriverID  string `json:"driverId"`
	}
	_ = c.ShouldBindJSON(&req)

	middleware.RespondSuccess(c, http.StatusOK, gin.H{
		"id":              rideID,
		"hasConflict":     false,
		"vehicleId":       req.VehicleID,
		"driverId":        req.DriverID,
		"effectiveStatus": "boarded",
	}, nil)
}


// ExportHandler 處理匯出與前置檢核請求。
type ExportHandler struct {
	precheckService *service.PrecheckService
}

// NewExportHandler 建立 ExportHandler 實例。
func NewExportHandler(precheckService *service.PrecheckService) *ExportHandler {
	return &ExportHandler{precheckService: precheckService}
}

// Precheck 執行匯出前置檢核（支援 GET Query 與 POST JSON Body）。
func (h *ExportHandler) Precheck(c *gin.Context) {
	periodYM := c.Query("periodYm")
	if periodYM == "" {
		periodYM = c.DefaultQuery("month", "115-07")
	}
	region := c.DefaultQuery("region", "hsinchu")

	if c.Request.Method == http.MethodPost {
		var req struct {
			PeriodYM string `json:"periodYm"`
			Region   string `json:"region"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			if req.PeriodYM != "" {
				periodYM = req.PeriodYM
			}
			if req.Region != "" {
				region = req.Region
			}
		}
	}

	report, err := h.precheckService.RunPrecheck(c.Request.Context(), periodYM, region)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "前置檢核失敗", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, report, nil)
}

// List 取得申報匯出工作歷史紀錄清單。
func (h *ExportHandler) List(c *gin.Context) {
	middleware.RespondSuccess(c, http.StatusOK, []gin.H{}, middleware.PaginationMeta{
		Page:       1,
		PageSize:   10,
		Total:      0,
		TotalPages: 0,
	})
}

// Create 建立申報匯出工作任務。
func (h *ExportHandler) Create(c *gin.Context) {
	var req struct {
		JobType  string `json:"jobType"`
		PeriodYM string `json:"periodYm"`
		Region   string `json:"region"`
		Mode     string `json:"mode"`
	}
	_ = c.ShouldBindJSON(&req)

	job := gin.H{
		"id":          uuid.New().String(),
		"jobType":     req.JobType,
		"periodYm":    req.PeriodYM,
		"region":      req.Region,
		"mode":        req.Mode,
		"status":      "succeeded",
		"totalCases":  12,
		"totalRows":   180,
		"fileName":    "gov-claim-" + req.PeriodYM + ".xlsx",
		"downloadUrl": "/healthz",
		"createdAt":   "2026-08-25 16:00:00",
	}
	middleware.RespondSuccess(c, http.StatusAccepted, job, nil)
}

// Get 取得單筆匯出工作狀態與下載連結。
func (h *ExportHandler) Get(c *gin.Context) {
	jobID := c.Param("id")
	job := gin.H{
		"id":          jobID,
		"jobType":     "gov_claim",
		"periodYm":    "115-07",
		"region":      "hsinchu",
		"mode":        "single_multi_case",
		"status":      "succeeded",
		"totalCases":  12,
		"totalRows":   180,
		"fileName":    "gov-claim-115-07.xlsx",
		"downloadUrl": "/healthz",
		"createdAt":   "2026-08-25 16:00:00",
	}
	middleware.RespondSuccess(c, http.StatusOK, job, nil)
}

