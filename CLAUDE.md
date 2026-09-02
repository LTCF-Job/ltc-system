# Claude project instructions

本文件是 Claude 的相容入口，專案規則的主要來源是根目錄的 [`AGENTS.md`](AGENTS.md)。先讀取 `AGENTS.md`，再依任務讀取相關 skill；本文件不重複定義架構規則。

## Skills

本專案的 canonical skills 位於 `.agents/skills/`。開始任何架構、程式碼、測試、mock 或重構工作前：

1. 列出 `.agents/skills/*/SKILL.md`，依檔案 frontmatter 的 `description` 判斷適用範圍。
2. 讀取所有與當前任務相關的完整 `SKILL.md`。
3. 依 skill 的指示執行工作，並將未驗證事項明確標示。

適用範圍如下：

- Go backend：`.agents/skills/backend-architecture/SKILL.md`
- Go backend code style：`.agents/skills/go-backend-code-style/SKILL.md`
- Go unit testing：`.agents/skills/golang-unit-testing/SKILL.md`
- Vue 3 frontend：`.agents/skills/frontend-architecture/SKILL.md`
- Admin UI design：`.agents/skills/admin-ui-design/SKILL.md`
- LTC dashboard visual language：`.agents/skills/ltc-dashboard-visual-language/SKILL.md`
- Frontend accessibility：`.agents/skills/accessibility/SKILL.md`
- mock／demo／seed／MSW：`.agents/skills/mock-and-demo-boundaries/SKILL.md`
- architecture audit／refactoring review：`.agents/skills/architecture-review/SKILL.md`
- Excel 匯入匯出、範本下載（僅支援 .xlsx，不支援 CSV）：`.agents/skills/excel-import-export-integrity/SKILL.md`

若 Claude 執行環境支援 Agent Skills 的自動發現，使用 `.agents/skills/` 作為專案 skills 來源；若該環境未自動發現，仍依上述步驟直接讀取檔案。請保留 `.agents/skills/` 為唯一 canonical 位置，不另建內容相同的 `.claude/skills/` 副本。

## Project scope

這是由 `apps/api` 與 `apps/web` 組成的前後端分離 monorepo。後端是 Go、Gin、PostgreSQL；前端是 Vue 3、TypeScript、Vite、Vue Router、Pinia、Axios、Element Plus 與 MSW。架構改善以 incremental modular-monolith migration 為方向。

## Working rules

- 先讀 `AGENTS.md`、相關 skills 與目前工作樹狀態。
- 使用目前分支的原始碼作為現況證據，保留既有未提交修改。
- 測試執行原則：
  - 不主動執行 E2E 測試：嚴禁主動執行 E2E 測試（如 Playwright、`npm run test:e2e*`），只有在使用者明確指示或要求時才執行。平時前端驗證以 `npm run type-check`、`npm run build` 或單元測試為主。
  - 只有在修改應用程式原始碼（application source code logic）時才執行對應的單元測試（如後端 Go tests）；文件、指令、測試檔案、註解、設定檔等與程式碼無關的任務一律不跑測試。
- mock、demo、seed 與 production behavior 必須標示清楚的啟用條件。
- 不自行 commit、push、rebase、merge 或重設使用者修改。
- 回報驗證結果時區分 static inspection、automated test、build/type-check 與 runtime proof。
