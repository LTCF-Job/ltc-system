---
doc_type: flow
covers:
  - apps/web/src/router/
  - apps/web/src/stores/auth.ts
  - apps/web/src/api/auth.ts
---

# 前端權限判斷邏輯

前端不再自行維護任何角色到權限的對照表。所有畫面顯示與操作限制都依據登入時向後端 `GET /api/v1/auth/me` 取得的 effective permissions，這份資料就是後端 `RequirePermission` 實際查詢並合併（角色矩陣＋個人 `customPermissions` 覆蓋）後的同一份結果——前端看到的能不能做，跟 API 實際放不放行必然一致，不會有兩套規則分歧的問題。

## 權限的取得與生效時機（`stores/auth.ts`）

`loadPermissions()` 呼叫 `GET /api/v1/auth/me`，把回傳的 `permissions` 存入 store，並在成功時標記 `permissionsLoaded = true`。函式回傳 `true` 表示權限已載入，回傳 `false` 表示請求失敗並將狀態標為 `error`。觸發時機：

1. 登入成功後：local mock 登入由 `setSession()` 建立 token 後呼叫；Supabase 登入由 `syncSession()` 同步 session 後呼叫，涵蓋 `ltcf-admin` email 代稱（實際仍走 Supabase）。只有權限載入成功才會導向目標頁面。
2. 分頁重新整理時：store 建構階段若偵測到 localStorage 已有本機 mock `token`／`user`（沿用既有 session），立即補打一次 `loadPermissions()`。**Permissions 本身不寫入 localStorage**，每次還原 session 都是向後端要最新的一份，避免權限異動後舊分頁還讀到過期快取。

`loadPermissions()` 內部用一個閉包變數快取進行中的 promise，避免 router guard 與 store 初始化同時觸發造成重複請求。

權限請求失敗時，登入流程會清除 session，不會同時顯示「系統發生錯誤」與「登入成功」。路由守衛遇到既有的 `error` 狀態也會清除 session；受保護路由導回 `/login`，公開路由則繼續放行。

`hasPermission(module, action)`：`permissionsLoaded === false` 時一律回傳 `false`（安全預設，不放行），其餘情況單純查 `permissions[module][action]`。**沒有任何角色字串的短路判斷**——包含 `admin` 在內，都是後端矩陣給 `true` 才會是 `true`，前端不額外開後門。這是刻意的設計：自訂角色與內建角色使用同一套判斷路徑，不會有「前端多信任 admin 一點」的分歧。

`action` 型別是 `'view' | 'edit' | 'delete'`，跟後端 `ModulePermission{View, Edit, Delete}` 三軸對齊。

## 路由守衛（`router/guards.ts`）

`beforeEach` 依序檢查：

1. 已登入但權限尚未載入完成（`!permissionsLoaded`，例如剛按 F5）→ `await authStore.loadPermissions()` 待其完成。
2. 權限載入失敗 → 清除 session；受保護路由導去 `/login`，公開路由繼續放行。
3. 路由不是 `meta.public` 且使用者未登入 → 導去 `/login`。
4. 已登入卻要進 `/login` → 導向第一個有 view permission 的功能路由；若沒有任何可進入路由才顯示 `Forbidden`。
5. 路由若有 `meta.module`，或 `meta.anyPermissions`，就依指定的 view permission 判斷；沒有權限時導向獨立的 `Forbidden` 頁，不再導回可能本身也無權限的首頁。

**沒有 `meta.roles` 這回事了**——`router/index.ts` 的每個路由只保留 `meta.module`，不再有平行存在、且早已跟真實判斷脫鉤的角色字串陣列。過去 `meta.roles` 只是文件性質的標註、不影響實際放行，這個誤導來源已經整個拿掉。

## 模組定義（`src/types/domain.ts`）

`SYSTEM_MODULES` 仍是模組 id 到顯示名稱的對照表，供「角色身分管理」頁渲染可勾選的模組清單，例如 `masters_cases`、`rides_calendar`、`settings_users`、`settings_holidays`、`ops_tasks`。**新增頁面／新增模組時只需要做兩件事**：`SYSTEM_MODULES` 加一筆、路由 `meta.module` 指到這個新模組 id——角色的權限值完全由後端 `roles.permissions`（「角色身分管理」頁編輯的那份資料）決定，前端不再需要為每個角色手動列一份預設權限表。

`ops_tasks` 目前沒有對應的前端頁面（純後端排程維運任務），`SYSTEM_MODULES` 收錄它只是讓角色管理頁能夠授權，屬預期行為。

`/driver-reports/import` 是 upload 與待維護共用頁：路由允許 `driver_reports:view` 或 `driver_report_mappings:view` 任一權限；頁面內的車輛、個案、司機、出勤查詢與各項寫入按鈕仍依各自 API 所需 permission gate，避免進入頁面就呼叫未授權端點。

## 個人自訂權限覆蓋

「使用者管理」頁對單一使用者疊加的 `customPermissions`，後端 `RequirePermission` 已經在合併時套用（整個模組物件覆蓋語意，見 [custom-permission-admin-api-enforcement.md](../decisions/custom-permission-admin-api-enforcement.md)），前端拿到的 `/auth/me` 回應本來就是合併後的最終結果，不需要在前端再合併一次。

## 跟後端授權的對應關係

前端 `hasPermission(module, action)` 與後端 `auth.RequirePermission(module, action)` 現在是同一套資料的兩個消費端，兩者共用 `auth.ResolveEffectivePermissions`：`RequirePermission` 直接拿它的結果判斷放不放行，`/auth/me` 拿同一個結果回給前端顯示。所有路由（含過去維持 `RequireRoles` 白名單的 `/users`、`/roles`、`/auth/change-password`、`/tasks/*`、`/holidays*`）現在都走這一套，`auth.RequireRoles` 已刪除，詳見 [role-permission-api-authorization.md](../decisions/role-permission-api-authorization.md) 的 2026-09 修訂（`/demo/reset` 本身已隨 demo 資料平面移除，系統現在只有 local／production 兩個環境）。

## 已知限制

`SUPABASE_SERVICE_ROLE_KEY` 在 production 環境未設定時，服務會直接拒絕啟動（見 `internal/platform/config/config.go`），不會再有「個人覆蓋悄悄失效」的情況；這條限制只在 local 環境仍成立（fail-open 為「沒有個人覆蓋」），詳見 [custom-permission-admin-api-enforcement.md](../decisions/custom-permission-admin-api-enforcement.md) 的追記。
