# ltc-system agent instructions

## Source of truth

本文件是專案層級的 agent 入口。具體架構規則放在 `.agents/skills/`，修改規則時以該目錄的 skill 為唯一來源；本文件只負責說明何時讀取它們。

每次開始工作時，先確認目前分支、工作樹狀態與實際專案結構。保留使用者既有修改，將目前程式碼與文件視為現況證據；規劃文件只代表目標，不取代原始碼證據。

## Skill routing

先依任務讀取對應的完整 `SKILL.md`：

- 後端架構、Go 分層、use case、repository、SQL 或 adapter：讀 `.agents/skills/backend-architecture/SKILL.md`。
- Go backend 程式碼風格、錯誤處理、pgx、transaction、API response 或 dependency wiring：讀 `.agents/skills/go-backend-code-style/SKILL.md`。
- Go unit test、table-driven test、domain rule、parser 或 service test：讀 `.agents/skills/golang-unit-testing/SKILL.md`。
- 前端架構、Vue 3、頁面拆分、composable、Pinia、API client 或 TypeScript contract：讀 `.agents/skills/frontend-architecture/SKILL.md`。
- 後台 UI 資訊架構、CRUD 工作台、表格、篩選、批次操作、審核流程或稽核頁面：讀 `.agents/skills/admin-ui-design/SKILL.md`。
- LTC dashboard 視覺語言、後台 Dashboard、KPI 卡片、Sidebar、alerts、charts 或資料面板 UI：讀 `.agents/skills/ltc-dashboard-visual-language/SKILL.md`。
- 前端 accessibility、鍵盤操作、focus、ARIA、表單錯誤、dialog／drawer、表格或圖表語意：讀 `.agents/skills/accessibility/SKILL.md`。
- mock、demo、seed、fixture、offline mode 或 MSW：讀 `.agents/skills/mock-and-demo-boundaries/SKILL.md`。
- 架構盤點、跨層依賴、模組過大、契約漂移或重構建議：讀 `.agents/skills/architecture-review/SKILL.md`。
- Excel 匯入、範本下載或匯出功能（本專案僅支援 .xlsx，不支援 CSV）：讀 `.agents/skills/excel-import-export-integrity/SKILL.md`。

若同一任務跨越多個範圍，依「審查 → 後端／前端 → mock 邊界」順序讀取；只讀與當前任務有關的 skill。新增或修改 agent 文件時，遵守 `.agents/skills` 既有 skill 的 progressive disclosure、single source of truth 與最小必要內容原則。

## Architecture direction

本專案採前後端分離的 modular monolith：

- Go API 已完成模組化：每個業務能力是 `internal/modules/<capability>/{transport,app,infra}`，模組之間只透過 `cmd/server` 注入的 port 協作。邊界由 `internal/arch/arch_test.go` 強制，其 baseline 為空。
- Vue 3 SPA 逐步朝 `app / features / shared / mocks` 的 feature-oriented 結構遷移。
- 前端時間顯示規格：時間一律只顯示到秒數（`YYYY-MM-DD HH:mm:ss`，純時間 `HH:mm:ss`），統一透過 `@/utils/formatters` 格式化，嚴禁直接輸出 raw ISO 8601、毫秒或時區字尾。
- API DTO、domain model、persistence model 與 mock fixture 保持不同責任。
- 重構採逐功能切片進行，先建立新邊界，再移動被觸碰的功能；保留既有路由、回應 envelope、權限規則與業務行為。

## Change boundaries

- 先檢查 `git status`，不覆寫、刪除或重設未被要求處理的修改。
- application source code logic 變更前，讀取相關架構 skill，並依目前專案的 workflow／註解規則執行。
- 前端修改完成後，一律執行對應的 E2E 測試或智慧改動偵測指令（`npm run test:e2e:changed`），確保修改未破壞既有功能。
- 任何 UI/UX 修改前，一律先讀取 `.agents/skills/ltc-dashboard-visual-language/SKILL.md`，並依其視覺、互動與資訊真實性規則執行。
- 文件與 agent 規則修改應保持最小範圍；不因新增入口而複製完整 skill 內容。
- 不自行 commit、push、rebase、merge 或執行破壞性 Git 操作。
- 任何「已驗證」的說法都要對應實際執行的檢查、測試或明確的靜態證據；未執行的 runtime 行為要標示為未驗證。

## Repository commands

- Backend：在 `apps/api` 執行 `go test ./...`，需要編譯檢查時執行 `go build ./...`。
- Frontend Type Check & Build：在 `apps/web` 執行 `npm run type-check` 與 `npm run build`。
- Frontend E2E Tests（Playwright + MSW）：
  - 跑全量 E2E 測試：`npm run test:e2e`（或根目錄 `make test-web-e2e`）。
  - 跑改動對應 E2E 測試：`npm run test:e2e:changed`（或根目錄 `make test-web-e2e-changed`）。
  - 單獨跑特定功能模組 E2E 測試：
    - 認證與權限：`npm run test:e2e:auth`
    - 總覽儀表板：`npm run test:e2e:dashboard`
    - 個案與排班：`npm run test:e2e:cases`
    - 基礎主檔：`npm run test:e2e:masters`
    - 表單與對應：`npm run test:e2e:forms`
    - 搭乘月曆表：`npm run test:e2e:rides`
    - 異常集中處理：`npm run test:e2e:issues`
    - 營運報表：`npm run test:e2e:reports`
    - 車輛與出勤：`npm run test:e2e:operations`
    - 系統設定與稽核：`npm run test:e2e:settings`
    - 政府申報匯出：`npm run test:e2e:exports`

## Language

面向使用者的說明、文件與註解使用臺灣繁體中文；程式碼、API 名稱、路徑與技術術語保留原文。
