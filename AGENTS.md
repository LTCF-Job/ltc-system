# ltc-system agent instructions

## Source of truth

本文件是專案層級的 agent 入口。具體架構規則放在 `.agents/skills/`，修改規則時以該目錄的 skill 為唯一來源；本文件只負責說明何時讀取它們。

每次開始工作時，先確認目前分支、工作樹狀態與實際專案結構。保留使用者既有修改，將目前程式碼與文件視為現況證據；規劃文件只代表目標，不取代原始碼證據。

## Skill routing

先依任務讀取對應的完整 `SKILL.md`：

- 所有開發、修正、重構或測試工作：先讀 `.agents/skills/development-guidelines/SKILL.md`，再依實際範圍讀專項規範。
- API endpoint、route、DTO、request／response、API client、query parameter 或 error mapping：讀 `.agents/skills/api-contract-guidelines/SKILL.md`。
- 資料寫入、transaction、刪除、稽核、stale protection 或併發處理：讀 `.agents/skills/mutation-guidelines/SKILL.md`。
- JWT、登入、actor、角色、權限矩陣、權限 cache 或使用者管理：讀 `.agents/skills/auth-permission-guidelines/SKILL.md`。
- migration、schema、index、constraint、seed 或資料庫版本：讀 `.agents/skills/migration-guidelines/SKILL.md`。
- CI/CD、Docker、Vercel、Cloud Run、環境變數、secret 或部署檢查：讀 `.agents/skills/deployment-guidelines/SKILL.md`。
- 業務日期、民國日期、時區、排班、搭乘、假日、狀態、合併或 UUID array filter：讀 `.agents/skills/domain-data-guidelines/SKILL.md`。
- 後端架構、Go 分層、use case、repository、SQL 或 adapter：讀 `.agents/skills/backend-architecture/SKILL.md`。
- Go backend 程式碼風格、錯誤處理、pgx、transaction、API response 或 dependency wiring：讀 `.agents/skills/go-backend-code-style/SKILL.md`。
- Go unit test、table-driven test、domain rule、parser 或 service test：讀 `.agents/skills/golang-unit-testing/SKILL.md`。
- 前端架構、Vue 3、頁面拆分、composable、Pinia、API client 或 TypeScript contract：讀 `.agents/skills/frontend-architecture/SKILL.md`。
- 後台 UI 資訊架構、CRUD 工作台、表格、篩選、批次操作、審核流程或稽核頁面：讀 `.agents/skills/admin-ui-design/SKILL.md`。
- LTC dashboard 視覺語言、後台 Dashboard、KPI 卡片、Sidebar、alerts、charts 或資料面板 UI：讀 `.agents/skills/ltc-dashboard-visual-language/SKILL.md`。
- 前端 accessibility、鍵盤操作、focus、ARIA、表單錯誤、dialog／drawer、表格或圖表語意：讀 `.agents/skills/accessibility/SKILL.md`。
- demo、seed、fixture 或 offline mode：讀 `.agents/skills/mock-and-demo-boundaries/SKILL.md`。
- 架構盤點、跨層依賴、模組過大、契約漂移或重構建議：讀 `.agents/skills/architecture-review/SKILL.md`。
- Excel 匯入、範本下載或匯出功能（本專案僅支援 .xlsx，不支援 CSV）：讀 `.agents/skills/excel-import-export-integrity/SKILL.md`。

若同一任務跨越多個範圍，依「審查 → 後端／前端 → mock 邊界」順序讀取；只讀與當前任務有關的 skill。新增或修改 agent 文件時，遵守 `.agents/skills` 既有 skill 的 progressive disclosure、single source of truth 與最小必要內容原則。

## Architecture direction

本專案採前後端分離的 modular monolith：

- Go API 已完成模組化：每個業務能力是 `internal/modules/<capability>/{transport,app,infra}`，模組之間只透過 `cmd/server` 注入的 port 協作。邊界由 `internal/arch/arch_test.go` 強制，其 baseline 為空。
- Vue 3 SPA 逐步朝 `app / features / shared` 的 feature-oriented 結構遷移。
- 前端時間顯示規格：時間一律只顯示到秒數（`YYYY-MM-DD HH:mm:ss`，純時間 `HH:mm:ss`），統一透過 `@/utils/formatters` 格式化，嚴禁直接輸出 raw ISO 8601、毫秒或時區字尾。
- API DTO、domain model、persistence model 與 mock fixture 保持不同責任。
- 重構採逐功能切片進行，先建立新邊界，再移動被觸碰的功能；保留既有路由、回應 envelope、權限規則與業務行為。

## Change boundaries

- 先檢查 `git status`，不覆寫、刪除或重設未被要求處理的修改。
- **測試執行原則**：
  - **不主動執行 E2E 測試**：嚴禁主動執行 E2E 測試（如 Playwright、`npm run test:e2e*`），只有在使用者明確指示或要求時才執行。平時前端以 `npm run type-check`、`npm run build` 或單元測試進行驗證。
  - **執行測試時機**：只有在修改應用程式原始碼（application source code logic）時，才執行對應的單元測試或相關測試（如後端 `go test`）；嚴禁無差別執行全量測試。
  - **略過測試時機**：文件、指令、測試檔案（test code）、註解、設定檔、script、除錯、規劃等與原始碼業務邏輯無關的任務，一律不跑測試。
- 任何 UI/UX 修改前，一律先讀取 `.agents/skills/ltc-dashboard-visual-language/SKILL.md`，並依其視覺、互動與資訊真實性規則執行。
- 文件與 agent 規則修改應保持最小範圍；不因新增入口而複製完整 skill 內容。
- 不自行 commit、push、rebase、merge 或執行破壞性 Git 操作。
- 任何「已驗證」的說法都要對應實際執行的檢查、測試或明確的靜態證據；未執行的 runtime 行為要標示為未驗證。

## Repository commands

- Backend：在 `apps/api` 執行 `go test ./...`，需要編譯檢查時執行 `go build ./...`。
- Frontend Type Check & Build：在 `apps/web` 執行 `npm run type-check` 與 `npm run build`。
- Frontend E2E Tests（Playwright，使用真實 API）：`npm run test:e2e:live`

## Language

面向使用者的說明、文件與註解使用臺灣繁體中文；程式碼、API 名稱、路徑與技術術語保留原文。
