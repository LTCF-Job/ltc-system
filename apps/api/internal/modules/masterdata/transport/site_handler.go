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

// SiteHandler 處理單位相關請求。
type SiteHandler struct {
	svc *app.SiteService
}

// NewSiteHandler 建立 SiteHandler 實例。
func NewSiteHandler(svc *app.SiteService) *SiteHandler {
	return &SiteHandler{svc: svc}
}

// List 查詢單位清單。
func (h *SiteHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	sites, total, err := h.svc.List(c.Request.Context(), c.Query("region"), c.Query("q"), page, pageSize)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "查詢單位失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newSiteResponses(sites), httpx.PaginationMeta{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}

// Create 新增單位。
func (h *SiteHandler) Create(c *gin.Context) {
	var req CreateSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		details := httpx.ExtractValidationDetails(err)
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, details)
		return
	}

	site, err := h.svc.Create(c.Request.Context(), app.CreateSiteInput{
		Name:     req.Name,
		Address:  req.Address,
		Region:   req.Region,
		OpenDays: req.OpenDays,
		Status:   req.Status,
	})
	if err != nil {
		if errors.Is(err, app.ErrSiteNameRequired) {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "name", Reason: "請輸入單位名稱"},
			})
			return
		}
		if errors.Is(err, app.ErrSiteAddressRequired) {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "address", Reason: "請輸入單位地址"},
			})
			return
		}
		if errors.Is(err, app.ErrSiteRegionRequired) {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "region", Reason: "請選擇所屬區域"},
			})
			return
		}
		if errors.Is(err, app.ErrDuplicateSiteName) {
			httpx.RespondErrorCode(c, http.StatusConflict, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "name", Reason: "該區域已存在相同名稱的單位"},
			})
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "建立單位失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusCreated, newSiteResponse(*site), nil)
}

// Update 更新單位。
func (h *SiteHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "無效的單位 ID")
		return
	}

	var req UpdateSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		details := httpx.ExtractValidationDetails(err)
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, details)
		return
	}

	site, err := h.svc.Update(c.Request.Context(), id, app.UpdateSiteInput{
		Name:     req.Name,
		Address:  req.Address,
		Region:   req.Region,
		OpenDays: req.OpenDays,
		Status:   req.Status,
	})
	if err != nil {
		if errors.Is(err, app.ErrSiteNotFound) {
			respondNotFound(c, "查無此單位")
			return
		}
		if errors.Is(err, app.ErrSiteNameRequired) {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "name", Reason: "請輸入單位名稱"},
			})
			return
		}
		if errors.Is(err, app.ErrSiteAddressRequired) {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "address", Reason: "請輸入單位地址"},
			})
			return
		}
		if errors.Is(err, app.ErrSiteRegionRequired) {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "region", Reason: "請選擇所屬區域"},
			})
			return
		}
		if errors.Is(err, app.ErrDuplicateSiteName) {
			httpx.RespondErrorCode(c, http.StatusConflict, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "name", Reason: "該區域已存在相同名稱的單位"},
			})
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "更新單位失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newSiteResponse(*site), nil)
}

// Delete 刪除單位。
func (h *SiteHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "無效的單位 ID")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
			{Field: "id", Reason: "該單位仍有相關資料參照，無法刪除"},
		})
		return
	}

	httpx.RespondSuccess(c, http.StatusNoContent, nil, nil)
}
