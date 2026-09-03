---
doc_type: decision
covers:
  - apps/api/internal/platform/auth/permission.go
  - apps/api/cmd/server/permission_adapter.go
  - apps/api/cmd/server/routes.go
  - apps/api/internal/platform/config/config.go
  - apps/web/src/stores/auth.ts
---

# 個人層級 customPermissions 覆蓋接上 API 授權（查詢＋TTL 快取，與角色矩陣同一套機制）

## Context

`docs/decisions/role-permission-api-authorization.md` 已把「角色層級」的模組權限矩陣接上 `auth.RequirePermission`（DB 查詢＋30 秒 TTL 快取）。「個人層級」的 `customPermissions` 覆蓋——「使用者管理」頁對單一使用者疊加的權限例外——當時只影響前端 UX，後端完全不讀取，是 `docs/tech/frontend-permission-logic.md` 記載的已知落差。

`customPermissions` 唯一的資料來源是 Supabase Auth `app_metadata.custom_permissions`（`identity.UserService.UpdatePermissions` 透過 `AdminIdentityProvider.SetCustomPermissions` 寫入），只能經由 Supabase Admin API 取得。第一版實作曾改走「JWT claim」（在 `setActorFromClaims` 解析 `app_metadata.custom_permissions`，不查任何外部服務），理由是當時部署環境的 `SUPABASE_SERVICE_ROLE_KEY` 刻意留空，若查 Admin API 會讓所有受保護路由意外依賴一個選配的外部服務。但這個方案讓角色矩陣（查詢＋快取，30 秒生效）與個人覆蓋（JWT claim，需重新登入才生效）用了兩種不同新鮮度、不同機制的資料來源，維護與說明成本高。使用者要求統一成同一種做法。

## Decision

1. 新增 `auth.CustomPermissionResolver` 介面（`Resolve(ctx, actorID) (map[string]ModulePermission, error)`），比照 `PermissionResolver` 的形狀，只是查詢鍵從角色 key 換成使用者 ID。
2. 新增 `auth.CachedCustomPermissionResolver`，比照 `CachedPermissionResolver` 包一層同樣 `permissionCacheTTL`（30 秒）的行程內快取，key 為 `actorID`。
3. `cmd/server/permission_adapter.go` 新增 `userCustomPermissionResolver`，透過既有的 `identityapp.AdminIdentityProvider.GetUser` 取得該使用者的 `AuthUser.CustomPermissions`。**明確的 fail-open 例外**：`AdminIdentityProvider.Configured()` 回 `false`（Service Role Key 未設定）時，`Resolve` 回 `(nil, nil)`（視為「沒有個人覆蓋」），不回傳 `ErrIdentityProviderUnconfigured`——這是 `AdminIdentityProvider` 介面文件「未設定必須 fail-loud」規則的刻意例外，只適用於這一條讀取路徑：customPermissions 是疊加在角色矩陣之上的可選層，未設定時應該退回純角色矩陣判斷，不能讓它拖垮系統內所有走 `RequirePermission` 的路由。已設定但查詢本身失敗（網路、逾時等）則正常回傳 error，交由 `RequirePermission` 視為系統錯誤（500），不會被誤判為「沒有個人覆蓋」而放行。
4. `RequirePermission(resolver, customResolver, module, action)` 簽章新增 `customResolver` 參數：先查角色矩陣，再用 `GetActorID(c)` 查個人覆蓋，兩者用「整個模組物件覆蓋」語意合併（customResolver 有該模組 key 就整包取代角色矩陣的值）。`cmd/server/routes.go` 全部 89 個 `RequirePermission(perm, "...")` 呼叫點機械式改為 `RequirePermission(perm, customPerm, "...")`（單一模式取代，函式邏輯不因呼叫點而異）。前端不再自行推導這份合併結果——`apps/web/src/stores/auth.ts` 改為呼叫 `GET /api/v1/auth/me` 取得後端已經合併好的 effective permissions，兩邊共用同一份 `auth.ResolveEffectivePermissions` 邏輯，避免前後端各自實作一次合併規則而產生分歧（見 `role-permission-api-authorization.md` 2026-09 修訂第 5 點）。
5. `main.go` 用既有的 `adminClient`（`SupabaseAdminClient`，已經是 `roleSvc`/`userSvc` 在用的同一個實例）包一層 `NewCachedCustomPermissionResolver(userCustomPermissionResolver{admin: adminClient})`，與 `permResolver` 一起傳入 `newRouter`。

