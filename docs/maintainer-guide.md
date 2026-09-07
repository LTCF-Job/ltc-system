---
doc_type: architecture
covers: []
---

# ltc-system 維護者指南

本文件是接手 `ltc-system` 時的第一個入口，描述目前程式碼實際形成的架構、請求流程、主要功能與資料責任。內容以目前 `main` 的實作與設定為準；規劃文件不取代程式碼證據。

## 閱讀順序與現況標記

建議依下列順序閱讀：

1. 本文件：先建立系統與模組地圖。
2. [維護 Runbook](tech/maintainer-runbook.md)：啟動、migration、health check 與故障排查。
3. [Backend API Reference](tech/backend-api-reference.md)：確認現行 route、request、response 與權限。
4. [Backend Flows](tech/backend-flows.md) 與 [Frontend Flows](tech/frontend-flows.md)：追蹤跨層業務流程。
5. [Full-stack Review](reviews/2026-09-04-full-stack-review.md)：查看已發現的 bug、契約落差與未驗證項目。

文件中的證據分成兩種：

- **靜態已確認**：由目前 source、migration、設定或文件內容直接確認。
- **待 runtime 驗證**：需要實際 PostgreSQL、Supabase、外部服務、Docker 或部署環境才能確認；不把 mock 或型別宣告當成 runtime 證據。

## 系統拓樸

```text
使用者瀏覽器
    │
    ▼
Vue 3 SPA (apps/web)
    │  Axios /api/v1、JWT、module permission
    ▼
Go API / Gin (apps/api)
    │
    ├─ platform/auth       JWT 驗證、JWKS、role / permission
    ├─ platform/httpx      response envelope、錯誤碼
    ├─ modules/*/transport HTTP handler、DTO、route
    ├─ modules/*/app       use case、業務規則、port
    └─ modules/*/infra     PostgreSQL repository、外部 adapter
          │
          ├─ pgxpool ───────────────► Supabase PostgreSQL
          ├─ Supabase Admin API ────► Supabase Auth user management
          └─ Holiday provider ─────► 政府假日資料來源

Migration / seed
    └─ apps/api/cmd/migrate ───────► PostgreSQL schema、RLS、seed data
```

目前部署邊界是前後端分離：web 依部署設定提供靜態 SPA，API 以 Go service 執行。正式環境的 JWT、資料庫連線、Supabase Admin 與外部 provider 必須分別驗證；本機 local/offline 模式不能代表正式整合已可用。

## Repository 結構

```text
ltc-system/
├─ apps/
│  ├─ api/
│  │  ├─ cmd/server/       API composition root、route wiring
│  │  ├─ cmd/migrate/      migration runner、seed runner
│  │  ├─ internal/
│  │  │  ├─ modules/       audit、case、caregiver、driver report、ride...
│  │  │  ├─ platform/      auth、config、db、httpx、middleware
│  │  │  └─ arch/          modular boundary architecture test
│  │  ├─ migrations/       PostgreSQL schema migrations
│  │  └─ seed/             baseline、demo seed、reference data
│  └─ web/
│     └─ src/
│        ├─ api/            Axios API client、DTO/types
│        ├─ app/            app bootstrap、permission state
│        ├─ features/       頁面與功能模組
│        ├─ layouts/        DefaultLayout、navigation
│        ├─ router/         route、auth guard、module guard
│        ├─ shared/         shared UI / composables / types
│        └─ utils/          formatters、共用工具
├─ docs/
│  ├─ maintainer-guide.md  維護者總覽（本文件）
│  ├─ tech/                 現行技術、API、流程與操作文件
│  ├─ reviews/              審查報告與 backlog
│  ├─ operations/           操作頁面與視覺驗收產物，不覆寫既有檔案
│  └─ flows/                歷史流程文件，需看文件頂端標記
└─ tests/                   repository-level 或跨層測試資產
```

## Backend modular monolith

每個業務能力位於 `apps/api/internal/modules/<capability>/`，原則上分為：

| 層 | 責任 | 不應承擔的責任 |
| --- | --- | --- |
| `transport` | Gin handler、request/response DTO、參數解析、HTTP 狀態碼 | SQL、跨模組業務決策 |
| `app` | use case、domain rule、port、跨 repository 協作 | 直接依賴 Gin 或 PostgreSQL schema 細節 |
| `infra` | PostgreSQL repository、Supabase／外部 adapter | 取代 use case 的權限與流程規則 |

目前主要模組如下：

