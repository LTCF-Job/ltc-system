---
doc_type: api
covers:
  - apps/api/internal/modules/demo/transport/reset_handler.go
  - apps/api/internal/modules/demo/app/reset_service.go
  - apps/api/internal/modules/demo/infra/reset_repo.go
  - apps/api/cmd/server/routes.go
---

# Demo 資料集重置

## Endpoint

`POST /api/v1/demo/reset`

只在服務以 `DATA_PLANE=demo` 啟動時註冊；`DATA_PLANE=production` 的服務完全沒有這條路由（呼叫會得到一般的 404，而不是權限錯誤）。

## Auth

需要有效 JWT，且該 JWT 的 `app_metadata.data_plane` 必須等於 `demo`（與其他所有端點一樣，由 `auth.Middleware` 統一驗證）。角色不限——`viewer`／`staff`／`admin` 皆可觸發，任何登入的 Demo 使用者都能重置整個共享資料集。

## Request

無 request body。

## Response

```json
{
  "data": {
    "datasetVersion": "0001_baseline",
    "resetAt": "2026-09-02T08:09:53Z"
  }
}
```

- `datasetVersion`：目前套用的種子檔版本（對應 `apps/api/seed/demo/0001_baseline.up.sql`）。
- `resetAt`：重置完成時間（UTC，RFC3339）。

## Errors

- `401 UNAUTHENTICATED`：JWT 缺失、無效，或 `data_plane` 與本服務不符。
- `500 DEMO_RESET_FAILED`：重置交易失敗（清空或套用種子資料任一步出錯），交易整筆回滾，資料庫維持重置前的狀態。

## Invariants and gotchas

- 重置在單一 PostgreSQL 交易內完成：先 `TRUNCATE` 所有業務資料表（`regions` 除外，見 `migrations/000002_seed_reference_data.up.sql`），再套用種子 SQL，最後重新加密種子資料的假身分證欄位（見 `docs/decisions/demo-data-plane-architecture.md`）；任何一步失敗都整筆回滾，不會留下半套資料。
- 呼叫這支端點的請求，以及重置期間所有其他 `/api/v1/*` 請求，共用同一個行程內 `sync.RWMutex`（`internal/modules/demo/app.ConcurrencyGuard`）：一般請求持共享鎖，重置需要獨佔鎖，因此重置一定會等既有請求做完，重置進行中新進來的請求會排隊等待，不會讀到重置到一半的中間狀態。這個機制假設 Demo 只有單一 Cloud Run instance；多 instance 時不會互相阻擋。
- 沒有速率限制，也沒有「只有特定使用者能重置」的限制——任何 Demo 使用者都能觸發，且會影響所有正在使用 Demo 的人；前端已在呼叫前以 `ElMessageBox.confirm` 要求二次確認，見 [`DefaultLayout.vue`](../../apps/web/src/layouts/DefaultLayout.vue)。
- 冪等：重複呼叫只會把資料集重設回同一份基準狀態，不會累積或報錯。
