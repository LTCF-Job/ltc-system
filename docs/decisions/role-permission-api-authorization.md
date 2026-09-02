---
doc_type: decision
covers:
  - apps/api/internal/platform/auth/permission.go
  - apps/api/cmd/server/routes.go
  - apps/api/cmd/server/permission_adapter.go
  - apps/api/internal/modules/identity/app/model.go
  - apps/api/migrations/000018_role_permission_delete_axis.up.sql
  - apps/web/src/types/domain.ts
  - apps/web/src/views/settings/RoleManagementView.vue
---

# 自訂角色的 API 存取層級改依模組權限矩陣

## Context

`auth.RequireRoles` 過去是逐路由寫死 `"viewer"`／`"staff"`／`"admin"` 三個字面值的白名單。但 Supabase JWT 的 `app_metadata.role` 寫入的其實是角色的 `key`（例如 `"dispatcher"`），不是 `roles.base_role`。兩者疊加的結果是：連系統內建的「調度員」角色，在真實環境 API 層一律被打 403——因為它的 JWT 角色字串永遠對不上任何路由要求的三個字面值之一。`roles` 表的 `permissions` JSONB（`view`／`edit` 兩軸模組矩陣）在「角色身分管理」頁可以精細設定，但這個設定從未真正影響 API 層的存取判斷；`docs/tech/frontend-permission-logic.md` 已把這件事定性為「已知落差」。

翻查現有路由發現後端的授權粒度其實比矩陣的兩軸更細：`masters_cases`／`masters_sites`／`masters_vehicles`／`masters_drivers`／`masters_caregivers`／`masters_regions`／`settings_notifications` 的刪除路由僅限 `admin`，但 `driver_reports`／`vehicles_maintenance`／`attendance_fuel` 的刪除路由跟編輯同層級（`staff`＋`admin`）。矩陣若只有兩軸，套用矩陣會讓前者的刪除權限意外鬆綁給所有能編輯的角色。

## Decision

1. `roles.permissions` 加第三軸 `delete`（`ModulePermission{View,Edit,Delete}`），對既有五個系統角色的既有值做一次性 migration 回填（見 `000018_role_permission_delete_axis.up.sql`）：`driver_reports`／`vehicles_maintenance`／`attendance_fuel` 的 `delete` 跟隨各角色自己的 `edit` 值；其餘有刪除路由的模組 `delete` 只有 `base_role = admin` 為 `true`；沒有刪除路由的模組一律 `false`。
2. 新增 `auth.RequirePermission(resolver, module, action)`，取代原本對應到模組 CRUD 的 `~60` 條 `auth.RequireRoles(...)` 呼叫；中介層查角色目前的 JWT `role`（即角色 `key`）在 `roles.permissions` 的模組矩陣，判斷 `view`／`edit`／`delete` 對應動作。
3. `PermissionResolver` 介面定義在 `internal/platform/auth`（不匯入 `identity` 模組型別，遵守 platform 不依賴業務模組的分層規則），由 `cmd/server` 的 `rolePermissionResolver` adapter 接上 `identity` 模組既有的 `RoleStore.GetByKey`。查詢結果包一層 30 秒 TTL 的行程內快取（`auth.CachedPermissionResolver`），讓「角色身分管理」頁改權限後不需要使用者重新登入即可生效，同時不讓每個受保護請求都直接查一次 `roles` 表。
4. `/users`、`/roles`、`/auth/change-password`、`/demo/reset`、`/tasks/*`、`/holidays*` 維持 `RequireRoles` 粗粒度白名單，不改走模組矩陣——使用者／角色管理本身能授予他人權限，若也讓矩陣的 `settings_users`／`settings_roles` 開關直接控制這些端點，等於讓自訂角色可能透過調整矩陣間接自我提權；`/holidays*` 則是因為 `SYSTEM_MODULES` 從未定義對應模組，矩陣本來就不涵蓋它。

## Alternatives

- **只修正 JWT 寫入 `base_role`（不動 `RequireRoles`）**：範圍更小，能讓 `dispatcher` 等內建角色恢復正常，但自訂角色永遠只能落在 `viewer`/`staff`/`admin` 三檔，管理員在角色頁勾的模組矩陣仍然管不到 API，沒有真正解決落差。
- **權限直接寫進 JWT custom claim，中介層不查 DB**：免查詢、效能最好，但改角色權限後要等使用者重新登入才生效，犧牲了「角色身分管理」頁的即時性。
- **不加 `delete` 軸，直接把刪除動作併入 `edit` 判斷**：改動範圍最小，但會讓 `masters_cases` 等模組的刪除權限從「僅限 admin」意外鬆綁給所有能編輯的角色，是安全倒退，故未採用。

## Consequences

- 個人層級的 `custom_permissions` 覆蓋（「使用者管理」頁對單一使用者的自訂權限）當時**沒有一併接上** API 層，後續已在 [custom-permission-admin-api-enforcement.md](custom-permission-admin-api-enforcement.md) 補上——與本文件描述的角色矩陣採同一套「查詢＋30 秒 TTL 快取」機制，只是資料來源改為 Supabase Admin API，取捨見該文件。
- 新增自訂角色、或修改既有角色的模組矩陣，會在 30 秒內反映到 API 存取範圍；反過來說，撤銷某角色的權限後最多有 30 秒的延遲視窗。
- `roles.base_role` 欄位在這次改動後不再被任何執行路徑讀取（僅 migration 000018 一次性回填時用過），保留欄位本身供未來「哪些角色屬於高信任層級」之類的判斷使用，但目前是死資料，之後若徹底不需要可以另開 migration 移除。
- 這個檔案一旦再被改動（尤其是 `permission.go` 的快取 TTL、或 routes.go 新增／調整模組路由）就會被標記 stale，需要重新核對「哪些路由走矩陣、哪些維持白名單」的邊界是否還成立。
