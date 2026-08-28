package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/service"
)

// SiteHandler 處理據點相關請求。
type SiteHandler struct {
	siteService *service.SiteService
}

// NewSiteHandler 建立 SiteHandler 實例。
func NewSiteHandler(siteService *service.SiteService) *SiteHandler {
	return &SiteHandler{siteService: siteService}
}

// List 查詢據點清單。
func (h *SiteHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	region := c.Query("region")
	q := c.Query("q")

	sites, total, err := h.siteService.List(c.Request.Context(), region, q, page, pageSize)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "查詢據點失敗", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, sites, middleware.PaginationMeta{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}

// Create 新增據點。
func (h *SiteHandler) Create(c *gin.Context) {
	var req CreateSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	site, err := h.siteService.Create(c.Request.Context(), service.CreateSiteInput{
		Code:     req.Code,
		Name:     req.Name,
		Address:  req.Address,
		Region:   req.Region,
		OpenDays: req.OpenDays,
		Status:   req.Status,
	})
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, site, nil)
}

// Update 更新據點。
func (h *SiteHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的據點 ID", nil)
		return
	}

	var req UpdateSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	site, err := h.siteService.Update(c.Request.Context(), id, service.UpdateSiteInput{
		Code:     req.Code,
		Name:     req.Name,
		Address:  req.Address,
		Region:   req.Region,
		OpenDays: req.OpenDays,
		Status:   req.Status,
	})
	if err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, site, nil)
}

// Delete 刪除據點。
func (h *SiteHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的據點 ID", nil)
		return
	}

	if err := h.siteService.Delete(c.Request.Context(), id); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusNoContent, nil, nil)
}
