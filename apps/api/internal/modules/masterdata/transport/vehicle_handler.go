package transport

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/masterdata/app"
	"ltc-system/apps/api/internal/platform/httpx"
)

// VehicleHandler 處理車輛相關請求。
type VehicleHandler struct {
	svc *app.VehicleService
}

// NewVehicleHandler 建立 VehicleHandler 實例。
func NewVehicleHandler(svc *app.VehicleService) *VehicleHandler {
	return &VehicleHandler{svc: svc}
}

// List 查詢車輛清單。
func (h *VehicleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	filter := app.VehicleFilter{Region: c.Query("region"), Q: c.Query("q")}
	if raw := c.Query("siteId"); raw != "" {
		siteID, err := uuid.Parse(raw)
		if err != nil {
			respondInvalidID(c, "無效的單位 ID")
			return
		}
		filter.SiteID = &siteID
	}

	vehicles, total, err := h.svc.List(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "查詢車輛失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newVehicleResponses(vehicles), httpx.PaginationMeta{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}

// Create 新增車輛。
func (h *VehicleHandler) Create(c *gin.Context) {
	var req CreateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	v, err := h.svc.Create(c.Request.Context(), req.toInput())
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusCreated, newVehicleResponse(*v), nil)
}

// Update 更新車輛。
func (h *VehicleHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "無效的車輛 ID")
		return
	}

	var req UpdateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	v, err := h.svc.Update(c.Request.Context(), id, req.toInput())
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newVehicleResponse(*v), nil)
}

// SetDrivers 整批設定車輛目前的司機。一位司機同期只會有一台車，被指派到本車時
// 其他車上尚未結束的指派會一併收掉。
func (h *VehicleHandler) SetDrivers(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "無效的車輛 ID")
		return
	}

	var req SetVehicleDriversRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	effectiveFrom := time.Now()
	if req.EffectiveFrom != nil {
		effectiveFrom = *req.EffectiveFrom
	}

	if err := h.svc.SetDrivers(c.Request.Context(), id, req.DriverIDs, effectiveFrom); err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{"vehicleId": id, "driverIds": req.DriverIDs}, nil)
}
