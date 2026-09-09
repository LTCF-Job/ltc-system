package app

import (
	"fmt"
	"sort"
	"strings"
)

// ModuleKeys 是權限矩陣認可的所有功能模組 key，也是 API 授權（cmd/server/routes.go 的
// RequirePermission）與前端權限設定畫面共用的權威清單。新增受權限控管的功能模組時，
// 必須同時在此登記並以 migration 回填既有角色，否則該模組對所有角色都會是拒絕。
var ModuleKeys = []string{
	"dashboard",
	"masters_regions",
	"masters_cases",
	"masters_sites",
	"masters_vehicles",
	"masters_drivers",
	"masters_caregivers",
	"driver_reports",
	"driver_report_mappings",
	"rides_calendar",
	"rides_issues",
	"rides_missing",
	"reports_trip_summary",
	"reports_hsinchu_schedule",
	"vehicles_maintenance",
	"attendance_fuel",
	"exports",
	"audit_logs",
	"settings_notifications",
	"settings_users",
	"settings_roles",
	"settings_holidays",
	"ops_tasks",
}

var moduleKeySet = func() map[string]bool {
	m := make(map[string]bool, len(ModuleKeys))
	for _, k := range ModuleKeys {
		m[k] = true
	}
	return m
}()

// IsModuleKey 回報 key 是否為已登記的功能模組。
func IsModuleKey(key string) bool { return moduleKeySet[key] }

// validateModuleKeys 檢查權限矩陣只包含已登記的模組 key；未登記的 key 寫進 JSONB 後
// 永遠不會被任何路由讀到，等同無聲失效，故在寫入前就擋下並回報全部不合法的 key。
func validateModuleKeys(perms map[string]ModulePermission) error {
	var unknown []string
	for k := range perms {
		if !moduleKeySet[k] {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	return fmt.Errorf("%w: %s", ErrUnknownModuleKey, strings.Join(unknown, ", "))
}
