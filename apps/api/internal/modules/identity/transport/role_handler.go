package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/identity/app"
	"ltc-system/apps/api/internal/platform/auth"
	"ltc-system/apps/api/internal/platform/httpx"
)

// RoleHandler 處理角色身分相關之 HTTP 請求。
type RoleHandler struct {
	svc *app.RoleService
}

// NewRoleHandler 建立 RoleHandler 實例。
func NewRoleHandler(svc *app.RoleService) *RoleHandler {
	return &RoleHandler{svc: svc}
}

// ListRoles 取得角色清單。
func (h *RoleHandler) ListRoles(c *gin.Context) {
	roles, err := h.svc.List(c.Request.Context())
	if err != nil {
		respondIdentityError(c, err)
		return
	}
	list := make([]roleResponse, 0, len(roles))
	for _, r := range roles {
		list = append(list, toRoleResponse(r))
	}
	httpx.RespondSuccess(c, http.StatusOK, list, nil)
}

// GetRole 取得單一角色。
func (h *RoleHandler) GetRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的角色 ID", nil)
		return
	}
	role, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		respondIdentityError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, toRoleResponse(*role), nil)
}

// CreateRole 新增自訂角色。
func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req createRoleRequest
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	role, err := h.svc.Create(c.Request.Context(), app.CreateRoleInput{
		Key:         req.Key,
		Name:        req.Name,
		Description: req.Description,
		TagType:     req.TagType,
		Permissions: toPermissionsModel(req.Permissions),
	}, auth.GetActorID(c), auth.GetActorRole(c))
	if err != nil {
		respondIdentityError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusCreated, toRoleResponse(*role), nil)
}

// UpdateRole 修改自訂角色。
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的角色 ID", nil)
		return
	}

	var req updateRoleRequest
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	role, err := h.svc.Update(c.Request.Context(), id, app.UpdateRoleInput{
		Name:        req.Name,
		Description: req.Description,
		TagType:     req.TagType,
		Permissions: toPermissionsModel(req.Permissions),
	}, auth.GetActorID(c), auth.GetActorRole(c))
	if err != nil {
		respondIdentityError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, toRoleResponse(*role), nil)
}

// DeleteRole 刪除自訂角色。
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的角色 ID", nil)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id, auth.GetActorID(c), auth.GetActorRole(c)); err != nil {
		respondIdentityError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, gin.H{"deleted": true}, nil)
}
