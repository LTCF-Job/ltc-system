---
doc_type: flow
covers:
  - apps/api/internal/modules/demo/
  - apps/api/seed/demo/
  - apps/api/ops/
  - apps/api/migrations/000002_seed_reference_data.up.sql
  - apps/api/migrations/000011_backfill_admin_identity.up.sql
  - .github/workflows/deploy-api.yml
  - apps/web/tests/e2e-live/
---

# Demo data-plane 從程式碼到可用環境

整體架構決策見 [demo-data-plane-architecture.md](../decisions/demo-data-plane-architecture.md)；本文件記錄「目前實作到哪裡、驗證到哪裡」，供下一輪開發接續，不重複那份文件已經寫過的設計理由。

## Trigger

- 自動：push 到 `develop`／`main` 且改到 `apps/api/**` 觸發 `.github/workflows/deploy-api.yml`。
- 手動：依 [environment-bootstrap.md](../tech/environment-bootstrap.md)「如果還要建一個 Demo 環境（選用）」與「Demo／Live E2E 專用設定」兩節，第一次建置 Demo 環境時的手動步驟。

## Steps

```
push apps/api/**
  > test（go vet + go test -race）
  > build-image（gcloud builds submit，image tag = commit SHA）
  > deploy-demo
      > 更新並執行 DEMO_MIGRATION_JOB（migrations 對 auth.* 的操作在無 auth schema 時自動跳過）
      > gcloud run deploy DEMO_API_SERVICE，輸出服務網址
  > e2e-demo（apps/web/tests/e2e-live/demo-data-plane.spec.ts，不啟動 MSW）
      > 真實 Supabase signInWithPassword 登入 Demo 測試帳號
      > 呼叫已部署 Demo API 驗證 JWT 被接受
      > 呼叫 POST /demo/reset，驗證 datasetVersion／resetAt 與重置後可讀到基準資料
      > [選用] 正式／Demo JWT 互相打對方 API，驗證 401 拒絕矩陣
  > deploy-prod
      > compare-demo-prod-schema.sh：比對 schema_migrations 版本與 public schema，不一致就中止
      > 更新並執行 MIGRATION_JOB
      > gcloud run deploy API_SERVICE
```

`POST /api/v1/demo/reset` 本身的請求流程（交易、鎖、重新加密）見 [demo-reset.md](../api/demo-reset.md)。

## Failure modes

- `e2e-demo` 缺 `LIVE_SUPABASE_URL`／`LIVE_SUPABASE_ANON_KEY`／`LIVE_DEMO_API_BASE_URL`／`LIVE_DEMO_TEST_EMAIL`／`LIVE_DEMO_TEST_PASSWORD` 任一項：對應測試全部 `test.skip`，**pipeline 視為通過**、照樣往下部署正式環境。這是刻意設計成「不因本機/新專案缺憑證就擋住部署」，但代價是「忘記在 GitHub Environment 設定這些變數」跟「Demo 真的沒問題」在 CI 結果上長得一樣，需要人工核對 Actions log 裡是否真的跑了測試而不是跳過。
- `compare-demo-prod-schema.sh` 缺 `SCHEMA_CHECK_PROD_DATABASE_URL`／`SCHEMA_CHECK_DEMO_DATABASE_URL`：直接以非零狀態碼失敗，會擋住部署（跟上面 Live E2E 的「缺憑證就跳過」相反，這裡是「缺憑證就擋」）。
- Demo 重置失敗：交易整筆回滾，資料庫維持重置前狀態；呼叫端收到 `500 DEMO_RESET_FAILED`。
- Demo／正式資料庫其中一邊手動改過 schema（沒有透過 migration）：`compare-demo-prod-schema.sh` 會在下一次部署時擋下，錯誤訊息是完整 `diff`。

## Unverified

