# CI/CD 部署操作說明

給要部署 `apps/api`（Cloud Run）或 `apps/web`（Vercel）的人看。涵蓋兩邊各自需要哪些環境變數、GitHub Actions 自動部署怎麼串、以及手動部署／除錯指令。與程式碼技術細節相關的說明見 [`README.md`](README.md)；分層邊界與程式碼風格見 [`AGENTS.md`](../../AGENTS.md)。

## 服務總覽

| 服務 | 平台 | 用途 |
|---|---|---|
| `apps/api` | Google Cloud Run（service `ltc-api` + migration job `ltc-api-migrate`），region `asia-east1` | Go 後端 API |
| `apps/web` | Vercel（project `ltc-system`） | Vue 3 前端靜態站台 |
| Supabase 專案 `oywacuduaiulnfxzmpxs` | Supabase | PostgreSQL 資料庫、Auth（GoTrue）、Storage |

正式環境（`main` 分支）與 develop 環境（`develop` 分支）目前共用**同一個** Supabase 專案；沒有各自獨立的資料庫。

## Supabase（資料庫與 Auth）

### 跑 migration

```bash
cd apps/api
make migrate-up      # 或 go run ./cmd/migrate up
make migrate-down    # 回滾最新一支
```

`DATABASE_URL` 要指向 Supabase 的連線池網址（port 6543，pgbouncer transaction pooling）。

### 已知坑：手動塞 `auth.users` 一定要補 `auth.identities`

Supabase 目前版本的 GoTrue 驗證 email/password 登入時，除了比對 `auth.users.encrypted_password`，還要求該使用者在 `auth.identities` 有一筆 `provider = 'email'` 的對應紀錄。只 `INSERT INTO auth.users` 不補 `auth.identities`，密碼即使完全正確，`signInWithPassword` 也會回 `Invalid login credentials`。

`auth.identities.email` 是唯讀的 generated column，INSERT／UPDATE 都不能寫這欄，寫了會噴 `column "email" can only be updated to DEFAULT`；`provider_id` 用 `auth.users.id` 的文字值即可。範例見 [`../../apps/api/migrations/000002_seed_reference_data.up.sql`](../../apps/api/migrations/000002_seed_reference_data.up.sql)。

### 已知坑：pgx 預設協定在 Supabase pgbouncer 下會併發撞名

`DATABASE_URL` 走的是 pgbouncer transaction pooling，同一個連線池物件的不同請求可能被路由到不同後端連線。`jackc/pgx/v5` 預設用 extended protocol，會替每個 SQL 語句快取一個 prepared statement 名稱；在 transaction pooling 下，並發量稍高就會出現：

```
ERROR: prepared statement "stmtcache_xxxx" already exists (SQLSTATE 42P05)
```

實測：40 個並發查詢，預設協定下 32 個失敗（80%），改用 simple protocol 後 0 個失敗。`apps/api/cmd/server/main.go` 與 `apps/api/cmd/migrate/main.go` 都已經在建立 `pgxpool` 時強制指定：

```go
poolCfg, _ := pgxpool.ParseConfig(cfg.DatabaseURL)
poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
pool, _ := pgxpool.NewWithConfig(ctx, poolCfg)
```

日後任何新增的入口（例如獨立的 worker、one-off script）只要用同一個 `DATABASE_URL` 連 Supabase pooler，都要照這個寫法，不能直接 `pgxpool.New(ctx, dsn)`。

## `apps/api` 環境變數

