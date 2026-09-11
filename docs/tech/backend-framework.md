---
doc_type: architecture
covers:
  - apps/api/
---

# 後端框架與分層架構

> 本文件描述目前實作。後續 Clean Architecture＋DDD 的目標邊界、開發規範與遷移順序，請先讀 [應用程式目標架構](application-architecture.md)；目標尚未全面落實，現有架構檢查仍須遵守，切片遷移時同步更新檢查規則。

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

internal/modules/<capability>/
  transport   解 request、呼叫 app、組 response，是唯一有 binding tag 與 gin 的地方
  app         use case、業務規則，以及這個模組需要的 port interface
  infra       SQL、交易、外部服務呼叫、檔案產生，實作 app 的 port

internal/platform     跨模組的技術底層：config、httpx、auth、logging、pgxdb
internal/domain       跟框架無關的純業務邏輯（見下方）
internal/arch         架構測試：匯入矩陣檢查，跟著 go test ./... 一起跑
```

以上是現行命名。目標命名（`domain`／`application`／`adapters`，`internal/sharedkernel`、`internal/bootstrap`）見 [應用程式目標架構](application-architecture.md)；某模組要到完成切片且 `internal/arch/arch_test.go` 擴充到能辨識其新 layout 後，才切換到目標命名，切換前現行命名仍然有效。

目前的能力模組：

| module | 範圍 |
|---|---|
| `masterdata` | 據點、車輛、司機、區域主檔 |
| `casemgmt` | 個案主檔、排班設定、交通偏好、個案彙整表匯出 |
| `caseimport` | 個案批次 `.xlsx` 解析、預覽與匯入 |
| `ride` | 搭乘紀錄合併與人工更正、搭乘月曆、衝突裁決；接送匯報的展開與解析屬 `driverreport`，`ride` 透過其 port 消費 |
| `driverreport` | 車輛匯報表登錄、`.xlsx` 匯入、解析與欄位對應 |
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
| `domain/merge` | 混車合併演算法：多個匯報來源回報同一趟時，「同一台車取最新回報、跨車用 OR」合併成單一 `ride_records` 狀態，同時保護已被人工裁決／更正過的紀錄不被自動覆蓋 |
| `domain/namenorm` | 姓名／表單欄名正規化：NFKC 正規化、去空白、異體字轉標準字、Levenshtein 編輯距離比對，用來把司機在匯報表填的姓名（可能有錯字、簡繁混用）配對回司機主檔 |
| `domain/rocdate` | 西元 ↔ 民國年（ROC）互轉，政府申報表格式要求 |
| `domain/timeslot` | 依出發時間＋服務時長算結束時間，處理跨小時、防止跨日 |
| `domain/crypto` | 身分證字號的檢查碼驗證、AES-256-GCM 加密／解密、HMAC 索引（用於唯一性比對又不明碼儲存）、遮罩顯示（`A20***9750`） |
| `domain/govform` | 政府申報 Excel 的 33 欄標題定義、資料列組裝、排序規則 |

## Auth

正式流程：前端帶 `Authorization: Bearer <supabase JWT>` → `middleware.AuthMiddleware` 用 Supabase JWKS 驗簽 → 從 claim 取得 actor identity，並以共享的 user security state projection 解析停用狀態、角色與個人覆寫權限 → 路由層用 `auth.RequirePermission(module, action)` 判斷。角色目前大致有 `admin` / `staff` / `dispatcher` / `driver` / `viewer`，實際哪個角色能打哪個 API 以 [backend-api-reference.md](backend-api-reference.md) 為準。

`APP_ENV=local` 時額外接受 `Authorization: Bearer mock_jwt_admin` 這種 token 字串（token 裡包含角色名稱關鍵字就吃該角色，預設 `staff`），讓本機沒接 Supabase 也能用正常的登入表單進系統；它仍要通過 data plane 檢查。`X-Mock-Role` header 後門已移除。

這條降級只在 `local` 生效，寫死在 `internal/platform/auth/auth.go`，改動時要非常小心不要讓它漏到 production 判斷分支裡。

司機接送匯報沒有免驗證入口：資料一律由已登入的 staff／admin 透過 `POST /api/v1/driver-reports/:id/import` 上傳 `.xlsx` 進來，走一般的 JWT 驗證。

## Response 格式

統一封裝在 `internal/platform/httpx/response.go`：

```jsonc
// 成功
{ "data": ..., "meta": {...可省略，通常是分頁資訊...} }

