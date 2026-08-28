package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/service"
)

// FuelHandler 處理車輛油資 API 請求。
type FuelHandler struct {
	fuelSvc *service.FuelService
}

// NewFuelHandler 建立 FuelHandler 實例。
func NewFuelHandler(fuelSvc *service.FuelService) *FuelHandler {
	return &FuelHandler{fuelSvc: fuelSvc}
}

// List 查詢油資清單。
func (h *FuelHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	var vehicleID, driverID *uuid.UUID
	if vIDStr := c.Query("vehicleId"); vIDStr != "" {
		if id, err := uuid.Parse(vIDStr); err == nil {
			vehicleID = &id
		}
	}
	if dIDStr := c.Query("driverId"); dIDStr != "" {
		if id, err := uuid.Parse(dIDStr); err == nil {
			driverID = &id
		}
	}

	var startDate, endDate *time.Time
	if startStr := c.Query("startDate"); startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = &t
		}
	}
	if endStr := c.Query("endDate"); endStr != "" {
		if t, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = &t
		}
	}

	q := c.Query("q")

	list, total, err := h.fuelSvc.List(c.Request.Context(), page, pageSize, vehicleID, driverID, startDate, endDate, q)
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusInternalServerError, middleware.CodeInternalError, err, nil)
		return
	}

	totalPages := (total + pageSize - 1) / pageSize
	middleware.RespondSuccess(c, http.StatusOK, list, &middleware.PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      int64(total),
		TotalPages: totalPages,
	})
}

// Create 新增油資紀錄。
func (h *FuelHandler) Create(c *gin.Context) {
	var req struct {
		VehicleID  uuid.UUID  `json:"vehicleId" binding:"required"`
		DriverID   *uuid.UUID `json:"driverId"`
		FuelDate   string     `json:"fuelDate" binding:"required"`
		Liters     float64    `json:"liters" binding:"required,gt=0"`
		Cost       float64    `json:"cost" binding:"required,gt=0"`
		ReceiptURL *string    `json:"receiptUrl"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	fuelDate, err := time.Parse("2006-01-02", req.FuelDate)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "加油日期格式必須為 YYYY-MM-DD", nil)
		return
	}

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)

	item, err := h.fuelSvc.Create(c.Request.Context(), service.FuelLogInput{
		VehicleID:  req.VehicleID,
		DriverID:   req.DriverID,
		FuelDate:   fuelDate,
		Liters:     req.Liters,
		Cost:       req.Cost,
		ReceiptURL: req.ReceiptURL,
		CreatedBy:  actorID,
	}, &actorID, &actorRole)
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusInternalServerError, middleware.CodeInternalError, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, item, nil)
}

// Update 修改油資紀錄。
func (h *FuelHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的紀錄 ID", nil)
		return
	}

	var req struct {
		VehicleID  uuid.UUID  `json:"vehicleId" binding:"required"`
		DriverID   *uuid.UUID `json:"driverId"`
		FuelDate   string     `json:"fuelDate" binding:"required"`
		Liters     float64    `json:"liters" binding:"required,gt=0"`
		Cost       float64    `json:"cost" binding:"required,gt=0"`
		ReceiptURL *string    `json:"receiptUrl"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	fuelDate, err := time.Parse("2006-01-02", req.FuelDate)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "加油日期格式必須為 YYYY-MM-DD", nil)
		return
	}

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)

	item, err := h.fuelSvc.Update(c.Request.Context(), id, service.FuelLogInput{
		VehicleID:  req.VehicleID,
		DriverID:   req.DriverID,
		FuelDate:   fuelDate,
		Liters:     req.Liters,
		Cost:       req.Cost,
		ReceiptURL: req.ReceiptURL,
	}, &actorID, &actorRole)
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusInternalServerError, middleware.CodeInternalError, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, item, nil)
}

// Delete 刪除油資紀錄。
func (h *FuelHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的紀錄 ID", nil)
		return
	}

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)
	if err := h.fuelSvc.Delete(c.Request.Context(), id, &actorID, &actorRole); err != nil {
		middleware.RespondErrorCode(c, http.StatusInternalServerError, middleware.CodeInternalError, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, gin.H{"success": true}, nil)
}
