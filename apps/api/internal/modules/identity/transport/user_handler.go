package transport

import (
	"errors"
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
	page, pageSize, err := httpx.ParsePagination(c)
	if err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, nil)
		return
	}
	users, total, err := h.svc.List(c.Request.Context(), c.Query("q"), c.Query("role"), page, pageSize)
	if err != nil {
		respondIdentityError(c, err)
		return
	}
	list := make([]userResponse, 0, len(users))
	for _, u := range users {
		list = append(list, toUserResponse(u))
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}
	httpx.RespondSuccess(c, http.StatusOK, list, httpx.PaginationMeta{Page: page, PageSize: pageSize, Total: int64(total), TotalPages: totalPages})
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
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	u, err := h.svc.Create(c.Request.Context(), app.CreateAuthUserInput{
		Email:             req.Email,
		Password:          req.Password,
		DisplayName:       req.DisplayName,
		Phone:             req.Phone,
		RoleKey:           req.Role,
		Status:            req.Status,
		CustomPermissions: toPermissionsModel(req.CustomPermissions),
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
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
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
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
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
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	actorID := auth.GetActorID(c)
	email := auth.GetActorEmail(c)

	if err := h.svc.ChangeSelfPassword(c.Request.Context(), actorID, email, req.OldPassword, req.NewPassword, auth.GetActorRole(c)); err != nil {
		// ErrInvalidCredentials 在這條「使用者自行修改密碼」路徑代表舊密碼輸入錯誤，屬於可由使用者
		// 修正的輸入問題；respondIdentityError 對此錯誤的預設映射是 401 UNAUTHENTICATED，會觸發前端
		// 全域登出流程，因此在這裡攔截後改回 422 + 專屬訊息，不落到共用映射。該共用映射仍保留給其他
		// （目前沒有，但未來可能出現的）代表真正 JWT 驗證失敗的呼叫端使用。
		if errors.Is(err, app.ErrInvalidCredentials) {
			httpx.RespondError(c, http.StatusUnprocessableEntity, httpx.CodeValidationFailed, "目前密碼不正確", nil)
			return
		}
		respondIdentityError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, gin.H{"changed": true}, nil)
}

// ResetPassword 讓管理員直接設定他人密碼，不需舊密碼。
func (h *IdentityHandler) ResetPassword(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusBadRequest, httpx.CodeValidationFailed, "無效的使用者 ID", nil)
		return
	}

	var req resetUserPasswordRequest
	if err := httpx.BindJSONStrict(c, &req); err != nil {
		httpx.RespondErrorCode(c, http.StatusBadRequest, httpx.CodeValidationFailed, err, httpx.ExtractValidationDetails(err))
		return
	}

	if err := h.svc.ResetPassword(c.Request.Context(), id, auth.GetActorID(c), auth.GetActorRole(c), req.NewPassword); err != nil {
		respondIdentityError(c, err)
		return
	}
	httpx.RespondSuccess(c, http.StatusOK, gin.H{"changed": true}, nil)
}
