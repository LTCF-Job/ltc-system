package transport

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/identity/app"
	"ltc-system/apps/api/internal/platform/auth"
	"ltc-system/apps/api/internal/platform/httpx"
)

// IdentityHandler 處理使用者帳號管理與密碼變更相關之 HTTP 請求。
type IdentityHandler struct {
	svc *app.UserService
}

// NewIdentityHandler 建立 IdentityHandler 實例。
func NewIdentityHandler(svc *app.UserService) *IdentityHandler {
	return &IdentityHandler{svc: svc}
}

// ListUsers 取得使用者清單。
func (h *IdentityHandler) ListUsers(c *gin.Context) {
	users, err := h.svc.List(c.Request.Context(), c.Query("keyword"), c.Query("role"))
	if err != nil {
		respondIdentityError(c, err)
		return
	}
	list := make([]userResponse, 0, len(users))
	for _, u := range users {
		list = append(list, toUserResponse(u))
	}
	httpx.RespondSuccess(c, http.StatusOK, list, nil)
}

// GetUser 取得單一使用者。
func (h *IdentityHandler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的使用者 ID", nil)
		return
	}
	u, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		respondIdentityError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, toUserResponse(*u), nil)
}

// CreateUser 建立新使用者。
func (h *IdentityHandler) CreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	u, err := h.svc.Create(c.Request.Context(), app.CreateAuthUserInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
		Phone:       req.Phone,
		RoleKey:     req.Role,
	}, auth.GetActorID(c), auth.GetActorRole(c))
	if err != nil {
		respondIdentityError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusCreated, toUserResponse(*u), nil)
}

// UpdateUser 更新使用者基本資料與角色。
func (h *IdentityHandler) UpdateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的使用者 ID", nil)
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	u, err := h.svc.Update(c.Request.Context(), id, app.UpdateAuthUserInput{
		DisplayName: req.DisplayName,
		Phone:       req.Phone,
		RoleKey:     req.Role,
		Status:      req.Status,
	}, auth.GetActorID(c), auth.GetActorRole(c))
	if err != nil {
		respondIdentityError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, toUserResponse(*u), nil)
}

// UpdateUserPermissions 覆寫使用者個人自訂權限。
func (h *IdentityHandler) UpdateUserPermissions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的使用者 ID", nil)
		return
	}

	var req updateUserPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	if err := h.svc.UpdatePermissions(c.Request.Context(), id, toPermissionsModel(req.CustomPermissions), auth.GetActorID(c), auth.GetActorRole(c)); err != nil {
		respondIdentityError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, gin.H{"updated": true}, nil)
}

// DeleteUser 刪除使用者。
func (h *IdentityHandler) DeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的使用者 ID", nil)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id, auth.GetActorID(c), auth.GetActorRole(c)); err != nil {
		respondIdentityError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, gin.H{"deleted": true}, nil)
}

// ChangeSelfPassword 讓已登入使用者變更自己的密碼。
func (h *IdentityHandler) ChangeSelfPassword(c *gin.Context) {
	var req changeSelfPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}

	actorID := auth.GetActorID(c)
	email := auth.GetActorEmail(c)

	if err := h.svc.ChangeSelfPassword(c.Request.Context(), actorID, email, req.OldPassword, req.NewPassword); err != nil {
		respondIdentityError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, gin.H{"changed": true}, nil)
}
