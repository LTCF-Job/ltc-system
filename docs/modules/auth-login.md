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

- 本機環境（local）未設定 Supabase 時不驗證 password，依輸入帳號是否包含 `viewer` 發出 `mock_jwt_viewer`，其餘發出 `mock_jwt_admin`；這只供 local 開發，不是正式登入證據。
- 正式環境必須透過真實 Supabase Auth 驗證。
- 登入頁品牌名稱固定為「好安心關懷協會-後臺系統」。

## Unverified

- 真實 Supabase Auth 的 password、JWT refresh／expiry、`app_metadata.role` 更新與 provider logout 尚未在本文件建立時以 runtime 驗證。
- local mock JWT、沒有 DB 的 offline server 與正式 Supabase data plane 的 CRUD／permission 行為不可互相推論。
