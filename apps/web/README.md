# 長照交通接送後台系統 — 前端應用程式 (@ltc/web)

本專案為長照交通接送後台系統的前端 SPA 應用程式，使用 Vue 3 + Vite + TypeScript + Element Plus 建置。

## 1. 環境變數設定

請複製 `.env.example` 為 `.env.development`：

```bash
cp .env.example .env.development
```

變數說明：
- `VITE_API_BASE_URL`: 後端 API 基礎路徑（預設 `/api/v1`）
- `VITE_SUPABASE_URL`: Supabase 專案網址，登入頁呼叫 `signInWithPassword` 取得真實 JWT 用；正式環境必填
- `VITE_SUPABASE_ANON_KEY`: Supabase 專案 anon public key；正式環境必填
- `VITE_API_SPEC_URL`: OpenAPI 規範 Swagger JSON 網址
- `VITE_ENABLE_MSW`: 是否啟用本機 MSW 模擬伺服器與展示模式快速登入（`true` / `false`）；正式環境必須為 `false` 或不設定

## 2. 啟動與開發指令

安裝相依套件：
```bash
npm install
```

啟動本機開發伺服器（含 MSW 模擬環境）：
```bash
npm run dev
```

執行型別檢查：
```bash
npm run type-check
```

建置生產打包檔案：
```bash
npm run build
```

## 3. 產出 API 型別指令

當後端 OpenAPI 規範更新時，執行以下指令重新產生 `src/types/api.d.ts`：
```bash
npm run gen:types
```

## 4. Docker 容器化建置

獨立建置前端映像檔：
```bash
docker build -t ltc-web ./apps/web
```

透過專案根目錄之 Docker Compose 啟動完整環境（含 Postgres、API 與 Web）：
```bash
docker compose up -d --build
```
啟動後前端可透過 `http://localhost:3000` 瀏覽，Nginx 會自動將 `/api/` 請求反向代理至後端 API 服務。
