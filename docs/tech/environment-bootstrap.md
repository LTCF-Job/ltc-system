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

**這一步會建立資料表與參考資料（全臺 22 個縣市）；管理員帳號採條件式 bootstrap。** migration runner 在同時提供 `DEFAULT_ADMIN_EMAIL`、`DEFAULT_ADMIN_PASSWORD`、`SUPABASE_URL` 與 `SUPABASE_SERVICE_ROLE_KEY` 時，會嘗試建立／補上 idempotent default admin；若不提供這些設定，才需要到 Supabase Dashboard 的「Authentication → Users」按「Add user」，填一組**只有你知道的 email 與密碼**，建立後把 `app_metadata`（有些版本顯示為 Raw App Meta Data）設成 `{"role":"admin"}`。沒有這個 `role` 欄位的話，登入得進去但 API permission 可能不足。密碼請直接存進你自己的密碼管理工具，**不要寫進程式碼、環境變數或任何文件**。如果登入時卡在「帳號密碼錯誤」，先確認專案的認證方式（Authentication → Providers）有啟用「Email」，這是最常見的漏設原因。

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

1. 到你的 GitHub repo 頁面 →「Settings」→「Environments」→「New environment」，輸入 `Production`（注意大寫 P，一定要跟這個名稱完全一樣）。
2. 進入 `Production` 這個 Environment 頁面：
   - 「Environment variables」區塊按「Add variable」，逐一新增以下五筆：
     - `GCP_PROJECT_ID` = 你的 Google Cloud 專案 ID
     - `GCP_REGION` = 步驟二選的區域（例如 `asia-east1`）
     - `ARTIFACT_REPOSITORY` = `cloud-run-source-deploy`
     - `API_SERVICE` = `ltc-api`
     - `MIGRATION_JOB` = `ltc-api-migrate`
   - 「Environment secrets」區塊按「Add secret」，新增兩筆：
     - `GCP_WIF_PROVIDER` = 步驟二最後記下的那串值
     - `GCP_DEPLOY_SA` = 步驟二最後記下的服務帳戶信箱

> 如果步驟五你是照建議做法（Vercel 內建 Git 整合），到這裡整個建置流程就完成了，可以直接跳到步驟七驗證。

## 步驟七：實際測試一次

完成設定並確認 workflow／secret 後，請在專案資料夾推送一個已審核、包含實際變更的 `main` commit 觸發部署；不要為了觸發流程建立無內容的空 commit。推送前先確認工作樹與目標 branch：

```bash
git status
git push origin main
```

部署流程由 `main` push 觸發；本文件不替代 code review、migration 授權或 rollback 準備。

推上去之後：

- 到 GitHub repo 的「Actions」分頁，應該會看到 `Deploy API to Cloud Run` 開始執行，等個幾分鐘確認它全部打勾成功。
- 到 Vercel 專案的「Deployments」頁面，應該會看到這次上傳觸發了一次新的部署，等它跑完。
- 都成功後，打開 Vercel 分配給你的網站網址，用步驟一在 Supabase「Authentication → Users」自行建立的那組管理員帳號試著登入一次，確認畫面能正常顯示資料。

如果哪一步卡住了，先看下面的檢查清單，逐項核對；更詳細的錯誤排解在 [`deployment.md`](deployment.md)。

## 檢查清單

- [ ] 已確認 default admin bootstrap 是否使用；若未提供 bootstrap 設定，至少在 Supabase「Authentication → Users」建立一組管理員帳號，且 `app_metadata` 有 `"role":"admin"`
- [ ] 資料庫初始化跑到最新版本（`schema_migrations` 表對得上 `apps/api/migrations/` 底下最大的檔案編號；若走 SQL Editor 手動貼的方式，自行核對每一支 `.up.sql` 都依序執行成功）
- [ ] `ltc-api-migrate` 這個工作跟 `ltc-api` 這個服務兩邊都各自設了 `APP_ENV=production`／`SUPABASE_JWKS_URL`／`SUPABASE_JWT_ISSUER` 或 `SUPABASE_PROJECT_REF`／`ALLOWED_ORIGINS`／`SUPABASE_SERVICE_ROLE_KEY`（拼字也要對）
- [ ] `ltc-api-migrate` 工作的「指令」欄位確認是 `/app/migrate`（或 `//app/migrate`），不是一串看起來像 Windows 路徑（`D:/...`）的亂碼
- [ ] `ltc-api` 服務的 `ALLOWED_ORIGINS` 已經從佔位值換成 Vercel 實際分配的網址
- [ ] Vercel 的 Production 環境變數已設過一次 `VITE_SUPABASE_URL`／`VITE_SUPABASE_ANON_KEY`／`VITE_API_BASE_URL`
- [ ] GitHub `Production` 這個 Environment 名稱一字不差，且有五個一般設定與兩個機密設定
- [ ] 上傳一次空白紀錄到 `main` 分支，能在 Actions 分頁看到 `Deploy API to Cloud Run` 成功、Vercel Deployments 頁面看到新的部署

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

設定完成後，之後每次上傳程式碼到 `main` 分支，`Deploy Web to Vercel` 這條自動化流程就會接手部署前端，行為與步驟七驗證的方式相同。

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

# 步驟六：建立 GitHub Environment 並填入設定值
gh api --method PUT repos/<GITHUB_ORG_OR_USER>/<GITHUB_REPO_NAME>/environments/Production

gh variable set GCP_PROJECT_ID --env Production --body "<GCP_PROJECT_ID>"
gh variable set GCP_REGION --env Production --body "<GCP_REGION>"
gh variable set ARTIFACT_REPOSITORY --env Production --body "cloud-run-source-deploy"
gh variable set API_SERVICE --env Production --body "ltc-api"
gh variable set MIGRATION_JOB --env Production --body "ltc-api-migrate"
gh secret set GCP_WIF_PROVIDER --env Production --body "projects/<GCP_PROJECT_NUMBER>/locations/global/workloadIdentityPools/github-pool/providers/github-provider"
gh secret set GCP_DEPLOY_SA --env Production --body "github-actions-deployer@<GCP_PROJECT_ID>.iam.gserviceaccount.com"
```