// 失敗
{ "error": { "code": "VALIDATION_FAILED", "message": "...", "details": [{"field":"...", "reason":"..."}], "requestId": "9F2C1A4B7E0D" } }
```

錯誤碼常數的權威清單在 `httpx`，可由 `httpx.ErrorCodes()` 取得；目前為 `VALIDATION_FAILED`、`UNAUTHENTICATED`、`FORBIDDEN`、`NOT_FOUND`、`ASSIGNMENT_OVERLAP`、`EXPORT_IN_PROGRESS`、`PRECHECK_FAILED`、`NO_EXPORT_DATA`、`MAPPING_REQUIRED`、`DRIVER_REPORT_IMPORT_FAILED`、`FORM_MAPPING_FAILED`、`INTERNAL_ERROR`、`SERVICE_UNAVAILABLE`、`RESOURCE_IN_USE`、`UNSUPPORTED_FILE_TYPE`、`FILE_TOO_LARGE`、`FILE_UNREADABLE`、`IMPORT_TEMPLATE_MISMATCH`、`ROUTE_NOT_FOUND`。新增錯誤情境優先看有沒有現成碼可以用，真的沒有再加新常數，不要在 handler 裡面手打字串。

### 三個欄位的分工

- `code`：機器可讀的分類，決定前端的處置（例如 401 導回登入、`RESOURCE_IN_USE` 不重試）。新增常數時 `codeMessages` 必須同步登記，否則 `internal/platform/httpx` 的測試會失敗。
- `message`：直接顯示給使用者的具體原因，由 handler 寫死。`internal/arch` 的 `TestErrorMessagesAreStaticText` 禁止把函式呼叫結果（例如 `err.Error()`）當成 message，因此前端可以原樣顯示；傳空字串時 `RespondError` 會自動填入該錯誤碼的預設文字。`RespondErrorCode` 則只寫預設文字，底層錯誤僅進 slog。
- `details`：欄位層級原因。request body 綁定失敗一律附 `httpx.ExtractValidationDetails(err)`，前端才說得出是哪一欄不合規則。
- `requestId`：`httpx.RequestIDMiddleware()` 產生，同時寫入 `X-Request-Id` header 與 slog 的 `request_id`。前端在 5xx 時把它一起顯示為「錯誤編號」，供反查伺服器 log。

前端字典在 `apps/web/src/api/errorCodes.ts`，由 `apps/web/tests/unit/error-codes.test.ts` 直接讀本檔對應的 Go 原始碼比對，漏改會在測試階段被擋下。

## 設定（環境變數）

集中在 `internal/platform/config/config.go`，啟動時直接驗證並在缺漏必填值時拒絕啟動：

- 必填：`APP_ENV`（僅接受 `local` 或 `production`）、`DATABASE_URL`、`ENCRYPTION_KEY`、`HMAC_KEY`（兩把 32 bytes base64 金鑰，且不可相同）。
- `APP_ENV=production` 時額外必填：`SUPABASE_JWKS_URL`、`ALLOWED_ORIGINS`、`TRUSTED_PROXIES`。
- 選填：`PORT`、`DB_MAX_CONNS`、`DB_MIN_CONNS`、`DB_MAX_CONN_LIFETIME`、`DB_MAX_CONN_IDLE_TIME`、`SUPABASE_PROJECT_REF`、`STORAGE_BUCKET`、`STORAGE_SIGNED_URL_TTL`、`GOOGLE_SA_JSON`、`SENTRY_DSN`、`LOG_LEVEL`。
- `NOTIFICATION_EMAIL_ENABLED` 明確控制寄信；`false` 時一律使用 `LogEmailSender`，即使 `RESEND_API_KEY` 存在也只寫入資料庫與 log。設為 `true` 時 `RESEND_API_KEY` 與 `NOTIFY_FROM` 都必填。
