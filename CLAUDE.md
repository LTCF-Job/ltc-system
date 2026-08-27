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

若 Claude 執行環境支援 Agent Skills 的自動發現，使用 `.agents/skills/` 作為專案 skills 來源；若該環境未自動發現，仍依上述步驟直接讀取檔案。請保留 `.agents/skills/` 為唯一 canonical 位置，不另建內容相同的 `.claude/skills/` 副本。

## Project scope

這是由 `apps/api` 與 `apps/web` 組成的前後端分離 monorepo。後端是 Go、Gin、PostgreSQL；前端是 Vue 3、TypeScript、Vite、Vue Router、Pinia、Axios、Element Plus 與 MSW。架構改善以 incremental modular-monolith migration 為方向。

## Working rules

- 先讀 `AGENTS.md`、相關 skills 與目前工作樹狀態。
- 使用目前分支的原始碼作為現況證據，保留既有未提交修改。
- 修改 application source code 前，先完成相關架構 skill 的邊界判斷；以最小功能切片實作，避免一次進行全專案搬遷。
- 前端修改完成後，執行 `npm run test:e2e:changed` 或對應的 `npm run test:e2e:<feature>`（共 11 組功能模組），確保未破壞既有功能。
- mock、demo、seed 與 production behavior 必須標示清楚的啟用條件。
- 不自行 commit、push、rebase、merge 或重設使用者修改。
- 回報驗證結果時區分 static inspection、automated test、build/type-check 與 runtime proof。
