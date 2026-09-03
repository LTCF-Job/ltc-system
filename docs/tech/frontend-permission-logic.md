# 前端權限判斷邏輯

前端不再自行維護任何角色到權限的對照表。所有畫面顯示與操作限制都依據登入時向後端 `GET /api/v1/auth/me` 取得的 effective permissions，這份資料就是後端 `RequirePermission` 實際查詢並合併（角色矩陣＋個人 `customPermissions` 覆蓋）後的同一份結果——前端看到的能不能做，跟 API 實際放不放行必然一致，不會有兩套規則分歧的問題。

## 權限的取得與生效時機（`stores/auth.ts`）

`loadPermissions()` 呼叫 `GET /api/v1/auth/me`，把回傳的 `permissions` 存入 store，並標記 `permissionsLoaded = true`。觸發時機：

1. 登入成功後（`setSession()` 內部呼叫），涵蓋三條登入路徑：正常 Supabase 登入、demo／`ltcf-admin` email 代稱登入（實際仍走 Supabase）、local 環境的 mock JWT 登入（Supabase 未設定時，表單送出後端直接發一張 `mock_jwt_<role>` token）。
2. 分頁重新整理時：store 建構階段若偵測到 localStorage 已有 `token`／`user`（沿用既有 session），立即補打一次 `loadPermissions()`。**Permissions 本身不寫入 localStorage**，每次還原 session 都是向後端要最新的一份，避免權限異動後舊分頁還讀到過期快取。

`loadPermissions()` 內部用一個閉包變數快取進行中的 promise，避免 router guard 與 store 初始化同時觸發造成重複請求。

`hasPermission(module, action)`：`permissionsLoaded === false` 時一律回傳 `false`（安全預設，不放行），其餘情況單純查 `permissions[module][action]`。**沒有任何角色字串的短路判斷**——包含 `admin` 在內，都是後端矩陣給 `true` 才會是 `true`，前端不額外開後門。這是刻意的設計：自訂角色與內建角色使用同一套判斷路徑，不會有「前端多信任 admin 一點」的分歧。

`action` 型別是 `'view' | 'edit' | 'delete'`，跟後端 `ModulePermission{View, Edit, Delete}` 三軸對齊。

## 路由守衛（`router/guards.ts`）

`beforeEach` 依序檢查：

1. 路由不是 `meta.public` 且使用者未登入 → 導去 `/login`。
2. 已登入卻要進 `/login` → 導去首頁。
3. 已登入但權限尚未載入完成（`!permissionsLoaded`，例如剛按 F5）→ `await authStore.loadPermissions()` 待其完成後才繼續判斷，不會因為請求還沒回來就誤判為無權限而把使用者踢出當前頁。
4. `meta.module` 有設定值 → 呼叫 `authStore.hasPermission(module, 'view')`，沒權限就導回首頁並跳警告訊息。

**沒有 `meta.roles` 這回事了**——`router/index.ts` 的每個路由只保留 `meta.module`，不再有平行存在、且早已跟真實判斷脫鉤的角色字串陣列。過去 `meta.roles` 只是文件性質的標註、不影響實際放行，這個誤導來源已經整個拿掉。

## 模組定義（`src/types/domain.ts`）

`SYSTEM_MODULES` 仍是模組 id 到顯示名稱的對照表，供「角色身分管理」頁渲染可勾選的模組清單，例如 `masters_cases`、`rides_calendar`、`settings_users`、`settings_holidays`、`ops_tasks`。**新增頁面／新增模組時只需要做兩件事**：`SYSTEM_MODULES` 加一筆、路由 `meta.module` 指到這個新模組 id——角色的權限值完全由後端 `roles.permissions`（「角色身分管理」頁編輯的那份資料）決定，前端不再需要為每個角色手動列一份預設權限表。

`ops_tasks` 目前沒有對應的前端頁面（純後端排程維運任務），`SYSTEM_MODULES` 收錄它只是讓角色管理頁能夠授權，屬預期行為。

## 個人自訂權限覆蓋

「使用者管理」頁對單一使用者疊加的 `customPermissions`，後端 `RequirePermission` 已經在合併時套用（整個模組物件覆蓋語意，見 [custom-permission-admin-api-enforcement.md](../decisions/custom-permission-admin-api-enforcement.md)），前端拿到的 `/auth/me` 回應本來就是合併後的最終結果，不需要在前端再合併一次。

## 跟後端授權的對應關係

前端 `hasPermission(module, action)` 與後端 `auth.RequirePermission(module, action)` 現在是同一套資料的兩個消費端，兩者共用 `auth.ResolveEffectivePermissions`：`RequirePermission` 直接拿它的結果判斷放不放行，`/auth/me` 拿同一個結果回給前端顯示。所有路由（含過去維持 `RequireRoles` 白名單的 `/users`、`/roles`、`/auth/change-password`、`/demo/reset`、`/tasks/*`、`/holidays*`）現在都走這一套，`auth.RequireRoles` 已刪除，詳見 [role-permission-api-authorization.md](../decisions/role-permission-api-authorization.md) 的 2026-09 修訂。

## 已知限制

`SUPABASE_SERVICE_ROLE_KEY` 在 production 環境未設定時，服務會直接拒絕啟動（見 `internal/platform/config/config.go`），不會再有「個人覆蓋悄悄失效」的情況；這條限制只在 local／demo 環境仍成立（fail-open 為「沒有個人覆蓋」），詳見 [custom-permission-admin-api-enforcement.md](../decisions/custom-permission-admin-api-enforcement.md) 的追記。
