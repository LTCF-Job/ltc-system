package transport

import (
	"net/http"
	"strconv"

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

	vehicles, total, err := h.svc.List(c.Request.Context(), c.Query("region"), c.Query("q"), page, pageSize)
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

	v, err := h.svc.Create(c.Request.Context(), app.CreateVehicleInput{
		PlateNo:     req.PlateNo,
		DisplayName: req.DisplayName,
		Region:      req.Region,
		Status:      req.Status,
	})
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

	v, err := h.svc.Update(c.Request.Context(), id, app.UpdateVehicleInput{
		PlateNo:     req.PlateNo,
		DisplayName: req.DisplayName,
		Region:      req.Region,
		Status:      req.Status,
	})
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newVehicleResponse(*v), nil)
}
