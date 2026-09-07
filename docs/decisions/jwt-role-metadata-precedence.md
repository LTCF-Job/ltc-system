---
doc_type: decision
covers:
  - apps/api/internal/platform/auth/auth.go
---

# JWT 角色解析：只信 app_metadata，不採 user_metadata 或頂層 role claim

## Context

`setActorFromClaims` 需要從 Supabase JWT 取出角色供 `RequirePermission`／`RequireRoles` 授權判斷。Supabase 的慣例是：`app_metadata` 只能經由持 service role key 的 Admin API 寫入，使用者無法自行竄改，適合放授權用的角色；`user_metadata` 使用者可透過 `supabase.auth.updateUser({data:{...}})` 自行編輯，只適合放顯示名稱等非敏感資訊；JWT 頂層的 `role` claim 則是 Supabase 內建的 Postgres role（`authenticated`／`anon`／`service_role`），並非本系統的業務角色。

本文件先前記載的版本採「頂層 role → user_metadata.role 覆蓋 → app_metadata.role 覆蓋」的疊加寫法，理由是修正一支特定帳號因 `user_metadata` 存在但缺 `role` 欄位而誤判為 `viewer` 的 bug。但這個寫法本身是一個權限提升漏洞：只要 `app_metadata` 缺 `role`（例如帳號不是經本系統 Admin API 建立），使用者可自行呼叫 `updateUser` 把 `user_metadata.role` 寫成 `"admin"`，`setActorFromClaims` 會採用它，接著即可通過 `/users`、`/roles` 等以角色字串把關的端點修改任何人的權限。

## Decision

角色解析改為只從 `app_metadata.role` 取值，完全移除 `user_metadata.role` 與頂層 `role` claim 這兩條來源。`app_metadata` 缺 `role`、或型別不是字串時，一律降為預設值 `"viewer"`（不拒絕請求，讓後續的 `RequirePermission` 因查不到權限矩陣而回 403，錯誤語意才正確）。

角色解析**不做靜態白名單**：系統支援管理員自建角色（`RoleService.Create` 以 slugify 產生任意 key，見 `role-permission-api-authorization.md`），合法的 role key 集合是動態的，寫死清單會讓自訂角色使用者被靜默降級成 viewer。未登記的 role key 由 `RequirePermission` 查不到對應的權限矩陣而自然擋下，不需要在這一層額外過濾。

顯示名稱維持兩邊都找，`user_metadata.display_name` 與 `app_metadata.display_name` 皆可提供——這不是授權欄位，使用者能自行竄改顯示名稱不構成安全問題。

## Alternatives

- 維持「app_metadata 覆蓋 user_metadata」的疊加寫法（本文件先前採用的方案）：只要 `app_metadata` 缺 `role` 就會被 `user_metadata` 頂替，正是本次要修的提權路徑，故放棄。
- 角色解析加靜態白名單（`admin`／`dispatcher`／`staff`／`driver`／`viewer`）：曾短暫實作，但會讓管理員自建的自訂角色（`role key` 為動態 slug）全部被靜默降級為 viewer，形同關掉自訂角色功能；且來源已限縮為 `app_metadata`（使用者寫不進去），白名單並未增加實質防護，故不採用。

## Consequences

- 新增使用者角色一律只能經由本系統的 Admin API 寫入 `app_metadata.role`；任何只寫 `user_metadata.role` 的建帳號方式（例如手動經 Supabase Dashboard 的一般使用者自助流程）都不會被系統辨識為對應角色，會落為 `viewer`。
- 既有帳號若只有 `user_metadata.role` 沒有 `app_metadata.role`，這次修正後權限會變小（被視為 viewer），而不是像過去可能被冒用提權；上線前應盤點既有帳號的 `app_metadata.role` 是否齊備。
- 這個檔案一旦再被改動（尤其是 role 解析來源、或改為採用白名單）就會被標記 stale，需要重新核對是否重新引入了本次修掉的提權路徑。
