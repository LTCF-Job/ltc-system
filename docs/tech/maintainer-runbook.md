---
doc_type: flow
covers:
  - README.md
  - Makefile
  - docker-compose.local.yml
  - apps/api/cmd/server/main.go
  - apps/api/cmd/migrate/main.go
  - apps/api/Dockerfile
  - apps/web/Dockerfile
---

# 維護 Runbook

本 Runbook 描述目前 repository 可找到的本機啟動、migration、health check 與故障排查流程。它不保證外部 Supabase、政府資料來源、寄信 provider 或正式部署已可用；那些項目請看「待驗證」章節與 [Full-stack Review](../reviews/2026-09-04-full-stack-review.md)。

## Trigger

以下情境使用本文件：

- 新開發人員第一次在本機啟動 `ltc-system`。
- 修改 migration、environment、API route 或 frontend API contract 後，需要做文件範圍內的靜態確認。
- API 啟動失敗、前端無法登入、出現 401／403、查不到資料或 migration 失敗。

## 本機啟動（Docker）

### 前置條件

- Docker Desktop 與 Docker Compose 可用。
- 已準備本機 `.env`／shell environment；不要把真實 secret 寫入 repository 或文件。
- 若需要真實 DB／Auth，準備 Supabase project URL、JWT／JWKS 設定、database URL 與必要的 Admin API key。

### 步驟

在 repository root 執行：

    docker compose -f docker-compose.local.yml up -d --build

查看服務：

    docker compose -f docker-compose.local.yml ps
    docker compose -f docker-compose.local.yml logs -f api
    docker compose -f docker-compose.local.yml logs -f web

第一次建立或更新 schema：

    Set-Location apps/api
    go run ./cmd/migrate up

若使用 root `Makefile`，`make migrate-up` 會切到 `apps/api` 執行相同 migration runner；`make docker-up` 沒有隱含指定 local compose file，維護時請優先使用上面的明確 `docker compose -f ...` 指令。

## 本機啟動（原生開發）

### API

    Set-Location apps/api
    go run ./cmd/server

API 預設 listen address 依 `PORT`／config 設定，常見本機位址是 `http://localhost:8080`。若 `APP_ENV=local` 且沒有 database，server 可能仍啟動；這只代表 process 能啟動，不代表需要資料庫的 route 可正常使用。

### Web

    Set-Location apps/web
    npm install
    npm run dev

前端透過 `VITE_API_BASE_URL` 或既定 proxy 設定呼叫 API。若前端沒有有效的 Supabase client 設定，local login path 可能使用 local mock session；不要把該路徑當成 production authentication 證據。

## Environment 最小檢查

| 類別 | 主要設定 | 用途與注意事項 |
| --- | --- | --- |
| API runtime | `APP_ENV`、`PORT`、`ALLOWED_ORIGINS`、`LOG_LEVEL` | `local` 允許開發降級；production 應 fail closed |
| Database | `DATABASE_URL`、`DB_MAX_OPEN_CONNS`、`DB_MAX_IDLE_CONNS` | PostgreSQL／Supabase DB pool；migration 與 server 應指向預期資料庫 |
| JWT | `SUPABASE_JWT_ISSUER`、`SUPABASE_PROJECT_REF`、`SUPABASE_JWKS_URL` | production 使用 JWKS、issuer 與固定 audience `authenticated`；不要在文件記錄 token／secret |
| Supabase Admin | `SUPABASE_SERVICE_ROLE_KEY`、`SUPABASE_ADMIN_API_TIMEOUT` | user、role bootstrap 與管理 API；缺少時 identity admin endpoints 可能回 503 |
| Bootstrap | `DEFAULT_ADMIN_EMAIL`、`DEFAULT_ADMIN_PASSWORD` | 兩者和 Admin 設定都具備時才會嘗試 idempotent default admin bootstrap；只在 secret manager／本機安全環境提供 |
| Frontend | `VITE_API_BASE_URL`、Supabase public URL／anon key 等 Vite 設定 | build 時注入；不要將 service role key 放入 `VITE_*` |
| External | `RESEND_API_KEY`、`NOTIFY_FROM`、holiday provider 設定 | Resend 設定目前存在但 notification service 預設仍使用 simulated sender |

設定名稱與目前程式碼不一致時，以 `apps/api/internal/platform/config/config.go` 和 `apps/web` 的 runtime config 為準；不要沿用歷史文件中的 `GOOGLE_SA_JSON`、`/healthz` 或其他已移除名稱。

`config.go` 內仍有公開的 development default encryption／HMAC key，只能供 local 開發；不可讓含真實個資的資料庫使用，也不可因 `APP_ENV` 誤設而帶入 production。正式環境必須使用 secret manager 中可輪替的金鑰，文件只記錄變數名稱，不記錄值。

## Migration 與 seed

