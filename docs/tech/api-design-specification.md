---
doc_type: flow
covers:
  - apps/api/internal/modules/
  - apps/web/src/api/
  - apps/web/src/types/api.d.ts
---

# API 設計與契約規格手冊 (API Design & Contract Specification)

本文件定義本專案 API 端點設計、Request / Response 契約、HTTP 狀態碼語意與錯誤對應之全域技術規範。

---

## 1. 資源識別與 URL 規範
- **Canonical Resource ID**：資源唯一識別碼（Resource ID）一律以 URL Path 為唯一來源（例如：`/api/v1/cases/:id`）。
- **禁止重複承載**：Request Body 中嚴禁重複傳遞與 URL Path 相同的 ID。若因相容性需求必須同時存在，後端一律以 URL Path 為準，並在測試中覆蓋衝突情境。
- **命名慣例**：URL Path 一律使用小寫 kebab-case（例如 `/driver-reports`、`/ride-records`）；Query 參數與 JSON 欄位使用 camelCase（例如 `claimStartDate`、`plateNo`）。

---

## 2. DTO 與模型責任隔離
Transport DTO、Domain Model、Persistence Model 與 Frontend TypeScript Type 必須各自維持清晰的職責邊界：
- **Transport DTO**：僅定義於各模組的 `transport/*_dto.go`，負責 JSON 序列化、反序列化與 `binding` 標籤校驗。
- **Domain Model**：僅封裝核心業務概念與不變量，不包含 JSON tag 或資料庫標籤。
- **Persistence Model**：僅映射 PostgreSQL 資料表與欄位。
- **嚴禁跨層污染**：禁止將資料庫 Entity 直接作為 API Response 回傳給前端，亦禁止將前端 Request DTO 直接傳遞至 Repository 層。

---

## 3. 統一 Response Envelope 格式

所有非檔案下載類型的 API，回應資料一律封裝於統一 Envelope：

### 成功回應 (Success Envelope)
```json
{
  "data": { ... } 或 [ ... ],
  "meta": {
    "total": 100,
    "page": 1,
    "pageSize": 20
  }
}
```
- 前端 Axios 攔截器會自動解包 `response.data`，業務呼叫端取得的即為 `{ data, meta }`。
- 清單為空時，`data` 應為空陣列 `[]`，不可回傳 `null`。

### 錯誤回應 (Error Envelope)
```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "輸入資料不符合規則，請確認後再試",
    "details": [
      {
        "field": "nationalId",
        "message": "身分證字號格式不正確"
      }
    ]
  }
}
```
- `code`：全系統唯一且結構化的大寫錯誤代碼。
- `message`：非技術性、使用者易懂的中文說明（不含程式堆疊或 SQL 錯誤）。
- `details`：選填，當存在具體欄位錯誤時以陣列逐項列出。

---

## 4. HTTP 狀態碼與查無資料語意規範

「查無資料」不得一律拋出 404，必須依 API 資源語意嚴格判定（依據 `docs/decisions/not-found-vs-empty-result.md`）：

| 查詢情境 | API 類型範例 | 預期 HTTP 狀態碼 | 回傳 Payload 格式 | 說明 |
| :--- | :--- | :---: | :--- | :--- |
| **特定單一資源** | `GET /cases/:id`<br>`PUT /vehicles/:id` | **404 Not Found** | 錯誤 Envelope<br>`code: "NOT_FOUND"` | 該資源明確指向特定主鍵，查無此人屬於異常情況。 |
| **集合／清單搜尋** | `GET /cases?q=王`<br>`GET /drivers?status=active` | **200 OK** | 成功 Envelope<br>`data: []` | 搜尋或過濾結果為空屬於正常查詢業務分支，非錯誤。 |
| **可選子資源** | `GET /cases/:id/schedule` | **200 OK** | 成功 Envelope<br>`data: null` | 個案尚未設定排班屬於合法狀態，不應當作 404 報錯。 |

---

## 5. 前後端整合驗證標準
每次新增或修改端點時，必須同時完成以下四道驗證：
1. **Wire Contract 測試**：驗證真實 HTTP Method、路徑、Query、Body、Status 與 Envelope。
2. **邊界異常測試**：驗證缺少必填欄位、無效 ID、查無資料與格式錯誤等邊界路徑。
3. **前端 TypeScript 型別同步**：同步更新 `apps/web/src/types/api.d.ts`（或 `src/api/*.ts`）。
4. **錯誤代碼登記**：若新增錯誤代碼，需在後端 `httpx` 與前端 `errorCodes.ts` 同步登記並測試。
