# tests/qa-crud — 前端 CRUD 全功能測試

用 Playwright 以真實使用者操作驅動前端，記錄前端送出的 request、後端回應與畫面回饋，再回 PostgreSQL 驗證資料。

測試流程與判準（測什麼、怎麼判、踩過哪些坑）寫在 [docs/tech/qa-crud-testing.md](../../docs/tech/qa-crud-testing.md)，這份只放怎麼跑。

## 前置條件

- 本機服務已啟動：`docker compose -f docker-compose.local.yml up -d`（web `:3000`、api `:8080`、postgres `:5432`）。
- `apps/web/node_modules` 已安裝（harness 直接 require 該目錄的 `playwright`），且 Playwright 的 chromium 已下載。
- 匯入匯出相關 suite 需要 Python 3 與 openpyxl（`tests/qa-crud/xlsx.py` 使用）。

## 執行

```bash
node tests/qa-crud/run.cjs
```

不帶參數會依序跑完 `suites/` 底下所有 suite。帶參數則只跑檔名包含該字串的 suite：

```bash
node tests/qa-crud/run.cjs 03-vehicles 04-drivers
```

結果同時輸出到 stdout 與 `tests/qa-crud/reports/report-<timestamp>.json`。

### 環境變數

| 變數 | 預設 | 用途 |
|---|---|---|
| `QA_BASE_URL` | `http://localhost:3000` | 前端位址 |
| `QA_LOGIN_EMAIL` | `admin@example.com` | 本機 mock 登入帳號；含 `viewer` 字樣會以檢視人員身分登入 |
| `QA_LOGIN_PASSWORD` | `pw123456` | 本機 mock 登入不驗密碼，任意非空值即可 |
| `QA_TAG` | `QA` + 時間戳末 6 碼 | 測試資料名稱前綴 |
| `QA_HEADED` | 未設定（headless） | 設為 `1` 開啟有頭瀏覽器觀察操作過程 |
| `QA_DESTRUCTIVE` | 未設定 | 設為 `1` 才會執行會破壞既有 demo 資料的刪除檢查 |

## Suite 清單

| 檔案 | 範圍 |
|---|---|
| `00-route-sweep.cjs` | 全路由煙霧測試：逐頁載入，收集 console error 與非 2xx 回應 |
| `01-regions.cjs` | 地區管理 CRUD |
| `02-sites.cjs` | 單位管理 CRUD、區域下拉值域比對 |
| `03-vehicles.cjs` | 車輛 CRUD、司機指派、刪除保護 |
| `04-drivers.cjs` | 司機 CRUD、身分證驗證、車輛指派 |
| `05-caregivers.cjs` | 照護人員 CRUD |
| `10-templates-download.cjs` | 各功能匯入範本與匯出檔下載 |
| `11-case-import.cjs` | 個案匯入：直接上傳官方範本 |
| `12-case-import-combinations.cjs` | 個案匯入：欄位組合、冪等性 |
| `13-caregiver-import.cjs` | 照護人員匯入 |
| `14-maintenance.cjs` | 車輛維修保養 CRUD |
| `15-attendance-fuel.cjs` | 出勤登記與油資 CRUD |
| `16-holidays.cjs` | 假日 CRUD 與政府行事曆匯入 |
| `17-notifications.cjs` | 通知收件人 CRUD |
| `18-roles-users-audit.cjs` | 角色、使用者、稽核紀錄 |
| `19-cases-crud.cjs` | 個案 CRUD 與編輯頁 |
| `20-driver-report-import.cjs` | 司機接送匯報批次上傳 |
| `21-rides.cjs` | 搭乘月曆、異常集中處理、未回報催報 |
| `22-exports.cjs` | 政府申報前置檢核與匯出 |
| `90-discovery.cjs` | 頁面控制項探索，寫新 suite 前用來確認實際按鈕與頁籤 |

## 重建匯入測試檔

`fixtures/*.xlsx` 是由當下下載的官方範本填出來的執行產物（不進版控），範本改版時要重新產生。先跑一次 `10-templates-download` 把範本抓到 `downloads/`，再執行：

```bash
python tests/qa-crud/xlsx.py fill "tests/qa-crud/downloads/個案批次匯入範本.xlsx" tests/qa-crud/fixtures/case-import-filled.xlsx tests/qa-crud/fixtures/case-import-rows.json 2
```

```bash
python tests/qa-crud/xlsx.py fill "tests/qa-crud/downloads/照護人員批次匯入範本.xlsx" tests/qa-crud/fixtures/caregiver-import-filled.xlsx tests/qa-crud/fixtures/caregiver-import-rows.json 2
```

司機匯報測試檔不套範本，直接依欄位順序（民國日期／駕駛人／各個案欄／備註）產生，檔名要含車輛代稱才會自動比對到車輛：

```bash
python tests/qa-crud/xlsx.py create "tests/qa-crud/fixtures/AAA-123 (回覆).xlsx" tests/qa-crud/fixtures/driver-report-rows.json 司機接送匯報
```

`*-rows.json` 是二維陣列，一個內層陣列一列，可直接編輯來調整測試組合。

## 新增 suite

在 `suites/` 建立 `NN-<name>.cjs`，export `name` 與 `run`：

```js
exports.name = '模組名稱 CRUD'

exports.run = async ({ page, net, record, step, L }) => {
  await L.goto(page, '/your/route')
  await L.openCreate(page, '新增XX')
  await step('空白送出', () => L.submit(page, net))
}
```

- `step(名稱, fn)`：包住會拋錯的操作，單一步驟失敗不會中斷整個 suite。
- `record(名稱, 物件)`：直接記一筆不會失敗的觀察結果。
- `L`：`lib.cjs` 的操作輔助（`fill` / `pick` / `radio` / `pickDate` / `selectOptions` / `submit` / `confirmBox` / `download` / `tableRows` / `resolveRow`…）。
- `net.calls`：本次執行所有寫入類 request 與其回應，用 `net.calls.length` 取切片界定某個步驟的區間。

## 測試資料清除

```bash
docker exec -i ltc-postgres psql -U postgres -d ltc_system < tests/qa-crud/cleanup.sql
```

清除所有名稱帶 `QA` 前綴的測試資料。`downloads/` 與 `reports/` 是執行產物，可直接刪除。