- **CI/CD pipeline 從未在真實 GCP 專案上執行過**：`deploy-api.yml` 的新增 job（`build-image`／`deploy-demo`／`e2e-demo`／`deploy-prod`）只驗證過 YAML 語法正確，`gcloud` 指令、`GITHUB_OUTPUT` 傳遞 Demo 服務網址給下一個 job 的寫法都是照現有正式部署 job 的既有慣例類推，尚未實際跑過一次確認 job 之間的資料傳遞（尤其是 `needs.deploy-demo.outputs.demo_api_url`）真的可用。第一次在真實專案跑這條 pipeline 時要盯著 Actions log 逐步確認。
- **各功能模組的 live CRUD 覆蓋度遠低於規劃**：原規劃要求「各功能模組至少一組 create → read/list → update → delete」，目前 `demo-data-plane.spec.ts` 只做了登入、reset、以及 `GET /regions` 一個唯讀端點；個案／排班／車輛／司機／接送／匯入／通知／出勤／維修／報表／稽核等模組完全沒有 live E2E。下一輪要做的話，可以照同一份檔案的 `test.skip` 模式（缺憑證就跳過、不擋 CI）逐模組擴充。
- ~~重置期間併發寫入的行為只在 Go 單元測試層級驗證~~ **已補上真實 Postgres 的整合測試**：`internal/modules/demo/infra/reset_repo_integration_test.go`（`//go:build integration`）用真實 `ResetRepository` + 真實資料庫工作負載驗證 `ConcurrencyGuard` 的互斥順序，並驗證重置後的種子資料筆數、身分證加密可被正確解密、重跑一次的冪等性；執行方式見檔案內註解（`go test -tags=integration ./internal/modules/demo/... -v`，需要 `DATABASE_URL` 指向已跑過 migration 的真實 Postgres）。撰寫過程中這支測試抓到一個真的 bug：`loadSeedFile` 原本只嘗試兩個相對路徑，`go test` 的工作目錄是套件目錄，兩個都對不到，已修正為多加一個相對於原始檔位置的候選路徑。仍未驗證的是「已部署的真實 Cloud Run API」在併發下的行為——這支測試驗證的是同一個 Go process 內的鎖語意，多 instance（架構決策裡已知的限制）或網路層級的併發仍未涵蓋。
- **重新整理／登出再登入後資料仍存在**：架構上成立（資料在 `ltc_demo` 資料庫，登出只清前端 session），但沒有寫成任何自動化測試。
- **Storage bucket／測試寄件網域隔離未實作，且目前判斷是不適用**：規劃裡「Demo API 強制使用 Demo bucket、測試寄件網域及固定測試收件人」這一段，實際查證後發現這個專案目前根本沒有真正的外部整合可以隔離——`internal/modules/notification/app/notification_service.go` 的 `EmailSender` 只有 `LogEmailSender`（純記 log，不寄信）在用，`main.go` 组装通知服務時傳的是 `nil`；匯出檔案（`export_handler.go` 的 `Download`）是即時渲染直接串流下載，不落地到任何雲端儲存，`config.go` 裡的 `STORAGE_BUCKET`／`RESEND_API_KEY`／`StorageSignedURLTTL` 三個設定值目前沒有任何程式碼讀取。下一輪如果要做這件事，前提是先實作真正的 email／storage 整合，屆時才需要一併做 data-plane 隔離；在那之前這條待辦沒有實體可以動工。
- ~~前端「重置 Demo 資料」按鈕只驗證過 MSW 模擬~~ **已對真實後端驗證過一次**：手動起一組真實 Postgres + `DATA_PLANE=demo` 的真實 Go server（無 `SUPABASE_JWKS_URL`，本機模式下用可解析但未簽章的 JWT 帶 `app_metadata.data_plane=demo`）+ 指向該 server 的 Vite dev server（`VITE_ENABLE_MSW=false`），用 Playwright 直接點擊「重置 Demo 資料」→ 確認送出，確認 `POST /demo/reset` 回應 200，且資料庫的 `cases` 筆數從重置前的 0 變成種子資料的 8 筆。這驗證了按鈕到真實 API 到真實資料庫的完整路徑；**仍未驗證的是真實 Supabase 簽發的 JWT**（這次用的是本機未驗證解析模式，不是 JWKS 驗證路徑），以及對著真正部署在 Cloud Run 上的服務按過這顆按鈕。
- ~~`apps/web/package-lock.json` 有一筆與本次功能無關的 npm 版本 metadata 噪音，需要使用者手動 `git checkout` 還原~~ **已隨後續 commit 一併收斂**：目前 working tree 乾淨，無殘留噪音。
