# 從零建置新環境

給要把這個專案部署到一個全新環境（新的 Supabase 專案、新的 GCP 專案、新的 Vercel 專案）的人看，照順序做完就能讓 `git push` 到 `develop`／`main` 自動部署。日常維運（改環境變數、手動部署、常見坑）見 [`deployment.md`](deployment.md)；這份文件只管「從零開始」那一次性的建置流程。

## 兩種操作方式

每個步驟都同時列出兩種做法，挑一種跟到底即可（也可以每步各自挑，兩種做法設定完的結果是一樣的）：

- **CLI 版**：全程用 `gcloud`／`vercel`／`gh` 指令，適合要重複建置多個環境、或想把步驟存成腳本的人。
- **頁面版**：全程用瀏覽器點 Supabase／GCP Console／Vercel／GitHub 的網頁介面，適合不想安裝多套 CLI、只建置一次的人。

不管選哪一種，設定完成後**往後每次 `git push` 都是全自動部署**，兩種做法都不會留下「以後每次部署都要手動做」的殘留步驟——差別只在「第一次建置」這一次要不要開終端機打指令。

Step 5（Vercel）的頁面版額外提供一種能再省掉 GitHub Actions 這一段的做法，見該步驟說明。

## 前置需求

**CLI 版**本機要有：`gcloud`（已登入且有目標 GCP 專案的 Owner 或等同權限）、`vercel` CLI（已登入）、`gh` CLI（已登入且對目標 repo 有 admin 權限，用來設定 Environments／secrets）、`go`、`node`／`npm`。

**頁面版**只要有瀏覽器與各平台的登入帳號（Supabase、Google Cloud Console、Vercel、GitHub），本機仍需要 `go`（跑一次性的資料庫 migration 工具）；若改用頁面版 Step 1 的 SQL Editor 做法則連 `go` 都不需要。

需要先有：一個 Supabase 專案、一個 GCP 專案（已啟用計費）、一個 GitHub repo（本專案的 fork 或原 repo）。

## Step 1：Supabase

