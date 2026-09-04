# 技術文件

給要維護 `apps/api` 或 `apps/web` 的開發者看的現況說明，依現有程式碼撰寫，不是規劃或估時文件。第一次接手請先讀 [maintainer-guide.md](../maintainer-guide.md)，操作與排查請讀 [maintainer-runbook.md](maintainer-runbook.md)。`docs/` 根目錄與 `docs/flows/` 中標示 historical 的文件保留作為背景資料，與程式碼衝突時，以程式碼與本區現況文件為準。

| 文件 | 內容 |
|---|---|
| [../maintainer-guide.md](../maintainer-guide.md) | 系統拓樸、模組責任、前後端資料流、資料責任與接手檢查表 |
| [maintainer-runbook.md](maintainer-runbook.md) | 本機 Docker／原生啟動、migration、health check、故障排查與未驗證邊界 |
| [../reviews/2026-09-04-full-stack-review.md](../reviews/2026-09-04-full-stack-review.md) | 2026-09-04 靜態 full-stack review、bug findings、文件落差與後續 backlog |
| [backend-framework.md](backend-framework.md) | 後端技術棧、分層架構、domain 套件、Auth 機制、response 格式、環境變數 |
| [backend-api-reference.md](backend-api-reference.md) | 現行 API 路由表（method／path／permission／說明）；不存在的歷史 route 另行標示 |
| [backend-flows.md](backend-flows.md) | 後端核心業務流程逐步拆解：表單 ingestion、更正與衝突處理、未回報偵測、政府申報匯出、主檔匯入、通知、稽核留痕 |
| [backend-business-rules.md](backend-business-rules.md) | 核心演算法與驗證規則的實際判斷邏輯：混車合併、應搭日曆計算、四趟展開、姓名比對評分、身分證驗證與加密、匯出前置檢核項目、申報表排序規則、申報表尚無資料來源的欄位 |
| [frontend-framework.md](frontend-framework.md) | 前端技術棧、目錄結構、資料流與 axios 攔截器、路由權限、Mock 邊界、狀態管理原則 |
| [frontend-pages.md](frontend-pages.md) | 完整頁面對照表：路徑、元件、module permission、對應打的後端 API |
| [frontend-flows.md](frontend-flows.md) | 前端核心功能流程：月曆補登、異常處理、未回報清單、表單欄位對應、匯出、批次匯入 |
| [frontend-permission-logic.md](frontend-permission-logic.md) | 前端權限判斷的實際邏輯：路由守衛判斷順序、模組權限表、個人自訂權限覆蓋規則與已知落差 |
| [integration-contract.md](integration-contract.md) | 前後端整合契約：一次 API 呼叫從畫面觸發到顯示錯誤，完整經過的認證、角色檢查、response envelope 解包與錯誤處理關卡 |
| [deployment.md](deployment.md) | 部署設定：GitHub Actions、Cloud Run、Vercel、Supabase 環境變數怎麼串，與部署時踩過的坑 |
| [environment-bootstrap.md](environment-bootstrap.md) | 從零把這個專案部署到全新的 Supabase／GCP／Vercel 環境，第一次建置專用 |

`docs/flows/e2e-local-backend-migration.md` 已明確標示為 **Historical**，只保留移除 MSW／demo path 的調查脈絡；現行本機啟動與驗證請以 [maintainer-runbook.md](maintainer-runbook.md) 為準。

分層邊界與程式碼風格的權威規則來源是 [`AGENTS.md`](../../AGENTS.md) 底下的 `.agents/skills/`。
