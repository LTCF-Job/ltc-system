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

登入頁 > Supabase 登入（正式／Demo）或本機直接登入 > 建立 session > 導向原目的路由

## Shared state

- `authStore`：保存登入 session 與使用者角色。
- `project-logo.png`：favicon、登入頁與品牌元件共用的 logo 資產。

## Invariants and gotchas

- 本機環境（local）未設定 Supabase 時不強制驗證，可直接以本機開發身分或快速切換登入，避免阻礙本機開發與測試。
- 正式與預覽環境必須透過真實 Supabase Auth 驗證。
- Demo 使用者依 JWT 的 `app_metadata.data_plane` 導向獨立的 Demo API 資料平面。
- 登入頁品牌名稱固定為「好安心關懷協會-後臺系統」。

## Unverified

none

