# apps/web

Vue 3（Composition API）+ TypeScript + Vite 前端 SPA。UI 元件庫用 Element Plus（自動 import），狀態管理用 Pinia，路由用 Vue Router。

## 跑起來

```bash
cp .env.example .env.development
npm install
npm run dev
```

`.env.development` 常用變數：

```
VITE_API_BASE_URL=/api/v1        # API base path，本機開發搭配 vite proxy 用相對路徑就好
VITE_API_TARGET=http://localhost:8080   # vite dev server 把 /api 轉發到這裡（見 vite.config.ts）
VITE_SUPABASE_URL=                # Supabase 專案網址，登入頁 signInWithPassword 用；正式環境必填
VITE_SUPABASE_ANON_KEY=           # Supabase anon public key；正式環境必填
```

其他指令：

```bash
npm run type-check   # vue-tsc --noEmit
npm run build         # type-check + vite build
npm run gen:types    # 依 VITE_API_SPEC_URL 指向的 OpenAPI spec 重新產生 src/types/api.d.ts，後端 API 改了記得跑這個
```

## 技術文件

這份 README 只放怎麼跑起來。架構、頁面、功能流程分開放在 [`docs/tech/`](../../docs/tech/README.md)：

- [前端框架與專案結構](../../docs/tech/frontend-framework.md)
- [前端頁面總覽](../../docs/tech/frontend-pages.md)
- [前端核心功能流程](../../docs/tech/frontend-flows.md)

## Docker

```bash
docker build -t ltc-web .
```

或用專案根目錄的 compose 一次啟動含 Postgres、API：

```bash
docker compose -f ../../docker-compose.local.yml up -d --build
```

啟動後前端在 `http://localhost:3000`，容器內跑 Nginx 把 `/api/` 反向代理到後端。
