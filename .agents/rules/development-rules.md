# 開發前置與變更守則 (Development Rules)

本規則為全域常駐守則，Agent 與開發者在執行任何修改前必須遵守。

## 1. 現況查核與最小變更
- **保留既有修改**：動手前先檢查 `git status`，絕對不覆寫、重設或刪除使用者未要求處理的檔案與修改。
- **最小必要變更**：只做需求直接需要的變更，不進行無關的重構、格式化清理或多餘的程式碼搬遷。
- **以原始碼為準**：以當前分支工作樹的實際程式碼與設定為現況依據；歷史規劃文件僅供參考，不取代原始碼證據。

## 2. 分層邊界防護
- **後端模組化單體邊界**：
  - `transport`：只處理 Gin HTTP 綁定、DTO 解析、HTTP 狀態碼與錯誤轉譯，嚴禁撰寫 SQL 或業務邏輯。
  - `app`：擁有業務邏輯、use case、domain rules 與 port 定義，嚴禁直接依賴 Gin 上下文或資料庫驅動細節。
  - `infra`：擁有 PostgreSQL repository、外部 adapter，負責實體儲存與查詢。
- **邊界檢查保證**：後端邊界由 `apps/api/internal/arch/arch_test.go` 強制驗證，其 baseline 為空，違反邊界即為 bug。

## 3. 前端通用規格守則
- **時間顯示格式**：全站時間一律只顯示到秒數（`YYYY-MM-DD HH:mm:ss`，純時間 `HH:mm:ss`），統一透過 `@/utils/formatters` 格式化，嚴禁直接輸出 raw ISO 8601、毫秒或時區字尾。
- **非技術性提示**：前端 UI 嚴禁輸出 raw SQL、panic、500 或堆疊追蹤；一律轉譯為使用者視角之白話提示，並附帶統一錯誤代碼（如 `[ERROR_CODE]`）。

## 4. 驗證證據紀律
- 嚴禁宣稱未經確認的「已完成」或「已驗證」。
- 回報檢查結果時，必須明確區分證據層級：
  - `Static Inspection`：靜態檢查、型別宣告、設定檢視。
  - `Automated Test`：單元測試、整合測試執行輸出。
  - `Build / Type-check`：`npm run type-check` 或 `go build` 編譯通過。
  - `Runtime Proof`：實際啟動資料庫或服務後的端到端操作證據。未執行的項目誠實標示為「未驗證」。
