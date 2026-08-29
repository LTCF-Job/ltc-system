# CI/CD 部署操作說明

本專案的部署分成三部分：GitHub Actions 驗證程式碼、Cloud Run 部署 Go API，以及 Vercel 部署 Vue 前端。

## 推送後的行為

| 動作 | 結果 |
|---|---|
| Pull Request 到 `develop` 或 `main` | 執行 API `go vet`、`go test -race`，以及前端 type-check、build |
| Push 到 `develop` | CI 通過後，`deploy-api.yml`／`deploy-web.yml` 一樣會觸發部署，只是 gate 在 GitHub Environment `develop`（Vercel 端對應 `preview` 部署），**不是「不部署」** |
| Push 到 `main` | CI 成功後，gate 在 GitHub Environment `production`：先執行 Cloud Run migration job 更新 Supabase 並部署 API，同時透過 Vercel CLI 部署前端（Vercel 端為 `production` 部署） |

`deploy-api.yml`／`deploy-web.yml` 都是用 `workflow_run` 監聽 CI 完成，再依觸發分支選擇 `environment: production` 或 `environment: develop`（見兩份 workflow 的 `if:`／`environment:` 設定），兩個分支都會實際跑一次部署，差別只在 GitHub Environment 與其對應的 secrets／variables。**目前 `GCP_PROJECT_ID`／`API_SERVICE`／`MIGRATION_JOB` 等變數只在下方「GitHub 設定」以 repository 層級設定**，若沒有另外在 `develop` 這個 GitHub Environment 覆寫成不同的專案或服務名稱，develop 分支的部署會打到跟 production 完全相同的 Cloud Run service／migration job——等於 develop 分支的每次 push 都會直接覆蓋正式環境。要讓 develop 真的只影響測試環境，必須在 GitHub repository 的 **Settings → Environments** 分別建立 `production`／`develop` 兩個 Environment，並在 `develop` Environment 覆寫指向測試用的 GCP 專案／Cloud Run service。

目前資料庫 migration 使用 `apps/api/migrations/*.up.sql`，由 `apps/api/cmd/migrate` 執行。不要同時使用 Supabase CLI 的 `supabase/migrations`，避免兩套 migration history 互相衝突。

## 三個服務的關聯參數總覽

Vercel（前端）、Cloud Run（API／migration job）、Supabase（資料庫與 Auth）三者部署時彼此依賴下列參數，設定順序建議：先建好 Supabase 專案 → 取得對應值 → 設定 Cloud Run → 取得 Cloud Run API 網址 → 設定 Vercel。

| 參數 | 來源服務 | 用途服務 | 說明 |
|---|---|---|---|
| Supabase Transaction Pooler 連線字串 | Supabase → Project Settings → Database | Cloud Run API service、migration job 的 `DATABASE_URL` | 需保留 `sslmode=require`，範例見 `.env.prod.example` |
| Supabase JWKS URL | Supabase → Project Settings → API（`https://<project-ref>.supabase.co/auth/v1/.well-known/jwks.json`） | Cloud Run API service 的 `SUPABASE_JWKS_URL` | `APP_ENV=production` 時必填，缺少會導致啟動失敗（見 `internal/platform/config/config.go`） |
| Supabase Project Ref | Supabase → Project Settings → General | Cloud Run API service 的 `SUPABASE_PROJECT_REF` | 用於驗證 JWT 的 issuer |
| Cloud Run API service 的公開網址 | Cloud Run 部署後由 `gcloud run deploy` 輸出 | Vercel Production 環境變數 `VITE_API_BASE_URL` | 前端所有 API 請求的 base URL，需在 Vercel Dashboard 手動填入完整網址（含 `https://`） |
| Vercel Production 網址 | Vercel 部署後由 `vercel deploy --prod` 輸出（或 Dashboard 上的 Domains） | Cloud Run API service 的 `ALLOWED_ORIGINS` | CORS 白名單，逗號分隔可填多個網域；`APP_ENV=production` 時必填，否則 API 啟動即失敗（見 `internal/platform/config/config.go`） |

migration job 與 API service 共用同一份 config（`apps/api/internal/platform/config/config.go`），因此 `APP_ENV`、`ENCRYPTION_KEY`、`HMAC_KEY` 兩邊都要設定一致，即使 migration job 本身只使用 `DATABASE_URL`。

## GitHub 設定

在 GitHub repository 的 **Settings → Secrets and variables → Actions** 設定：

### Variables

| 名稱 | 範例 |
|---|---|
| `GCP_PROJECT_ID` | `ltc-production` |
| `GCP_REGION` | `asia-east1` |
| `ARTIFACT_REPOSITORY` | `ltc` |
| `API_SERVICE` | `ltc-api`，可省略 |
| `MIGRATION_JOB` | `ltc-api-migrate`，可省略 |

### Secrets

| 名稱 | 內容 |
|---|---|
| `GCP_WIF_PROVIDER` | Google Cloud Workload Identity Federation provider resource name |
| `GCP_DEPLOY_SA` | 允許 Cloud Build、Artifact Registry、Cloud Run 部署的 service account email，IAM 角色需求見下方「Cloud Run 前置條件」 |

## Cloud Run 前置條件

先在 Google Cloud 建立並設定：

