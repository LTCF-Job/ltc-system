---
doc_type: module
covers:
  - apps/web/src/views/auth/LoginView.vue
  - apps/web/src/components/AppLogo.vue
  - apps/web/src/layouts/DefaultLayout.vue
  - apps/web/public/project-logo.png
---

# 登入與品牌識別

## Responsibility

登入頁提供使用者登入。`AppLogo` 統一呈現協會 logo，供登入頁與側邊欄使用。

## Entrypoints

- `/login`：登入頁路由。
- `DefaultLayout`：登入後的側邊欄品牌入口。

## Flow

登入頁 > Supabase 登入或本機直接登入 > 建立 session > 導向原目的路由

## Shared state

- `authStore`：保存登入 session 與使用者角色。
- `project-logo.png`：favicon、登入頁與品牌元件共用的 logo 資產。

## Invariants and gotchas

- 本機環境（local）一律不呼叫 Supabase，使用 `mock_jwt_<role>` 建立本機開發 session；即使 `.env.local` 殘留 Supabase 設定也不會切換登入模式。
- 本機 API 必須明確設定 `ALLOW_INSECURE_MOCK_AUTH=true` 才接受 mock token；正式環境不得開啟此設定。
- 非本機環境必須透過真實 Supabase Auth 驗證。
- `/auth/me` 權限載入失敗時不得顯示登入成功，前端會清除 session 並回到登入頁。
- 登入頁品牌名稱固定為「好安心關懷協會-後臺系統」。

## Unverified

none

