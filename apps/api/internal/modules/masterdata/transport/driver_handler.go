package transport

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/masterdata/app"
	"ltc-system/apps/api/internal/platform/auth"
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
	page, pageSize, err := httpx.ParsePagination(c)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "分頁參數格式錯誤", nil)
		return
	}

	drivers, total, err := h.svc.List(c.Request.Context(), c.Query("region"), c.Query("q"), c.Query("status"), page, pageSize)
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
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	d, err := h.svc.Create(c.Request.Context(), app.CreateDriverInput{
		Name:              req.Name,
		NationalID:        req.NationalID,
		Email:             req.Email,
		Region:            req.Region,
		LicenseClass:      req.LicenseClass,
		LicenseExpiryDate: req.LicenseExpiryDate.toTimePtr(),
	})
	if err != nil {
		if errors.Is(err, app.ErrDriverNameRequired) {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "司機姓名不可為空白", nil)
			return
		}
		if errors.Is(err, app.ErrInvalidDriverNationalID) {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "身分證檢查碼錯誤", nil)
			return
		}
		if errors.Is(err, app.ErrInvalidDriverLicenseClass) {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "駕照類別不正確", nil)
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
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	d, err := h.svc.Update(c.Request.Context(), id, app.UpdateDriverInput{
		Name:                   req.Name,
		Email:                  req.Email,
		Region:                 req.Region,
		Status:                 req.Status,
		LicenseClass:           req.LicenseClass,
		LicenseExpiryDate:      req.LicenseExpiryDate.Value,
		ClearLicenseExpiryDate: req.LicenseExpiryDate.Present && req.LicenseExpiryDate.Value == nil,
	})
	if err != nil {
		if errors.Is(err, app.ErrDriverNameRequired) {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "司機姓名不可為空白", nil)
			return
		}
		if errors.Is(err, app.ErrDriverNotFound) {
			respondNotFound(c, "查無司機資料")
			return
		}
		if errors.Is(err, app.ErrInvalidStatus) {
			httpx.RespondError(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "status 必須為 active 或 inactive", nil)
			return
		}
		if errors.Is(err, app.ErrInvalidDriverLicenseClass) {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "駕照類別不正確", nil)
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

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)
	plainID, err := h.svc.Reveal(c.Request.Context(), id, actorID, actorRole, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, app.ErrDriverNotFound) {
			respondNotFound(c, "查無司機資料")
			return
		}
		if errors.Is(err, app.ErrNationalIDNotConfigured) {
			httpx.RespondError(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "司機尚未設定身分證資料", nil)
			return
		}
		if errors.Is(err, app.ErrRevealAuditUnavailable) {
			httpx.RespondErrorCode(c, http.StatusServiceUnavailable, httpx.CodeServiceUnavailable, err, nil)
			return
		}
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, gin.H{"nationalId": plainID}, nil)
}

// Delete 軟刪除司機。
func (h *DriverHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "無效的司機 ID")
		return
	}

	actorID := auth.GetActorID(c)
	actorRole := auth.GetActorRole(c)

	if err := h.svc.Delete(c.Request.Context(), id, actorID, actorRole); err != nil {
		if errors.Is(err, app.ErrDriverNotFound) {
			respondNotFound(c, "查無司機資料")
			return
		}
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	c.Status(http.StatusNoContent)
}

// AssignVehicle 指派司機車輛。
func (h *DriverHandler) AssignVehicle(c *gin.Context) {
	driverID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "無效的司機 ID")
		return
	}

	var req AssignVehicleRequest
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	assignment, err := h.svc.AssignVehicle(c.Request.Context(), driverID, app.AssignVehicleInput{
		VehicleID:     req.VehicleID,
		EffectiveFrom: req.EffectiveFrom.toTime(),
		EffectiveTo:   req.EffectiveTo.toTimePtr(),
	})
	if err != nil {
		if errors.Is(err, app.ErrInvalidAssignmentRange) {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "司機指派日期區間無效", nil)
			return
		}
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusCreated, newDriverAssignmentResponse(*assignment), nil)
}