0. `GCP_DEPLOY_SA` 這個 service account 除了送出 Cloud Build、部署 Cloud Run 的權限，**還要加 `roles/logging.viewer`**（Logs Viewer）。`deploy-api.yml` 的 `gcloud builds submit` 預設會即時串流建置 log，若 SA 沒有讀 log 權限，指令會直接失敗並顯示 `This tool can only stream logs if you are Viewer/Owner of the project`——即使 Cloud Build 本身有被成功觸發、建置繼續在背景跑，GitHub Actions 這一步還是會判定失敗、擋住後續部署步驟。實際起碼要有：
   - `roles/cloudbuild.builds.editor`（送出並管理 Cloud Build 建置）
   - `roles/run.admin`（部署 Cloud Run service／job）
   - `roles/artifactregistry.writer`（推送映像檔到 Artifact Registry）
   - `roles/iam.serviceAccountUser`（Cloud Run 服務需要用這個 SA 執行時）
   - `roles/logging.viewer`（**串流 Cloud Build log 用，最容易漏掉**）
1. Artifact Registry repository，名稱需與 `ARTIFACT_REPOSITORY` 相同。
2. Cloud Run API service，名稱需與 `API_SERVICE` 相同。
3. Cloud Run Job，名稱需與 `MIGRATION_JOB` 相同，執行指令設定為 `/app/migrate up`。
4. API service 與 migration job 都要設定下列環境變數或 Secret Manager 參照（對應 `apps/api/internal/platform/config/config.go`）：
   - `APP_ENV=production`（必填，缺少或非 `local`/`production` 會拒絕啟動）
   - `DATABASE_URL`（必填，見下方 Supabase Transaction Pooler 連線字串）
   - `ENCRYPTION_KEY`、`HMAC_KEY`（必填，32 bytes base64，且兩者不可相同）
   - `SUPABASE_JWKS_URL`（`APP_ENV=production` 時必填，否則 API 啟動即失敗）
   - `SUPABASE_PROJECT_REF`（驗證 JWT issuer 用）
   - `ALLOWED_ORIGINS`（`APP_ENV=production` 時必填，否則 API 啟動即失敗；逗號分隔的前端網域白名單，例如 `https://ltc-system-inky.vercel.app`，用於 CORS）
   - 選填：`STORAGE_BUCKET`、`STORAGE_SIGNED_URL_TTL`、`GOOGLE_SA_JSON`、`RESEND_API_KEY`、`NOTIFY_FROM`、`SENTRY_DSN`、`LOG_LEVEL`
5. migration job 使用 Supabase Transaction Pooler 的連線字串（`postgres://postgres.<project-ref>:<password>@aws-0-<region>.pooler.supabase.com:6543/postgres?sslmode=require`），並保留 `sslmode=require`。

Migration job 成功後才會部署 API；migration 失敗時 workflow 會停止，不會更新 API service。

## Vercel 前置條件

因為這個 repo 掛在 GitHub Organization（`LTCF-Job`）底下，Vercel Hobby（免費版）的原生 Git Integration 無法直接匯入 org 底下的 repo（會被要求升級付費 Team）。因此前端改用 **Vercel CLI 手動部署**，由 `.github/workflows/deploy-web.yml` 在 `main` 分支 CI 成功後自動觸發，不使用 Vercel Git Integration。

### 一次性設定（本機執行）

1. 安裝並登入 CLI：`npm install -g vercel` → `vercel login`
2. 在 `apps/web` 目錄執行 `vercel link`，依提示建立或連結 Vercel 專案（選個人帳號 scope，不要選 Team）
3. `vercel link` 完成後會產生 `apps/web/.vercel/project.json`，裡面的 `orgId`、`projectId` 分別對應下方 GitHub Secrets 的 `VERCEL_ORG_ID`、`VERCEL_PROJECT_ID`
4. 到 Vercel Dashboard 該專案的 **Settings → Environment Variables**，針對 Production 環境設定：
   - `VITE_API_BASE_URL`（後端 Cloud Run API 的完整網址）
5. 到 Vercel Dashboard **Account Settings → Tokens** 建立一組 Personal Access Token，作為 GitHub Secrets 的 `VERCEL_TOKEN`

### GitHub 設定

在 repository 的 **Settings → Secrets and variables → Actions → Secrets** 新增：

| 名稱 | 內容 |
|---|---|
| `VERCEL_TOKEN` | Vercel Personal Access Token |
| `VERCEL_ORG_ID` | `apps/web/.vercel/project.json` 裡的 `orgId` |
| `VERCEL_PROJECT_ID` | `apps/web/.vercel/project.json` 裡的 `projectId` |

`apps/web/.vercel/` 已加入 `.gitignore`，不會被提交到 git。

## Supabase GitHub Integration 注意事項

本專案目前不是 Supabase CLI migration 結構，因此不應直接開啟 Supabase 的 migration 自動部署來處理 `apps/api/migrations`。`main` 的 CI 成功後，會透過 Cloud Run migration job 執行 `apps/api/cmd/migrate`，使用 `DATABASE_URL` 連到 Supabase Transaction Pooler；migration 成功後才更新 API service。若未來要改用 Supabase CLI，必須先完整遷移至 `supabase/migrations`，並重新核對遠端 migration history。
