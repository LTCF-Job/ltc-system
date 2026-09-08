# Claude project instructions

本文件是 Claude 的相容入口，專案規則的主要來源是根目錄的 [`AGENTS.md`](AGENTS.md)。先讀取 `AGENTS.md`，再依任務讀取相關 skill；本文件不重複定義架構規則。

## Skills and documents

本專案的 canonical skills 位於 `.agents/skills/`，技術與業務規範文件位於 `docs/tech/`。開始任何架構、程式碼、測試、mock 或重構工作前：

1. 先讀取 `AGENTS.md`，並依任務變更範圍查閱相關的 Skill 與 `docs/tech/` 規範文件。
2. 涉及全域系統核心邏輯（主檔 CRUD、啟用停用、匯出檢核非阻擋、匯入檢核、待維護隔離、錯誤代碼與 E2E 測試），**必須閱讀 `docs/tech/system-logic-specification.md`**。
3. 讀取所有與當前任務相關的完整 `SKILL.md` 與對應文件後再行動手，嚴禁盲目變更程式碼。

適用範圍如下：

- 所有開發、修正、重構或測試工作：`.agents/skills/development-guidelines/SKILL.md`、`docs/tech/system-logic-specification.md`
- API endpoint、route、DTO、request／response、API client 或 error mapping：`docs/tech/api-design-specification.md`、`docs/tech/integration-contract.md`、`docs/tech/backend-api-reference.md`
- 資料寫入、transaction、刪除、稽核、stale protection 或併發處理：`docs/tech/mutation-audit-specification.md`、`docs/tech/system-logic-specification.md`、`docs/decisions/mutation-audit-policy.md`
- JWT、登入、actor、角色、權限矩陣、權限 cache 或使用者管理：`.agents/skills/auth-permission-guidelines/SKILL.md`、`docs/tech/frontend-permission-logic.md`、`docs/decisions/role-permission-api-authorization.md`
- migration、schema、index、constraint、seed 或資料庫版本：`.agents/skills/migration-guidelines/SKILL.md`、`.agents/skills/supabase-postgres-best-practices/SKILL.md`、`docs/tech/maintainer-runbook.md`
- 業務日期、民國日期、時區、排班、搭乘、假日、狀態、合併或待維護隔離：`docs/tech/backend-business-rules.md`、`docs/decisions/pending-data-visibility.md`、`docs/tech/system-logic-specification.md`
- Go backend 架構與程式碼風格：`.agents/skills/backend-architecture/SKILL.md`、`.agents/skills/go-backend-code-style/SKILL.md`、`docs/tech/backend-flows.md`、`docs/tech/backend-framework.md`
- Go unit testing：`.agents/skills/golang-unit-testing/SKILL.md`、`docs/tech/backend-business-rules.md`
- Vue 3 frontend：`.agents/skills/frontend-architecture/SKILL.md`、`docs/tech/frontend-flows.md`、`docs/tech/frontend-pages.md`、`docs/tech/frontend-framework.md`
- Admin UI design：`.agents/skills/admin-ui-design/SKILL.md`、`docs/tech/system-logic-specification.md`
- LTC dashboard visual language：`.agents/skills/ltc-dashboard-visual-language/SKILL.md`
- Frontend accessibility：`.agents/skills/accessibility/SKILL.md`
- demo／seed／fixture：`.agents/skills/mock-and-demo-boundaries/SKILL.md`
- architecture audit／refactoring review：`.agents/skills/architecture-review/SKILL.md`
- Excel 匯入匯出、範本下載（僅支援 .xlsx，不支援 CSV）：`.agents/skills/excel-import-export-integrity/SKILL.md`、`docs/tech/system-logic-specification.md`
- 前端 E2E 測試：`docs/tech/system-logic-specification.md`（準則七）、`docs/flows/e2e-local-backend-migration.md`

若 Claude 執行環境支援 Agent Skills 的自動發現，使用 `.agents/skills/` 作為專案 skills 來源；若該環境未自動發現，仍依上述步驟直接讀取檔案。請保留 `.agents/skills/` 為唯一 canonical 位置，不另建內容相同的 `.claude/skills/` 副本。

## Project scope

這是由 `apps/api` 與 `apps/web` 組成的前後端分離 monorepo。後端是 Go、Gin、PostgreSQL；前端是 Vue 3、TypeScript、Vite、Vue Router、Pinia、Axios 與 Element Plus。架構改善以 incremental modular-monolith migration 為方向。

## Working rules

- 先讀 `AGENTS.md`、相關 skills 與目前工作樹狀態。
- 使用目前分支的原始碼作為現況證據，保留既有未提交修改。
- 測試執行原則：
  - 不主動執行 E2E 測試：嚴禁主動執行 E2E 測試（如 Playwright、`npm run test:e2e*`），只有在使用者明確指示或要求時才執行。平時前端驗證以 `npm run type-check`、`npm run build` 或單元測試為主。
  - 只有在修改應用程式原始碼（application source code logic）時才執行對應的單元測試（如後端 Go tests）；文件、指令、測試檔案、註解、設定檔等與程式碼無關的任務一律不跑測試。
- demo、seed 與 production behavior 必須標示清楚的啟用條件。
- 不自行 commit、push、rebase、merge 或重設使用者修改。
- 回報驗證結果時區分 static inspection、automated test、build/type-check 與 runtime proof。