| 模組 | 主要能力 | 主要資料／外部責任 |
| --- | --- | --- |
| `identity` | 登入後的 user、role、permission 與密碼管理 | Supabase Auth Admin、`roles`、`permissions`、`user_roles`、`user_permissions` |
| `masterdata` | 區域、服務據點、車輛、司機、照服員 | master data tables、soft delete、active 狀態 |
| `caseimport` | Excel 個案／司機匯入與 mapping | `.xlsx` parser、case／driver import staging 與 upsert |
| `casemgmt` | 個案查詢、建檔、更新與刪除 | `cases`、個案服務設定 |
| `ride` | 行程、行程問題、缺漏報表、行程合併 | rides、ride issues、driver reports、case／vehicle 關聯 |
| `driverreport` | 司機回報、狀態、批次匯入與 mapping | driver report records、import result |
| `reporting` | 行程摘要、排班、匯出與 export job | report query、precheck、export metadata／file |
| `task` | 缺漏檢查、排程任務、回報狀態協作 | task／missing report use case，可能觸發 notification |
| `attendance`（位於 `ops`） | 出勤與油資 | attendance query／write、holiday lookup |
| `audit` | 操作稽核紀錄 | append-oriented audit records |
| `holiday` | 假日查詢與快取 | government holiday provider、holiday table |
| `notification` | 通知設定、通知紀錄、寄信 port | notification settings/logs；目前 sender 是 simulated |

`apps/api/cmd/server/routes.go` 是現行 HTTP route 的入口；`apps/api/cmd/server/main.go` 負責依賴注入。模組之間應透過 composition root 注入的 port 協作，不能把另一模組的 `infra` 或 repository 當成公共 API。`internal/arch/arch_test.go` 是邊界的自動化約束，但仍需搭配契約與 runtime 檢查。

## API request、JWT 與權限流程

```text
Browser
  │ Authorization: Bearer <JWT>
  ▼
Gin route
  │ 1. CORS / auth middleware
  │ 2. JWT signature、issuer、audience、expiry
  │ 3. app_metadata.role → built-in role
  │ 4. RequirePermission(module, action)
  ▼
Handler
  │ request DTO / query parse
  ▼
Use case (app)
  │ repository / external port
  ▼
Response envelope
  ├─ success: { data, meta }
  └─ error:   { error: { code, message, details } }
```

### JWT 與角色來源

- production 由 Supabase JWKS 驗證 JWT；`issuer`、`audience`、signature 與 expiry 必須符合設定。
- built-in role 只從 JWT `app_metadata.role` 讀取；`user_metadata.role` 或 top-level `role` 不應被當成後端授權依據。
- local 可接受 `mock_jwt_` token，這是開發便利功能，不是 production authentication。
- route 現況以 `RequirePermission` 為主；`/api/v1/auth/me` 與 `/api/v1/auth/change-password` 只要求已驗證身分。不存在的 `/demo/reset` 與 `RequireRoles` 不應再寫入現行流程。
- custom permission 會由資料庫載入並在 process 內短暫 cache；多 instance 的撤權不會即時同步，詳見 review report。

後端是最後授權邊界。前端 route guard、menu visibility 與 module permission 是使用者體驗與預防性控制，不能取代 API 的 401／403。

## Frontend 結構與 route

前端是 Vue 3 SPA，主要路徑如下：

| 區域 | 目前責任 |
| --- | --- |
| `src/router/index.ts` | 所有頁面 route、redirect、route meta |
| `src/router/guards.ts` | token／user session、module permission guard |
| `src/layouts/DefaultLayout.vue` | sidebar、navigation、logout、頁面容器 |
| `src/api/client.ts` | Axios base URL、JWT header、401 handling、錯誤 mapping |
| `src/api/*.ts` | 各功能 API client 與前端 DTO |
| `src/stores/auth.ts` | session、user、permissions 的 local state 與 localStorage |
| `src/features/*` | 頁面、表單、表格、互動與功能專屬 composable |
| `src/utils/formatters.ts` | 日期／時間與顯示格式；時間規格只到秒 |

目前 route 入口：

```text
/login
/
/cases                  /cases/:id
/masters/regions        /masters/sites
/masters/vehicles       /masters/drivers       /masters/caregivers
/driver-reports         /driver-reports/status /driver-reports/import
/driver-reports/mappings
/rides                  /rides/issues          /rides/missing
/reports/trip-summary   /reports/hsinchu-schedule
/vehicles/maintenance  /attendance             /audit
/settings/users         /settings/roles        /settings/notifications
/settings/holidays      /exports
```

route meta 目前以 `title`、`module` 等頁面資訊為主，沒有一份獨立的 role whitelist 可取代後端 permission matrix。新增頁面時，需同時更新 route、sidebar/menu、API client、permission module 與本文件／技術文件。

### Frontend 時間規格

使用 `@/utils/formatters` 顯示時間，規格是 `YYYY-MM-DD HH:mm:ss`；純時間是 `HH:mm:ss`。不要直接把 raw ISO 8601、毫秒或時區字尾渲染給使用者。輸入日期、API query 的 timezone 轉換則必須在功能文件中說明，不能只依瀏覽器 local timezone 猜測。

