---
doc_type: module
covers: ["apps/web/src/router/index.ts"]
---

# 前端頁面總覽

依 `apps/web/src/router/index.ts` 的實際路由整理。`允許角色` 欄是路由 `meta.roles`，**只是標註用**，實際權限判斷是另一套模組權限表，見 [frontend-permission-logic.md](frontend-permission-logic.md)。`API` 欄對應到 [backend-api-reference.md](backend-api-reference.md) 的路由分組，方便追一個功能的前後端兩端程式碼。「全部」＝`['admin', 'staff', 'dispatcher', 'driver', 'viewer']`。

| 路徑 | 頁面元件 | 允許角色 | 主要打的 API |
|---|---|---|---|
| `/login` | `views/auth/LoginView.vue` | 公開 | Supabase `signInWithPassword` |
| `/` | `views/dashboard/DashboardView.vue` | 全部 | `dashboard/*` |
| `/cases` | `views/cases/CaseListView.vue` | admin/staff/dispatcher/viewer | `cases`、`cases/import`、`cases/template` |
| `/cases/:id` | `views/cases/CaseDetailView.vue`（含 `ScheduleEditor.vue`） | admin/staff/dispatcher/viewer | `cases/:id`、`cases/:id/reveal`、`cases/:id/schedule` |
| `/masters/regions` | `views/masters/RegionListView.vue` | admin/staff/dispatcher/viewer | `regions/*` |
| `/masters/sites` | `views/masters/SiteListView.vue` | admin/staff/dispatcher/viewer | `sites/*` |
| `/masters/vehicles` | `views/masters/VehicleListView.vue` | 全部 | `vehicles/*` |
| `/masters/drivers` | `views/masters/DriverListView.vue` | 全部 | `drivers/*` |
| `/masters/caregivers` | `views/masters/CaregiverListView.vue` | admin/staff/dispatcher/viewer | `caregivers/*`、`caregivers/import`、`caregivers/template` |
| `/driver-reports` | `views/driverReports/DriverReportListView.vue` | admin/staff/dispatcher/viewer | `driver-reports`、`driver-reports/:id/import`、`driver-reports/:id/template` |
| `/driver-reports/batch-import` | `views/driverReports/DriverReportBatchImportView.vue` | admin/staff/dispatcher/viewer | `vehicles`、`driver-reports`、`driver-reports/imported-months`、`driver-reports`（建表）、`driver-reports/:id/import` |
| `/driver-reports/mappings` | `views/driverReports/FieldMappingView.vue` | admin/staff/dispatcher/viewer | `driver-reports/columns*` |
| `/rides` | `views/rides/RideCalendarView.vue`（含 `RideManualEntryDialog.vue`） | 全部 | `rides/calendar`、`rides/manual-report` |
| `/rides/issues` | `views/rides/RideIssuesView.vue`（含 `RideCorrectionDrawer.vue`） | admin/staff/dispatcher/viewer | `rides/issues`、`rides/:id`、`rides/:id/resolve-conflict` |
| `/rides/missing` | `views/rides/MissingRidesView.vue` | admin/staff/dispatcher/viewer | `rides/missing` |
| `/reports/trip-summary` | `views/reports/TripSummaryView.vue` | admin/staff/dispatcher/viewer | `reports/trip-summary*` |
| `/reports/hsinchu-schedule` | `views/reports/HsinchuScheduleView.vue` | admin/staff/dispatcher/viewer | `reports/hsinchu-schedule*` |
| `/vehicles/maintenance` | `views/vehicles/MaintenanceView.vue` | 全部 | `vehicles/maintenance*` |
| `/attendance` | `views/attendance/AttendanceFuelView.vue` | 全部 | `attendance`、`fuel-logs` |
| `/audit` | `views/audit/AuditLogView.vue` | admin only | `audit` |
| `/settings/users` | `views/settings/UserManagementView.vue` | admin only | `users/*`（⚠️ 後端未實作，見 [backend-api-reference.md](backend-api-reference.md)） |
| `/settings/roles` | `views/settings/RoleManagementView.vue` | admin only | `roles/*`（⚠️ 後端未實作，見 [backend-api-reference.md](backend-api-reference.md)） |
| `/settings/notifications` | `views/settings/NotificationSettingsView.vue` | admin/staff/dispatcher/viewer | `settings/notification-recipients*`、`notifications/logs` |
| `/settings/holidays` | `views/settings/HolidayCalendarView.vue` | admin/staff/dispatcher/viewer | `holidays*` |
| `/exports` | `views/exports/ExportView.vue` | admin/staff/dispatcher | `exports/precheck`、`exports/*` |

## 共用元件（非路由）

不對應獨立路由，是被上面頁面內嵌使用的：

| 元件 | 被誰用 | 用途 |
|---|---|---|
| `views/cases/ScheduleEditor.vue` | `CaseDetailView` | 編輯個案的星期／時段排班 |
| `views/rides/RideManualEntryDialog.vue` | `RideCalendarView` | 月曆空白格人工補登回報的彈窗 |
| `views/rides/RideCorrectionDrawer.vue` | `RideIssuesView` | 異常搭乘的更正／衝突裁決側邊欄 |
