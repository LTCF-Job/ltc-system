---
name: api-contract-guidelines
description: Use when changing an API endpoint, route, DTO, request or response shape, API client, TypeScript API type, query parameter, response envelope, or HTTP error mapping.
---

# API contract guidelines

## 契約定義

- 每個公開入口先列出 method、URL、path parameter、query、request body、response envelope、錯誤狀態與 mock 對應。
- resource ID 以 URL path 為 canonical source；body 不重複承載同一 ID。若相容性需求必須同時存在，明確定義優先順序並測試衝突值。
- DTO、domain model、persistence model、API client type 與 mock fixture 各自維持責任邊界；不要用任一層型別直接代替其他層。

## 實作一致性

- response envelope 在單一 client boundary 解開；分頁資料、meta、錯誤 envelope 與所有呼叫端使用同一契約。
- endpoint 變更同步檢查 handler、service、repository interface、client、TypeScript type、mock、文件與測試。
- transport layer 集中進行 domain error mapping；`not found`、`validation`、`conflict` 與 infrastructure failure 保留可區分的 HTTP 結果。
- 無效 resource ID、關聯 ID 或請求格式進入明確錯誤路徑；成功結果只代表實際完成，不以空陣列、假分頁或假成功訊息替代失敗。

## 驗證

- 測試實際 wire contract：HTTP method、path、query、body、status、response envelope 與錯誤 payload。
- 為 path／body 識別碼不同、缺少必要欄位、查無資料、關聯不存在與下游失敗建立邊界案例。
- 前端呼叫端完成後再檢查頁面狀態、錯誤提示與型別推導；只驗證 handler 或只驗證 client 都不算契約完成。