| 變數 | 本機 `.env` | Cloud Run | 說明 |
|---|---|---|---|
| `PORT` | `8080` | Cloud Run 自動注入，不用設 | HTTP 監聽埠 |
| `APP_ENV` | `local` | `production` | `production` 時會強制要求 `SUPABASE_JWKS_URL` 與 `ALLOWED_ORIGINS`，否則直接拒絕啟動 |
| `DATABASE_URL` | Supabase 連線池網址 | 同左，存在 Secret Manager | 見上方 pgbouncer 說明 |
| `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` | `5` / `2` | 同左 | 目前設定值有讀進 config，但尚未接到 `pgxpool` 的 `MaxConns`／`MinConns`，實際連線數上限吃 pgx 預設值，非本次修復範圍 |
| `ENCRYPTION_KEY` / `HMAC_KEY` | 32 bytes base64 | 同左，存在 Secret Manager | 個案身分證等敏感欄位加密用 |
| `SUPABASE_JWKS_URL` | 可留空（本機不驗簽） | `https://<project-ref>.supabase.co/auth/v1/.well-known/jwks.json` | `production` 必填 |
| `SUPABASE_PROJECT_REF` | Supabase 專案 ref | 同左 | |
| `ALLOWED_ORIGINS` | 不需要（`local` 時 CORS 全開） | 逗號分隔的網域清單 | `production` 必填，見下方常見錯誤 |
| `STORAGE_BUCKET` | `ltc-exports` | 同左 | |
| `STORAGE_SIGNED_URL_TTL` | `24h` | 同左 | |
| `RESEND_API_KEY` / `NOTIFY_FROM` | 可留空 | 通知信件用 | |
| `SENTRY_DSN` | 可留空 | 錯誤追蹤 | |
| `LOG_LEVEL` | `info` | 同左 | |

## `apps/web` 環境變數（Vite，`VITE_` 前綴才會被打包進前端）

| 變數 | 說明 |
|---|---|
| `VITE_API_BASE_URL` | 後端 API base URL。本機用 `/api/v1`（走 dev server proxy），部署環境要填完整網址，例如 `https://ltc-api-<hash>.<region>.run.app/api/v1` |
| `VITE_SUPABASE_URL` | Supabase 專案網址，例如 `https://oywacuduaiulnfxzmpxs.supabase.co` |
| `VITE_SUPABASE_ANON_KEY` | Supabase anon key（公開金鑰，非機密） |
| `VITE_GOOGLE_CLIENT_ID` / `VITE_GOOGLE_API_KEY` / `VITE_GOOGLE_APP_ID` | Google Picker／Identity Services，選填 |
| `VITE_ENABLE_MSW` | 是否啟用 mock service worker，部署環境一律 `false` |

`VITE_SUPABASE_URL`／`VITE_SUPABASE_ANON_KEY` 沒設定時，[`apps/web/src/lib/supabase.ts`](../../apps/web/src/lib/supabase.ts) 會讓 `supabase` client 維持 `null`；登入頁看到 `!supabase` 就會直接顯示「帳號密碼錯誤或無此使用者」，不會真的呼叫 Supabase Auth，容易誤判成帳密問題。

### 已知坑：Vercel Preview 環境變數要另外設

`vercel env ls` 顯示的環境變數清單是分 Production／Preview／Development 各自獨立設定的。只在 Production 設過 `VITE_API_BASE_URL` 不代表 Preview（例如 `*-git-develop-*.vercel.app` 這種分支預覽網址）也有——曾經發生過 Preview 完全沒設 Supabase 相關變數，導致 develop 分支預覽網址永遠無法用真實帳密登入。

新增或檢查 Preview 環境變數：

```bash
cd apps/web
vercel env ls
vercel env add VITE_SUPABASE_URL preview --no-sensitive --value 'https://oywacuduaiulnfxzmpxs.supabase.co'
vercel env add VITE_SUPABASE_ANON_KEY preview --no-sensitive --value '<anon key>'
vercel env add VITE_API_BASE_URL preview --no-sensitive --value 'https://<cloud-run-url>/api/v1'
```

`VITE_` 開頭的變數一定會被打包進前端 bundle、任何訪客都看得到，`vercel env add` 對這種變數不允許用 `--sensitive`，一律用 `--no-sensitive`。

### 已知坑：`vercel deploy` 不會自動更新分支預覽網址

用 `vercel env add` 改完 Preview 環境變數後，既有的 build 產物不會自動重新打包——環境變數是**建置期**注入的，一定要重新觸發一次 build。

`vercel deploy --prebuilt` 這個旗標是「有出現就生效」，`--prebuilt=false` 不會真的關掉它，反而會因為找不到本機預建置產物而報錯。要雲端重新建置，直接不要加這個旗標：

```bash
vercel deploy
```

