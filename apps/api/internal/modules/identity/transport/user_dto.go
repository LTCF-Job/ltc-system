package transport

import (
	"time"

	"ltc-system/apps/api/internal/modules/identity/app"
)

// userResponse 是使用者的對外回應形狀。
type userResponse struct {
	ID                string                         `json:"id"`
	Email             string                         `json:"email"`
	DisplayName       string                         `json:"displayName"`
	Role              string                         `json:"role"`
	Phone             string                         `json:"phone,omitempty"`
	Status            string                         `json:"status"`
	CustomPermissions map[string]modulePermissionDTO `json:"customPermissions,omitempty"`
	LastLoginAt       string                         `json:"lastLoginAt,omitempty"`
	CreatedAt         string                         `json:"createdAt,omitempty"`
}

func toUserResponse(u app.AuthUser) userResponse {
	resp := userResponse{
		ID:          u.ID.String(),
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Role:        u.RoleKey,
		Phone:       u.Phone,
		Status:      u.Status,
		CreatedAt:   u.CreatedAt.Format(time.RFC3339),
	}
	if u.CustomPermissions != nil {
		resp.CustomPermissions = toPermissionsDTO(u.CustomPermissions)
	}
	if u.LastSignInAt != nil {
		resp.LastLoginAt = u.LastSignInAt.Format(time.RFC3339)
	}
	return resp
}

type createUserRequest struct {
	Email             string                         `json:"email" binding:"required,email"`
	DisplayName       string                         `json:"displayName" binding:"required"`
	Role              string                         `json:"role" binding:"required"`
	Phone             string                         `json:"phone"`
	Password          string                         `json:"password" binding:"required,min=8"`
	Status            string                         `json:"status"`
	CustomPermissions map[string]modulePermissionDTO `json:"customPermissions"`
}

type updateUserRequest struct {
	DisplayName *string `json:"displayName"`
	Role        *string `json:"role"`
	Phone       *string `json:"phone"`
	Status      *string `json:"status"`
}

type updateUserPermissionsRequest struct {
	CustomPermissions map[string]modulePermissionDTO `json:"customPermissions" binding:"required"`
}

type changeSelfPasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

type resetUserPasswordRequest struct {
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}
