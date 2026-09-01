# 從零建置新環境

給要把這個專案部署到一個全新環境（新的 Supabase 專案、新的 GCP 專案、新的 Vercel 專案）的人看，照順序做完就能讓 `git push` 到 `develop`／`main` 自動部署。日常維運（改環境變數、手動部署、常見坑）見 [`deployment.md`](deployment.md)；這份文件只管「從零開始」那一次性的建置流程。

## 前置需求

本機要有：`gcloud`（已登入且有目標 GCP 專案的 Owner 或等同權限）、`vercel` CLI（已登入）、`gh` CLI（已登入且對目標 repo 有 admin 權限，用來設定 Environments／secrets）、`go`、`node`／`npm`。

需要先有：一個 Supabase 專案、一個 GCP 專案（已啟用計費）、一個 GitHub repo（本專案的 fork 或原 repo）。

## Step 1：Supabase

1. 到 Supabase 建一個新專案，記下：
   - 專案 URL：`https://<project-ref>.supabase.co`
   - 專案 ref（`<project-ref>` 那段）
   - anon key（Settings → API）
   - 連線池（Connection Pooling）DATABASE_URL，port `6543`，transaction mode
2. 跑 migration：

   ```bash
   cd apps/api
   DATABASE_URL='<連線池 URL>' APP_ENV=local go run ./cmd/migrate up
   ```

   `APP_ENV=local` 只是為了跳過 `production` 模式底下 `SUPABASE_JWKS_URL`／`ALLOWED_ORIGINS` 必填的檢查，migration 本身不受影響。

3. `000002_seed_reference_data.up.sql` 會種一個預設管理員帳號（`ltcf-admin@ltc.example.com`），同時建好 `auth.users` 與對應的 `auth.identities`（缺其中一個都會導致密碼正確也無法登入，見 [`deployment.md`](deployment.md) 的已知坑）。要換帳密就直接改這支 migration 裡的 bcrypt hash，或改用 Supabase Dashboard 的 Auth 介面另外建帳號。

## Step 2：GCP 專案、Artifact Registry、Service Account、Workload Identity Federation

以下指令假設你已經 `gcloud config set project <GCP_PROJECT_ID>`。

### 啟用必要 API

```bash
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  iamcredentials.googleapis.com \
  secretmanager.googleapis.com
```

### Artifact Registry（存放 Docker image）

```bash
gcloud artifacts repositories create cloud-run-source-deploy \
  --repository-format=docker \
  --location=<GCP_REGION>
```

`ARTIFACT_REPOSITORY` 這個名字沒有強制規定，跟 GitHub variable `ARTIFACT_REPOSITORY` 對上即可。

### 部署用 Service Account

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

`roles/logging.viewer` 不是嚴格必要（`deploy-api.yml` 撈 build log 那步已經用 `|| true` 容錯），但建議還是加，串流 log 才看得到。

### Workload Identity Federation（讓 GitHub Actions 免存 GCP 金鑰就能登入）

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

`<GCP_PROJECT_NUMBER>` 是專案編號（純數字），不是專案 ID，用 `gcloud projects describe <GCP_PROJECT_ID> --format="value(projectNumber)"` 查。`GCP_WIF_PROVIDER` 這個 GitHub secret 的值就是：

```
projects/<GCP_PROJECT_NUMBER>/locations/global/workloadIdentityPools/github-pool/providers/github-provider
```

`GCP_DEPLOY_SA` 這個 secret 的值就是 `$SA_EMAIL`。

## Step 3：Cloud Run 服務與 migration job（第一次手動建立）

GitHub Actions 只會「更新」既有的 service／job，第一次要手動建。先準備一份 `.env`（不進版控），內容參考 [`../../apps/api/.env.example`](../../apps/api/.env.example)，`ALLOWED_ORIGINS` 先留空稍後再補（要等 Vercel 網域確定）。

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
  --command="./migrate" \
  --args="up"
```

image 先隨便給一個佔位值，GitHub Actions 的「Update migration job image」那步會在每次部署時換成最新 tag，這裡建立時的 image 只是讓 job 能被建立起來。

## Step 4：Cloud Run 環境變數與 Secret Manager

機密值（`DATABASE_URL`／`ENCRYPTION_KEY`／`HMAC_KEY`）放 Secret Manager，其餘用一般環境變數：

```bash
echo -n '<DATABASE_URL>'   | gcloud secrets create DATABASE_URL   --data-file=-
echo -n '<ENCRYPTION_KEY>' | gcloud secrets create ENCRYPTION_KEY --data-file=-
echo -n '<HMAC_KEY>'       | gcloud secrets create HMAC_KEY       --data-file=-

