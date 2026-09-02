package transport

import (
	"time"

	"ltc-system/apps/api/internal/modules/identity/app"
)

// roleResponse 是角色的對外回應形狀；base_role 屬於後端內部授權對映機制，不對前端揭露。
type roleResponse struct {
	ID          string                         `json:"id"`
	Key         string                         `json:"key"`
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	TagType     string                         `json:"tagType"`
	IsSystem    bool                           `json:"isSystem"`
	Permissions map[string]modulePermissionDTO `json:"permissions"`
	UserCount   int                            `json:"userCount"`
	CreatedAt   string                         `json:"createdAt"`
	UpdatedAt   string                         `json:"updatedAt"`
}

func toRoleResponse(r app.Role) roleResponse {
	return roleResponse{
		ID:          r.ID.String(),
		Key:         r.Key,
		Name:        r.Name,
		Description: r.Description,
		TagType:     r.TagType,
		IsSystem:    r.IsSystem,
		Permissions: toPermissionsDTO(r.Permissions),
		UserCount:   r.UserCount,
		CreatedAt:   r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   r.UpdatedAt.Format(time.RFC3339),
	}
}

// createRoleRequest 對應前端 CreateRoleRequest。
type createRoleRequest struct {
	Key         string                         `json:"key"`
	Name        string                         `json:"name" binding:"required"`
	Description string                         `json:"description"`
	TagType     string                         `json:"tagType"`
	Permissions map[string]modulePermissionDTO `json:"permissions" binding:"required"`
}

// updateRoleRequest 對應前端 UpdateRoleRequest。
type updateRoleRequest struct {
	Name        *string                        `json:"name"`
	Description *string                        `json:"description"`
	TagType     *string                        `json:"tagType"`
	Permissions map[string]modulePermissionDTO `json:"permissions"`
}
