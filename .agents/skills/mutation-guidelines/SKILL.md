---
name: mutation-guidelines
description: Use when changing create, update, delete, import, transaction, audit logging, stale protection, concurrency handling, or any operation with multiple persistent side effects.
---

# Mutation guidelines

## 寫入邊界

- 先定義一次使用者可觀察的 mutation 包含哪些主資料、關聯資料與稽核事件，以及全有或全無的原子性需求。
- 需要原子性的流程使用同一 transaction context；所有 repository 從該 context 取得同一 connection，不能由下游自行開啟獨立連線。
- 每一項副作用都定義完成條件、失敗結果、重試語意與 idempotency key；部分成功必須是明確的業務設計，而不是未定義狀態。

## 稽核與錯誤

- 明確決定 audit write failure 是阻斷條件或非阻斷觀測；阻斷條件回滾並回報錯誤，非阻斷降級須留下可追蹤的 log／metric 證據。
- repository 未設定、查無資料、資料庫錯誤與權限拒絕使用可辨識的 domain／infrastructure error；空清單只表示查詢成功但沒有資料。
- delete、soft-delete 與關聯檢查要在同一個一致性邊界內完成，並保留使用者可理解的 conflict／validation 結果。

## Stale 與併發

- 可被更正或覆寫的資料使用明確的 fingerprint、version 或 `updated_at` 條件；提交時重新比對，過期資料回傳 conflict，不覆蓋較新的變更。
- correction request 帶入其依據版本；缺少依據、依據不符與重複提交各自定義結果。

## 驗證

- 覆蓋成功、任一中途步驟失敗、rollback、audit 失敗、查無資料、重試與 stale conflict。
- 對批次匯入驗證有效列、無效列、重複列、原始值／原因保留與稽核結果，不只驗證總筆數。
