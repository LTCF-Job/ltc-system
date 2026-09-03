---
doc_type: flow
covers:
  - apps/web/tests/e2e/
  - apps/web/playwright.config.ts
  - apps/web/playwright.config.local.ts
  - apps/api/seed/
  - docker-compose.local.yml
---

# 本機 E2E 從 MSW 假資料改打真實 API — 待開發

MSW 已在本輪移除（`apps/web/src/mocks/` 整目錄、`demoMode.ts` 相關死碼皆已刪除），local／demo／production 三環境現在功能路徑一致，只差資料庫。但 `apps/web/tests/e2e/`（`01~12-*.spec.ts`，約 1,276 行）依賴 MSW 攔截與假資料斷言，**目前全部會壞**。本文件記錄調查到的現況與設計方向，供下一輪接續；不是實作紀錄，是待辦。

## 為什麼還沒做完

嘗試過一次自動化改寫，執行到「複製 demo 種子資料改給本機測試用」這一步時被中止，尚未完成、未經驗證，因此**沒有隨這輪一起 commit**，避免留下看似完整、實際跑不通的程式碼。以下是已確認的現況與應該怎麼做的方向。

## 已確認的現況

- **CI 目前不受影響**：`.github/workflows/ci.yml` 只跑 `go test` / `type-check` / `build`；唯一在 CI 上跑的 Playwright 是 `.github/workflows/deploy-api.yml` 的 `e2e-demo` job，測的是 `apps/web/tests/e2e-live/`，這套本來就打真 API、不碰 MSW，**不受本次影響，不要動它**。
- **`tests/e2e/` 從未在 CI 跑過**，只是本機手動的回歸網，所以拆 MSW不會讓 CI 變紅，但本機失去了唯一的 UI 回歸覆蓋。
- **三層依賴，MSW 拿掉後全部要處理**：
  1. `playwright.config.ts` / `playwright.config.local.ts` 把 `VITE_SUPABASE_URL` 指向 `http://mock.supabase.local`，靠 MSW 的 `handlers/supabaseAuth.ts` 攔截 `/auth/v1/token` 回傳假 session——這個攔截機制已經不存在。
  2. `tests/e2e/helpers/auth.ts` 的 `loginAs()` 原本點快速登入按鈕（已在 `LoginView.vue` 移除）或寫入 `demo_token_<role>` 到 localStorage（格式跟後端現在只認的 `mock_jwt_` 前綴對不上）。
  3. 12 支 spec 的斷言大量寫死 MSW 假資料的具體內容（例如個案姓名「1.張詹竹妹 [去程]」、故意用「王小明」當查無資料案例、斷言匯出前置檢核**一定**跳警告框——因為假資料故意留了瑕疵）。這些斷言的前提資料來源已經不存在。
- **local 登入流程已改好**：`LoginView.vue` 的表單在所有環境長得一樣，使用者照常輸入帳號密碼、按登入；local 環境（Supabase 未設定時）不驗證密碼，直接發一張 `mock_jwt_<role>` token 建立 session（角色推斷：email 含 `viewer` → viewer，否則 admin）。後端 `auth.go` 現在只接受這一條 local 分支，且會正確跑 `enforceDataPlane`。E2E 的登入 helper 應該改用「填表單、按登入」這個真實流程，而不是繼續走已被拿掉的捷徑。
- **權限矩陣已統一**（見 [role-permission-api-authorization.md](../decisions/role-permission-api-authorization.md)）：後端所有路由都走 `RequirePermission(module, action)`，前端依 `/auth/me` 回傳的權限決定畫面。這代表新版 E2E 的斷言基礎應該是「權限矩陣的行為」而不是「角色字串」——例如驗證某個權限受限的使用者看不到編輯按鈕、或操作被 API 擋下（403），而不是假設「這個角色永遠能做這件事」。
- **既有可能複用、但不能直接套用的機制**：`apps/api/seed/demo/0001_baseline.up.sql`（約 8 筆 cases 的種子資料）＋ `POST /demo/reset` 端點可重置 demo 資料庫。但 demo 資料平面要求 `DATA_PLANE=demo` 且 JWT 帶 `app_metadata.data_plane=demo`，跟 local 分支（`mock_jwt_` token，`DATA_PLANE` 預設 `production`）是不同的資料平面設定。**這兩套機制能不能直接借給 local E2E 用、還是要另立一套本機測試種子，需要下一輪重新調查判斷，不要假設能直接套用**——上一輪的中止點就停在「打算照抄 demo 種子檔案改給本機用」，這個假設本身還沒驗證過是否可行。