但這樣建出來的是一個全新、隨機雜湊網址的 Preview 部署（例如 `ltc-system-<hash>-<team>.vercel.app`），**不會**自動指到 `ltc-system-git-<branch>-<team>.vercel.app` 這個固定的分支別名——那個別名只有透過 GitHub 原生 Git 整合（push 觸發）的部署才會自動更新；這個專案走的是 GitHub Actions＋CLI 部署（不是原生 Git 整合），沒有這個機制。

`deploy-web.yml` 已經處理過這件事：`Deploy prebuilt output` 那步把 `vercel deploy` 印出的網址存成 step output，接著的 `Update branch alias` 步驟會自動 `vercel alias set` 到 GitHub Environment variable `VERCEL_ALIAS_DOMAIN`（`production` 分支走 `--prod`、會自動套用專案設定的 Production Domains，不需要這個變數）。沒設這個變數就跳過，不影響部署本身成功與否。手動用 CLI 部署、或這個自動化本身故障時才需要手動補：

```bash
vercel alias set https://ltc-system-<hash>-<team>.vercel.app ltc-system-git-<branch>-<team>.vercel.app
```

### 已知坑：`vercel build` 在 CI／本機不會自動把 `node_modules/.bin` 加進 PATH

`vercel build` 在 Vercel 平台上跑時，build container 本身就把 `node_modules/.bin` 放進 PATH；但透過 CLI 在本機或 GitHub Actions 執行 `vercel build`（[`deploy-web.yml`](../../.github/workflows/deploy-web.yml) 就是這樣用），偵測到的 build command（這個專案是 `vite build`）會直接當一般 shell 指令執行，不會像 `npm run build` 那樣自動把 `node_modules/.bin` 加進 PATH，即使 `npm ci` 已經裝好依賴，還是會報：

```
sh: 1: vite: not found
Error: Command "vite build" exited with 127
```

`deploy-web.yml` 已經在呼叫 `vercel build` 前手動 `export PATH="$PWD/node_modules/.bin:$PATH"` 處理這個問題；日後若改寫這段 workflow，記得保留這一行，不要因為「本機測試時 `vite` 是全域指令所以正常」而誤刪。

## `ltc-api`（Cloud Run）常見部署操作

### 查目前設定（唯讀）

```bash
gcloud run services describe ltc-api --region=asia-east1 \
  --format="value(spec.template.spec.containers[0].env)"
```

### 改環境變數

```bash
gcloud run services update ltc-api --region=asia-east1 \
  --update-env-vars="KEY1=value1,KEY2=value2"
```

### 已知坑：值本身含逗號時要換分隔字元

`--update-env-vars` 預設用逗號分隔多個 `KEY=VALUE`。像 `ALLOWED_ORIGINS` 這種值本身就是逗號分隔網域清單的變數，直接這樣寫會被誤判成要設第二個環境變數：

```bash
# 錯誤：gcloud 會把第二個網址當成新的 KEY=VALUE 解析，報 Bad syntax for dict arg
--update-env-vars="ALLOWED_ORIGINS=https://a.example.com,https://b.example.com"
```

用 `^分隔字元^` 語法把分隔字元從逗號改成別的字元（例如 `;`），並且整段值要用引號包起來（避免 PowerShell 把 `;` 當成指令分隔符號處理）：

```bash
gcloud run services update ltc-api --region=asia-east1 \
  --update-env-vars="^;^ALLOWED_ORIGINS=https://a.example.com,https://b.example.com"
```

`ALLOWED_ORIGINS` 要包含**每一個**會呼叫這支 API 的前端網域，包括 Vercel 的分支預覽網址（`*-git-<branch>-*.vercel.app`）——測試 develop 分支功能時如果後端一直 CORS 擋掉，先檢查這裡有沒有漏放。

### 手動部署（source-based deploy）

正常情況應該讓下面「GitHub Actions 自動部署」那條路徑處理部署；只有在自動部署本身壞掉、或要在 GitHub Actions 跑完前先手動驗證修正時，才用這個指令暫時頂上：

```bash
gcloud run deploy ltc-api --source apps/api --region asia-east1 --project <GCP_PROJECT_ID>
```

這個指令只會重新建置並部署 API service，**不會**執行 database migration；改了 migration 檔案要另外自己跑 `make migrate-up`（見上方 Supabase 段落），或用下面的 migration job。