gcloud run services update ltc-api --region <GCP_REGION> \
  --set-secrets="DATABASE_URL=DATABASE_URL:latest,ENCRYPTION_KEY=ENCRYPTION_KEY:latest,HMAC_KEY=HMAC_KEY:latest" \
  --update-env-vars="APP_ENV=production,SUPABASE_PROJECT_REF=<project-ref>,SUPABASE_JWKS_URL=https://<project-ref>.supabase.co/auth/v1/.well-known/jwks.json,STORAGE_BUCKET=ltc-exports,STORAGE_SIGNED_URL_TTL=24h,LOG_LEVEL=info"

gcloud run jobs update ltc-api-migrate --region <GCP_REGION> \
  --set-secrets="DATABASE_URL=DATABASE_URL:latest"
```

`ENCRYPTION_KEY`／`HMAC_KEY` 要是 32 bytes 的 base64：`openssl rand -base64 32`。`ALLOWED_ORIGINS` 留到 Step 5 拿到 Vercel 網域後再補（見 [`deployment.md`](deployment.md) 的逗號轉義寫法）。

## Step 5：Vercel

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

Preview 環境變數要另外設一次，不會沿用 Production 的設定，見 [`deployment.md`](deployment.md) 的已知坑。拿到 Vercel 分配的網域（`https://<project>.vercel.app` 與 `https://<project>-git-<branch>-<team>.vercel.app`）後回頭補 Cloud Run 的 `ALLOWED_ORIGINS`。

## Step 6：GitHub repo 設定

### Environments

到 repo 的 Settings → Environments 建立兩個 Environment，名字要跟 `.github/workflows/deploy-api.yml`／`deploy-web.yml` 裡 `environment:` 欄位的值完全一致（大小寫敏感）：

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

用 `gh` CLI 設定（比網頁快）：

```bash
gh variable set GCP_PROJECT_ID --env develop --body "<GCP_PROJECT_ID>"
gh variable set GCP_REGION --env develop --body "<GCP_REGION>"
gh variable set ARTIFACT_REPOSITORY --env develop --body "cloud-run-source-deploy"
gh variable set API_SERVICE --env develop --body "ltc-api"
gh variable set MIGRATION_JOB --env develop --body "ltc-api-migrate"
gh secret set GCP_WIF_PROVIDER --env develop --body "projects/<GCP_PROJECT_NUMBER>/locations/global/workloadIdentityPools/github-pool/providers/github-provider"
gh secret set GCP_DEPLOY_SA --env develop --body "github-actions-deployer@<GCP_PROJECT_ID>.iam.gserviceaccount.com"
```

`Production` 環境依樣重複一次（正式環境是否要用獨立的 GCP 專案／Supabase 專案是團隊決定，這份文件不預設答案）。

### Repo 層級 Secrets（不分環境，Vercel 用）

```bash
gh secret set VERCEL_TOKEN --body "<vercel token>"
gh secret set VERCEL_ORG_ID --body "<vercel org id>"       # vercel link 後看 apps/web/.vercel/project.json
gh secret set VERCEL_PROJECT_ID --body "<vercel project id>"  # 同上
```

`VERCEL_TOKEN` 到 Vercel 帳號設定的 Tokens 頁面建立；範圍要能存取 Step 5 建立的那個 project。

## Step 7：驗證自動部署

```bash
git checkout develop
git commit --allow-empty -m "chore: trigger deploy"
git push origin develop
```

到 GitHub repo 的 Actions 分頁確認 `Deploy API to Cloud Run` 與 `Deploy Web to Vercel` 兩條都以 `push` 事件觸發（不是 `workflow_run`）並且成功。成功後照 [`deployment.md`](deployment.md) 的「部署後檢查清單」實際登入測試一次。

## 檢查清單

- [ ] Supabase migration 跑到最新版本（`schema_migrations` 表對得上 `apps/api/migrations/` 底下最大的檔號）
- [ ] Cloud Run service／migration job 都能用 GitHub Actions 的 service account 部署（`roles/run.admin` 等七個角色都給了）
- [ ] Cloud Run 的 `ALLOWED_ORIGINS` 包含 Vercel 分配的每一個會用到的網域
- [ ] Vercel Production／Preview 兩邊都各自設過 `VITE_SUPABASE_URL`／`VITE_SUPABASE_ANON_KEY`／`VITE_API_BASE_URL`
- [ ] GitHub `develop`／`Production` 兩個 Environment 名稱與 workflow 檔案裡的字串完全一致，且各自有七個 GCP variables／secrets
- [ ] repo 層級有三個 `VERCEL_*` secrets
- [ ] push 一個空 commit 到 `develop` 能看到兩條 workflow 都成功
