package transport

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/masterdata/app"
	"ltc-system/apps/api/internal/platform/httpx"
)

// SiteHandler 處理據點相關請求。
type SiteHandler struct {
	svc *app.SiteService
}

// NewSiteHandler 建立 SiteHandler 實例。
func NewSiteHandler(svc *app.SiteService) *SiteHandler {
	return &SiteHandler{svc: svc}
}

// List 查詢據點清單。
func (h *SiteHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	sites, total, err := h.svc.List(c.Request.Context(), c.Query("region"), c.Query("q"), page, pageSize)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "查詢據點失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newSiteResponses(sites), httpx.PaginationMeta{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}

// Create 新增據點。
func (h *SiteHandler) Create(c *gin.Context) {
	var req CreateSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	site, err := h.svc.Create(c.Request.Context(), app.CreateSiteInput{
		Code:     req.Code,
		Name:     req.Name,
		Address:  req.Address,
		Region:   req.Region,
		OpenDays: req.OpenDays,
		Status:   req.Status,
	})
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusCreated, newSiteResponse(*site), nil)
}

// Update 更新據點。
func (h *SiteHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "無效的據點 ID")
		return
	}

	var req UpdateSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	site, err := h.svc.Update(c.Request.Context(), id, app.UpdateSiteInput{
		Code:     req.Code,
		Name:     req.Name,
		Address:  req.Address,
		Region:   req.Region,
		OpenDays: req.OpenDays,
		Status:   req.Status,
	})
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newSiteResponse(*site), nil)
}

// Delete 刪除據點。
func (h *SiteHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "無效的據點 ID")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusNoContent, nil, nil)
}