### migration job（`ltc-api-migrate`）

```bash
gcloud run jobs execute ltc-api-migrate --region=asia-east1 --wait
```

這個 job 是為了讓 GitHub Actions 部署流程能在部署新版 API 之前自動跑 migration（見下方），手動 source-based deploy 不會觸發它，需要的話要自己執行。

### 已知坑：migration job 跟 API service 要各自設定同一套 production 必填變數

`cmd/migrate` 跟 `cmd/server`（`ltc-api` 服務本體）共用同一套設定驗證（`internal/platform/config`）：`APP_ENV=production` 時少了 `SUPABASE_JWKS_URL` 或 `ALLOWED_ORIGINS` 任何一個都會直接拒絕啟動。這兩個環境變數（以及 `APP_ENV`、`SUPABASE_PROJECT_REF`）**migration job 跟 API service 是各自獨立的環境變數集合**，只在 `ltc-api` 服務上設定過不代表 `ltc-api-migrate` job 也有——曾經發生過 job 只設了 `DATABASE_URL` 這個 secret，`APP_ENV` 從未設定，實際跑起來因為 `config.LoadFromEnv()` 沒驗證過就直接把整包環境變數丟給 job，結果是不知道哪來的舊設定殘留了 `APP_ENV=develope`（打錯字，不是 `develop` 也不是 `production`），導致 `gcloud run jobs execute` 每次都以 `Failed to load config` 失敗，連帶讓 `deploy-api.yml` 卡在「Run database migrations」那步。

`ALLOWED_ORIGINS` 在 migration job 上只是為了通過設定檢查，job 不會真的處理 HTTP 請求，填什麼網域都不影響功能。用下面指令核對兩邊變數是否一致：

```bash
gcloud run jobs describe ltc-api-migrate --region=asia-east1 --format="value(spec.template.spec.template.spec.containers[0].env)"
gcloud run services describe ltc-api --region=asia-east1 --format="value(spec.template.spec.containers[0].env)"
```

### 已知坑：Windows Git Bash 會把 `--command` 的路徑參數轉換成 Windows 路徑

在 Windows 用 Git Bash 執行 `gcloud run jobs update ltc-api-migrate --command="/app/migrate"` 這類指令時，Git Bash（MSYS）會自動把看起來像 Unix 絕對路徑的參數轉換成 Windows 路徑，實際送給 gcloud 的值會變成 `D:/Program Files/Git/app/migrate` 這種在 Linux 容器裡不存在的路徑，job 執行時會直接 `Application failed to start`，且錯誤訊息完全看不出是路徑被轉換，只會看到 `terminated: Application failed to start`。

```bash
gcloud run jobs describe ltc-api-migrate --region=asia-east1 --format="value(spec.template.spec.template.spec.containers[0].command)"
```

查出來若是一串 `D:/...` 就是中招了。解法：把值前面多加一個斜線變成 `--command="//app/migrate"`（雙斜線），Git Bash 就不會轉換；Linux 容器內部會把 `//app/migrate` 正常解析成 `/app/migrate`。改用 PowerShell 或 cmd.exe 執行則不會有這個問題，直接填單斜線即可。

## GitHub Actions 自動部署（`develop`／`main`）

`deploy-api.yml` 與 `deploy-web.yml` 各自直接由 `push` 到 `develop`／`main` 觸發（也可以用 `workflow_dispatch` 手動觸發），每個檔案內都有一個 `test` job（vet／test／build 或 type-check／build）跑完才會進 `deploy` job；`ci.yml` 只在 PR 上跑，不重複跑 push 的測試。

`deploy-api.yml` 的 `deploy` job 依序：`gcloud builds submit` 建 image → 更新 `ltc-api-migrate` job 的 image → `gcloud run jobs execute` 跑 migration（`--wait`，失敗會擋住下一步）→ `gcloud run deploy` 部署 API service。

`deploy-web.yml` 的 `deploy` job：`npm ci` → `vercel pull` → `vercel build` → `vercel deploy --prebuilt`，依分支決定 `--prod` 與否。

### 已知坑：`workflow_run` 讀的是 default branch 上的工作流程檔，不是觸發它的分支

