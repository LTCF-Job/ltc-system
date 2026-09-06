---
doc_type: decision
covers:
  - apps/api/internal/modules/casemgmt
  - apps/api/internal/modules/caseimport
  - apps/api/internal/modules/driverreport
  - apps/api/internal/modules/identity
  - apps/api/internal/modules/masterdata
  - apps/api/internal/modules/notification
  - apps/api/internal/modules/ops
  - apps/api/internal/modules/reporting
  - apps/api/internal/modules/ride
---

# 全系統 Mutation Audit Policy

## Context

Mutation 的稽核行為曾由各 service 自行決定，造成有些異動直接序列化 domain
entity、有些忽略稽核錯誤，且外部 side effect 與稽核成功被誤當成同一件事。

## Decision

下列 action 必須留下稽核紀錄：

```text
create, update, delete, status_change, permission_change,
reveal_pii, import, export_requested, export_succeeded, export_failed,
conflict_resolve, manual_correction
```

每筆紀錄固定包含 `actorId`、`actorRole`、`action`、`entityType`、
`entityId`、`before`、`after`、`ip`、`userAgent` 與由 storage 產生的時間戳。
`before`／`after` 一律使用明確的 audit snapshot DTO；不得直接把 domain entity、
request 或含有身分證密文、HMAC、明文身分證、地址與聯絡資料的物件序列化。

目前系統尚未有共用 outbox，因此採下列過渡策略：

```text
本地資料庫 mutation
  ├─ 同交易內必須維持的關鍵狀態（例如衝突處理、刪除）
  │    └─ audit failure → rollback / 回傳錯誤
  └─ 已完成的外部 side effect 或非關鍵事後稽核
       └─ mutation success → audit failure 只記 server log，排入既有重試／監控責任
```

Supabase user mutation 先以外部結果判定成功，再把 audit failure 視為可補償的
觀測事件；密碼、PII reveal 與 permission change 仍須採用各自 service 的高風險
阻擋規則。匯入與匯出會分別記錄實際的 import／export 狀態，不把 request accepted
誤記成成功。

## Consequences

- 新增 mutation 時，需同時提供 actor context、明確 snapshot 與對應 action。
- audit writer 失敗不可使用 `_ = audit.Write(...)` 靜默吞掉；至少要記錄結構化
  server log，且不得把已成功的非交易外部 side effect 回報為失敗。
- 若未來需要跨 instance 的可靠補送，應以共用 outbox／queue 取代各 service 的
  best-effort 寫入，不在 application layer 重新實作另一套佇列。
