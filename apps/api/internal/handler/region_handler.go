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

// RegionHandler 處理區域主檔相關 HTTP 請求。
type RegionHandler struct {
	svc *service.RegionService
}

// NewRegionHandler 建立 RegionHandler 實例。
func NewRegionHandler(svc *service.RegionService) *RegionHandler {
	return &RegionHandler{svc: svc}
}

// List 查詢區域分頁清單，支援全部載入（all=true）。
func (h *RegionHandler) List(c *gin.Context) {
	if c.Query("all") == "true" {
		list, err := h.svc.ListAllRegions(c.Request.Context())
		if err != nil {
			middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "查詢區域清單失敗", nil)
			return
		}
		middleware.RespondSuccess(c, http.StatusOK, list, nil)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	q := c.Query("q")
	status := c.Query("status")

	list, total, err := h.svc.ListRegions(c.Request.Context(), q, status, page, pageSize)
	if err != nil {
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "查詢區域清單失敗", nil)
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	middleware.RespondSuccess(c, http.StatusOK, list, middleware.PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

// Get 依 ID 取得單一區域。
func (h *RegionHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的區域 ID", nil)
		return
	}

	reg, err := h.svc.GetRegion(c.Request.Context(), id)
	if err != nil {
		middleware.RespondError(c, http.StatusNotFound, middleware.CodeNotFound, "查無此區域", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, reg, nil)
}

// Create 新增區域主檔。
func (h *RegionHandler) Create(c *gin.Context) {
	var req service.CreateRegionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)
	reg, err := h.svc.CreateRegion(c.Request.Context(), req, actorID, actorRole, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, service.ErrDuplicateRegionName) {
			middleware.RespondError(c, http.StatusConflict, middleware.CodeValidationFailed, "區域名稱已存在", nil)
			return
		}
		if errors.Is(err, service.ErrRegionNameRequired) {
			middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
			return
		}
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "建立區域失敗", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusCreated, reg, nil)
}

// Update 更新區域主檔。
func (h *RegionHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的區域 ID", nil)
		return
	}

	var req service.UpdateRegionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.RespondErrorCode(c, http.StatusBadRequest, middleware.CodeValidationFailed, err, nil)
		return
	}

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)
	reg, err := h.svc.UpdateRegion(c.Request.Context(), id, req, actorID, actorRole, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, service.ErrRegionNotFound) {
			middleware.RespondError(c, http.StatusNotFound, middleware.CodeNotFound, "查無此區域", nil)
			return
		}
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "更新區域失敗", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusOK, reg, nil)
}

// Delete 刪除區域。
func (h *RegionHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		middleware.RespondError(c, http.StatusBadRequest, middleware.CodeValidationFailed, "無效的區域 ID", nil)
		return
	}

	actorID := middleware.GetActorID(c)
	actorRole := middleware.GetActorRole(c)
	if err := h.svc.DeleteRegion(c.Request.Context(), id, actorID, actorRole, c.ClientIP(), c.Request.UserAgent()); err != nil {
		if errors.Is(err, service.ErrRegionNotFound) {
			middleware.RespondError(c, http.StatusNotFound, middleware.CodeNotFound, "查無此區域", nil)
			return
		}
		middleware.RespondError(c, http.StatusInternalServerError, middleware.CodeInternalError, "刪除區域失敗", nil)
		return
	}

	middleware.RespondSuccess(c, http.StatusNoContent, nil, nil)
}
