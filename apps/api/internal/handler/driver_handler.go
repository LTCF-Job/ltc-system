package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/config"
	"ltc-system/apps/api/internal/domain/crypto"
	"ltc-system/apps/api/internal/domain/namenorm"
	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/repository"
)

// VehicleHandler 處理車輛相關請求。
type VehicleHandler struct {
	vehicleRepo *repository.VehicleRepository
}

// NewVehicleHandler 建立 VehicleHandler 實例。
func NewVehicleHandler(vehicleRepo *repository.VehicleRepository) *VehicleHandler {
	return &VehicleHandler{vehicleRepo: vehicleRepo}
}

// List 查詢車輛清單。
func (h *VehicleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	region := c.Query("region")
	q := c.Query("q")

	vehicles, total, err := h.vehicleRepo.List(c.Request.Context(), region, q, page, pageSize)
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

// DriverHandler 處理司機相關請求。
type DriverHandler struct {
	cfg        *config.Config
	driverRepo *repository.DriverRepository
}

// NewDriverHandler 建立 DriverHandler 實例。
func NewDriverHandler(cfg *config.Config, driverRepo *repository.DriverRepository) *DriverHandler {
	return &DriverHandler{cfg: cfg, driverRepo: driverRepo}
}

// List 查詢司機清單。
func (h *DriverHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	region := c.Query("region")
	q := c.Query("q")

	drivers, total, err := h.driverRepo.List(c.Request.Context(), region, q, page, pageSize)
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

// CreateDriverRequest 代表新增司機請求。
type CreateDriverRequest struct {
	Code       string  `json:"code"`
	Name       string  `json:"name" binding:"required"`
	NationalID string  `json:"nationalId" binding:"required"`
	Email      *string `json:"email"`
	Region     string  `json:"region" binding:"required"`
}

// Create 新增司機（身分證加密與 HMAC 索引）。
func (h *DriverHandler) Create(c *gin.Context) {
	var req CreateDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	if !crypto.ValidateNationalID(req.NationalID) {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "身分證檢查碼錯誤", nil)
		return
	}

	hmacIdx := crypto.Index(req.NationalID, h.cfg.HMACKey)
	cipherText, err := crypto.Encrypt(req.NationalID, h.cfg.EncryptionKey)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "加密失敗", nil)
		return
	}

	d := repository.DriverEntity{
		ID:               uuid.New(),
		Code:             req.Code,
		Name:             req.Name,
		NameNormalized:   namenorm.Normalize(req.Name),
		NationalIDCipher: cipherText,
		NationalIDHMAC:   hmacIdx,
		NationalIDMasked: crypto.Mask(req.NationalID),
		Email:            req.Email,
		Region:           req.Region,
		Status:           "active",
	}

	if err := h.driverRepo.Create(c.Request.Context(), &d); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, d, nil)
}
