# 技術與規範文件總覽 (Technical & Specification Index)

本目錄包含系統所有核心業務邏輯規範、前後端架構、API 對照、操作流程與運維文件。

---

## 1. 核心業務與工程規範 (Specifications & Standards)

所有模組開發與維護必須嚴格遵守以下規範：

| 規範文件 | 內容與核心守則 |
|---|---|
| [system-logic-specification.md](system-logic-specification.md) | **全系統 7 大核心業務與工程規範**：主檔 CRUD 與狀態管理、匯出前置檢核與非阻擋原則、匯入比照檢核、待維護全站隔離與生命週期、使用者友善錯誤代碼、前後端同步規格、前端 E2E 測試規範。 |
| [backend-business-rules.md](backend-business-rules.md) | **後端核心業務演算與驗證規則**：混車合併演算法、司機車輛歸屬、應搭日曆計算、四趟展開、姓名比對評分、身分證驗證與加密、申報表排序規則、申報表留白欄位。 |
| [api-design-specification.md](api-design-specification.md) | **API 設計與契約規範**：Canonical Resource ID、DTO 隔離、統一 Response Envelope、404 vs 200 狀態碼語意、前後端整合驗證標準。 |
| [mutation-audit-specification.md](mutation-audit-specification.md) | **資料寫入、併發防護與審計規範**：單一交易上下文原則、樂觀鎖防過期覆寫（`updated_at`）、阻斷與非阻斷審計政策、軟刪除與物理刪除語意。 |
| [integration-contract.md](integration-contract.md) | **前後端整合通訊契約**：一次 API 呼叫從畫面觸發到錯誤顯示的全流程、JWT 驗證、模組權限檢查、Axios 攔截器、Response 解包。 |

---

## 2. 架構與系統總覽 (Architecture & Frameworks)

| 文件 | 內容說明 |
|---|---|
| [backend-framework.md](backend-framework.md) | 後端 Go 技術棧、Gin 分層架構、Domain 套件、Auth 機制、環境變數。 |
| [frontend-framework.md](frontend-framework.md) | 前端 Vue 3 技術棧、目錄結構、資料流與 Axios 攔截器、狀態管理原則。 |
| [frontend-permission-logic.md](frontend-permission-logic.md) | 前端權限判斷邏輯：路由守衛順序、模組權限表、個人權限覆蓋規則。 |
| [supabase-security.md](supabase-security.md) | Supabase 安全設定與連線架構。 |

---

## 3. 端點與頁面對照表 (References & Mappings)

| 文件 | 內容說明 |
|---|---|
| [backend-api-reference.md](backend-api-reference.md) | 完整後端 API 路由表（Method / Path / 角色權限 / 功能說明）。 |
| [frontend-pages.md](frontend-pages.md) | 前端完整頁面對照表（路由路徑 / 元件 / 允許角色 / 對應後端 API）。 |
| [pending-integrations.md](pending-integrations.md) | 待確認與待環境驗證之外部整合清單。 |

---

## 4. 業務流程圖解 (End-to-End Flows)

| 文件 | 內容說明 |
|---|---|
| [backend-flows.md](backend-flows.md) | 後端業務流程逐步拆解：司機匯報 Ingestion、更正與衝突處理、未回報偵測、政府申報匯出、主檔匯入、稽核留痕。 |
| [frontend-flows.md](frontend-flows.md) | 前端核心功能流程：搭乘月曆補登、異常集中處理、未回報清單、表單欄位對應、匯出與批次匯入。 |

---

## 5. 運維與部署手冊 (Operations & Runbooks)

| 文件 | 內容說明 |
|---|---|
| [qa-crud-testing.md](qa-crud-testing.md) | 前端 CRUD 全功能測試流程：五道驗證層級、每個模組的固定測試矩陣、匯入匯出驗證順序、測試資料標記與清除。實作在 `tests/qa-crud/`。 |
| [maintainer-runbook.md](maintainer-runbook.md) | 維護 Runbook：本機啟動、Migration 執行、Health Check 與故障排查。 |
| [deployment.md](deployment.md) | 部署設定：GitHub Actions、Cloud Run、Vercel、Supabase 環境變數串接與踩坑紀錄。 |
| [environment-bootstrap.md](environment-bootstrap.md) | 從零把專案部署到全新 Supabase / GCP / Vercel 環境之建置手冊。 |

---

> 常駐 Agent 守則請參閱根目錄 [AGENTS.md](../../AGENTS.md) 與 `.agents/rules/`。
