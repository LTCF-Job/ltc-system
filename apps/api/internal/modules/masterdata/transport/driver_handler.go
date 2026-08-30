package transport

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/masterdata/app"
	"ltc-system/apps/api/internal/platform/httpx"
)

// DriverHandler 處理司機相關請求。
type DriverHandler struct {
	svc *app.DriverService
}

// NewDriverHandler 建立 DriverHandler 實例。
func NewDriverHandler(svc *app.DriverService) *DriverHandler {
	return &DriverHandler{svc: svc}
}

// List 查詢司機清單。
func (h *DriverHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	drivers, total, err := h.svc.List(c.Request.Context(), c.Query("region"), c.Query("q"), page, pageSize)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "查詢司機失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newDriverResponses(drivers), httpx.PaginationMeta{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}

// Create 新增司機（身分證加密與 HMAC 索引）。
func (h *DriverHandler) Create(c *gin.Context) {
	var req CreateDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	d, err := h.svc.Create(c.Request.Context(), app.CreateDriverInput{
		Code:       req.Code,
		Name:       req.Name,
		NationalID: req.NationalID,
		Email:      req.Email,
		Region:     req.Region,
	})
	if err != nil {
		if errors.Is(err, app.ErrInvalidDriverNationalID) {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "身分證檢查碼錯誤", nil)
			return
		}
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusCreated, newDriverResponse(*d), nil)
}

// Update 更新司機。
func (h *DriverHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "無效的司機 ID")
		return
	}

	var req UpdateDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	d, err := h.svc.Update(c.Request.Context(), id, app.UpdateDriverInput{
		Name:   req.Name,
		Email:  req.Email,
		Region: req.Region,
		Status: req.Status,
	})
	if err != nil {
		if errors.Is(err, app.ErrDriverNotFound) {
			respondNotFound(c, "查無司機資料")
			return
		}
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newDriverResponse(*d), nil)
}

// Reveal 解密司機身分證明碼。
func (h *DriverHandler) Reveal(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "無效的司機 ID")
		return
	}

	plainID, err := h.svc.Reveal(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, app.ErrDriverNotFound) {
			respondNotFound(c, "查無司機資料")
			return
		}
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{"nationalId": plainID}, nil)
}

// AssignVehicle 指派司機車輛。
func (h *DriverHandler) AssignVehicle(c *gin.Context) {
	driverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "無效的司機 ID")
		return
	}

	var req AssignVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	assignment, err := h.svc.AssignVehicle(c.Request.Context(), driverID, app.AssignVehicleInput{
		VehicleID:     req.VehicleID,
		EffectiveFrom: req.EffectiveFrom,
		EffectiveTo:   req.EffectiveTo,
	})
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusCreated, newDriverAssignmentResponse(*assignment), nil)
}