## PostgreSQL migration、seed 與資料責任

### Migration

- schema migration 位於 `apps/api/migrations/`，以序號檔名決定執行順序。
- up／down 內容、index、foreign key、soft-delete 欄位與 constraint 是資料契約；修改欄位時必須同步 repository、DTO、seed、報表 query 與文件。
- `apps/api/cmd/migrate` 會依 migration runner 執行 schema，並可在設定允許時執行 default admin bootstrap。
- migration runner 目前沒有可取代人工協調的跨 instance lock；正式環境要避免多個 migration job 同時執行。

### Seed

- `apps/api/seed/` 的 baseline／reference data 是開發與測試資料來源，不是 production 真實資料。
- seed 必須跟目前 schema 的 constraint、欄位與 enum/status 保持一致；舊 demo seed 可能與最新 migration 漂移，執行前先看 review report。
- default admin 只有在同時提供 `DEFAULT_ADMIN_EMAIL`、`DEFAULT_ADMIN_PASSWORD` 以及可用的 Supabase Admin 設定時才會嘗試建立／補上 admin metadata；文件與 repository 不能記錄真實密碼。

### 資料責任

| 資料類型 | Source of truth | 注意事項 |
| --- | --- | --- |
| Authentication identity | Supabase Auth | JWT 與 Admin API 的 user identity 不等於業務角色資料 |
| Authorization | `roles`／`permissions`／user binding tables + JWT built-in role | app metadata、DB custom permission、前端 session 可能有同步延遲 |
| 業務資料 | Supabase PostgreSQL tables | repository 應負責 soft-delete、FK、pagination 與 transaction 契約 |
| API contract | Go DTO／handler + route wiring | `apps/web/src/api` 型別是 consumer view，不能單獨視為 OpenAPI source |
| Demo／mock | `apps/web` mock/demo boundary、seed | 不可用來宣稱正式 Auth、DB、Storage 或外部 provider 已驗證 |
| 通知 | `notification` port 與設定／log | 目前預設注入 `LogEmailSender`，只模擬寄送，不代表 Resend 已接通 |

## 主要功能與跨模組流程

### 個案與主檔

```text
Region / Site / Vehicle / Driver / Caregiver
                    │
                    ▼
              Case management
                    │
                    ▼
         Ride / schedule / report queries
```

主檔先建立可供個案與行程引用的資料。soft delete、active status、FK 與查詢 filter 需一起維護；不能只在前端隱藏停用資料。

### 司機回報到缺漏檢查

```text
Driver report (manual / .xlsx import)
        │
        ├─ parse / validate / mapping
        ├─ persist report records
        └─ query by date / driver / ride
                    │
                    ▼
              Missing / issue views
```

`/rides/missing` 的現行 backend handler 會呼叫 task layer 的 missing-report check；目前靜態證據顯示它可能觸發 notification 且只解析少數 query，與前端的 pagination/filter 宣告不完全一致。這是高優先修正項目，詳見 [審查報告](reviews/2026-09-04-full-stack-review.md)。

### Report、precheck 與 export

```text
Report query
    │
    ├─ summary / schedule data
    ├─ export precheck (data completeness / business rules)
    └─ export job / file metadata
           │
           ▼
      frontend download / result display
```

匯出流程需要區分「查詢成功」、「precheck 通過」、「檔案產生成功」與「下載成功」。任何一層的 error code、empty result、partial result 與 job status 都應寫入 API／流程文件；不可只用前端的 success toast 判斷。

### Attendance 與 holidays

Attendance 會依月份查詢司機／出勤與油資資料，holiday 模組提供假日資料。月份、日期與 timezone 的邊界目前仍有前端 hardcode 與 backend `time.Now`／parse 混用的風險，接手修正時先閱讀 review report 的日期與時區項目。

### Audit 與 notification

重要 mutation 應產生 audit record，notification 應透過 port 觸發並可追蹤結果；目前程式碼存在 audit coverage、error swallowing 與通知 sender 尚未接上真實 provider 的落差。不要把 log 中的 simulated email 當成外部寄信成功。

## 維護者變更檢查表

新增或修改功能時，至少同步檢查：

- route、permission module/action、sidebar visibility 與 API client。
- transport DTO、app port/use case、infra repository 與 response envelope。
- migration、seed、soft-delete／FK／index／unique constraint。
- pagination、query filter、timezone、空資料與錯誤碼。
- audit、notification、transaction／rollback 與 idempotency。
- mock/demo 是否仍在允許的邊界內，是否誤導成正式整合。
- 技術文件、flow 文件與歷史文件標記是否仍正確。

完整的目前問題與後續處理順序集中在 [2026-09-04 Full-stack Review](reviews/2026-09-04-full-stack-review.md)。
