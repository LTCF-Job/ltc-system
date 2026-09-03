---
doc_type: decision
covers:
  - apps/api/internal/platform/config/config.go
  - apps/api/internal/platform/auth/auth.go
  - apps/api/internal/modules/demo/
  - apps/api/migrations/000002_seed_reference_data.up.sql
  - apps/api/migrations/000011_backfill_admin_identity.up.sql
  - apps/api/seed/demo/
  - apps/api/ops/demo-db-roles.sql
  - apps/web/src/api/client.ts
  - apps/web/src/views/auth/LoginView.vue
---

# Demo data-plane：真實流程展示環境

## Context

系統需要一個公開、可讓外部使用者實際操作（不只是看預錄畫面）的展示環境，但展示環境的 CRUD 不能碰到正式業務資料，且不應該讓使用者以為展示環境是走假資料（MSW mock）——展示的可信度來自「這是真的 Go API + 真的 PostgreSQL + 真的 Supabase Auth」。同時，這個展示環境不值得比照正式環境獨立開一個全新 Supabase 專案（多一份帳單、多一份 Auth 使用者管理、多一份維運負擔）。

## Decision

Demo 與正式環境共用同一個 Supabase 專案（含 Auth），但業務資料落在同專案內的第二個 PostgreSQL 資料庫（`ltc_demo`），並各自部署一個 Cloud Run 服務指向不同資料庫：

- 兩個服務執行同一份 image、同一份 migration，只用 `DATA_PLANE`（`production`／`demo`）與 `DATABASE_URL` 兩個環境變數區分。
- migration 對 `auth.*` 的操作全部包在 `IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'auth')` 內：`ltc_demo` 沒有 `auth` schema（Auth 只掛在預設 `postgres` 資料庫），業務 schema 仍照跑。
- 授權層面不能只靠前端挑網址：JWT 的 `app_metadata.data_plane` 必須與服務自己的 `DATA_PLANE` 相符，`internal/platform/auth.Middleware` 對每個請求強制比對，不符就 401——即使有人直接對 Demo API 發送正式帳號的合法 JWT 也會被擋。
- Demo 資料庫的連線角色（`ltc_demo_app`，見 `apps/api/ops/demo-db-roles.sql`）在 Postgres 權限層級被擋掉 `CONNECT` 正式 `postgres` 資料庫的權限，反向亦然；這是比「只在應用層檢查」更底層的一道防線。
- Demo 重置（`POST /api/v1/demo/reset`）用一個行程內的 `sync.RWMutex` 讓一般請求與重置互斥：重置需要獨佔鎖，必須等所有進行中請求釋放共享鎖才能取得；這個機制只在 Demo 是單一 Cloud Run instance 的前提下成立，多 instance 會需要換成資料庫層級的鎖。
- 種子資料（`apps/api/seed/demo/0001_baseline.up.sql`）用純 SQL 撰寫、固定 UUID、`ON CONFLICT` 保證可重複執行；但個案／司機的身分證加密欄位無法用靜態 SQL 產生真正密文（金鑰只有執行中的服務持有），改由 `internal/modules/demo/infra` 在每次重置時，用服務自己的 `EncryptionKey`／`HMACKey` 對種子資料重新加密一輪假身分證字號。
- 前端「demo/demo」快速登入改成真的呼叫 `supabase.auth.signInWithPassword`，不再是純前端的 MSW bypass；`dataPlane` 隨 JWT 的 `app_metadata.data_plane` 存進 session，`apps/web/src/api/client.ts` 依這個欄位切換 `VITE_API_BASE_URL`／`VITE_DEMO_API_BASE_URL`。

## Alternatives

- **完全獨立的第二個 Supabase 專案**：資料、Auth、Storage 全部物理隔離，安全性最徹底，但多一份帳單與一組要另外維護的 Auth 使用者資料庫，對一個「只是給人試用」的展示環境不成比例。
- **同一個資料庫、用 tenant_id 欄位邏輯隔離**：不需要 capability-aware migration，但任何一支查詢忘記帶 tenant 過濾條件就會外洩或污染正式資料，風險遠高於現在這種「實體上就是兩個資料庫」的做法。
- **Demo 端仍用 MSW**：實作成本最低，但使用者能感覺出來是假資料、且完全不能驗證真實的 Auth／API／DB 串接，違背這次要做「真實流程 Demo」的目的。

## Consequences

- 之後新增任何會寫入 `auth.*` 的 migration，都要記得比照 000002／000011 做 capability-aware 包裝，否則對 `ltc_demo` 跑 migration 會直接炸掉。
- 新增業務資料表時，`internal/modules/demo/infra` 的 truncate 清單與 `apps/api/seed/demo/` 的種子資料要一併更新，否則 Demo 重置不會清到新表，長期會累積跨次重置的髒資料。
- Demo 只能跑單一 Cloud Run instance；如果之後 Demo 流量大到需要多 instance，這裡的行程內鎖必須換掉，且要重新設計「重置期間拒絕寫入」的機制。
- 這份文件涵蓋的檔案一旦再被改動就會被標記 stale，需要重新核對上述假設是否還成立。
