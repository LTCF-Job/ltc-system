package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/middleware"
	"ltc-system/apps/api/internal/repository"
)

// SiteHandler 處理據點相關請求。
type SiteHandler struct {
	siteRepo *repository.SiteRepository
}

// NewSiteHandler 建立 SiteHandler 實例。
func NewSiteHandler(siteRepo *repository.SiteRepository) *SiteHandler {
	return &SiteHandler{siteRepo: siteRepo}
}

// List 查詢據點清單。
func (h *SiteHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	region := c.Query("region")
	q := c.Query("q")

	sites, total, err := h.siteRepo.List(c.Request.Context(), region, q, page, pageSize)
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
	var req repository.SiteEntity
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, req, nil)
}

// Update 更新據點。
func (h *SiteHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	var req repository.SiteEntity
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, err.Error(), nil)
		return
	}
	req.ID = uuid.MustParse(idStr)
	middleware.RespondSuccess(c, http.StatusOK, req, nil)
}

// Delete 刪除據點。
func (h *SiteHandler) Delete(c *gin.Context) {
	middleware.RespondSuccess(c, http.StatusNoContent, nil, nil)
}

