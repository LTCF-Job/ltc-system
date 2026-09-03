package transport

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"ltc-system/apps/api/internal/modules/holiday/app"
	"ltc-system/apps/api/internal/platform/auth"
	"ltc-system/apps/api/internal/platform/httpx"
)

// HolidayHandler 處理國定假日與行事曆之 HTTP 請求。
type HolidayHandler struct {
	svc *app.HolidayService
}

// NewHolidayHandler 建立 HolidayHandler 實例。
func NewHolidayHandler(svc *app.HolidayService) *HolidayHandler {
	return &HolidayHandler{svc: svc}
}

// CreateHolidayRequest 定義新增假日請求結構。
type CreateHolidayRequest struct {
	HolidayDate string  `json:"holidayDate" binding:"required"` // YYYY-MM-DD
	Name        string  `json:"name" binding:"required"`
	Region      *string `json:"region"`
	Source      string  `json:"source"`
	IsDayOff    *bool   `json:"isDayOff"`
}

// ImportHolidayRequest 定義批次匯入年份行事曆請求。
type ImportHolidayRequest struct {
	Year int `json:"year" binding:"required"`
}

// List 查詢特定日期範圍之國定假日。
func (h *HolidayHandler) List(c *gin.Context) {
	startStr := c.DefaultQuery("startDate", time.Now().Format("2006-01-01"))
	endStr := c.DefaultQuery("endDate", time.Now().AddDate(1, 0, 0).Format("2006-01-01"))
	region := c.Query("region")

	start, err1 := time.Parse("2006-01-02", startStr)
	end, err2 := time.Parse("2006-01-02", endStr)
	if err1 != nil || err2 != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "日期格式錯誤，請使用 YYYY-MM-DD", nil)
		return
	}

	holidays, err := h.svc.ListHolidays(c.Request.Context(), start, end, region)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "查詢國定假日失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, holidays, nil)
}

// Create 新增或更新單一國定假日。
func (h *HolidayHandler) Create(c *gin.Context) {
	var req CreateHolidayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	date, err := time.Parse("2006-01-02", req.HolidayDate)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "日期格式錯誤", nil)
		return
	}

	source := req.Source
	if source == "" {
		source = "manual"
	}
	isDayOff := true
	if req.IsDayOff != nil {
		isDayOff = *req.IsDayOff
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	item, err := h.svc.UpsertHoliday(c.Request.Context(), app.UpsertHolidayInput{
		HolidayDate: date,
		Name:        req.Name,
		Region:      req.Region,
		Source:      source,
		IsDayOff:    isDayOff,
	}, actorID, actorRole)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "儲存國定假日失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusCreated, item, nil)
}

// Import 匯入官方標準國定假日。
func (h *HolidayHandler) Import(c *gin.Context) {
	var req ImportHolidayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Year = time.Now().Year()
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	count, err := h.svc.ImportTaiwanGovHolidays(c.Request.Context(), req.Year, actorID, actorRole)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "匯入官方假日失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{"importedCount": count, "year": req.Year}, nil)
}

// Delete 刪除指定日期假日。
func (h *HolidayHandler) Delete(c *gin.Context) {
	dateStr := c.Param("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "日期格式錯誤", nil)
		return
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	if err := h.svc.DeleteHoliday(c.Request.Context(), date, actorID, actorRole); err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "刪除假日失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{"deleted": true}, nil)
}
