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

// RegionHandler 處理區域主檔相關 HTTP 請求。
type RegionHandler struct {
	svc *app.RegionService
}

// NewRegionHandler 建立 RegionHandler 實例。
func NewRegionHandler(svc *app.RegionService) *RegionHandler {
	return &RegionHandler{svc: svc}
}

// actorOf 自請求脈絡取出稽核留痕所需的操作者資訊。
func actorOf(c *gin.Context) app.ActorContext {
	return app.ActorContext{
		ActorID:   auth.GetActorID(c),
		ActorRole: auth.GetActorRole(c),
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}
}

// List 查詢區域分頁清單，支援全部載入（all=true）。
func (h *RegionHandler) List(c *gin.Context) {
	if c.Query("all") == "true" {
		list, err := h.svc.ListAllRegions(c.Request.Context())
		if err != nil {
			httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "查詢區域清單失敗", nil)
			return
		}
		httpx.RespondSuccess(c, http.StatusOK, newRegionResponses(list), nil)
		return
	}

	page, pageSize, err := httpx.ParsePagination(c)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "分頁參數格式錯誤", nil)
		return
	}

	list, total, err := h.svc.ListRegions(c.Request.Context(), c.Query("q"), c.Query("status"), page, pageSize)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "查詢區域清單失敗", nil)
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	httpx.RespondSuccess(c, http.StatusOK, newRegionResponses(list), httpx.PaginationMeta{
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
		respondInvalidID(c, "無效的區域 ID")
		return
	}

	reg, err := h.svc.GetRegion(c.Request.Context(), id)
	if err != nil {
		respondNotFound(c, "查無此區域")
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newRegionResponse(*reg), nil)
}

// Create 新增區域主檔。
func (h *RegionHandler) Create(c *gin.Context) {
	var req CreateRegionRequest
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	reg, err := h.svc.CreateRegion(c.Request.Context(), app.CreateRegionInput{
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		SortOrder:   req.SortOrder,
	}, actorOf(c))
	if err != nil {
		if errors.Is(err, app.ErrInvalidStatus) {
			httpx.RespondError(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "status 必須為 active 或 inactive", nil)
			return
		}
		if errors.Is(err, app.ErrDuplicateRegionName) {
			httpx.RespondError(c, http.StatusConflict, httpx.CodeValidationFailed, "區域名稱已存在", nil)
			return
		}
		if errors.Is(err, app.ErrRegionNameRequired) {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "建立區域失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusCreated, newRegionResponse(*reg), nil)
}

// Update 更新區域主檔。
func (h *RegionHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "無效的區域 ID")
		return
	}

	var req UpdateRegionRequest
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	reg, err := h.svc.UpdateRegion(c.Request.Context(), id, app.UpdateRegionInput{
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		SortOrder:   req.SortOrder,
	}, actorOf(c))
	if err != nil {
		if errors.Is(err, app.ErrInvalidStatus) {
			httpx.RespondError(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "status 必須為 active 或 inactive", nil)
			return
		}
		if errors.Is(err, app.ErrRegionNotFound) {
			respondNotFound(c, "查無此區域")
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "更新區域失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusOK, newRegionResponse(*reg), nil)
}

// Delete 刪除區域。
func (h *RegionHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondInvalidID(c, "無效的區域 ID")
		return
	}

	if err := h.svc.DeleteRegion(c.Request.Context(), id, actorOf(c)); err != nil {
		if errors.Is(err, app.ErrRegionNotFound) {
			respondNotFound(c, "查無此區域")
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "刪除區域失敗", nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusNoContent, nil, nil)
}