## Alternatives

- **JWT claim（第一版採用的方案）**：不查任何外部服務，完全不受 Admin API 可用性影響，但個人覆蓋改動要等使用者重新登入或 token 刷新才生效，跟角色矩陣的 30 秒 TTL 不一致，是本次要解決的問題本身，故放棄。
- **角色矩陣也改走 JWT claim（反向統一）**：徹底不查任何 DB/API，機制最單純，但這正是 `role-permission-api-authorization.md` 當初否決的方案——管理員撤銷某角色的權限（例如緊急停用整個 `dispatcher` 角色的刪除權）不會立即生效，要等該角色所有使用者重新登入，對多人同時受影響的情境是安全倒退，不採用。
- **customPermissions 改存本地 Postgres（不透過 Supabase Admin API）**：能徹底不受 `SUPABASE_SERVICE_ROLE_KEY` 是否設定影響，機制最乾淨，但需要新增 migration、新的 store，且要處理「`UpdatePermissions` 現有寫入路徑（寫 Supabase `app_metadata`）要保留、改掉、還是雙寫」的問題，屬於另一輪較大改動；本次採用 fail-open 已能在不新增資料表的前提下達成「機制統一、且金鑰未設定時不阻斷任何路由」的效果，故先不做。

## 追記：production 已擋掉 fail-open 的觸發條件

`internal/platform/config/config.go` 的 `LoadFromEnv` 現在會在 `APP_ENV=production` 且 `SUPABASE_SERVICE_ROLE_KEY` 未設定時直接拒絕啟動（`return nil, error`），不再只是 `slog.Warn`。這代表本文件描述的 `Configured()==false` fail-open 分支在 production 環境下已經是不可達路徑——production 若跑得起來，`SUPABASE_SERVICE_ROLE_KEY` 必然已設定。fail-open 分支只在 local／demo 環境仍會被觸發，維持本文件原本的設計不變。

## Consequences

- 角色矩陣與個人覆蓋現在是同一套「查詢＋30 秒 TTL 快取」機制，行為一致：改角色矩陣或改某使用者的 customPermissions，最長 30 秒後在該使用者/角色的所有請求上生效，不再需要重新登入。
- 目前部署環境 `SUPABASE_SERVICE_ROLE_KEY` 留空時，`userCustomPermissionResolver.Resolve` 一律回 `(nil, nil)`，等同於「所有使用者都沒有個人覆蓋，一律照角色矩陣判斷」——`/cases`、`/vehicles` 等路由不受影響；但這也代表在金鑰設定之前，即使管理員在「使用者管理」頁設定了 customPermissions，後端實際上完全不會套用（前端 UX 仍會顯示已設定的覆蓋值，造成前後端不一致）。金鑰設定後才會真正生效，這點與 `docs/tech/pending-integrations.md` 記載的「Supabase Service Role Key 留空」已知限制是同一件事的延伸，不是新增的落差。
- 個人覆蓋現在每次快取過期後會多一次 `AdminIdentityProvider.GetUser` 呼叫（外部 HTTP request），跟角色矩陣的本地 DB 查詢相比延遲更高、更可能失敗；查詢失敗（非「未設定」）會讓該請求回 500，屬於預期行為，不是靜默降級。
- `RequirePermission` 簽章變動（新增 `customResolver` 參數）是這個檔案內部的介面變更，唯一呼叫端 `routes.go` 已同步更新；沒有對外 HTTP 契約，不影響前端。