之前 `deploy-api.yml`／`deploy-web.yml` 是用 `workflow_run` 接在 `ci.yml` 後面觸發，這個模式有一個不明顯的陷阱：GitHub Actions 執行 `workflow_run` 觸發的工作流程時，**用的是 repo default branch 上那份工作流程定義檔**，不是 `head_branch`（觸發它的那個分支）上的版本。這個 repo 的 default branch 是 `main`；`deploy-api.yml` 曾經在 `develop` 分支上修好一個 `gcloud builds submit` 串流 log 失敗的 bug，但因為沒回合併到 `main`，`workflow_run` 觸發時永遠讀到 `main` 上沒修過的舊版，導致不管從哪個分支 push，部署都用同一份壞掉的邏輯失敗。

現在改成 `push` 觸發：`push` 事件用的工作流程檔案就是**這次要部署的那個 commit 自己的版本**，不會有這個問題，但代價是這兩個檔案本身的修改也要各自進到 `develop`／`main` 才會對該分支生效——不能只改 `develop` 就期待 `main` 的部署也吃到。

### 已知坑：GitHub Environment 名稱要跟 workflow 裡的字串完全一致（大小寫敏感）

`deploy-api.yml`／`deploy-web.yml` 的 `environment:` 欄位用 `github.ref_name == 'main' && 'Production' || 'develop'` 決定要用哪個 GitHub Environment 的 secrets／variables。這個 repo 建立的 Environment 名稱是 `develop`、`Preview`、`Production`——注意是 `Production`（大寫開頭），不是 `production`。曾經因為 workflow 裡寫成小寫 `production`，導致 push 到 `main` 時 GitHub 自動建立一個全新、空的 `production` Environment（沒有任何 secret），部署卡在「Authenticate to Google Cloud」那步失敗。改 `environment:` 欄位或新增 Environment 時，兩邊名稱要逐字比對。

需要的 secrets／variables，**GCP 相關的是設在 `develop`／`Production` 兩個 GitHub Environment 底下**（不是 repo 層級），Vercel 相關的是 repo 層級（不分環境）：

| 名稱 | 類型 | 設定位置 | 用途 |
|---|---|---|---|
| `GCP_PROJECT_ID` / `GCP_REGION` / `ARTIFACT_REPOSITORY` | Variable | Environment | Cloud Build／Artifact Registry 位置 |
| `API_SERVICE`（預設 `ltc-api`）/ `MIGRATION_JOB`（預設 `ltc-api-migrate`） | Variable | Environment | 服務／job 名稱 |
| `GCP_WIF_PROVIDER` / `GCP_DEPLOY_SA` | Secret | Environment | Workload Identity Federation 部署身分 |
| `VERCEL_TOKEN` / `VERCEL_ORG_ID` / `VERCEL_PROJECT_ID` | Secret | Repo | Vercel CLI 部署憑證 |
| `VERCEL_ALIAS_DOMAIN` | Variable | Environment | 非 production 分支部署完自動 `vercel alias set` 的目標網域，見下方已知坑；沒設定就跳過這步 |

從零設定這些值（含 GCP service account、Workload Identity Federation、Artifact Registry 怎麼建）見 [`environment-bootstrap.md`](environment-bootstrap.md)。

## 部署後檢查清單

1. `curl -i https://<cloud-run-url>/api/v1/healthz`（若有健康檢查端點）或直接打一個需要認證的端點確認回 401 而不是 500／連不上。
2. 用實際帳密在目標網域登入一次，不要只信任「畫面沒有紅字」——CORS 失敗、`supabase` client 為 `null` 都不會讓瀏覽器整頁報錯，要看 DevTools console 有沒有 CORS 或網路錯誤。
3. 前端瀏覽器對同一批 API 快速觸發多個並發請求（例如快速切換好幾個選單頁面），確認沒有隨機出現的 500——這類 prepared statement 撞名的 bug 在低併發下不容易重現。
4. 若剛執行過 `vercel env add` 或改過 Cloud Run 環境變數，記得變數是**建置期**／**啟動時**生效，一定要有一次新的 build／新的 revision 才會套用，不能只改設定就期待既有部署自動吃到。