到 [supabase.com](https://supabase.com) 建一個新專案（頁面版與 CLI 版共用這步，Supabase 本身沒有對應的官方建置 CLI 指令），記下：

- 專案 URL：`https://<project-ref>.supabase.co`
- 專案 ref（`<project-ref>` 那段）
- anon key（Settings → API）
- 連線池（Connection Pooling）DATABASE_URL，port `6543`，transaction mode

### CLI 版：跑 migration

```bash
cd apps/api
DATABASE_URL='<連線池 URL>' APP_ENV=local go run ./cmd/migrate up
```

`APP_ENV=local` 只是為了跳過 `production` 模式底下 `SUPABASE_JWKS_URL`／`ALLOWED_ORIGINS` 必填的檢查，migration 本身不受影響。

### 頁面版：用 SQL Editor 跑 migration

Supabase Dashboard 左側選單 → SQL Editor → New query，依檔名數字順序，把 [`../../apps/api/migrations/`](../../apps/api/migrations/) 底下每一個 `*.up.sql` 貼進去執行（例如 `000001_xxx.up.sql`、`000002_seed_reference_data.up.sql`……依序往下），每貼一個就按 Run，確認沒有錯誤再貼下一個。這個做法完全不需要本機裝 `go`，缺點是沒有 `schema_migrations` 版本紀錄表自動維護，之後要接 CLI 版的 `go run ./cmd/migrate up` 前，需自行確認兩邊 schema 狀態一致。

不論哪種做法，`000002_seed_reference_data.up.sql` 都會種一個預設管理員帳號（`ltcf-admin@ltc.example.com`），同時建好 `auth.users` 與對應的 `auth.identities`（缺其中一個都會導致密碼正確也無法登入，見 [`deployment.md`](deployment.md) 的已知坑）。要換帳密就直接改這支 migration 裡的 bcrypt hash，或改用 Supabase Dashboard 的 Auth 介面另外建帳號（另建帳號時要自行處理 `auth.identities` 對應，否則會遇到同一個已知坑）。

## Step 2：GCP 專案、Artifact Registry、Service Account、Workload Identity Federation

以下 CLI 版指令假設你已經 `gcloud config set project <GCP_PROJECT_ID>`；頁面版操作前先在 [Google Cloud Console](https://console.cloud.google.com) 右上角選好目標專案。

### 啟用必要 API

**CLI 版：**

```bash
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  iamcredentials.googleapis.com \
  secretmanager.googleapis.com
```

**頁面版：** Console 左上選單 →「API 和服務」→「程式庫」，依序搜尋並個別點「啟用」：Cloud Run Admin API、Cloud Build API、Artifact Registry API、IAM Service Account Credentials API、Secret Manager API。

### Artifact Registry（存放 Docker image）

**CLI 版：**

```bash
gcloud artifacts repositories create cloud-run-source-deploy \
  --repository-format=docker \
  --location=<GCP_REGION>
```

**頁面版：** Console →「Artifact Registry」→「儲存庫」→「建立存放區」，名稱填 `cloud-run-source-deploy`，格式選 Docker，模式選「標準」，區域選 `<GCP_REGION>`（要跟後面 Cloud Run 服務同一個區域）。

`ARTIFACT_REPOSITORY` 這個名字沒有強制規定，跟 GitHub variable `ARTIFACT_REPOSITORY` 對上即可。

### 部署用 Service Account

**CLI 版：**

```bash
gcloud iam service-accounts create github-actions-deployer \
  --display-name="GitHub Actions Deployer"

SA_EMAIL="github-actions-deployer@<GCP_PROJECT_ID>.iam.gserviceaccount.com"

for ROLE in \
  roles/artifactregistry.writer \
  roles/cloudbuild.builds.editor \
  roles/iam.serviceAccountUser \
  roles/logging.viewer \
  roles/run.admin \
  roles/secretmanager.secretAccessor \
  roles/storage.admin
do
  gcloud projects add-iam-policy-binding <GCP_PROJECT_ID> \
    --member="serviceAccount:${SA_EMAIL}" \
    --role="$ROLE"
done
```

**頁面版：** Console →「IAM 與管理」→「服務帳戶」→「建立服務帳戶」，名稱填 `github-actions-deployer`，顯示名稱填「GitHub Actions Deployer」，直接按「完成」（不用在建立精靈裡加角色）。建立好後到「IAM 與管理」→「IAM」，找到剛剛這個服務帳戶（`github-actions-deployer@<GCP_PROJECT_ID>.iam.gserviceaccount.com`）→ 點鉛筆圖示編輯 → 逐一「新增另一個角色」，加滿以下七個角色後儲存：

- Artifact Registry 寫入者（`roles/artifactregistry.writer`）
- Cloud Build 編輯者（`roles/cloudbuild.builds.editor`）
- 服務帳戶使用者（`roles/iam.serviceAccountUser`）
- Logging 檢視者（`roles/logging.viewer`）
- Cloud Run 管理員（`roles/run.admin`）
- Secret Manager 密鑰存取者（`roles/secretmanager.secretAccessor`）
- Storage 管理員（`roles/storage.admin`）

`roles/logging.viewer` 不是嚴格必要（`deploy-api.yml` 撈 build log 那步已經用 `|| true` 容錯），但建議還是加，串流 log 才看得到。

### Workload Identity Federation（讓 GitHub Actions 免存 GCP 金鑰就能登入）

**CLI 版：**

```bash
gcloud iam workload-identity-pools create github-pool \
  --location=global \
  --display-name="GitHub Actions Pool"

gcloud iam workload-identity-pools providers create-oidc github-provider \
  --location=global \
  --workload-identity-pool=github-pool \
  --display-name="GitHub Provider" \
  --issuer-uri="https://token.actions.githubusercontent.com" \
  --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.repository_owner=assertion.repository_owner,attribute.ref=assertion.ref" \
  --attribute-condition="assertion.repository_owner == '<GITHUB_ORG_OR_USER>'"

gcloud iam service-accounts add-iam-policy-binding "$SA_EMAIL" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/<GCP_PROJECT_NUMBER>/locations/global/workloadIdentityPools/github-pool/attribute.repository/<GITHUB_ORG_OR_USER>/<GITHUB_REPO_NAME>"
```

**頁面版：** Console →「IAM 與管理」→「Workload Identity 聯盟」→「建立集區」，名稱填 `github-pool`。建好集區後在集區內「新增提供者」，提供者類型選 OpenID Connect (OIDC)，提供者名稱填 `github-provider`，發行者 (Issuer) URL 填 `https://token.actions.githubusercontent.com`；「屬性對應」區塊逐筆加入：

| Google 屬性 | OIDC 宣告值 |
|---|---|
| `google.subject` | `assertion.sub` |
| `attribute.repository` | `assertion.repository` |
| `attribute.repository_owner` | `assertion.repository_owner` |
| `attribute.ref` | `assertion.ref` |

「屬性條件」填 `assertion.repository_owner == '<GITHUB_ORG_OR_USER>'`。存檔後回到「服務帳戶」頁面，點進 `github-actions-deployer`，切到「權限」頁籤 →「授予存取權」，主體貼上：

```
principalSet://iam.googleapis.com/projects/<GCP_PROJECT_NUMBER>/locations/global/workloadIdentityPools/github-pool/attribute.repository/<GITHUB_ORG_OR_USER>/<GITHUB_REPO_NAME>
```

角色選「Workload Identity 使用者」。

> 這一步頁面操作的欄位多、字串（屬性對應、屬性條件、主體字串）容易打錯或漏字，就算其餘步驟都走頁面版，這一步也建議直接複製上面 CLI 版的指令貼到終端機執行，出錯機率低很多。

`<GCP_PROJECT_NUMBER>` 是專案編號（純數字），不是專案 ID；CLI 用 `gcloud projects describe <GCP_PROJECT_ID> --format="value(projectNumber)"` 查，頁面版在 Console 首頁儀表板卡片或「IAM 與管理」→「設定」都能看到。`GCP_WIF_PROVIDER` 這個 GitHub secret 的值就是：

```
projects/<GCP_PROJECT_NUMBER>/locations/global/workloadIdentityPools/github-pool/providers/github-provider
```

`GCP_DEPLOY_SA` 這個 secret 的值就是 `$SA_EMAIL`。

## Step 3：Cloud Run 服務與 migration job（第一次手動建立）

GitHub Actions 只會「更新」既有的 service／job，第一次要手動建。先準備一份 `.env`（不進版控），內容參考 [`../../apps/api/.env.example`](../../apps/api/.env.example)，`ALLOWED_ORIGINS` 先留空稍後再補（要等 Vercel 網域確定）。

**CLI 版：**

```bash
cd apps/api
gcloud run deploy ltc-api --source . --region <GCP_REGION>
```

第一次跑會用互動問答問你要不要允許未驗證存取（選否，前端一律帶 JWT）。跑完後在 Cloud Run 主控台或用 `gcloud run services update` 補齊環境變數（見下方）。

再建 migration job（讓 GitHub Actions 之後每次部署都能自動跑 migration）：

```bash
gcloud run jobs create ltc-api-migrate \
  --region <GCP_REGION> \
  --image <GCP_REGION>-docker.pkg.dev/<GCP_PROJECT_ID>/cloud-run-source-deploy/api:bootstrap \
  --command="/app/migrate" \
  --args="up"
```

image 先隨便給一個佔位值，GitHub Actions 的「Update migration job image」那步會在每次部署時換成最新 tag，這裡建立時的 image 只是讓 job 能被建立起來。指令要填 `/app/migrate`（對應 [`../../apps/api/Dockerfile`](../../apps/api/Dockerfile) 裡 `COPY --from=builder /app/bin/migrate /app/migrate`），不是 `./migrate`——容器裡沒有這個相對路徑，實測會直接 `Application failed to start`。

> **Windows 底下用 Git Bash／MINGW 跑這行指令要注意**：Git Bash 會自動把看起來像 Unix 路徑的參數（`/app/migrate`）轉換成 Windows 路徑，實際送給 gcloud 的值會變成類似 `D:/Program Files/Git/app/migrate` 這種亂碼，job 會一樣啟動失敗且不容易看出原因。遇到這狀況把 `--command` 的值前面多加一個斜線變成 `--command="//app/migrate"`（雙斜線），Git Bash 就不會轉換，Linux 容器內部會把 `//app/migrate` 正常解析成 `/app/migrate`。用 PowerShell 或 cmd.exe 執行則不會有這個問題，直接填單斜線 `/app/migrate` 即可。

**頁面版：**

Cloud Run 服務用「建立服務」精靈跑本機原始碼比較麻煩（精靈本身不支援直接上傳資料夾建置），實務上比較快的頁面做法是：先用任一佔位 image 把服務與 job 建起來（跟 CLI 版一樣，image 只是佔位、後面 GitHub Actions 會覆蓋），環境變數等 Step 4 再補：

1. Console →「Cloud Run」→「服務」→「部署容器」，服務名稱填 `ltc-api`，容器映像檔網址先填 `us-docker.pkg.dev/cloudrun/container/hello`（Google 官方佔位 image），區域選 `<GCP_REGION>`，「驗證」選「需要驗證」（不允許未驗證存取，前端一律帶 JWT），其餘保持預設按「建立」。
2. Console →「Cloud Run」→「工作」→「部署容器」，工作名稱填 `ltc-api-migrate`，容器映像檔網址一樣先填上面那個佔位 image，「容器、變數與密鑰、連線、安全性」區塊展開「容器」頁籤，指令欄填 `/app/migrate`，引數欄填 `up`，區域選 `<GCP_REGION>`，按「建立」。

兩者的 image 之後都會被 GitHub Actions 部署時自動換成正式 build 出來的版本，這裡佔位 image 跑不跑得起來不影響後續（job 甚至會因為 `/app/migrate` 在佔位 image 裡不存在而執行失敗，這是預期行為，不用理它，等 GitHub Actions 換過 image 後就正常了）。

## Step 4：Cloud Run 環境變數與 Secret Manager

機密值（`DATABASE_URL`／`ENCRYPTION_KEY`／`HMAC_KEY`）放 Secret Manager，其餘用一般環境變數。`ENCRYPTION_KEY`／`HMAC_KEY` 要是 32 bytes 的 base64：`openssl rand -base64 32`。

**CLI 版：**

```bash
echo -n '<DATABASE_URL>'   | gcloud secrets create DATABASE_URL   --data-file=-
echo -n '<ENCRYPTION_KEY>' | gcloud secrets create ENCRYPTION_KEY --data-file=-
echo -n '<HMAC_KEY>'       | gcloud secrets create HMAC_KEY       --data-file=-

gcloud run services update ltc-api --region <GCP_REGION> \
  --set-secrets="DATABASE_URL=DATABASE_URL:latest,ENCRYPTION_KEY=ENCRYPTION_KEY:latest,HMAC_KEY=HMAC_KEY:latest" \
  --update-env-vars="APP_ENV=production,SUPABASE_PROJECT_REF=<project-ref>,SUPABASE_JWKS_URL=https://<project-ref>.supabase.co/auth/v1/.well-known/jwks.json,STORAGE_BUCKET=ltc-exports,STORAGE_SIGNED_URL_TTL=24h,LOG_LEVEL=info,ALLOWED_ORIGINS=https://placeholder.example.com"

gcloud run jobs update ltc-api-migrate --region <GCP_REGION> \
  --set-secrets="DATABASE_URL=DATABASE_URL:latest" \
  --update-env-vars="APP_ENV=production,SUPABASE_PROJECT_REF=<project-ref>,SUPABASE_JWKS_URL=https://<project-ref>.supabase.co/auth/v1/.well-known/jwks.json,ALLOWED_ORIGINS=https://placeholder.example.com"
```

`cmd/migrate` 跟 `cmd/server` 共用同一套設定驗證（[`../../apps/api/internal/platform/config`](../../apps/api/internal/platform/config)）：`APP_ENV=production` 時 `SUPABASE_JWKS_URL`／`ALLOWED_ORIGINS` 沒填會直接拒絕啟動，**migration job 跟 API service 兩邊都要各自設好這幾個變數**，只設 API service 那邊、忘了 job 這邊，migration job 會在 `Failed to load config` 直接失敗、`gcloud run jobs execute` 回傳非 0 導致整條 GitHub Actions 卡住。`ALLOWED_ORIGINS` 在 migration job 上只是為了通過設定檢查，job 本身不會真的處理 HTTP 請求，填什麼值都不影響功能；等 Step 5 拿到真正的前端網域後，記得回頭把 API service（不是 migration job）的 `ALLOWED_ORIGINS` 換成實際網域。

**頁面版：**

1. Console →「Security」→「Secret Manager」→「建立密鑰」，重複三次，名稱分別填 `DATABASE_URL`、`ENCRYPTION_KEY`、`HMAC_KEY`，「密鑰值」欄位貼上對應的值。
2. Console →「Cloud Run」→「服務」→ 點進 `ltc-api` →「編輯並部署新修訂版本」，切到「變數與密鑰」頁籤：
   - 在「環境變數」區塊「新增變數」，逐一加入 `APP_ENV=production`、`SUPABASE_PROJECT_REF=<project-ref>`、`SUPABASE_JWKS_URL=https://<project-ref>.supabase.co/auth/v1/.well-known/jwks.json`、`STORAGE_BUCKET=ltc-exports`、`STORAGE_SIGNED_URL_TTL=24h`、`LOG_LEVEL=info`、`ALLOWED_ORIGINS=https://placeholder.example.com`（先填佔位值，Step 5 拿到 Vercel 網域後回來換成實際網域）。
   - 在「密鑰」區塊「參照密鑰」，逐一把 `DATABASE_URL`／`ENCRYPTION_KEY`／`HMAC_KEY` 三個密鑰各自掛成同名環境變數（版本選「最新」）。
   - 按「部署」。
3. Console →「Cloud Run」→「工作」→ 點進 `ltc-api-migrate` →「編輯」，在「變數與密鑰」頁籤：
   - 「環境變數」區塊加入 `APP_ENV=production`、`SUPABASE_PROJECT_REF=<project-ref>`、`SUPABASE_JWKS_URL=https://<project-ref>.supabase.co/auth/v1/.well-known/jwks.json`、`ALLOWED_ORIGINS=https://placeholder.example.com`。
   - 「密鑰」區塊掛上 `DATABASE_URL`。
   - 儲存。

`cmd/migrate` 跟 `cmd/server`（`ltc-api` 服務本體）共用同一套設定驗證，`APP_ENV=production` 時少了 `SUPABASE_JWKS_URL`／`ALLOWED_ORIGINS` 任何一個都會直接拒絕啟動——**這幾個變數 migration job 跟 API service 兩邊都要各自設，只設服務那邊會讓 migration job 用「Failed to load config」失敗，進而讓 GitHub Actions 卡在跑 migration 那步**。`ALLOWED_ORIGINS` 在 migration job 上只是為了通過設定檢查，job 不會真的處理 HTTP 請求，填什麼網域都不影響功能；只有 API service 那份 `ALLOWED_ORIGINS` 在 Step 5 拿到 Vercel 網域後需要換成實際值（見 [`deployment.md`](deployment.md) 的逗號轉義寫法；頁面版直接在「環境變數」區塊填完整逗號分隔網域清單即可，不會遇到 CLI 版那個分隔字元的坑）。

## Step 5：Vercel

### CLI 版

```bash
cd apps/web
vercel link            # 選對 team／建立新 project，Root Directory 選 apps/web
vercel env add VITE_SUPABASE_URL production
vercel env add VITE_SUPABASE_ANON_KEY production
vercel env add VITE_API_BASE_URL production      # 填 https://<cloud-run-url>/api/v1
vercel env add VITE_SUPABASE_URL preview --no-sensitive
vercel env add VITE_SUPABASE_ANON_KEY preview --no-sensitive
vercel env add VITE_API_BASE_URL preview --no-sensitive
```

Preview 環境變數要另外設一次，不會沿用 Production 的設定，見 [`deployment.md`](deployment.md) 的已知坑。

這種做法建立的 Vercel 專案本身**不會**因為 push 就自動部署——實際觸發部署的是 [`../../.github/workflows/deploy-web.yml`](../../.github/workflows/deploy-web.yml)（`vercel pull` → `vercel build` → `vercel deploy --prebuilt`），所以還要接著做 Step 6 的 `VERCEL_TOKEN`／`VERCEL_ORG_ID`／`VERCEL_PROJECT_ID` 三個 repo secrets。

### 頁面版（含可省掉 GitHub Actions 這段的做法）

1. [vercel.com](https://vercel.com) →「Add New...」→「Project」→「Import Git Repository」，選這個 repo。
2. 「Root Directory」點「Edit」選 `apps/web`，Framework Preset 通常會自動偵測成 Vite，維持預設即可。
3. 展開「Environment Variables」，依序新增三筆：`VITE_SUPABASE_URL`、`VITE_SUPABASE_ANON_KEY`、`VITE_API_BASE_URL`（值填 `https://<cloud-run-url>/api/v1`）。每一筆右側可以選要套用到 Production／Preview／Development 哪些環境；Production 與 Preview 的 `VITE_API_BASE_URL` 若指向不同後端網址，取消「Same value for all Environments」分開填。
4. 按「Deploy」完成第一次建置。

這樣建出來的 Vercel 專案，內建的 GitHub 整合本身就會在每次 push 時自動建置部署（`main` 部署到 Production、其他分支部署 Preview），**不需要**跑 [`deploy-web.yml`](../../.github/workflows/deploy-web.yml)，也就**不需要** Step 6 的 `VERCEL_TOKEN`／`VERCEL_ORG_ID`／`VERCEL_PROJECT_ID` 三個 repo secrets——是整份文件裡最能落實「減少人工操作」的一步，往後改前端環境變數也只要回到 Vercel 專案的「Settings → Environment Variables」頁面改，改完 Vercel 會提示重新部署，不用碰 GitHub。

> 若團隊希望所有部署都集中由 GitHub Actions 記錄與控管（例如要跟 API 部署綁在同一個 workflow 執行順序），才需要保留 `deploy-web.yml` 這條路徑；採用時記得到 Vercel 專案「Settings → Git」把「自動部署」關閉，避免同一次 push 觸發 Vercel 原生整合與 GitHub Actions 兩次部署互相打架。這是既有 CI/CD 流程的取捨，本文件不預設答案，僅提供兩種做法的差異。

## Step 6：GitHub repo 設定

### Environments

到 repo 的 Settings → Environments 建立兩個 Environment，名字要跟 [`../../.github/workflows/deploy-api.yml`](../../.github/workflows/deploy-api.yml)／[`../../.github/workflows/deploy-web.yml`](../../.github/workflows/deploy-web.yml) 裡 `environment:` 欄位的值完全一致（大小寫敏感）：

- `develop`
- `Production`

`deploy-api.yml` 用 `github.ref_name == 'main' && 'Production' || 'develop'` 決定要用哪個 Environment；名字對不上的話 GitHub 會自動建一個空的同名 Environment，裡面沒有任何 secret，部署會直接失敗在「Authenticate to Google Cloud」那步。

### 每個 Environment 各自要有的 Variables／Secrets

| 名稱 | 類型 | develop | Production |
|---|---|---|---|
| `GCP_PROJECT_ID` | Variable | 你的 GCP 專案 ID | 同左或另一個正式專案 |
| `GCP_REGION` | Variable | 例如 `asia-east1` | 同左 |
| `ARTIFACT_REPOSITORY` | Variable | `cloud-run-source-deploy` | 同左 |
| `API_SERVICE` | Variable | `ltc-api` | 同左或另一個服務名 |
| `MIGRATION_JOB` | Variable | `ltc-api-migrate` | 同左 |
| `GCP_WIF_PROVIDER` | Secret | Step 2 產出的 provider 資源名稱 | 同左或另一組 |
| `GCP_DEPLOY_SA` | Secret | Step 2 產出的 service account email | 同左或另一組 |
| `VERCEL_ALIAS_DOMAIN` | Variable | Step 5 拿到的 `<project>-git-<branch>-<team>.vercel.app` | `Production` 不需要，`--prod` 部署會自動套用 Production Domains |

若 Step 5 採用 Vercel 頁面版（原生 Git 整合），以上這張表就是這個 GitHub Environment 要設定的**全部**內容，不用再另外處理 Vercel 相關項目。

**CLI 版：**

```bash
gh api --method PUT repos/<GITHUB_ORG_OR_USER>/<GITHUB_REPO_NAME>/environments/develop
gh api --method PUT repos/<GITHUB_ORG_OR_USER>/<GITHUB_REPO_NAME>/environments/Production

gh variable set GCP_PROJECT_ID --env develop --body "<GCP_PROJECT_ID>"
gh variable set GCP_REGION --env develop --body "<GCP_REGION>"
gh variable set ARTIFACT_REPOSITORY --env develop --body "cloud-run-source-deploy"
gh variable set API_SERVICE --env develop --body "ltc-api"
gh variable set MIGRATION_JOB --env develop --body "ltc-api-migrate"
gh secret set GCP_WIF_PROVIDER --env develop --body "projects/<GCP_PROJECT_NUMBER>/locations/global/workloadIdentityPools/github-pool/providers/github-provider"
gh secret set GCP_DEPLOY_SA --env develop --body "github-actions-deployer@<GCP_PROJECT_ID>.iam.gserviceaccount.com"
gh variable set VERCEL_ALIAS_DOMAIN --env develop --body "<project>-git-develop-<team>.vercel.app"
```

`gh api --method PUT repos/.../environments/<name>` 是用來自動建立 Environment 本身（`gh` 沒有專屬的 `gh environment create` 子指令），連「到網頁點 New environment」這一步都省掉；建立後緊接著的 `gh variable set`／`gh secret set` 才是逐項填值。`Production` 環境依樣重複一次（正式環境是否要用獨立的 GCP 專案／Supabase 專案是團隊決定，這份文件不預設答案）。

**頁面版：**

1. Settings → Environments →「New environment」，輸入 `develop`，按「Configure environment」；重複一次輸入 `Production`。
2. 進入每個 Environment 頁面，「Environment variables」區塊按「Add variable」逐一新增 `GCP_PROJECT_ID`／`GCP_REGION`／`ARTIFACT_REPOSITORY`／`API_SERVICE`／`MIGRATION_JOB` 五筆；「Environment secrets」區塊按「Add secret」新增 `GCP_WIF_PROVIDER`／`GCP_DEPLOY_SA` 兩筆。
3. 兩個 Environment（`develop`／`Production`）都要各自填一次，內容依上表。

### Repo 層級 Secrets（不分環境，Vercel CLI 版才需要）

只有 Step 5 選 CLI 版時才需要這三筆；Step 5 選頁面版（原生 Git 整合）可以整段跳過。

**CLI 版：**

```bash
gh secret set VERCEL_TOKEN --body "<vercel token>"
gh secret set VERCEL_ORG_ID --body "<vercel org id>"       # vercel link 後看 apps/web/.vercel/project.json
gh secret set VERCEL_PROJECT_ID --body "<vercel project id>"  # 同上
```

`VERCEL_TOKEN` 到 Vercel 帳號設定的 Tokens 頁面建立；範圍要能存取 Step 5 建立的那個 project。

**頁面版：** Settings →「Secrets and variables」→「Actions」→「Repository secrets」→「New repository secret」，重複三次分別建立 `VERCEL_TOKEN`（值到 Vercel 帳號的 Settings → Tokens 頁面建立取得）、`VERCEL_ORG_ID`、`VERCEL_PROJECT_ID`（後兩者在 Vercel 專案的 Settings → General 頁面可以找到，分別對應 Team ID／Project ID）。

## Step 7：驗證自動部署

```bash
git checkout develop
git commit --allow-empty -m "chore: trigger deploy"
git push origin develop
```

到 GitHub repo 的 Actions 分頁確認 `Deploy API to Cloud Run` 以 `push` 事件觸發並成功；若 Step 5 選 CLI 版，`Deploy Web to Vercel` 也要一併確認成功。若 Step 5 選頁面版（原生 Git 整合），改到 Vercel 專案的「Deployments」頁面確認這次 push 觸發了新的部署。成功後照 [`deployment.md`](deployment.md) 的「部署後檢查清單」實際登入測試一次。

## 檢查清單

- [ ] Supabase migration 跑到最新版本（`schema_migrations` 表對得上 `apps/api/migrations/` 底下最大的檔號；若走頁面版 SQL Editor，自行核對每支 `.up.sql` 都已依序執行成功）
- [ ] Cloud Run service／migration job 都能用 GitHub Actions 的 service account 部署（`roles/run.admin` 等七個角色都給了）
- [ ] migration job 跟 API service 兩邊都各自設了 `APP_ENV=production`／`SUPABASE_JWKS_URL`／`ALLOWED_ORIGINS`（拼字也要對，例如 `APP_ENV=develope` 這種打錯字會讓 job 直接 `Failed to load config` 失敗，且錯誤訊息不會明確指出是哪個變數拼錯）
- [ ] Cloud Run 的 `ALLOWED_ORIGINS`（API service 那份，不是 migration job 的佔位值）包含 Vercel 分配的每一個會用到的網域
- [ ] `gcloud run jobs describe ltc-api-migrate ... --format="value(...containers[0].command)"` 印出來的是 `/app/migrate`（或等價的 `//app/migrate`），不是一串 Windows 路徑（`D:/...`）——在 Windows 用 Git Bash 打 `--command` 這類參數最容易中招
- [ ] 若 Step 5 選 CLI 版原生 `deploy-web.yml`：本機或另開一個乾淨環境重跑過一次 `vercel build` 確認不會 `vite: not found`（Vercel CLI 本機建置不會自動把 `node_modules/.bin` 加進 PATH，`deploy-web.yml` 已經處理過這點，但如果日後改寫這個 workflow 步驟要記得保留）
- [ ] Vercel Production／Preview 兩邊都各自設過 `VITE_SUPABASE_URL`／`VITE_SUPABASE_ANON_KEY`／`VITE_API_BASE_URL`
- [ ] GitHub `develop`／`Production` 兩個 Environment 名稱與 workflow 檔案裡的字串完全一致，且各自有七個 GCP variables／secrets
- [ ] 若 Step 5 選 CLI 版：repo 層級有三個 `VERCEL_*` secrets；若選頁面版原生整合：確認 Vercel 專案的「自動部署」是開著的，且沒有同時保留 `deploy-web.yml` 造成重複部署
- [ ] push 一個空 commit 到 `develop` 能看到 `Deploy API to Cloud Run`（以及選 CLI 版時的 `Deploy Web to Vercel`）成功；選頁面版原生整合則到 Vercel Deployments 頁面確認
