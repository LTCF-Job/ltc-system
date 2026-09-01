---
doc_type: decision
covers:
  - apps/api/internal/platform/auth/auth.go
---

# JWT 角色解析：app_metadata 優先於 user_metadata

## Context

`setActorFromClaims` 需要從 Supabase JWT 取出角色供 `RequireRoles` 授權判斷。Supabase 的慣例是：`app_metadata` 由伺服器端寫入、使用者無法自行竄改，適合放授權用的角色；`user_metadata` 使用者可自行編輯，只適合放顯示名稱等非敏感資訊。

修正前的邏輯用 `if / else if` 依序檢查 `user_metadata`、`app_metadata`、頂層 `role` claim，只要 `user_metadata` 這個 key 存在（即使裡面沒有 `role` 欄位）就不會再檢查 `app_metadata`。`migrations/000002_seed_reference_data.up.sql` 建立的預設管理員帳號把角色寫在 `raw_app_meta_data.role = "admin"`，`raw_user_meta_data` 只有 `display_name`（沒有 `role`）。結果是這個帳號的 `user_metadata` map 存在但沒有 `role`，導致角色永遠退回預設值 `"viewer"`，即使前端登入畫面正確顯示「系統管理員」。實際影響：所有 `RequireRoles("staff", "admin")` 的端點對這個帳號一律回傳 403（例如 `POST /exports/precheck`），且不限於這一個帳號——任何 `user_metadata` 有其他欄位但沒有 `role` 的使用者都會中招。

## Decision

角色解析改為：先看頂層 `role` claim 當基礎值，再用 `user_metadata.role`（若有）覆蓋，最後用 `app_metadata.role`（若有）覆蓋——`app_metadata` 永遠是最終權威來源。顯示名稱維持兩邊都找，`user_metadata.display_name` 與 `app_metadata.display_name` 皆可提供。

## Alternatives

- 維持 `user_metadata` 優先：與 Supabase RBAC 慣例相反，且會被使用者可編輯的欄位覆蓋伺服器授權角色，屬安全疑慮。
- 只看 `app_metadata`，完全不看 `user_metadata.role`：可行，但拿掉了原本設計的彈性（測試/種子資料可能仍會用到 `user_metadata.role`），改為「app_metadata 優先」的疊加寫法風險更低、改動更小。

## Consequences

- 之後新增使用者角色一律應寫入 `app_metadata.role`（`raw_app_meta_data`），不要只寫 `user_metadata`。
- 這個檔案一旦再被改動就會被標記 stale，需重新核對兩個 metadata 來源的優先順序是否還成立。
