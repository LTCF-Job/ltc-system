package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"ltc-system/apps/api/internal/platform/auth"
	"ltc-system/apps/api/internal/platform/httpx"
)

// meResponse 是 GET /auth/me 的對外回應形狀，供前端決定可見選單與可用操作。
type meResponse struct {
	ID          string                         `json:"id"`
	Email       string                         `json:"email"`
	DisplayName string                         `json:"displayName"`
	Role        string                         `json:"role"`
	DataPlane   string                         `json:"dataPlane"`
	Permissions map[string]modulePermissionDTO `json:"permissions"`
}

// MeHandler 回報目前登入者的身分與生效權限。
type MeHandler struct {
	perm       auth.PermissionResolver
	customPerm auth.CustomPermissionResolver
}

// NewMeHandler 建立 MeHandler 實例。
func NewMeHandler(perm auth.PermissionResolver, customPerm auth.CustomPermissionResolver) *MeHandler {
	return &MeHandler{perm: perm, customPerm: customPerm}
}

// Me 回傳目前登入者的身分與生效的模組權限矩陣。
func (h *MeHandler) Me(c *gin.Context) {
	actorID := auth.GetActorID(c)
	// 與 RequirePermission 共用 ResolveEffectivePermissions，前端據以隱藏的操作與 API 實際放行的範圍才會一致
	effective, err := auth.ResolveEffectivePermissions(c.Request.Context(), h.perm, h.customPerm, auth.GetActorRole(c), actorID)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, httpx.CodeInternalError, "無法解析權限", nil)
		return
	}

	perms := make(map[string]modulePermissionDTO, len(effective))
	for k, v := range effective {
		perms[k] = modulePermissionDTO{View: v.View, Edit: v.Edit, Delete: v.Delete}
	}

	httpx.RespondSuccess(c, http.StatusOK, meResponse{
		ID:          actorID.String(),
		Email:       auth.GetActorEmail(c),
		DisplayName: auth.GetActorName(c),
		Role:        auth.GetActorRole(c),
		DataPlane:   auth.GetActorDataPlane(c),
		Permissions: perms,
	}, nil)
}