### Migration 流程

    確認 DATABASE_URL
          │
          ▼
    go run ./cmd/migrate up
          │
          ├─ 依序執行 apps/api/migrations/*.up.sql
          ├─ 建立／更新 schema、index、constraint、RLS
          └─ 依設定執行 seed / default admin bootstrap

操作前請確認：

- 目前 connection 不是 production，或已取得正式 migration 授權。
- `schema_migrations`（或 runner 使用的 tracking table）狀態與目標版本。
- migration 與 seed 是否仍引用已刪除欄位、舊 unique constraint 或舊 status。
- 不要同時啟動多個 migration runner；目前 runner 沒有足夠的 cross-instance lock 保證。

### Seed 注意事項

`apps/api/seed/` 的資料是開發／測試資料。`demo` seed 不是正式資料匯入工具；目前 review 已找到它和最新 migration 可能漂移的項目。若 seed 失敗，先比對 migration schema、constraint、欄位與 enum/status，再決定是否修 seed。不要直接刪資料或重設 shared database。

## Health check

目前現行 endpoint 是：

    GET http://localhost:8080/api/health

可用 PowerShell 檢查：

    Invoke-WebRequest http://localhost:8080/api/health | Select-Object StatusCode, Content

`/api/health` 回傳的 HTTP status 目前可能在 database ping 失敗時仍是 200，因此必須同時查看 response body 的 `database` 狀態與 server log；HTTP 200 不等於資料庫或外部整合健康。歷史文件中的 `/api/v1/healthz` 不是現行 route。

## 常見故障排查

### API 能啟動但查詢失敗

1. 確認 `DATABASE_URL` 指向正確資料庫。
2. 呼叫 `/api/health` 並查看 `database` 欄位。
3. 查看 API log 是否有 connection、migration、foreign key 或 permission error。
4. 確認 migration 已完成且 seed 沒有引用舊欄位。
5. local offline mode 只能用來看啟動／部分 UI 行為；不能用來宣稱 CRUD、transaction 或報表 query 已驗證。

### 登入成功但 API 回 401

- 檢查 browser 是否送出 `Authorization: Bearer <JWT>`。
- 檢查 token 的 issuer、audience、expiry 與 API config。
- production 使用 Supabase JWKS；local `mock_jwt_` 僅在 `APP_ENV=local` 有效。
- 若是「前端看似登入、API 拒絕」的情況，檢查前端是否使用 `user_metadata.role`，而 backend 只接受 `app_metadata.role`。

### API 回 403

- 先確認使用者的 built-in role 與 module/action permission。
- 確認 DB 中的 role／permission binding 已寫入且 server 已重新載入或等待 process-local permission cache 到期。
- 後端 permission 是最後邊界；不要只靠 sidebar 是否顯示判斷授權正確。
- user／role administration 還有 self-role、last-admin 與跨 instance cache 風險，需看 review backlog。

### Identity users／roles endpoint 回 503

identity management 需要 Supabase Admin API 與 service role 設定。缺少或錯誤時，API 可能回 `SERVICE_UNAVAILABLE`；不要用 local mock user 資料判斷正式 Admin API 已可用。

### Notification 看起來成功但收不到信

目前 server composition root 以 `nil` sender 建立 notification service，service 會套用 `LogEmailSender`，只在 log 顯示 simulated email。`RESEND_API_KEY`／`NOTIFY_FROM` 目前不是「已實際寄信」的證據；需完成 adapter、provider credentials、delivery log 與 runtime verification 後才能宣稱寄信可用。

### 前端表格資料與 backend 不一致

- 檢查 query 名稱是否一致，例如 `q`／`keyword`、日期欄位、pagination 與 filter。
- 檢查 response 是否為 `{ data, meta }`，不要從 envelope 根層讀 `message`。
- 確認 `apps/web/src/api` 型別沒有超前現行 Go handler；目前 repository 沒有可直接視為 source of truth 的 OpenAPI spec。
- 先看 [API Reference](backend-api-reference.md) 和 [Full-stack Review](../reviews/2026-09-04-full-stack-review.md) 的 contract drift 清單。

### CORS、Docker 或部署啟動問題

- `ALLOWED_ORIGINS` 的逗號分隔值要確認是否包含實際 origin；目前 parsing 的 whitespace 行為需特別檢查。
- Docker image 使用與 runtime 相關的 base image／權限／healthcheck 設定；不要只看 container 是 running。
- server 目前以基本 `r.Run` 啟動，production 的 timeout、graceful shutdown、readiness 與 liveness 需要另外驗證。

## 靜態文件／變更驗收

文件或非程式碼變更可執行：

    git diff --check

再以 `rg` 檢查歷史 route／claim 是否只出現在明確標示的 historical 或 review evidence 中。文件-only 變更不執行 Go tests、frontend build、Playwright E2E 或實際 deployment verification。

## Failure modes

- **Migration partially applied**：先停止後續 deploy，保存 runner log 與 DB migration state；不要用 destructive reset 掩蓋原因。
- **API started in offline local mode**：process 可用但 DB-backed endpoints 可能 panic、回空資料或失敗；將其標示為未驗證。
- **JWT valid but permission stale**：可能是 app metadata、DB custom permissions 或 process-local cache 不一致；重新取得 token 並檢查 DB／server instance。
- **Mock／demo path hides production defect**：回到真實 API、資料庫與 Supabase Auth boundary 分別驗證。
- **Health 200 with dependency failure**：讀 body 與 log，不只依 HTTP status。

## Unverified

以下項目不在本文件建立時以 runtime 證據確認：

- 真實 Supabase Auth JWT／JWKS、Admin user listing、role metadata 更新與 logout 行為。
- PostgreSQL migration 全量執行、RLS／constraint、transaction rollback 與併發 migration 行為。
- 政府 holiday provider、Resend／任何 email delivery、Storage 與正式 export download。
- Docker Compose、Cloud Run／Vercel（或目前部署平台）的正式網路、secret、timeout、readiness 與 rollback。
- 前後端所有 CRUD、pagination、日期 timezone、mobile accessibility 與跨 instance permission revocation 的 runtime contract。
