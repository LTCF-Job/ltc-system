---
doc_type: decision
covers:
  - apps/api/internal/platform/auth/permission.go
  - apps/api/cmd/server/permission_adapter.go
  - apps/api/cmd/server/routes.go
  - apps/api/internal/modules/identity/app/user_service.go
  - apps/api/internal/modules/identity/infra/user_security_state_repo.go
  - apps/api/migrations/000032_auth_user_security_states.up.sql
  - apps/api/internal/platform/config/config.go
  - apps/web/src/stores/auth.ts
---

# 個人層級 customPermissions 覆蓋接上 API 授權（本地投影＋版本快取）

## Context

`docs/decisions/role-permission-api-authorization.md` 已把「角色層級」的模組權限矩陣接上 `auth.RequirePermission`（DB 查詢＋30 秒 TTL 快取）。「個人層級」的 `customPermissions` 覆蓋——「使用者管理」頁對單一使用者疊加的權限例外——當時只影響前端 UX，後端完全不讀取，是 `docs/tech/frontend-permission-logic.md` 記載的已知落差。

`customPermissions` 原本只寫入 Supabase Auth `app_metadata.custom_permissions`，授權路徑若直接查 Admin API 會讓每支受保護 API 都依賴外部服務。現在改由 `auth_user_security_states` 保存授權所需的最小本地投影；`UserService.UpdatePermissions` 仍更新 Supabase identity，同時更新本地投影，讓 Supabase 負責 identity、PostgreSQL 負責 API 授權讀取。首次遇到尚未投影的舊帳號時才回源一次並建立投影。

## Decision

1. 新增 `auth.CustomPermissionResolver` 與 `auth.VersionedCustomPermissionResolver` 介面，查詢鍵從角色 key 換成使用者 ID。
2. `auth.CachedCustomPermissionResolver` 以共享 `permission_version` 驗證快取版本；本地快取只保存結果，不決定撤權何時生效。
3. `cmd/server/permission_adapter.go` 的 `userSecurityStateResolver` 先查 PostgreSQL `auth_user_security_states`。只有尚未同步的舊帳號才回源 Supabase 一次；讀取或同步失敗會回報錯誤，不會把未知狀態當成有權限。
4. `RequirePermission(resolver, customResolver, module, action)` 先查角色矩陣，再以個人覆蓋整個模組物件；前端透過 `GET /api/v1/auth/me` 取得同一份 effective permissions。
5. `main.go` 將 `UserSecurityStateRepository` 接到 versioned resolver，`UserService` 的 create／update／permission mutation 同步更新安全狀態投影；投影寫入失敗只記錄，不讓已完成的外部 mutation 被回報成可重試失敗。

## Alternatives

- **JWT claim（第一版採用的方案）**：不查任何外部服務，完全不受 Admin API 可用性影響，但個人覆蓋改動要等使用者重新登入或 token 刷新才生效，跟角色矩陣的 30 秒 TTL 不一致，是本次要解決的問題本身，故放棄。
- **角色矩陣也改走 JWT claim（反向統一）**：徹底不查任何 DB/API，機制最單純，但這正是 `role-permission-api-authorization.md` 當初否決的方案——管理員撤銷某角色的權限（例如緊急停用整個 `dispatcher` 角色的刪除權）不會立即生效，要等該角色所有使用者重新登入，對多人同時受影響的情境是安全倒退，不採用。
- **只讀 Supabase Admin API**：能直接取得最新 metadata，但每支 API 都會增加外部 HTTP 依賴，且停權／撤權會受外部服務可用性影響，不採用。

## 追記：production 仍要求 Supabase identity 設定

`internal/platform/config/config.go` 的 `LoadFromEnv` 在 production 且 `SUPABASE_SERVICE_ROLE_KEY` 未設定時直接拒絕啟動。這是 identity mutation 與舊帳號首次投影同步的必要條件；一般已完成投影的授權讀取不會再呼叫 Supabase Admin API。

## Consequences

- 角色矩陣與個人覆蓋都透過共享資料來源版本判斷快取是否仍有效；更新投影會遞增 `permission_version`，撤權不依賴單一 replica 的本地 TTL。
- PostgreSQL 投影與 Supabase identity 採雙寫。Supabase mutation 成功後若投影寫入失敗，API 不要求使用者重送；伺服器會記錄錯誤，後續可用同步／reconciliation 補齊。
- 首次同步舊帳號仍可能需要一次 Supabase Admin API；完成投影後，一般受保護 API 只讀 PostgreSQL。
- `RequirePermission` 簽章變動（新增 `customResolver` 參數）是這個檔案內部的介面變更，唯一呼叫端 `routes.go` 已同步更新；沒有對外 HTTP 契約，不影響前端。
