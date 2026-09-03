package transport

import "ltc-system/apps/api/internal/modules/identity/app"

// modulePermissionDTO 是前端 SystemPermissions 單一模組項目的形狀。
type modulePermissionDTO struct {
	View   bool `json:"view"`
	Edit   bool `json:"edit"`
	Delete bool `json:"delete"`
}

func toPermissionsDTO(perms map[string]app.ModulePermission) map[string]modulePermissionDTO {
	if perms == nil {
		return map[string]modulePermissionDTO{}
	}
	out := make(map[string]modulePermissionDTO, len(perms))
	for k, v := range perms {
		out[k] = modulePermissionDTO{View: v.View, Edit: v.Edit, Delete: v.Delete}
	}
	return out
}

func toPermissionsModel(dto map[string]modulePermissionDTO) map[string]app.ModulePermission {
	if dto == nil {
		return nil
	}
	out := make(map[string]app.ModulePermission, len(dto))
	for k, v := range dto {
		out[k] = app.ModulePermission{View: v.View, Edit: v.Edit, Delete: v.Delete}
	}
	return out
}
