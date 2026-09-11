---
doc_type: api
covers: ["apps/api/internal/modules/casemgmt/transport/case_handler.go", "apps/api/internal/modules/casemgmt/transport/case_dto.go", "apps/api/internal/modules/casemgmt/infra/case_repo.go"]
---

## Endpoint

`GET /api/v1/cases`

同一份 `CaseResponse` 也用於 `GET /api/v1/cases/:id`。

## Auth

需要 `masters_cases:view` 權限（見 `apps/api/cmd/server/routes.go`）。

## Request

全部為 query 參數，皆為選填：

| 參數 | 型別 | 說明 |
|---|---|---|
| `page` | int | 頁碼，由 `httpx.ParsePagination` 解析 |
| `pageSize` | int | 每頁筆數，同上 |
| `status` | string | 個案狀態完全比對；空字串不過濾 |
| `q` | string | 對 `name` 與 `home_address` 做 `ILIKE` 模糊比對 |
| `unresolvedLink` | `"true"` | 只回傳待維護個案（定義見下方 gotchas） |
| `excludePending` | `"true"` | 反向排除待維護個案，供主清單與「待維護」分頁互斥呈現 |

`unresolvedLink` 與 `excludePending` 只認字串 `"true"`，其他值一律視為 false。

## Response

`data` 為 `CaseResponse[]`，另附 `meta` 分頁資訊（`page`／`pageSize`／`total`／`totalPages`）。

```json
{
  "id": "uuid",
  "name": "string",
  "nameNormalized": "string",
  "nationalId": "string",
  "nationalIdInvalid": false,
  "householdType": "string | null",
  "gender": "string | null",
  "birthDate": "RFC3339 | null",
  "birthDateRaw": "string | null",
  "careContactRole": "string | null",
  "careContactName": "string | null",
  "registeredAddress": "string | null",
  "siteId": "uuid | null",
  "siteName": "string",
  "siteNameRaw": "string | null",
  "homeAddress": "string | null",
  "ltcLevel": "string | null",
  "serviceCategory": 0,
  "serviceUsageType": 0,
  "claimEndDate": "RFC3339 | null",
  "status": "string",
  "remarks": "string | null",
  "createdAt": "RFC3339",
  "updatedAt": "RFC3339"
}
```

身分證密文與 HMAC 索引不對外輸出；`nationalId` 直接回傳解密後的明碼，不再需要另呼叫揭露 API。

## Errors

| 狀態 | code | 觸發條件 |
|---|---|---|
| 400 | `CodeValidationFailed` | 分頁參數格式錯誤 |
| 500 | `CodeInternalError` | 查詢失敗 |

查無資料回 `200` 搭配空陣列，不回 404（見 `docs/decisions/not-found-vs-empty-result.md`）。

## Invariants and gotchas

- 「待維護」的判定條件是以下任一成立：`site_id` 為 null 但 `site_name_raw` 有值、`caregiver_id` 為 null 但 `care_contact_name` 有值、`birth_date_raw` 有值、`national_id_invalid` 為 true。前端「待維護」分頁的問題標籤與補齊對話框都是直接依這幾個欄位推導，沒有另一個彙總欄位。
- `siteNameRaw` 是匯入當下抓到、但比對不到主檔的原始名稱。**這個欄位既是待維護的判定依據，也是前端唯一能顯示「差什麼」的資料來源**——曾經 SQL 的 `WHERE` 有用它過濾、`SELECT` 卻沒選出來，導致待維護清單抓得到人、問題欄與補齊對話框卻整片空白。改動 `case_repo.go` 的查詢時，`List`／`ListAll`／`GetByID` 三支共用同一組欄位順序，要一起改。
- 個案不再關聯去/回程車輛：`case_transport_preferences` 表與 `PUT /api/v1/cases/:id/transport-preference` 端點已於 migration `000055` 一併移除。
- 主檔（據點、照護人員、司機、個案）新增或改名時，若能唯一比對到本頁列出的待維護資料，系統會自動關聯並清除 `is_pending`，見 `docs/decisions/pending-data-visibility.md` §4。

## Unverified

none
