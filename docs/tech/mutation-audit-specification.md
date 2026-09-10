---
doc_type: flow
covers:
  - apps/api/internal/modules/
  - apps/api/internal/platform/db/
---

# 資料寫入、併發防護與審計規範手冊 (Data Mutation & Audit Specification)

本文件定義系統中資料新增（Create）、更新（Update）、刪除（Delete）、批次匯入、併發處理與操作審計（Audit Logging）之全域技術規範。

---

## 1. 寫入邊界與交易原子性 (Atomic Transactions)

### 1.1 單一交易上下文原則
- 當一次使用者操作引發多項資料變更（例如：新增個案同時建立聯絡人、記錄審計日誌、扣減額度），所有寫入操作必須在**同一個資料庫交易上下文（Transaction Context）**內完成。
- 所有相關 Repository 必須從該 Transaction Context 共享同一條資料庫連線，**嚴禁下游模組或方法自行私自開啟獨立連線或私開交易**。
- **現況缺口**：`internal/platform/pgxdb/txrunner.go` 的 `WithTx`（L37）沒有巢狀偵測，遇到 context 已帶交易仍會無條件再開一條新交易；`FromContext`（L62）的 fallback 設計會在 context 沒有交易時靜默改用 pool（一般查詢適用，但寫入路徑若誤用會變成無交易自動提交）；`NewTxRunner(nil)`（L29）回傳 `nil` 而非 error。這些是待收斂的差距，實作時不得假設現有機制已提供巢狀合併保護。

### 1.2 副作用與冪等性 (Idempotency)
- 外部 Webhook 或批次上傳等具備重試特性的端點，必須支援冪等性（Idempotency）。
- 實作方式：使用唯一業務鍵（如 Submission ID + 行號、或請求中的 Idempotency-Key 標頭）進行去重檢查。

---

## 2. 併發防護與過期寫入防範 (Stale Protection)

> **現況**：以下為目標規範，全系統尚未實作。`grep -rn "AND updated_at" apps/api/internal/modules/ --include=*.go` 無命中；全後端唯一的 409 是 `identity/transport/errors.go:22` 的 `RESOURCE_IN_USE`，與併發衝突無關。實作範圍見 [應用程式目標架構](application-architecture.md) §11 階段零。

為防止多位使用者同時編輯同一筆資料導致覆寫（Lost Updates）：
1. **樂觀鎖機制（Optimistic Concurrency Control）**：
   - 凡可被更新之核心主檔與業務資料表（如 `cases`、`vehicles`、`drivers`、`ride_records`），均應具備 `updated_at` 時間戳。
   - 前端發起更新請求時，必須攜帶讀取時的 `updated_at`（或版本雜湊）。
   - 後端執行 SQL 更新時，應加入檢查條件：
     ```sql
     UPDATE cases SET ... WHERE id = $1 AND updated_at = $2;
     ```
   - 若受影響列數為 0，表示資料已被其他人先一步修改，後端須終止交易並回傳 `409 Conflict`，前端提示使用者：「資料已被更新，請重新整理後再試」。

---

## 3. 操作審計留痕規範 (Audit Logging Policy)

依據 `docs/decisions/mutation-audit-policy.md`，系統必須完整記錄關鍵操作之稽核軌跡：

### 3.1 審計等級劃分

判準句（與 [`mutation-audit-policy.md`](../decisions/mutation-audit-policy.md) 一致，該文件為權威來源）：**若稽核寫入失敗而業務資料保留，是否會讓該操作事後無法被查核或追責？** 是 → 阻斷性；否 → 非阻斷性。

1. **阻斷性審計（Blocking Audit）**：
   - action：`delete`、`permission_change`、`conflict_resolve`、`manual_correction`。涉及核心身分識別、權限角色變更、刪除、人工裁決與衝突處理等高風險操作。`reveal_pii` 為歷史動作（身分證揭露 API 已移除，不再產生新紀錄），僅供舊稽核紀錄回溯查詢。
   - 審計日誌寫入與業務寫入處於同一交易內，若審計日誌寫入失敗，**整體操作必須回滾（Rollback）並報錯**。
2. **非阻斷性觀測（Non-blocking Audit）**：
   - action：一般主檔的 `create`／`update`／`status_change`、匯入匯出的狀態留痕、已完成外部 side effect 的事後觀測、一般日常讀取與排班檢視。**據點／車輛／司機的 create 與 update 屬此級**——現行 `masterdata/app/audit.go` 的 `writeAuditBestEffort`（失敗只 `slog.Error`，不回傳 error）是照此分級實作，不是缺陷。
   - 審計失敗時允許降級記錄系統 Warning Log，不阻礙主體業務流程。

### 3.2 審計日誌欄位標準
審計日誌一律必須包含：
- `actor_id`：操作者帳號 UUID。
- `actor_role`：操作時之有效角色。
- `action`：操作動作名稱（如 `case.create`, `driver_report.import`）。
- `target_type` 與 `target_id`：被操作之實體類型與主鍵。
- `ip_address` 與 `user_agent`：請求來源網路資訊。
- `created_at`：操作時間（UTC）。

---

## 4. 刪除與狀態終結語意
1. **主檔資料**：一律優先使用軟刪除（`deleted_at IS NOT NULL`）或狀態停用（`status = 'inactive'`），禁止物理硬刪除已被歷史趟次或統計關聯的實體。
2. **暫存與待維護資料**：待維護池中的暫存列（如司機匯報衝突列、疑似重複個案暫存列），若使用者點選「忽略此筆」，則執行物理硬刪除，不留假死資料。
