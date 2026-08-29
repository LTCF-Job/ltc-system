# 後端框架與分層架構

給要動 `apps/api` 程式碼的人看。技術棧、分層邊界、domain 套件、Auth 機制、response 格式。API 完整路由表另見 [backend-api-reference.md](backend-api-reference.md)，業務流程另見 [backend-flows.md](backend-flows.md)。

## 技術棧

| 項目 | 選擇 |
|---|---|
| 語言／版本 | Go 1.25 |
| HTTP 框架 | Gin（`gin-gonic/gin`） |
| 資料庫驅動 | pgx v5（`jackc/pgx/v5`），直接寫 SQL，沒有 ORM |
| 認證 | JWT，透過 Supabase JWKS 端點驗簽（`MicahParks/keyfunc`、`golang-jwt/jwt/v5`） |
| 設定管理 | `kelseyhightower/envconfig`，環境變數驅動，啟動時驗證 |
| Excel 匯出 | `xuri/excelize/v2` |
| 測試 | 標準庫 `testing` + `stretchr/testify` |

沒有用任何 DI 框架，所有物件組裝（infra → app → transport）都寫在 `cmd/server/main.go` 裡，一行一行手動 new 出來再傳進去。

## 目錄與分層

```
cmd/server    HTTP 服務入口。main.go 只做 dependency wiring，routes.go 放路由表
cmd/migrate   跑 migration 的 CLI
cmd/exporter  Cloud Run Job，跑政府申報 Excel 匯出

internal/modules/<capability>/
  transport   解 request、呼叫 app、組 response，是唯一有 binding tag 與 gin 的地方
  app         use case、業務規則，以及這個模組需要的 port interface
  infra       SQL、交易、外部服務呼叫、檔案產生，實作 app 的 port

internal/platform     跨模組的技術底層：config、httpx、auth、logging、pgxdb
internal/domain       跟框架無關的純業務邏輯（見下方）
internal/arch         架構測試：匯入矩陣檢查，跟著 go test ./... 一起跑
```

目前的能力模組：

| module | 範圍 |
|---|---|
| `masterdata` | 據點、車輛、司機、區域主檔 |
| `casemgmt` | 個案主檔、排班設定、交通偏好、個案彙整表匯出 |
| `caseimport` | 個案批次 Excel／CSV 解析、預覽與匯入 |
| `ride` | Google 表單回報、搭乘紀錄合併與人工更正 |
| `formsync` | Google 表單登錄與欄位對應 |
| `reporting` | 趟數表、新竹時刻表、儀表板、前置檢核、政府申報匯出 |
| `ops` | 司機出勤、油資、車輛維修 |
| `notification` | 通知收件人與寄送留痕 |
| `holiday` | 國定假日維護與政府行事曆同步 |
| `audit` | 稽核日誌寫入與查詢 |
| `task` | 缺報偵測與月結排程作業 |

模組之間**不互相 import**。跨模組協作一律由消費端在自己的 `app/ports.go` 宣告 port，
再由 `cmd/server` 注入 adapter（見 `cmd/server/module_adapters.go`）；稽核寫入也是同一個
模式（`cmd/server/audit_adapters.go`）。

新增或修改功能時，依賴方向、模型所有權、驗證位置、錯誤與交易歸屬、port 定義位置、檔案拆分時機，以 [`layering-rules.md`](../../.agents/skills/backend-architecture/references/layering-rules.md) 為準；那份文件裡的規則多數已被 `internal/arch/arch_test.go` 編碼，違反會讓 `go test ./...` 失敗。`arch_test.go` 的 baseline 目前是空的——過渡期的既有違規已全數清除，新的違規是要修的缺陷，不是可以加進 baseline 的項目。邊界背後的原則見 [`backend-architecture/SKILL.md`](../../.agents/skills/backend-architecture/SKILL.md) 與 [`go-backend-code-style/SKILL.md`](../../.agents/skills/go-backend-code-style/SKILL.md)。

`main.go` 裡每個 `xxxRepo := xxxinfra.NewXxxRepository(pool)` → `xxxSvc := xxxapp.NewXxxService(xxxRepo, ...)` → `handlers{...}` 欄位就是一組完整的 vertical slice，看不懂某個功能全貌時，從這幾行找起最快。

## domain 套件（純邏輯，不碰 DB／HTTP）

這幾個套件是整個系統最核心、最容易出錯也最值得先看懂的部分：

