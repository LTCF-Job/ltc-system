# 從零建置新環境（新手教學）

這份教學會帶你一步一步、用滑鼠點網頁的方式，把這個系統架設到你自己的環境上。**不需要會寫程式**，只要照著每個步驟點選、貼上文字即可；少數幾個步驟必須開一次「終端機」（電腦上打指令的黑色視窗），教學裡會清楚標明，並附上完整可以直接複製的指令，你不需要看懂指令在做什麼。

日常維運（例如之後要改設定、常見錯誤排解）另外寫在 [`deployment.md`](deployment.md)；這份文件只負責「從零開始」建置一次的流程，建置完成後就用不到了。

## 這份教學會用到的四個服務

先花一分鐘認識一下，之後照著做會比較清楚每一步在幹嘛：

| 服務 | 白話說明 |
|---|---|
| **Supabase** | 存放系統所有資料（個案、車輛、司機、登入帳號……）的資料庫，也負責處理「登入」這件事 |
| **Google Cloud（Cloud Run）** | 執行系統後端程式的地方，負責處理所有業務邏輯與跟資料庫溝通 |
| **Vercel** | 存放並顯示系統前端網頁的地方，就是使用者瀏覽器打開的那個網站 |
| **GitHub Actions** | 一個自動化機器人：你把程式改動上傳到 GitHub 之後，它會自動幫你把新版本部署到 Cloud Run 跟 Vercel，不用自己手動上傳 |

整套流程做完之後，日常使用就只剩：改程式 → 上傳到 GitHub → 系統自動幫你部署好，不用再重複這份教學的任何步驟。

## 開始之前，你需要準備

