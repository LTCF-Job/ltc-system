---
doc_type: decision
covers:
  - apps/web/src/components/ChangePasswordDialog.vue
  - apps/api/internal/modules/identity/app/user_service.go
  - apps/api/internal/modules/identity/infra/supabase_admin_client.go
---

# 密碼變更：舊密碼改由後端真正驗證

## Context

`ChangePasswordDialog.vue` 原本的「目前密碼」欄位只做前端必填檢查，實際改密碼呼叫的是 `supabase.auth.updateUser({ password: newPassword })`。這支 API 只需要一個有效的 session token 即可改密碼，完全不會檢查 `oldPassword` 欄位的值——任何持有有效登入 session 的人（例如使用者離開座位但瀏覽器分頁未關閉）可以在「目前密碼」欄位打任意非空字串，直接把密碼改成自己想要的值，原本的使用者會被立即登出且無法自行復原帳號。

這個問題在展示模式（MSW mock）下完全看不出來，因為 mock handler 不會真的呼叫 Supabase，只會回傳成功。

## Decision

密碼變更改為後端接管，前端移除 `supabase.auth.updateUser()` 直呼：

1. 前端只呼叫 `POST /auth/change-password`，帶 `{oldPassword, newPassword}`（`newPassword` 最短 8 碼，UI 提示同步從 6 碼調整為 8 碼）。
2. 後端 `UserService.ChangeSelfPassword` 先用 `oldPassword` 呼叫 Supabase `POST /auth/v1/token?grant_type=password`（帶當前使用者 email）驗證舊密碼，非 2xx 回應視為驗證失敗，回傳 `ErrInvalidCredentials`（HTTP 401），**不會**接著呼叫 Admin API 改密碼。
3. 驗證通過後才用 Admin API（`service_role` key）設定新密碼。
4. 寫入稽核紀錄 `action: "change_password"`，只記動作與操作者，不記任何密碼內容（明文或雜湊皆不記）。
5. `SUPABASE_SERVICE_ROLE_KEY` 未設定時（`identity` 模組 `Configured()` 為 false），此端點與其餘 `identity` 端點一致回 `503`，不接受任何密碼變更請求。

## Alternatives

- **維持前端直呼 Supabase，只是把 `oldPassword` 欄位拿掉**：治標不治本，沒有解決「有效 session 即可改密碼」的核心問題，只是移除了會造成使用者誤以為有驗證的欄位。
- **前端呼叫 Supabase 的 `signInWithPassword` 驗證舊密碼後才呼叫 `updateUser`**：驗證與寫入拆成兩次前端呼叫，若要用 Admin API 精準控制仍得把 `service_role` key 放進前端環境變數才能達到同等效果，而前端無法安全持有這把金鑰；後端接管才能同時做到「真正驗證」與「金鑰不外洩」。

## Consequences

- 之後任何新增的密碼相關流程（例如忘記密碼、後台重設密碼）都應遵循「驗證舊憑證在後端完成、`service_role` key 不進前端」的原則。
- `SUPABASE_SERVICE_ROLE_KEY` 未設定的環境（本輪交付時的預設狀態）下，使用者無法自行改密碼，需由資料庫管理員直接於 Supabase Dashboard 處理，這是已知且刻意接受的限制，待金鑰就位後自動解除。
