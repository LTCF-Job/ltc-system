package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/service"
)

// VehicleHandler 處理車輛相關請求。
type VehicleHandler struct {
	vehicleService *service.VehicleService
}

// NewVehicleHandler 建立 VehicleHandler 實例。
func NewVehicleHandler(vehicleService *service.VehicleService) *VehicleHandler {
	return &VehicleHandler{vehicleService: vehicleService}
}

// List 查詢車輛清單。
func (h *VehicleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	region := c.Query("region")
	q := c.Query("q")

	vehicles, total, err := h.vehicleService.List(c.Request.Context(), region, q, page, pageSize)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "查詢車輛失敗", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, vehicles, middleware.PaginationMeta{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}

// Create 新增車輛。
func (h *VehicleHandler) Create(c *gin.Context) {
	var req CreateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	v, err := h.vehicleService.Create(c.Request.Context(), service.CreateVehicleInput{
		PlateNo:     req.PlateNo,
		DisplayName: req.DisplayName,
		Region:      req.Region,
		Status:      req.Status,
	})
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, v, nil)
}

// Update 更新車輛。
func (h *VehicleHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的車輛 ID", nil)
		return
	}

	var req UpdateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	v, err := h.vehicleService.Update(c.Request.Context(), id, service.UpdateVehicleInput{
		PlateNo:     req.PlateNo,
		DisplayName: req.DisplayName,
		Region:      req.Region,
		Status:      req.Status,
	})
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, v, nil)
}

// DriverHandler 處理司機相關請求。
type DriverHandler struct {
	driverService *service.DriverService
}

// NewDriverHandler 建立 DriverHandler 實例。
func NewDriverHandler(driverService *service.DriverService) *DriverHandler {
	return &DriverHandler{driverService: driverService}
}

// List 查詢司機清單。
func (h *DriverHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	region := c.Query("region")
	q := c.Query("q")

	drivers, total, err := h.driverService.List(c.Request.Context(), region, q, page, pageSize)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "查詢司機失敗", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, drivers, middleware.PaginationMeta{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}

// Create 新增司機（身分證加密與 HMAC 索引）。
func (h *DriverHandler) Create(c *gin.Context) {
	var req CreateDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	d, err := h.driverService.Create(c.Request.Context(), service.CreateDriverInput{
		Code:       req.Code,
		Name:       req.Name,
		NationalID: req.NationalID,
		Email:      req.Email,
		Region:     req.Region,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidDriverNationalID) {
			middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "身分證檢查碼錯誤", nil)
			return
		}
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, d, nil)
}

// Update 更新司機。
func (h *DriverHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的司機 ID", nil)
		return
	}

	var req UpdateDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	d, err := h.driverService.Update(c.Request.Context(), id, service.UpdateDriverInput{
		Name:   req.Name,
		Email:  req.Email,
		Region: req.Region,
		Status: req.Status,
	})
	if err != nil {
		if errors.Is(err, service.ErrDriverNotFound) {
			middleware.RespondError(c, http.StatusNotFound, middleware.CodeNotFound, "查無司機資料", nil)
			return
		}
		middleware.RespondErrorCode(c, http.StatusInternalServerError, middleware.CodeInternalError, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, d, nil)
}

// Reveal 解密司機身分證，並寫入稽核紀錄。
func (h *DriverHandler) Reveal(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的司機 ID", nil)
		return
	}

	plainID, err := h.driverService.Reveal(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrDriverNotFound) {
			middleware.RespondError(c, http.StatusNotFound, middleware.CodeNotFound, "查無司機資料", nil)
			return
		}
		middleware.RespondErrorCode(c, http.StatusInternalServerError, middleware.CodeInternalError, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{"nationalId": plainID}, nil)
}

// AssignVehicle 指派司機車輛。
func (h *DriverHandler) AssignVehicle(c *gin.Context) {
	idStr := c.Param("id")
	driverID, err := uuid.Parse(idStr)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的司機 ID", nil)
		return
	}

	var req AssignVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	assignment, err := h.driverService.AssignVehicle(c.Request.Context(), driverID, service.AssignVehicleInput{
		VehicleID:     req.VehicleID,
		IsPrimary:     req.IsPrimary,
		EffectiveFrom: req.EffectiveFrom,
		EffectiveTo:   req.EffectiveTo,
	})
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusInternalServerError, middleware.CodeInternalError, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, assignment, nil)
}
