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
	page, pageSize, err := httpx.ParsePagination(c)
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "分頁參數格式錯誤", nil)
		return
	}

	sites, total, err := h.svc.List(c.Request.Context(), c.Query("region"), c.Query("q"), c.Query("status"), page, pageSize)
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
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		details := httpx.ExtractValidationDetails(err)
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, details)
		return
	}

	site, err := h.svc.Create(c.Request.Context(), app.CreateSiteInput{
		Name:    req.Name,
		Address: req.Address,
		Region:  req.Region,
		Status:  req.Status,
		Remarks: req.Remarks,
	}, app.ActorContext{
		ActorID:   auth.GetActorID(c),
		ActorRole: auth.GetActorRole(c),
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		if errors.Is(err, app.ErrInvalidStatus) {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "狀態設定不正確，僅接受「啟用」或「停用」", nil)
			return
		}
		if errors.Is(err, app.ErrSiteNameRequired) {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "name", Reason: "請輸入據點名稱"},
			})
			return
		}
		if errors.Is(err, app.ErrSiteAddressRequired) {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "address", Reason: "請輸入據點地址"},
			})
			return
		}
		if errors.Is(err, app.ErrDuplicateSiteName) {
			httpx.RespondErrorCode(c, http.StatusConflict, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "name", Reason: "該區域已存在相同名稱的據點"},
			})
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "建立據點失敗", nil)
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
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		details := httpx.ExtractValidationDetails(err)
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, details)
		return
	}

	site, err := h.svc.Update(c.Request.Context(), id, app.UpdateSiteInput{
		Name:    req.Name,
		Address: req.Address,
		Region:  req.Region,
		Status:  req.Status,
		Remarks: req.Remarks,
	}, app.ActorContext{
		ActorID:   auth.GetActorID(c),
		ActorRole: auth.GetActorRole(c),
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		if errors.Is(err, app.ErrInvalidStatus) {
			httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "狀態設定不正確，僅接受「啟用」或「停用」", nil)
			return
		}
		if errors.Is(err, app.ErrSiteNotFound) {
			respondNotFound(c, "查無此據點")
			return
		}
		if errors.Is(err, app.ErrSiteNameRequired) {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "name", Reason: "請輸入據點名稱"},
			})
			return
		}
		if errors.Is(err, app.ErrSiteAddressRequired) {
			httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "address", Reason: "請輸入據點地址"},
			})
			return
		}
		if errors.Is(err, app.ErrDuplicateSiteName) {
			httpx.RespondErrorCode(c, http.StatusConflict, httpx.CodeValidationFailed, err, []httpx.ErrorDetail{
				{Field: "name", Reason: "該區域已存在相同名稱的據點"},
			})
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "更新據點失敗", nil)
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

	if err := h.svc.Delete(c.Request.Context(), id, app.ActorContext{
		ActorID:   auth.GetActorID(c),
		ActorRole: auth.GetActorRole(c),
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}); err != nil {
		if errors.Is(err, app.ErrSiteNotFound) {
			respondNotFound(c, "查無此據點")
			return
		}
		if errors.Is(err, app.ErrSiteInUse) {
			httpx.RespondError(c, http.StatusConflict, httpx.CodeResourceInUse, "該據點仍有相關資料參照，請先解除關聯後再刪除", nil)
			return
		}
		httpx.RespondErrorCode(c, http.StatusInternalServerError, httpx.CodeInternalError, err, nil)
		return
	}

	httpx.RespondSuccess(c, http.StatusNoContent, nil, nil)
}