| 套件 | 做什麼 |
|---|---|
| `domain/calendar` | 依個案的排班規則（星期、起訖日、四趟制）算出某個月「應該搭乘」的完整日曆，是「未回報偵測」跟「異常比對」的比對基準 |
| `domain/merge` | 混車合併演算法：多個 Google 表單來源回報同一趟時，「同一台車取最新回報、跨車用 OR」合併成單一 `ride_records` 狀態，同時保護已被人工裁決／更正過的紀錄不被自動覆蓋 |
| `domain/namenorm` | 姓名／表單欄名正規化：NFKC 正規化、去空白、異體字轉標準字、Levenshtein 編輯距離比對，用來把司機在 Google 表單填的姓名（可能有錯字、簡繁混用）配對回司機主檔 |
| `domain/rocdate` | 西元 ↔ 民國年（ROC）互轉，政府申報表格式要求 |
| `domain/timeslot` | 依出發時間＋服務時長算結束時間，處理跨小時、防止跨日 |
| `domain/crypto` | 身分證字號的檢查碼驗證、AES-256-GCM 加密／解密、HMAC 索引（用於唯一性比對又不明碼儲存）、遮罩顯示（`A20***9750`） |
| `domain/govform` | 政府申報 Excel 的 33 欄標題定義、資料列組裝、排序規則 |

## Auth

正式流程：前端帶 `Authorization: Bearer <supabase JWT>` → `middleware.AuthMiddleware` 用 Supabase JWKS 驗簽 → 從 claim 的 `user_metadata.role`（或 `app_metadata.role`）拿角色，寫進 Gin context（`actor_id`、`actor_role`）→ 路由層用 `middleware.RequireRoles("staff", "admin")` 這種白名單擋。角色目前大致有 `admin` / `staff` / `dispatcher` / `driver` / `viewer`，實際哪個角色能打哪個 API 以 [backend-api-reference.md](backend-api-reference.md) 為準。

`APP_ENV=local` 時額外支援兩種免登入方式，方便本機測試不同角色：

- header 帶 `X-Mock-Role: admin`（可搭配 `X-Mock-User-ID`）直接跳過 JWT 驗證。
- `Authorization: Bearer mock_jwt_admin` 這種 token 字串（token 裡包含角色名稱關鍵字就吃該角色，預設 `staff`）。

這兩種只在 `local` 生效，寫死在 `internal/platform/auth/auth.go`，改動時要非常小心不要讓它漏到 production 判斷分支裡。

Google 表單回填走另一條路：`POST /api/v1/ingest/google-form`，不走 JWT，用 `X-Ingest-Token` header 驗證（token 存在 `forms.secret` 欄位，一個表單一組）。

## Response 格式

統一封裝在 `internal/platform/httpx/response.go`：

```jsonc
// 成功
{ "data": ..., "meta": {...可省略，通常是分頁資訊...} }

// 失敗
{ "error": { "code": "VALIDATION_FAILED", "message": "...", "details": [{"field":"...", "reason":"..."}] } }
```

錯誤碼是 `httpx` 的常數：`VALIDATION_FAILED`、`UNAUTHENTICATED`、`FORBIDDEN`、`NOT_FOUND`、`DUPLICATE_NATIONAL_ID`、`ASSIGNMENT_OVERLAP`、`EXPORT_IN_PROGRESS`、`PRECHECK_FAILED`、`MAPPING_REQUIRED`、`INGEST_TOKEN_INVALID`、`FORM_SOURCE_FAILED`、`FORM_SYNC_FAILED`、`FORM_MAPPING_FAILED`、`INTERNAL_ERROR`。新增錯誤情境優先看有沒有現成碼可以用，真的沒有再加新常數，不要在 handler 裡面手打字串。

## 設定（環境變數）

集中在 `internal/platform/config/config.go`，啟動時直接驗證並在缺漏必填值時拒絕啟動：

- 必填：`APP_ENV`（僅接受 `local` 或 `production`）、`DATABASE_URL`、`ENCRYPTION_KEY`、`HMAC_KEY`（兩把 32 bytes base64 金鑰，且不可相同）。
- `APP_ENV=production` 時額外必填：`SUPABASE_JWKS_URL`、`ALLOWED_ORIGINS`。
- 選填：`PORT`、`DB_MAX_OPEN_CONNS`、`DB_MAX_IDLE_CONNS`、`SUPABASE_PROJECT_REF`、`STORAGE_BUCKET`、`STORAGE_SIGNED_URL_TTL`、`GOOGLE_SA_JSON`、`RESEND_API_KEY`、`NOTIFY_FROM`、`SENTRY_DSN`、`LOG_LEVEL`。
