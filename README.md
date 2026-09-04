# ltc-transport-system

好安心關懷協會-後臺系統。前後端分離的 monorepo：Go API + Vue 3 SPA，資料庫用 Supabase（PostgreSQL）。

```
apps/api/   Go + Gin 後端 API，見 apps/api/README.md
apps/web/   Vue 3 + Vite 前端 SPA，見 apps/web/README.md
docs/       現況維護文件、技術文件、審查報告與 historical 規劃文件
tests/e2e/  瀏覽器 E2E 測試
```

先讀 [`AGENTS.md`](AGENTS.md)，裡面說明改哪個範圍的程式碼要先讀哪份架構規則（`.agents/skills/`），這些規則是目前團隊實際在遵守的分層與邊界共識，比自己猜規範可靠。

## 快速啟動（本機）

不想裝 Go／Node 環境的話，直接用 Docker：

```bash
docker compose -f docker-compose.local.yml up -d --build
```

啟動後：Postgres 在 `5432`、API 在 `8080`、前端在 `3000`。前端容器裡跑的是 Vite dev server（有 hot reload），`/api` 會被 proxy 到 API 容器。

第一次啟動要手動跑 migration：

```bash
cd apps/api && go run ./cmd/migrate up
```

不想用 Docker、想直接跑原生環境的話，分別看 `apps/api/README.md` 和 `apps/web/README.md`。

也可以用根目錄的 `Makefile`：

```bash
make docker-up      # 依 Makefile 設定啟動 compose；維護時優先明確指定 compose file
make migrate-up      # 套用 migration
make run             # go run 啟動 API（不進容器）
make web-dev          # npm run dev 啟動前端（不進容器）
make test             # 後端單元測試
make test-e2e         # tests/e2e 的瀏覽器測試
```

## 系統長相

- 前端是純靜態 SPA（Vercel 部署），所有資料都透過 REST API 打後端拿。
- 後端是單一 Go 服務（Cloud Run 部署），沒有拆微服務。政府申報 Excel 由 `POST /exports` 同步產出，沒有另外的批次程式。
- Auth 交給 Supabase：前端用 Supabase JS SDK 登入拿 JWT，後端只負責驗簽（打 Supabase 的 JWKS endpoint）跟看 JWT 裡帶的角色 claim 做授權，自己不存密碼。
- 資料庫 schema 是自己手刻 SQL migration（`apps/api/migrations/`，`cmd/migrate` 執行），沒有用 Supabase CLI 那一套 migration 機制，兩者不要混用。
- 現行 health endpoint 是 `GET /api/health`；notification service 目前預設寫 simulated email log，不代表 Resend 或其他 email provider 已接通。

想搞懂整個系統在幹嘛，先看這條核心資料流：司機在接送匯報表上填「今天有沒有載到某個案」→ 操作人員把匯報 .xlsx 匯入後端 → 姓名正規化配對司機、依排班規則展開成完整趟次、多來源回報用「同車取最新、跨車 OR」規則合併 → 存成搭乘紀錄 → 定期跟「應搭日曆」比對抓出未回報 → 月底跑前置檢核、通過才產出政府申報用的 Excel。這條流程橫跨了後端大部分模組，細節在 [`docs/tech/backend-flows.md`](docs/tech/backend-flows.md)。

## 技術文件

框架、API、業務流程與操作方式分別獨立成文件，放在 [`docs/tech/`](docs/tech/README.md)。第一次接手先讀 [`docs/maintainer-guide.md`](docs/maintainer-guide.md)，再讀 [`docs/tech/maintainer-runbook.md`](docs/tech/maintainer-runbook.md)；完整問題清單在 [`docs/reviews/2026-09-04-full-stack-review.md`](docs/reviews/2026-09-04-full-stack-review.md)。`apps/api/README.md`、`apps/web/README.md` 保留啟動指令與 package 細節。

分層與邊界規則的權威來源是 [`AGENTS.md`](AGENTS.md) 底下的 `.agents/skills/`，改架構前先讀。`docs/` 根目錄與 `docs/flows/` 中標示 historical 的文件保留作為背景資料，跟現有程式碼衝突時，以程式碼、`docs/maintainer-guide.md`、`docs/tech/` 與最新 review 為準。

## CI/CD

`.github/workflows/` 底下有兩條 pipeline：PR 主要跑靜態／編譯檢查；push 到 `main` 才會進入部署流程（migration → Cloud Run API → Vercel 前端）。部署參數怎麼串（Supabase / Cloud Run / Vercel 三邊環境變數怎麼對應）寫在 [`docs/tech/deployment.md`](docs/tech/deployment.md)，但正式 deploy、外部 provider 與 rollback 仍需依 Runbook 做環境驗證。