- 四個帳號：[Supabase](https://supabase.com)、[Google 帳號](https://console.cloud.google.com)（且已幫 Google Cloud 專案開通付款方式，Google 稱為「帳單帳戶」）、[Vercel](https://vercel.com)、[GitHub](https://github.com)。
- 一份這個專案的原始碼，放在你自己的 GitHub 帳號下（用 GitHub 的「Fork」功能，或直接是這個 repo 本身）。
- 你的電腦要能開「終端機」（Windows 用「PowerShell」，Mac／Linux 用「Terminal」）：只有兩個步驟需要用到（跑一次資料庫初始化指令、以及最後上傳程式碼），其餘都在瀏覽器裡完成。

準備好了嗎？開始吧。

## 名詞小百科（看不懂的詞先來這裡查）

| 名詞 | 白話說明 |
|---|---|
| 環境變數 | 一組「設定值」，讓程式在啟動時知道要連去哪個資料庫、用什麼密碼等等，不用把這些資訊寫死在程式碼裡 |
| 密鑰 / Secret | 不能公開的敏感設定值（例如資料庫密碼），跟一般環境變數分開管理、加密存放 |
| API Key／Token | 一長串英數字組成的通行證，讓一個系統證明「我有權限使用你」 |
| 部署 | 把寫好的程式正式上線、讓使用者能實際打開來用的動作 |
| Repo（倉庫） | GitHub 上存放這個專案所有程式碼的地方 |
| Push（推送） | 把你電腦上改好的程式碼上傳到 GitHub 上的動作 |

## 步驟一：建立 Supabase（資料庫）

1. 到 [supabase.com](https://supabase.com) 註冊並登入。
2. 按「New Project」，幫專案取個名字（例如 `ltc-system`），密碼欄位系統會幫你生一組，記得存起來備用（不影響後面步驟，只是資料庫的管理員密碼）。
3. 建立完成後，到左側選單「Project Settings → Data API」，把下面兩個值記下來，等一下會用到：
   - **Project URL**：長得像 `https://xxxxxxxxxxxx.supabase.co`
   - **Project Reference**：網址裡 `xxxxxxxxxxxx` 那一段
4. 再到「Project Settings → API Keys」，記下 **anon public** 這把金鑰（一長串英數字，是公開金鑰，不是機密，前端網站會直接用到）。
5. 到「Connect」（或「Project Settings → Database」），找到 **Connection Pooling** 那組連線字串，模式選 **Transaction**（連接埠通常是 `6543`），把完整字串記下來——這是資料庫的「地址＋密碼」，要保密，等一下叫它「資料庫連線字串」。

### 初始化資料庫（這一步需要開終端機）

打開終端機，切換到這個專案的資料夾，執行：

```bash
cd apps/api
DATABASE_URL="剛剛記下的資料庫連線字串" APP_ENV=local go run ./cmd/migrate up
```

`APP_ENV=local` 只是先跳過正式環境才需要的檢查，不影響資料庫初始化的結果。跑完之後畫面若出現一行 `Migration completed successfully`（或類似成功訊息），就代表資料表都建好了。

> **不想裝終端機工具的話**：也可以到 Supabase Dashboard 左側「SQL Editor」→「New query」，把 [`../../apps/api/migrations/`](../../apps/api/migrations/) 資料夾裡每一個檔名結尾是 `.up.sql` 的檔案，依檔名開頭數字由小到大（`000001_...`、`000002_...`……）依序貼進去按「Run」，一個成功再貼下一個，完全不用裝任何工具。缺點是之後如果想改回終端機那個指令，要自己確認兩邊資料庫版本有沒有對齊。

**這一步只會建立資料表與參考資料（全臺 22 個縣市），不會建立任何登入帳號。**（早期版本的 migration 會塞一組預設管理員，現已全面移除：`000002_seed_reference_data.up.sql` 只剩縣市資料，`000011_backfill_admin_identity.up.sql` 已改成不做事的 no-op。）所以初始化完之後你必須自己建一組管理員帳號才能登入：到 Supabase Dashboard 的「Authentication → Users」按「Add user」，填一組**只有你知道的 email 與密碼**，建立後點進該使用者，把 `app_metadata`（有些版本顯示為 Raw App Meta Data）改成 `{"role":"admin","data_plane":"production"}` 並儲存——沒有這個 `role` 欄位的話，登入得進去但每一支 API 都會回 403。密碼請直接存進你自己的密碼管理工具，**不要寫進程式碼、環境變數或任何文件**。如果登入時卡在「帳號密碼錯誤」，先確認專案的認證方式（Authentication → Providers）有啟用「Email」，這是最常見的漏設原因。

### 如果還要建一個 Demo 環境（選用）

只有需要另外開一個「給人試用、不影響正式資料」的 Demo 站台時才需要這一段；一般情況跳過即可。

Demo 環境跟正式環境共用同一個 Supabase 專案，但用另一個獨立的資料庫存資料，彼此在 Postgres 權限層級互相隔離，不會互相讀寫到對方的資料。下面步驟是實際跑過一次、確認可行的完整流程（含建置時踩到的坑）；`<project-ref>` 全部換成實際的 Supabase 專案 ref（例如 `oywacuduaiulnfxzmpxs`），`<GCP_PROJECT_ID>`／`<GCP_REGION>` 換成步驟二、三建立的實際值：

1. **建立 `ltc_demo` 資料庫**：到 Supabase Dashboard 的「SQL Editor」，先確認目前連線的是預設的 `postgres` 資料庫，執行 `CREATE DATABASE ltc_demo;`（`CREATE DATABASE` 不能在交易裡執行，也不能對自己所在的資料庫下達，所以要先連在 `postgres` 上才能建立 `ltc_demo`）。
2. **建立 Demo 專用連線角色**：把 SQL Editor 的連線切到剛剛新建的 `ltc_demo` 資料庫（不是 `postgres`），貼上 [`../../apps/api/ops/demo-db-roles.sql`](../../apps/api/ops/demo-db-roles.sql) 整份內容執行一次。這支腳本會建立 Demo 專用的資料庫連線角色（`ltc_demo_app`），只給它 `ltc_demo` 這個資料庫**既有表格**的 DML 讀寫權限（`SELECT/INSERT/UPDATE/DELETE`，不含 `CREATE`），並且擋掉它連到正式 `postgres` 資料庫的權限，這就是「Demo 站台碰不到正式資料」的實際防線。腳本裡有 `CHANGE ME` 標記的地方是佔位密碼，執行完之後記得立刻在 Supabase Dashboard 或用 `ALTER ROLE ltc_demo_app WITH PASSWORD '<新密碼>'` 換成一組真正的密碼。
3. **建立 Demo 登入帳號**：`demo@ltc.example.com` 這個帳號無法用一般的「Sign up」建立（需要同時設定 `app_metadata`），要在 Supabase Dashboard「Authentication → Users」手動新增，並把 `app_metadata` 設成 `{"role":"admin","data_plane":"demo"}`；若拿不到 Dashboard 存取權限、只有資料庫連線，也可以比照 [`000002_seed_reference_data.up.sql`](../../apps/api/migrations/000002_seed_reference_data.up.sql) 的做法直接對 `postgres`（不是 `ltc_demo`，Auth 使用者是專案層級共用的）執行 `INSERT INTO auth.users`／`INSERT INTO auth.identities`，密碼欄位用 `extensions.crypt('<密碼>', extensions.gen_salt('bf', 10))` 產生雜湊；兩種做法擇一即可。
4. **在 GCP Secret Manager 準備兩組 `ltc_demo` 連線字串**（不是一組——理由見下方第 6 步）：
   ```bash
   # 給 migrate job 用：Supabase 專案預設的 postgres 超級使用者，dbname 換成 ltc_demo
   echo -n 'postgres://postgres.<project-ref>:<postgres 密碼>@<pooler-host>:6543/ltc_demo?sslmode=require' \
     | gcloud secrets create DEMO_MIGRATE_DATABASE_URL --project=<GCP_PROJECT_ID> --replication-policy=automatic --data-file=-

   # 給 ltc-api-demo 服務本體用：步驟 2 建立的 ltc_demo_app，注意使用者名稱要帶專案 ref（見下方已知坑）
   echo -n 'postgres://ltc_demo_app.<project-ref>:<ltc_demo_app 密碼>@<pooler-host>:6543/ltc_demo?sslmode=require' \
     | gcloud secrets create DEMO_DATABASE_URL --project=<GCP_PROJECT_ID> --replication-policy=automatic --data-file=-

   for s in DEMO_MIGRATE_DATABASE_URL DEMO_DATABASE_URL; do
     gcloud secrets add-iam-policy-binding "$s" --project=<GCP_PROJECT_ID> \
       --member="serviceAccount:<Cloud Run 服務用的 service account>" \
       --role="roles/secretmanager.secretAccessor"
   done
   ```
   已知坑：Supavisor pooler（6543 埠）用使用者名稱判斷要連到哪個 Supabase 專案，自訂角色（`ltc_demo_app`）一定要寫成 `<角色>.<project-ref>`，只寫角色名稱會得到 `FATAL: (ENOIDENTIFIER) no tenant identifier provided`。詳細背景見 [`deployment.md`](deployment.md#已知坑連-supavisor-pooler-存取非預設資料庫非預設角色時使用者名稱要帶專案-ref)。
5. **建立並執行 migrate job**（沿用正式服務的 image）：
   ```bash
   gcloud run jobs create ltc-api-demo-migrate \
     --project=<GCP_PROJECT_ID> --region=<GCP_REGION> \
     --image=<跟 ltc-api-migrate 相同的 image> \
     --command=//app/migrate --args=up \
     --set-env-vars=APP_ENV=production,DATA_PLANE=demo,SUPABASE_JWKS_URL=https://<project-ref>.supabase.co/auth/v1/.well-known/jwks.json,SUPABASE_PROJECT_REF=<project-ref>,ALLOWED_ORIGINS=<任意值，job 不處理 HTTP 請求> \
     --set-secrets=DATABASE_URL=DEMO_MIGRATE_DATABASE_URL:latest,ENCRYPTION_KEY=ENCRYPTION_KEY:latest,HMAC_KEY=HMAC_KEY:latest \
     --service-account=<跟 ltc-api-migrate 相同的 service account>

   gcloud run jobs execute ltc-api-demo-migrate --project=<GCP_PROJECT_ID> --region=<GCP_REGION> --wait
   ```
   已知坑：這個 job 的 `DATABASE_URL` **必須**用第 4 步的 `DEMO_MIGRATE_DATABASE_URL`（superuser），不能用 `DEMO_DATABASE_URL`（`ltc_demo_app`）——`ltc_demo_app` 沒有 `CREATE TABLE` 權限，跑 migration 會在建立 `schema_migrations` 表格那步就失敗（`permission denied for schema public`）。只要第 2 步是用 `postgres` 超級使用者連進 `ltc_demo` 執行 `demo-db-roles.sql`，這裡用同一個超級使用者跑 migration 建出來的新表格就會自動套用 `ALTER DEFAULT PRIVILEGES` 授權給 `ltc_demo_app`，不用額外補權限。
6. **部署 `ltc-api-demo` 服務**（沿用正式服務的 image 與大部分設定，差異處見下方）：
   ```bash
   gcloud run deploy ltc-api-demo \
     --project=<GCP_PROJECT_ID> --region=<GCP_REGION> \
     --image=<跟 ltc-api 相同的 image> \
     --service-account=<跟 ltc-api 相同的 service account> \
     --port=8080 --allow-unauthenticated \
     --set-env-vars="^;^APP_ENV=production;DATA_PLANE=demo;DB_MAX_OPEN_CONNS=2;DB_MAX_IDLE_CONNS=1;SUPABASE_PROJECT_REF=<project-ref>;SUPABASE_JWKS_URL=https://<project-ref>.supabase.co/auth/v1/.well-known/jwks.json;ALLOWED_ORIGINS=<跟正式服務一樣的前端網域清單>" \
     --set-secrets=DATABASE_URL=DEMO_DATABASE_URL:latest,ENCRYPTION_KEY=ENCRYPTION_KEY:latest,HMAC_KEY=HMAC_KEY:latest
   ```
   跟正式服務不一樣的地方：
   - `DATABASE_URL` 密鑰用第 4 步的 `DEMO_DATABASE_URL`（`ltc_demo_app`，不是 superuser），而不是正式服務用的那組。
   - `DB_MAX_OPEN_CONNS`／`DB_MAX_IDLE_CONNS` 建議設得比正式服務小（例如各設 `2`／`1`）。這兩個資料庫是同一個 Supabase 專案共用同一份運算資源與連線數上限，Demo 站台把連線池開太大，會排擠到正式服務可用的連線額度。
   - 環境變數要多加一筆 `DATA_PLANE=demo`（跟 `APP_ENV=production` 是兩個獨立的設定：`APP_ENV` 決定要不要放行開發用的檢查捷徑，`DATA_PLANE` 決定這個服務只認 Demo 帳號的登入憑證、不接受正式帳號的憑證，兩個都要設，缺一個效果都不對）。
   - `ENCRYPTION_KEY`／`HMAC_KEY` 沿用跟正式服務**同一組** secret 即可，這兩把金鑰不區分 data plane。
7. **前端加一個環境變數**：Vercel 專案的 `VITE_DEMO_API_BASE_URL` 要指向這個新服務網址（`https://ltc-api-demo-<hash>.<region>.run.app/api/v1`），**Production 與 Preview 兩個環境都要各加一次**（`vercel env add VITE_DEMO_API_BASE_URL production --no-sensitive --value '...'`，`preview` 同理），加完要重新觸發一次 build 才會生效（Vite 環境變數是建置期注入的）。細節與 `vercel` CLI 的其他已知坑見 [`deployment.md`](deployment.md#已知坑vercel-cli-一律要在-repo-根目錄執行不能先-cd-appsweb)。
8. 建好以上兩個 Demo 用的 Cloud Run 資源後，`.github/workflows/deploy-api.yml` 會自動把它們接進部署流程：每次 push 都先建置一次 image，接著 migration／部署 Demo，跑一輪對 Demo 的真實 E2E 測試（[`apps/web/tests/e2e-live/`](../../apps/web/tests/e2e-live/)，直接打真正的 Supabase 與 Demo API，不透過 MSW），測試沒過就不會繼續 migration／部署正式環境。這一段需要在 GitHub 該環境（`develop` 或 `Production`）多設幾個變數與密鑰，見步驟六最後的「Demo／Live E2E 專用設定」。

## 步驟二：建立 Google Cloud 專案

1. 到 [Google Cloud Console](https://console.cloud.google.com)，右上角選單新增一個專案，取個名字。
2. 左上角選單 →「帳單」，確認這個專案已經連結一個有效的帳單帳戶（沒有的話系統會提示你設定信用卡，Google Cloud 有免費額度，正常使用小型系統費用非常低）。
3. 左上角選單 →「API 和服務」→「程式庫」，依序搜尋以下五個名稱並個別點「啟用」：
   - Cloud Run Admin API
   - Cloud Build API
   - Artifact Registry API
   - IAM Service Account Credentials API
   - Secret Manager API

### 建立一個「存放程式包裹」的地方（Artifact Registry）

Console 左側選單 →「Artifact Registry」→「儲存庫」→「建立存放區」：名稱填 `cloud-run-source-deploy`，格式選 **Docker**，模式選「標準」，區域選一個離台灣近的（例如 `asia-east1`，代表台灣）。記住你選的這個區域代號，後面會一直用到，這份教學都寫成 `<GCP_REGION>`。

### 建立一個「代替 GitHub 操作 Google Cloud 的身分」（Service Account）

這是讓 GitHub 的自動化機器人有權限幫你部署系統的必要設定。

1. Console →「IAM 與管理」→「服務帳戶」→「建立服務帳戶」，名稱填 `github-actions-deployer`，顯示名稱填「GitHub Actions Deployer」，直接按「完成」。
2. 到「IAM 與管理」→「IAM」，找到剛剛這個帳戶（`github-actions-deployer@你的專案ID.iam.gserviceaccount.com`）→ 點鉛筆圖示編輯 → 逐一「新增另一個角色」，把以下七個角色都加上去後儲存：

   - Artifact Registry 寫入者
   - Cloud Build 編輯者
   - 服務帳戶使用者
   - Logging 檢視者
   - Cloud Run 管理員
   - Secret Manager 密鑰存取者
   - Storage 管理員

### 讓 GitHub 能安全地登入 Google Cloud（Workload Identity Federation）

這一步是這份教學裡**唯一建議一定要用終端機**的步驟——牽涉到好幾個必須完全正確、不能打錯字的長字串，用複製貼上比在網頁表單裡一個一個手打安全很多。你的電腦要先裝好 [Google Cloud CLI（`gcloud` 指令工具）](https://cloud.google.com/sdk/docs/install)並登入（`gcloud auth login`）。

打開終端機，把下面指令裡的 `<GCP_PROJECT_ID>`（Google Cloud 專案 ID，不是專案名稱，在 Console 首頁能看到）跟 `<GITHUB_ORG_OR_USER>`（你的 GitHub 帳號名稱或組織名稱）換成你自己的值，依序執行：

```bash
gcloud config set project <GCP_PROJECT_ID>

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

gcloud iam service-accounts add-iam-policy-binding \
  "github-actions-deployer@<GCP_PROJECT_ID>.iam.gserviceaccount.com" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/<GCP_PROJECT_NUMBER>/locations/global/workloadIdentityPools/github-pool/attribute.repository/<GITHUB_ORG_OR_USER>/<GITHUB_REPO_NAME>"
```

`<GCP_PROJECT_NUMBER>` 是一串純數字的專案編號（跟專案 ID 不同），用這行指令查：

```bash
gcloud projects describe <GCP_PROJECT_ID> --format="value(projectNumber)"
```

跑完把下面這兩個值記下來，等一下步驟七會用到：

- `GCP_WIF_PROVIDER` 的值是：`projects/<GCP_PROJECT_NUMBER>/locations/global/workloadIdentityPools/github-pool/providers/github-provider`
- `GCP_DEPLOY_SA` 的值是：`github-actions-deployer@<GCP_PROJECT_ID>.iam.gserviceaccount.com`

## 步驟三：先手動建一次後端服務與資料庫初始化工作

GitHub 自動化機器人只會「更新」已經存在的服務，第一次要自己手動建立起來，之後才交給它接手。

1. Console →「Cloud Run」→「服務」→「部署容器」：服務名稱填 `ltc-api`，容器映像檔網址先填 `us-docker.pkg.dev/cloudrun/container/hello`（Google 提供的示範用容器，只是佔位，等一下會被真正的程式取代），區域選你在步驟二選的那個，「驗證」選「需要驗證」（不允許沒有登入就存取，因為我們的系統一律要求登入），其餘保持預設，按「建立」。
2. Console →「Cloud Run」→「工作」→「部署容器」：工作名稱填 `ltc-api-migrate`，容器映像檔網址一樣先填上面那個示範用容器，展開「容器」頁籤，指令欄填 `/app/migrate`，引數欄填 `up`，區域一樣選好，按「建立」。

   > 這個「工作」是每次部署新版程式前，自動幫你更新資料庫結構用的，之後完全不用手動理它。指令一定要填 `/app/migrate`（不是 `./migrate`），這是程式在容器裡實際存放的位置；如果你是在 Windows 用 Git Bash 這類終端機打字（而不是在網頁表單填），要注意它有時會自動把 `/app/migrate` 誤判成 Windows 路徑、變成一串奇怪的亂碼，把值改成兩個斜線 `//app/migrate` 就能避開這個問題，不影響容器內部的判讀。

3. 這兩個示範用容器現在都還跑不起來，這是正常現象，不用理它——等步驟七完成後，GitHub 的自動化機器人會把它們換成真正的程式版本。

## 步驟四：填後端需要的設定值

系統的後端程式要知道資料庫在哪裡、密碼是什麼、可以接受哪些網站呼叫它，這些都要先設定好。

### 先把三個機密值存起來（Secret Manager）

Console →「Security」→「Secret Manager」→「建立密鑰」，重複三次，分別建立以下三個名稱，「密鑰值」欄位貼上對應的值：

- `DATABASE_URL`：步驟一記下的資料庫連線字串
- `ENCRYPTION_KEY`：一組隨機產生的密碼，執行 `openssl rand -base64 32`（終端機指令）就能產生一組
- `HMAC_KEY`：跟上面一樣的方式再產生一組（兩個要不一樣）

### 幫服務跟工作補上其他設定

Console →「Cloud Run」→「服務」→ 點進 `ltc-api` →「編輯並部署新修訂版本」，切到「變數與密鑰」頁籤：

- 「環境變數」區塊，逐一新增以下六筆（等號後面換成你自己的值）：
  - `APP_ENV=production`
  - `SUPABASE_PROJECT_REF=<步驟一記下的 Project Reference>`
  - `SUPABASE_JWKS_URL=https://<步驟一記下的 Project Reference>.supabase.co/auth/v1/.well-known/jwks.json`
  - `STORAGE_BUCKET=ltc-exports`
  - `STORAGE_SIGNED_URL_TTL=24h`
  - `LOG_LEVEL=info`
  - `ALLOWED_ORIGINS=https://placeholder.example.com`（先填這個佔位值，步驟五拿到真正的網站網址後記得回來換掉，這欄位是「允許呼叫這個後端的網站清單」）
- 「密鑰」區塊「參照密鑰」，把 `DATABASE_URL`／`ENCRYPTION_KEY`／`HMAC_KEY` 三個密鑰各自掛成同名環境變數（版本選「最新」）。
- 按「部署」。

再到「Cloud Run」→「工作」→ 點進 `ltc-api-migrate` →「編輯」，一樣在「變數與密鑰」頁籤：

- 「環境變數」加入 `APP_ENV=production`、`SUPABASE_PROJECT_REF=...`、`SUPABASE_JWKS_URL=...`（值跟上面一樣）、`ALLOWED_ORIGINS=https://placeholder.example.com`。
- 「密鑰」掛上 `DATABASE_URL`。
- 儲存。

> **為什麼工作（job）也要設一次？** 因為「服務」跟「工作」是兩個獨立的東西，各自有各自的設定，不會互相共用。少設了任何一個，資料庫初始化工作在每次自動部署時都會失敗，錯誤訊息通常是 `Failed to load config`。填錯字也會有一樣的症狀（例如打成 `APP_ENV=produciton`），這種情況錯誤訊息不會明確告訴你是哪裡打錯，遇到問題先把這兩邊的設定值一字一字核對一次。

## 步驟五：建立前端網站（Vercel）

1. 登入 [vercel.com](https://vercel.com) →「Add New...」→「Project」→「Import Git Repository」，選你的這個 repo。
2. 「Root Directory」點「Edit」，選擇 `apps/web`（代表這個網站的原始碼放在專案裡的哪個資料夾），Framework Preset 通常會自動偵測成「Vite」，維持預設即可。
3. 展開「Environment Variables」，依序新增三筆（左側填名稱、右側填值）：
   - `VITE_SUPABASE_URL`：步驟一記下的 Project URL
   - `VITE_SUPABASE_ANON_KEY`：步驟一記下的 anon public 金鑰
   - `VITE_API_BASE_URL`：`https://<步驟三建立的 Cloud Run 服務網址>/api/v1`（服務網址在 Cloud Run 服務頁面最上方能看到，部署完成後才有）
   - 每一筆右側可以勾選要套用到 Production／Preview／Development 哪些環境；如果正式站與測試站的後端網址不一樣，把「Same value for all Environments」取消勾選，分開填。
4. 按「Deploy」，等它跑完第一次部署。

這樣建立的 Vercel 專案，本身內建的 GitHub 整合就會在你**每次**上傳程式碼時自動重新建置部署（`main` 分支部署到正式站、其他分支部署到測試站），**不需要**額外再設定任何東西，之後要改前端的環境變數，也只要回到這個 Vercel 專案的「Settings → Environment Variables」頁面改就好，改完 Vercel 會自動提示要不要重新部署，完全不用碰 GitHub 或終端機。這是整份教學裡最省事的一步。

> 如果你的團隊比較偏好把「後端部署」跟「前端部署」都集中在同一套 GitHub 自動化流程裡統一管理與記錄，這個專案也準備了另一種進階做法（用 GitHub Actions 部署前端，而不是用 Vercel 內建的 Git 整合），詳細步驟見文件最後的〈進階：改用 GitHub Actions 部署前端〉。一般情況不需要，直接照上面的做法即可。

## 步驟六：確認 GitHub 的自動化設定

前面步驟二記下的兩把 Google Cloud 金鑰，要放進 GitHub 才能讓自動化機器人使用。

1. 到你的 GitHub repo 頁面 →「Settings」→「Environments」→「New environment」，輸入 `develop`，按「Configure environment」；重複一次，這次輸入 `Production`（注意大寫 P，一定要跟這個名稱完全一樣）。
2. 進入 `develop` 這個 Environment 頁面：
   - 「Environment variables」區塊按「Add variable」，逐一新增以下五筆：
     - `GCP_PROJECT_ID` = 你的 Google Cloud 專案 ID
     - `GCP_REGION` = 步驟二選的區域（例如 `asia-east1`）
     - `ARTIFACT_REPOSITORY` = `cloud-run-source-deploy`
     - `API_SERVICE` = `ltc-api`
     - `MIGRATION_JOB` = `ltc-api-migrate`
   - 「Environment secrets」區塊按「Add secret」，新增兩筆：
     - `GCP_WIF_PROVIDER` = 步驟二最後記下的那串值
     - `GCP_DEPLOY_SA` = 步驟二最後記下的服務帳戶信箱
3. `Production` 這個 Environment 重複填一次一樣的內容（正式環境要不要另外用一個獨立的 Google Cloud 專案是你自己的選擇，這份教學不預設答案；不確定的話先填一樣的值也可以）。

> 如果步驟五你是照建議做法（Vercel 內建 Git 整合），且沒有建立步驟一的「如果還要建一個 Demo 環境（選用）」，到這裡整個建置流程就完成了，可以直接跳到步驟七驗證。

### Demo／Live E2E 專用設定（只有建了 Demo 環境才需要）

沒有照步驟一建立 Demo 環境的話可以跳過這一段；`.github/workflows/deploy-api.yml` 在偵測不到這些設定時仍會嘗試部署 Demo 相關資源，缺這些變數與密鑰會讓「Live E2E against Demo」這個 job 直接失敗，擋住後面正式環境的部署，此時只能先補齊設定或暫時 revert 這次修改。同樣在 `develop` 與 `Production` 兩個 Environment 各自新增：

- 「Environment variables」再加四筆：
  - `DEMO_API_SERVICE` = `ltc-api-demo`（跟步驟一建立 Cloud Run 服務時取的名字一致）
  - `DEMO_MIGRATION_JOB` = `ltc-api-demo-migrate`
  - `LIVE_SUPABASE_URL` = 步驟一記下的 Project URL（跟 Vercel 環境變數 `VITE_SUPABASE_URL` 同一個值）
  - `LIVE_SUPABASE_ANON_KEY` = 步驟一記下的 anon public 金鑰
- 「Environment secrets」再加一筆：
  - `LIVE_DEMO_TEST_PASSWORD` = Demo 測試帳號（`demo@ltc.example.com`）的登入密碼
- 這個 Demo 測試帳號就是上面步驟一第 3 點建立的 `demo@ltc.example.com`，`app_metadata` 需為 `{"role":"admin","data_plane":"demo"}`（`app_metadata` 只能透過 Dashboard 或 Admin API 設定，使用者自己登入後改不動，這是 [`internal/platform/auth/auth.go`](../../apps/api/internal/platform/auth/auth.go) 判斷 data plane 的唯一依據）；還沒建立的話回去看步驟一第 3 點。

以下為選用：想讓 Live E2E 順便驗證「正式帳號的憑證會被 Demo API 拒絕、Demo 帳號的憑證會被正式 API 拒絕」這組矩陣，再加：

- 「Environment variables」：`LIVE_PROD_API_BASE_URL`（正式 API 的網址 + `/api/v1`）、`LIVE_PROD_TEST_EMAIL`（一個正式環境的真實帳號信箱）
- 「Environment secrets」：`LIVE_PROD_TEST_PASSWORD`

缺這組選用設定時，對應的測試會自動跳過，不影響其餘部署流程。

想在部署正式環境前額外擋一層「正式與 Demo 資料庫 schema 是否一致」的檢查（[`apps/api/ops/compare-demo-prod-schema.sh`](../../apps/api/ops/compare-demo-prod-schema.sh)），再加兩筆「Environment secrets」：

- `SCHEMA_CHECK_PROD_DATABASE_URL` = 正式資料庫的連線字串
- `SCHEMA_CHECK_DEMO_DATABASE_URL` = `ltc_demo` 資料庫的連線字串

沒設這兩個密鑰時這一步會直接失敗並擋住部署，等於強制要求設定；如果暫時不想要這層防護，需要自行修改 `deploy-api.yml` 拿掉這個步驟。

## 步驟七：實際測試一次

打開終端機，在專案資料夾裡執行：

```bash
git checkout develop
git commit --allow-empty -m "chore: 觸發第一次自動部署"
git push origin develop
```

這行指令只是「上傳一次沒有任何程式改動的紀錄」，用來觸發自動部署，確認整套設定是通的。

推上去之後：

- 到 GitHub repo 的「Actions」分頁，應該會看到 `Deploy API to Cloud Run` 開始執行，等個幾分鐘確認它全部打勾成功。
- 到 Vercel 專案的「Deployments」頁面，應該會看到這次上傳觸發了一次新的部署，等它跑完。
- 都成功後，打開 Vercel 分配給你的網站網址，用步驟一在 Supabase「Authentication → Users」自行建立的那組管理員帳號試著登入一次，確認畫面能正常顯示資料。

如果哪一步卡住了，先看下面的檢查清單，逐項核對；更詳細的錯誤排解在 [`deployment.md`](deployment.md)。

## 檢查清單

- [ ] 已在 Supabase「Authentication → Users」自行建立至少一組管理員帳號，且 `app_metadata` 有 `"role":"admin"`（migration 不會幫你建帳號）
- [ ] 資料庫初始化跑到最新版本（`schema_migrations` 表對得上 `apps/api/migrations/` 底下最大的檔案編號；若走 SQL Editor 手動貼的方式，自行核對每一支 `.up.sql` 都依序執行成功）
- [ ] `ltc-api-migrate` 這個工作跟 `ltc-api` 這個服務兩邊都各自設了 `APP_ENV=production`／`SUPABASE_JWKS_URL`／`ALLOWED_ORIGINS`（拼字也要對，打錯字不會有明確的錯誤提示）
- [ ] `ltc-api-migrate` 工作的「指令」欄位確認是 `/app/migrate`（或 `//app/migrate`），不是一串看起來像 Windows 路徑（`D:/...`）的亂碼
- [ ] `ltc-api` 服務的 `ALLOWED_ORIGINS` 已經從佔位值換成 Vercel 實際分配的網址
- [ ] Vercel 的 Production／Preview（正式／測試）環境變數都各自設過一次 `VITE_SUPABASE_URL`／`VITE_SUPABASE_ANON_KEY`／`VITE_API_BASE_URL`（不會互相沿用）
- [ ] GitHub `develop`／`Production` 兩個 Environment 名稱一字不差，且各自有五個一般設定與兩個機密設定
- [ ] 上傳一次空白紀錄到 `develop` 分支，能在 Actions 分頁看到 `Deploy API to Cloud Run` 成功、Vercel Deployments 頁面看到新的部署
- [ ] 有建立 Demo 環境的話，`develop`／`Production` 兩個 Environment 都補齊「Demo／Live E2E 專用設定」列出的變數與密鑰，否則部署會卡在 `Live E2E against Demo` 這個 job

---

## 進階：改用 GitHub Actions 部署前端

只有想把前端部署也統一交給 GitHub Actions 管理時才需要這段；一般情況步驟五做完就結束了，不需要看這段。

### 額外要在 GitHub 設定的機密值

1. 到 Vercel 帳號的「Settings → Tokens」建立一個新的 Token，範圍要能存取步驟五建立的那個專案，複製起來。
2. 到 Vercel 專案的「Settings → General」頁面，找到 Team ID 與 Project ID。
3. 到 GitHub repo 的「Settings → Secrets and variables → Actions → Repository secrets」，新增三筆（這三筆不分環境，只要設一次）：
   - `VERCEL_TOKEN` = 剛剛建立的 Token
   - `VERCEL_ORG_ID` = Team ID
   - `VERCEL_PROJECT_ID` = Project ID

### 關掉 Vercel 自己的自動部署，避免部署兩次

到 Vercel 專案的「Settings → Git」，把「自動部署」關閉。不關掉的話，同一次上傳程式碼會同時觸發 Vercel 自己的部署跟 GitHub Actions 的部署，兩邊互相搶著部署，容易搞混哪個才是最新版本。

### 額外的一個 GitHub Environment 變數

`VERCEL_ALIAS_DOMAIN`：讓每次自動部署完，測試站都能反映在同一個固定網址上（不然 GitHub Actions 每次部署出來的網址都是一串隨機亂碼，很難分享給別人測試）。這個變數只要在 `develop` 這個 Environment 設定，值填 Vercel 分配給這個專案、這個分支的固定網址，格式通常是：

```
<專案名稱>-git-develop-<你的 Vercel 帳號或團隊名稱>.vercel.app
```

第一次部署完之後，去 Vercel 的「Deployments」頁面找這個固定網址填進去即可；`Production`（正式站）不需要設定這個，因為它會自動套用你在 Vercel 專案設定的正式網域。

設定完成後，之後每次上傳程式碼到 `develop`／`main` 分支，`Deploy Web to Vercel` 這條自動化流程就會接手部署前端，行為與步驟七驗證的方式相同。

### 給熟悉終端機操作的人：整段用指令做一次

如果你全程都想用終端機操作（例如要重複建置好幾個環境、想把整套流程存成腳本），以下是這一段對應的指令版本，前提是你已經在終端機登入 `vercel` CLI（`vercel login`）與 `gh` CLI（`gh auth login`，且對這個 repo 有管理權限）：

```bash
cd apps/web
vercel link            # 選對帳號或團隊、建立新專案，Root Directory 選 apps/web
vercel env add VITE_SUPABASE_URL production
vercel env add VITE_SUPABASE_ANON_KEY production
vercel env add VITE_API_BASE_URL production      # 填 https://<cloud-run-網址>/api/v1
vercel env add VITE_SUPABASE_URL preview --no-sensitive
vercel env add VITE_SUPABASE_ANON_KEY preview --no-sensitive
vercel env add VITE_API_BASE_URL preview --no-sensitive

gh secret set VERCEL_TOKEN --body "<vercel token>"
gh secret set VERCEL_ORG_ID --body "<vercel org id>"
gh secret set VERCEL_PROJECT_ID --body "<vercel project id>"
gh variable set VERCEL_ALIAS_DOMAIN --env develop --body "<專案名稱>-git-develop-<帳號或團隊名稱>.vercel.app"
```

Preview（測試站）的環境變數要另外設一次，不會沿用 Production（正式站）的設定，這是 Vercel 的既有行為。

### 給熟悉終端機操作的人：步驟二、四、六的指令版本

步驟二的「啟用 API」「建立 Artifact Registry」「建立 Service Account」，以及步驟四、六，用指令做的版本如下（`<...>` 換成你自己的值）：

```bash
# 步驟二：啟用必要 API
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  iamcredentials.googleapis.com \
  secretmanager.googleapis.com

# 步驟二：建立 Artifact Registry
gcloud artifacts repositories create cloud-run-source-deploy \
  --repository-format=docker \
  --location=<GCP_REGION>

# 步驟二：建立 Service Account 並加上七個角色
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

# 步驟三：建立 Cloud Run 服務與 migration 工作（佔位 image，之後會被 GitHub Actions 換掉）
gcloud run deploy ltc-api --source apps/api --region <GCP_REGION>

gcloud run jobs create ltc-api-migrate \
  --region <GCP_REGION> \
  --image <GCP_REGION>-docker.pkg.dev/<GCP_PROJECT_ID>/cloud-run-source-deploy/api:bootstrap \
  --command="/app/migrate" \
  --args="up"

# 步驟四：Secret Manager 與環境變數
echo -n '<DATABASE_URL>'   | gcloud secrets create DATABASE_URL   --data-file=-
echo -n '<ENCRYPTION_KEY>' | gcloud secrets create ENCRYPTION_KEY --data-file=-
echo -n '<HMAC_KEY>'       | gcloud secrets create HMAC_KEY       --data-file=-

gcloud run services update ltc-api --region <GCP_REGION> \
  --set-secrets="DATABASE_URL=DATABASE_URL:latest,ENCRYPTION_KEY=ENCRYPTION_KEY:latest,HMAC_KEY=HMAC_KEY:latest" \
  --update-env-vars="APP_ENV=production,SUPABASE_PROJECT_REF=<project-ref>,SUPABASE_JWKS_URL=https://<project-ref>.supabase.co/auth/v1/.well-known/jwks.json,STORAGE_BUCKET=ltc-exports,STORAGE_SIGNED_URL_TTL=24h,LOG_LEVEL=info,ALLOWED_ORIGINS=https://placeholder.example.com"

gcloud run jobs update ltc-api-migrate --region <GCP_REGION> \
  --set-secrets="DATABASE_URL=DATABASE_URL:latest" \
  --update-env-vars="APP_ENV=production,SUPABASE_PROJECT_REF=<project-ref>,SUPABASE_JWKS_URL=https://<project-ref>.supabase.co/auth/v1/.well-known/jwks.json,ALLOWED_ORIGINS=https://placeholder.example.com"

# 步驟六：建立 GitHub Environments 並填入設定值
gh api --method PUT repos/<GITHUB_ORG_OR_USER>/<GITHUB_REPO_NAME>/environments/develop
gh api --method PUT repos/<GITHUB_ORG_OR_USER>/<GITHUB_REPO_NAME>/environments/Production

gh variable set GCP_PROJECT_ID --env develop --body "<GCP_PROJECT_ID>"
gh variable set GCP_REGION --env develop --body "<GCP_REGION>"
gh variable set ARTIFACT_REPOSITORY --env develop --body "cloud-run-source-deploy"
gh variable set API_SERVICE --env develop --body "ltc-api"
gh variable set MIGRATION_JOB --env develop --body "ltc-api-migrate"
gh secret set GCP_WIF_PROVIDER --env develop --body "projects/<GCP_PROJECT_NUMBER>/locations/global/workloadIdentityPools/github-pool/providers/github-provider"
gh secret set GCP_DEPLOY_SA --env develop --body "github-actions-deployer@<GCP_PROJECT_ID>.iam.gserviceaccount.com"
```

`Production` 環境依樣重複一次 `gh variable set`／`gh secret set` 那幾行（把 `--env develop` 換成 `--env Production`）。
