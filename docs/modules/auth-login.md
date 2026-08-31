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

登入頁提供使用者登入與展示模式快速登入。`AppLogo` 統一呈現協會 logo，供登入頁與側邊欄使用。

## Entrypoints

- `/login`：登入頁路由。
- `DefaultLayout`：登入後的側邊欄品牌入口。

## Flow

登入頁 > Supabase 或 Demo 登入 > 建立 session > 導向原目的路由

## Shared state

- `authStore`：保存登入 session 與使用者角色。
- `project-logo.png`：favicon、登入頁與品牌元件共用的 logo 資產。

## Invariants and gotchas

- Demo 快速登入只在明確啟用 mock runtime 時顯示。
- 登入頁品牌名稱固定為「好安心關懷協會-後臺系統」。

## Unverified

none
