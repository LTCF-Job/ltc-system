# 程式碼風格與防護常駐守則 (Code Style Rules)

本規則定義本專案跨前後端核心編碼規範與邊界防護要求，Agent 與開發者必須遵守。

## 1. 後端 Go 編碼風格與防護
- **命名與結構**：遵循標準 Go 命名慣例、error wrapping（`fmt.Errorf("...: %w", err)`），保持函式單一責任與易讀性。
- **DTO 與模型隔離**：
  - Transport DTO、Domain Model 與 Persistence Model 各自維持獨立，嚴禁以單一層型別通套全站。
  - DTO 只存在於邊界，嚴禁將 HTTP 請求直接傳入 Repository 或將資料庫 Entity 直接丟給前端。
- **錯誤處理**：在 transport 層集中進行 domain error mapping；系統內部錯誤（DB 失敗、第三方服務異常）必須保留可識別的 domain error 或 HTTP 錯誤代碼，不回傳模糊的通用錯誤。

## 2. 前端 Vue 3 與 TypeScript 編碼風格
- **型別安全**：所有 API 呼叫均應有完整的 TypeScript DTO 定義，嚴禁濫用 `any`。
- **響應式狀態**：遵循 Pinia 狀態管理原則，元件內部以 `ref` / `computed` 為主，避免在跨元件中傳遞 mutable 物件。
- **UI 元件庫規格**：統一使用 Element Plus 與系統自訂 Design Tokens，遵循無障礙鍵盤操作與易讀性要求。

## 3. 資料異動與併發防護
- **交易原子性**：一次使用者可觀察的寫入（主資料、關聯資料與稽核事件）必須在同一 Transaction Context 內完成，連線必須統一從該 context 取得。
- **防過期覆寫（Stale Protection）**：可被更正或覆寫的資料必須檢查版本號或 `updated_at` 時間戳；若資料已過期必須拒絕寫入並回傳衝突提示，嚴禁直接覆蓋較新的變更。
