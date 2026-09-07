---
doc_type: api
covers: ["apps/api/internal/modules/casemgmt/transport/case_handler.go", "apps/api/internal/modules/casemgmt/app/case_service.go"]
---

## Endpoint

`GET /api/v1/cases/:id/schedule`

## Auth

需要 `masters_cases:view` 權限（見 `apps/api/cmd/server/routes.go`）。

## Request

- Path：`id`（UUID，個案 ID）
- 無 query／body 參數，一律查詢當日（`clock.Today()`）生效中的排班

## Response

成功時 `data` 為 `CaseScheduleResponse`：

```json
{
  "id": "uuid",
  "caseId": "uuid",
  "siteId": "uuid",
  "siteName": "string",
  "effectiveFrom": "RFC3339 date",
  "effectiveTo": "RFC3339 date | null",
  "weekdays": [1, 3, 5],
  "tripPattern": 0,
  "unitPrice": 0,
  "distanceKm": 0,
  "serviceDurationMin": 0,
  "serviceCode": "string",
  "note": "string | null",
  "legs": [ /* ScheduleLegResponse[] */ ],
  "createdAt": "RFC3339",
  "updatedAt": "RFC3339"
}
```

個案在查詢當日**尚無現行排班**時，`data` 為 `null`，HTTP 狀態碼仍為 `200`——這是合法的業務狀態，不是錯誤。呼叫端（`apps/web/src/api/cases.ts` 的 `getCaseSchedule`）依此回傳 `CaseScheduleDTO | null`，前端顯示「尚無現行排班」等提示，不觸發全域錯誤攔截器。

## Errors

- `400 VALIDATION_FAILED`：`id` 不是合法 UUID
- `500 INTERNAL_ERROR`：查詢排班時發生非預期錯誤

## Invariants and gotchas

- 「查無現行排班」刻意不使用 `404`，避免與「個案本身不存在」（`GET /cases/:id` 的 404）混淆；兩者語意不同，不可互相替代。
- 排班的建立／更新走 `PUT /api/v1/cases/:id/schedule`（`SaveSchedule`），此文件僅涵蓋查詢端點。

## Unverified

none