## 下一輪要做的事

1. **調查 local 測試環境的資料庫啟動與重置機制**：`docker-compose.local.yml` 目前的 `APP_ENV`／`DATABASE_URL`／`SUPABASE_JWKS_URL` 設定、`apps/api/cmd/migrate/main.go` 怎麼跑 migration、`apps/api/internal/modules/demo/`（`reset_service.go`／`reset_repo.go`）的重置邏輯能否抽出「truncate + 重跑種子」這段給 local 測試共用，或者需要獨立一支不掛 HTTP 端點、直接在 Playwright `globalSetup` 執行的本機專用種子腳本。
2. **確認 `mock_jwt_` 分支的 `actorID`**：`auth.go` 目前這條分支用固定 UUID 建立 actor，設計本機種子資料時要考慮讓種子使用者的 ID 對得上這個固定值，`/auth/me` 才能查到正確的角色與權限。
3. **playwright config 要能同時起後端與資料庫**：目前 `playwright.config.ts` 的 `webServer` 只起 vite dev server，沒有起 Go API server 或確保 Postgres 已就緒並跑過 migration，這部分需要補上。
4. **重新設計測試資料策略，不要重蹈覆轍**：MSW 版本的問題是斷言寫死具體假資料內容（`mockData.ts`）。新版原則：能由 spec 自己在測試中建立的資料（透過 UI 操作或直接呼叫 API），就自己建立、自己斷言、測試後清理；只有「列表本來就該有基準資料」（例如空狀態測試）才需要種子資料。
5. **測試隔離**：每次 `npm run test:e2e` 都要能從已知的資料庫狀態開始，不因上一輪跑過而累積髒資料。方案（跑 migration 到全新測試專用 DB、每個 spec file 用 `beforeAll`/`beforeEach` 重置、或整個 test run 開始前 truncate+reseed 一次）需要下一輪評估取捨後決定。
6. **至少一支 spec 要驗證「不同權限的使用者看到不同畫面」**——呼應這輪權限矩陣統一的核心目的，不能只測 admin 一種身分。可以用 email 推斷規則登入 viewer 角色，驗證看不到編輯/刪除按鈕、或操作被 API 擋下。
7. **12 支既有 spec 逐支處理**：哪些整支重寫、哪些只是替換資料來源，需要先讀過全部 12 支列出每一支依賴 MSW 假資料的具體斷言點，再決定改法（詳見 `apps/web/tests/e2e/01~12-*.spec.ts` 與 `tests/e2e/helpers/{auth,ui}.ts`）。

## 驗收標準

- `npm run test:e2e`（本機、非 CI）能在乾淨環境從頭跑到尾，不依賴任何已刪除的 MSW 機制。
- 斷言不再有寫死的假資料具體內容（姓名、筆數等），改為測試自建資料或明確定義的種子基準。
- 至少一支 spec 涵蓋權限矩陣行為（不同角色看到不同畫面／操作被擋）。
- `tests/e2e-live/` 維持不動，CI 的 `e2e-demo` job 不受影響。
- 若受限於環境無法起本機 DB/Postgres 驗證到底，至少要誠實記錄卡在哪裡，不要宣稱測試通過但實際沒跑過。
