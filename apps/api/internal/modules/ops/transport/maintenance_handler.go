package transport

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/ops/app"
	"ltc-system/apps/api/internal/platform/auth"
	"ltc-system/apps/api/internal/platform/httpx"
)

// MaintenanceHandler 處理車輛維修保養 API 請求。
type MaintenanceHandler struct {
	maintenanceSvc *app.MaintenanceService
}

// NewMaintenanceHandler 建立 MaintenanceHandler 實例。
func NewMaintenanceHandler(maintenanceSvc *app.MaintenanceService) *MaintenanceHandler {
	return &MaintenanceHandler{maintenanceSvc: maintenanceSvc}
}

// List 查詢維修保養清單。
func (h *MaintenanceHandler) List(c *gin.Context) {
	page, pageSize, err := httpx.ParsePagination(c)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	var vehicleID *uuid.UUID
	if vIDStr := c.Query("vehicleId"); vIDStr != "" {
		id, parseErr := uuid.Parse(vIDStr)
		if parseErr != nil {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, parseErr, nil)
			return
		}
		vehicleID = &id
	}

	var startDate, endDate *time.Time
	if startStr := c.Query("startDate"); startStr != "" {
		t, parseErr := time.Parse("2006-01-02", startStr)
		if parseErr != nil {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, parseErr, nil)
			return
		}
		startDate = &t
	}
	if endStr := c.Query("endDate"); endStr != "" {
		t, parseErr := time.Parse("2006-01-02", endStr)
		if parseErr != nil {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, parseErr, nil)
			return
		}
		endDate = &t
	}
	if startDate != nil && endDate != nil && endDate.Before(*startDate) {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, errInvalidDateRange, nil)
		return
	}

	q := c.Query("q")

	list, total, err := h.maintenanceSvc.List(c.Request.Context(), page, pageSize, vehicleID, startDate, endDate, q)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	totalPages := (total + pageSize - 1) / pageSize
	httpx.RespondSuccess(c, http.StatusOK, newMaintenanceLogResponses(list), &httpx.PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      int64(total),
		TotalPages: totalPages,
	})
}

// Create 新增維修保養紀錄。
func (h *MaintenanceHandler) Create(c *gin.Context) {
	var req struct {
		VehicleID   uuid.UUID `json:"vehicleId" binding:"required"`
		ServiceDate string    `json:"serviceDate" binding:"required"`
		Mileage     float64   `json:"mileage" binding:"required,gte=0"`
		Items       string    `json:"items" binding:"required"`
		Vendor      *string   `json:"vendor"`
		Cost        float64   `json:"cost" binding:"gte=0"`
		ReceiptURL  *string   `json:"receiptUrl"`
		Note        *string   `json:"note"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	svcDate, err := time.Parse("2006-01-02", req.ServiceDate)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "保養日期格式必須為 YYYY-MM-DD", nil)
		return
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	item, err := h.maintenanceSvc.Create(c.Request.Context(), app.MaintenanceLogInput{
		VehicleID:   req.VehicleID,
		ServiceDate: svcDate,
		Mileage:     req.Mileage,
		Items:       req.Items,
		Vendor:      req.Vendor,
		Cost:        req.Cost,
		ReceiptURL:  req.ReceiptURL,
		Note:        req.Note,
		CreatedBy:   actorID,
	}, &actorID, &actorRole, auditContext(c))
	if err != nil {
		if errors.Is(err, app.ErrInvalidReceiptURL) {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "收據連結必須是 http 或 https 開頭的網址", nil)
			return
		}
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusCreated, newMaintenanceLogResponse(*item), nil)
}

// Update 修改維修保養紀錄。
func (h *MaintenanceHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的紀錄 ID", nil)
		return
	}

	var req struct {
		VehicleID   uuid.UUID `json:"vehicleId" binding:"required"`
		ServiceDate string    `json:"serviceDate" binding:"required"`
		Mileage     float64   `json:"mileage" binding:"required,gte=0"`
		Items       string    `json:"items" binding:"required"`
		Vendor      *string   `json:"vendor"`
		Cost        float64   `json:"cost" binding:"gte=0"`
		ReceiptURL  *string   `json:"receiptUrl"`
		Note        *string   `json:"note"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	svcDate, err := time.Parse("2006-01-02", req.ServiceDate)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "保養日期格式必須為 YYYY-MM-DD", nil)
		return
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	item, err := h.maintenanceSvc.Update(c.Request.Context(), id, app.MaintenanceLogInput{
		VehicleID:   req.VehicleID,
		ServiceDate: svcDate,
		Mileage:     req.Mileage,
		Items:       req.Items,
		Vendor:      req.Vendor,
		Cost:        req.Cost,
		ReceiptURL:  req.ReceiptURL,
		Note:        req.Note,
	}, &actorID, &actorRole, auditContext(c))
	if err != nil {
		if errors.Is(err, app.ErrInvalidReceiptURL) {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "收據連結必須是 http 或 https 開頭的網址", nil)
			return
		}
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newMaintenanceLogResponse(*item), nil)
}

// Delete 刪除維修保養紀錄。
func (h *MaintenanceHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的紀錄 ID", nil)
		return
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)
	if err := h.maintenanceSvc.Delete(c.Request.Context(), id, &actorID, &actorRole, auditContext(c)); err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{"success": true}, nil)
}

// DownloadBlankTemplate 下載 15 車空白維修保養檢查表格 Excel。
func (h *MaintenanceHandler) DownloadBlankTemplate(c *gin.Context) {
	excelBytes, err := h.maintenanceSvc.GenerateBlankMaintenanceExcel(c.Request.Context())
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	fileName := "maintenance-blank.xlsx"
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", excelBytes)
}
