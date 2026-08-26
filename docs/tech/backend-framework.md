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

沒有用任何 DI 框架，所有物件組裝（repository → service → handler）都寫在 `cmd/server/main.go` 裡，一行一行手動 new 出來再傳進去。

## 目錄與分層

```
cmd/server    HTTP 服務入口，路由表跟 dependency wiring 都在這
cmd/migrate   跑 migration 的 CLI
cmd/exporter  Cloud Run Job，跑政府申報 Excel 匯出

internal/handler      解 request、呼叫 service、組 response，不碰 SQL
internal/service       商業邏輯：驗證、狀態轉換、跨 repo 協調
internal/repository    SQL 查詢、transaction，回傳 repository 自己的 entity struct
internal/domain        跟框架無關的純邏輯（見下方）
internal/middleware    JWT 驗證、角色檢查、CORS、audit log、統一 response 格式
internal/export        Excel 產生
internal/config        環境變數載入跟啟動檢查
```

依賴方向固定是 `handler → service → repository`，`domain` 誰都能呼叫但不反過來依賴任何人。加新功能照這個順序寫：

1. 在對應 `repository` 加查詢方法，回傳 repository 自己的 entity struct（不是 domain 型別、也不是 API DTO）。
2. 在對應 `service` 加業務方法：吃 repository，做驗證、狀態轉換、跨 repository 協調。
3. 在對應 `handler` 加一個方法：解 request（binding）、呼叫 service、用 `middleware.RespondSuccess` / `RespondError` 組 response。
4. 在 `cmd/server/main.go` 建構這三層物件，掛路由並指定 `middleware.RequireRoles(...)`。

`main.go` 裡每個 `xxxRepo := repository.NewXxxRepository(pool)` → `xxxSvc := service.NewXxxService(xxxRepo, ...)` → `xxxH := handler.NewXxxHandler(xxxSvc)` 就是一組完整的 vertical slice，看不懂某個功能全貌時，從這三行找起最快。

想確認邊界細節時讀 [`.agents/skills/backend-architecture/SKILL.md`](../../.agents/skills/backend-architecture/SKILL.md) 跟 [`.agents/skills/go-backend-code-style/SKILL.md`](../../.agents/skills/go-backend-code-style/SKILL.md)，那是團隊目前實際在遵守的規則。

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

這兩種只在 `local` 生效，寫死在 `internal/middleware/auth.go`，改動時要非常小心不要讓它漏到 production 判斷分支裡。

Google 表單回填走另一條路：`POST /api/v1/ingest/google-form`，不走 JWT，用 `X-Ingest-Token` header 驗證（token 存在 `forms.secret` 欄位，一個表單一組）。

## Response 格式

統一封裝在 `internal/middleware/response.go`：

```jsonc
// 成功
{ "data": ..., "meta": {...可省略，通常是分頁資訊...} }

// 失敗
{ "error": { "code": "VALIDATION_FAILED", "message": "...", "details": [{"field":"...", "reason":"..."}] } }
```

錯誤碼是常數：`VALIDATION_FAILED`、`UNAUTHENTICATED`、`FORBIDDEN`、`NOT_FOUND`、`DUPLICATE_NATIONAL_ID`、`ASSIGNMENT_OVERLAP`、`EXPORT_IN_PROGRESS`、`PRECHECK_FAILED`、`MAPPING_REQUIRED`、`INGEST_TOKEN_INVALID`、`INTERNAL_ERROR`。新增錯誤情境優先看有沒有現成碼可以用，真的沒有再加新常數，不要在 handler 裡面手打字串。

## 設定（環境變數）

集中在 `internal/config/config.go`，啟動時直接驗證並在缺漏必填值時拒絕啟動：

- 必填：`APP_ENV`（僅接受 `local` 或 `production`）、`DATABASE_URL`、`ENCRYPTION_KEY`、`HMAC_KEY`（兩把 32 bytes base64 金鑰，且不可相同）。
- `APP_ENV=production` 時額外必填：`SUPABASE_JWKS_URL`、`ALLOWED_ORIGINS`。
- 選填：`PORT`、`DB_MAX_OPEN_CONNS`、`DB_MAX_IDLE_CONNS`、`SUPABASE_PROJECT_REF`、`STORAGE_BUCKET`、`STORAGE_SIGNED_URL_TTL`、`GOOGLE_SA_JSON`、`RESEND_API_KEY`、`NOTIFY_FROM`、`SENTRY_DSN`、`LOG_LEVEL`。
