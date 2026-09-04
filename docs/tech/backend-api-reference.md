---
doc_type: api
covers: ["apps/api/cmd/server/routes.go"]
---

# 後端 API 路由總覽

Base path：`/api/v1`，全部要帶 JWT（`auth.Middleware`）；公開 health endpoint 是 `GET /api/health`。目前沒有 `/api/v1/ingest/google-form` 或 `X-Ingest-Token` route。實作對應各能力模組的 `internal/modules/<capability>/transport/*.go`。路由表以 `apps/api/cmd/server/routes.go` 為唯一事實來源，改路由記得同步更新這份文件。

下表「角色」欄只作為**內建 role 的預設權限基準**，不是授權機制本身：現行 `/api/v1` 業務 route 一律以 `auth.RequirePermission(module, action)` 查角色的模組權限矩陣（`roles.permissions` 與 user custom permission）。`/auth/me` 與 `/auth/change-password` 只要求已通過 JWT authentication；不存在 `/demo/reset`，也沒有現行 `auth.RequireRoles` route。自訂角色的實際存取範圍以 permission matrix 為準，不受下表文字侷限。機制細節與撤權／cache 邊界見 [role-permission-api-authorization.md](../decisions/role-permission-api-authorization.md)。

架構背景見 [backend-framework.md](backend-framework.md)，每支端點背後的業務流程見 [backend-flows.md](backend-flows.md)。

## 區域主檔 `regionH`

| Method | Path | 角色 |
|---|---|---|
| GET | `/regions` | viewer, staff, admin |
| GET | `/regions/:id` | viewer, staff, admin |
| POST | `/regions` | staff, admin |
| PATCH | `/regions/:id` | staff, admin |
| DELETE | `/regions/:id` | admin |

## 個案主檔與排班 `caseH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/cases` | viewer, staff, admin | 個案清單（回傳遮罩身分證） |
| POST | `/cases` | staff, admin | 新增個案 |
| GET | `/cases/template` | viewer, staff, admin | 下載批次匯入用 Excel 範本 |
| GET | `/cases/:id` | viewer, staff, admin | |
| PATCH | `/cases/:id` | staff, admin | |
| DELETE | `/cases/:id` | admin | 軟刪除（`deleted_at`/`deleted_by`），同交易內收斂生效中排班 |
| POST | `/cases/:id/reveal` | staff, admin | 明文顯示身分證字號（會寫 audit log 的 `reveal_pii`） |
| GET | `/cases/:id/schedule` | viewer, staff, admin | 取得排班（星期、時段、四趟制設定） |
| PUT | `/cases/:id/schedule` | staff, admin | 覆寫排班 |
| POST | `/cases/schedules` | staff, admin | 批次建立排班 |
| POST | `/cases/import` | staff, admin | 批次匯入個案 Excel |
| POST | `/masters/import` | staff, admin | 同上，走另一條相容路徑（歷史因素，實際都打 `caseH.ImportExcel`） |
| GET | `/cases/export?caseIds=` | viewer, staff, admin | 匯出個案彙整表；`caseIds` 為逗號分隔的個案 ID，省略則匯出全部個案 |
| PUT | `/cases/:id/transport-preference` | staff, admin | 更新個案交通偏好設定 |

## 單位主檔 `siteH`

| Method | Path | 角色 |
|---|---|---|
| GET | `/sites` | viewer, staff, admin |
| POST | `/sites` | staff, admin |
| PATCH | `/sites/:id` | staff, admin |
| DELETE | `/sites/:id` | admin |

## 車輛主檔 `vehicleH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/vehicles` | viewer, staff, admin | 支援 `siteId`、`region`、`q`、`status`（`active`／`inactive`）篩選；每筆帶 `drivers`（該車今日生效的司機，一台車可有多位），以及所屬單位帶出的 `siteName` 與唯讀 `region` |
| POST | `/vehicles` | staff, admin | `siteId`、`plateNo` 與 `displayName`（代稱）為必填，其餘車籍欄位皆為選填（支援 `null` 與空值）；`status` 非 `active`／`inactive` 一律預設 `active`；車號或代稱重複時回 409 並帶 `details` |
| PATCH | `/vehicles/:id` | staff, admin | 整筆覆寫，必填欄位同 POST；車號或代稱重複時回 409 並帶 `details` |
| DELETE | `/vehicles/:id` | admin | 軟刪除（僅標記 `deleted_at`，不影響 `status` 啟用/停用狀態）；仍有生效中司機指派或排班趟次綁定時回 409（`CodeResourceInUse`） |
| PUT | `/vehicles/:id/drivers` | staff, admin | 整批設定本車司機：`{ driverIds: string[], effectiveFrom?: date }`；`driverIds` 為空代表清空 |

## 司機主檔 `driverH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/drivers` | viewer, staff, admin | 支援 `region`、`q`、`status`（`active`／`inactive`）篩選；每筆帶 `licenseClass`（駕照類別代碼）與 `licenseExpiryDate`（駕照有效日期），未補登為 `null` |
| POST | `/drivers` | staff, admin | `licenseClass`／`licenseExpiryDate` 選填，新增一律為 `active` |
| PATCH | `/drivers/:id` | staff, admin | 欄位未提供代表不變更；`licenseExpiryDate` 明確給 `null` 才會清空；`status` 非 `active`／`inactive` 時保留原值不變更 |
| DELETE | `/drivers/:id` | admin | 軟刪除（僅標記 `deleted_at`，不影響 `status` 啟用/停用狀態），同交易內收斂生效中的司機指派區間 |
| POST | `/drivers/:id/reveal` | staff, admin | 明文顯示司機個資 |
| POST | `/drivers/:id/assignments` | staff, admin | 指派車輛給司機；一位司機同期只會有一台車，指派新車即取代原本的指派 |

## 司機接送匯報與欄位對應 `driverReportH`

一台車一份匯報表；資料來源是使用者上傳的 `.xlsx`，沒有任何 Google 串接。

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/driver-reports` | viewer, staff, admin | 各車匯報表清單與欄位對應進度 |
| POST | `/driver-reports` | staff, admin | 為一台車建立匯報表；該車已有匯報表時只更新名稱並回傳既有那一份 |
| DELETE | `/driver-reports/:id` | staff, admin | 刪除匯報表（欄位對應與匯報紀錄一併移除） |
| GET | `/driver-reports/:id/template` | staff, admin | 下載該車空白匯報範本（`.xlsx`，只有表頭） |
| POST | `/driver-reports/:id/import?dryRun=&yearMonth=` | staff, admin | 上傳匯報檔；`dryRun=true`（預設）回傳預覽，`dryRun=false` 正式寫入。`yearMonth`（`YYYY-MM`）選填，宣告後整月覆蓋並拒收該月以外的日期 |
| GET | `/driver-reports/imported-months` | viewer, staff, admin | 每份匯報表各月份已匯入的筆數與最後匯入時間 |
| GET | `/driver-reports/:id/months/:yearMonth` | viewer, staff, admin | 取得指定匯報表月份明細 |
| GET | `/driver-reports/columns` | viewer, staff, admin | 欄位清單與對應狀態（可帶 `formId`、`mappingStatus`） |
| GET | `/driver-reports/columns/name-matches?name=` | viewer, staff, admin | 找出目前待維護欄位中姓名與傳入姓名相符（含近似）的欄位 |
| PATCH | `/driver-reports/columns/:id/mapping` | staff, admin | 設定單一欄位對應到哪個個案的哪一趟；剛從待維護變成已對應時同一交易內立即回填搭乘紀錄 |
| POST | `/driver-reports/columns/batch-mapping` | staff, admin | 批次設定欄位對應 |
| GET | `/driver-reports/submissions/review` | viewer, staff, admin | 以匯報表列（一天一筆提交）為單位列出待維護資料，一列可能同時有個案欄位與駕駛人兩種問題 |
| POST | `/driver-reports/drivers/bind` | staff, admin | 把某個比對不到司機主檔的原始姓名綁定到指定司機，立即回填所有正規化姓名相符的既有回報 |

匯入檔的欄位順序固定為：民國日期、駕駛人、各個案趟次欄、備註。個案趟次欄只接受
「有坐」「沒坐」，其餘（含空白）視為未回報不建立紀錄。`dryRun=false` 時可另外以
form field `columnDecisions` 帶入預覽畫面就地確認的欄位對應（JSON 陣列，元素為
`{columnHeader, mappingStatus, caseId, legSeq}`）。

## 搭乘月曆、異常與更正 `rideH` / `taskH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/rides/calendar` | viewer, staff, admin | 月曆矩陣視圖（個案 × 日期 × 趟次的搭乘狀態） |
| GET | `/rides/issues` | viewer, staff, admin | 異常搭乘集中清單（衝突、待確認等） |
| GET | `/rides/missing` | viewer, staff, admin | 未回報清單（目前接到 `taskH.GetMissingReports`；query／pagination 與前端 contract 仍有落差，且需注意可能進入 notification-capable path） |
| GET | `/rides/:id` | viewer, staff, admin | 單筆搭乘紀錄詳情 |
| PATCH | `/rides/:id` | staff, admin | 人工更正搭乘紀錄（寫 audit log） |
| POST | `/rides/manual-report` | staff, admin | 人工補登整筆回報（月曆空白格填寫） |
| POST | `/rides/:id/resolve-conflict` | staff, admin | 裁決同車衝突回報：`{vehicleId, driverId, reason}`，寫入 `conflict_resolution_note` 並記稽核（`resolve_conflict`） |

`GET /rides/issues` 支援 `issueType=conflict\|unreported\|import_error`（三擇一）、`month`（`YYYY-MM`，省略則預設當月）、`keyword`、`page`、`pageSize`。三種類型的資料路徑完全不同：`conflict` 讀 `ride_records` 聚合 `ride_sources` 車輛陣列；`unreported` 重用 `task/app.TaskService.ListMissingReportsForMonth` 的整月查詢（不觸發催報通知）；`import_error` 讀 `form_submissions.anomaly_flags` 非空的列。

## 匯出前置檢核與工作管理 `exportH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/exports/precheck` | staff, admin | 執行匯出前置檢核 |
| POST | `/exports/precheck` | staff, admin | 同上 |
| GET | `/exports` | viewer, staff, admin | 匯出工作歷史清單（不含檔案明細與下載連結） |
| POST | `/exports` | staff, admin | 建立政府申報匯出工作並同步產檔；body 需帶 `periodYm`(民國 5 碼)、`mode`(`direct`\|`zip`)、`caseIds`(至少一筆) |
| GET | `/exports/:id` | viewer, staff, admin | 單筆匯出工作詳情，含逐案檔案清單 `files` |
| GET | `/exports/:id/files/:caseId/download` | viewer, staff, admin | 下載單一個案的申報 `.xlsx` |
| GET | `/exports/:id/download` | viewer, staff, admin | 下載整包 `.zip`；非壓縮檔模式回 400 |

## 假日主檔 `holidayH`

| Method | Path | 角色 |
|---|---|---|
| GET | `/holidays` | viewer, staff, admin |
| POST | `/holidays` | staff, admin |
| POST | `/holidays/import` | staff, admin |
| DELETE | `/holidays/:date` | admin |

## 通知設定 `notificationH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/settings/notification-recipients` | viewer, staff, admin | 通知收件人清單 |
| POST | `/settings/notification-recipients` | admin | 新增收件人（僅支援 `email` 型別） |
| POST | `/settings/notification-recipients/batch` | admin | 批次新增，`{recipients: [{topic, email, displayName?}]}`；`topic`+`email` 重複者靜默略過，回傳只含實際新增的列 |
| POST | `/settings/notification-recipients/batch-delete` | admin | 批次刪除，`{ids: string[]}`，回 `{count}` |
| PATCH | `/settings/notification-recipients/:id` | admin | |
| DELETE | `/settings/notification-recipients/:id` | admin | |
| GET | `/notifications/logs` | viewer, staff, admin | 通知發送歷史 |

`notification_recipients` 除 `email` 型別外，資料庫已加 `recipient_type`/`target_role`/`user_id` 欄位（`role`／`user` 型別），但目前沒有完整的前端建立／更新流程會使用這兩種型別。寄送時若收件人未能解析出 email 會略過並記 log，不讓整批通知失敗；目前 server 預設 sender 是 simulated `LogEmailSender`，不代表 Resend delivery 已完成。

## 報表 `reportH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/reports/trip-summary` | viewer, staff, admin | 車輛趟數表 |
| GET | `/reports/trip-summary/export` | viewer, staff, admin | 匯出 Excel |
| GET | `/reports/hsinchu-schedule` | viewer, staff, admin | 新竹接送時刻表 |
| GET | `/reports/hsinchu-schedule/export` | viewer, staff, admin | 匯出 Excel |

## 車輛維修保養 `maintenanceH`

| Method | Path | 角色 |
|---|---|---|
| GET | `/vehicles/maintenance` | viewer, staff, admin |
| POST | `/vehicles/maintenance` | staff, admin |
| PATCH | `/vehicles/maintenance/:id` | staff, admin |
| DELETE | `/vehicles/maintenance/:id` | staff, admin |
| GET | `/vehicles/maintenance/blank-template` | viewer, staff, admin |

## 出勤與油資 `attendanceH` / `fuelH`

| Method | Path | 角色 |
|---|---|---|
| GET | `/attendance` | viewer, staff, admin |
| POST | `/attendance` | staff, admin |
| GET | `/attendance/conflicts` | viewer, staff, admin |
| POST | `/attendance/conflicts/:id/resolve` | staff, admin |
| GET | `/fuel-logs` | viewer, staff, admin |
| POST | `/fuel-logs` | staff, admin |
| PATCH | `/fuel-logs/:id` | staff, admin |
| DELETE | `/fuel-logs/:id` | staff, admin |

## 儀表板 `dashboardH`

| Method | Path | 角色 |
|---|---|---|
| GET | `/dashboard/metrics` | viewer, staff, admin |
| GET | `/dashboard/stats` | viewer, staff, admin |

`/dashboard/stats` 的 `recentExports` 目前是最近 5 筆申報匯出工作（重用 `exportH` 的 `ExportJobDTO` 形狀）；其餘欄位與 dashboard metrics 的資料來源見 [integration-contract.md](integration-contract.md) 及 review report，部分 KPI 仍需 runtime／業務資料驗證。

## 稽核紀錄 `auditH`

| Method | Path | 角色 |
|---|---|---|
| GET | `/audit` | admin |

## 排程任務（給 Cloud Scheduler 打的內部端點）`taskH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| POST | `/tasks/check-missing-reports` | staff, admin | 觸發「未回報偵測」批次 |
| POST | `/tasks/month-end-reminder` | staff, admin | 觸發「月底申報提醒」批次 |

## 照護人員主檔 `caregiverH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/caregivers` | viewer, staff, admin | 支援 `q`、`status`（`active`／`inactive`）、`unresolvedLink`（單位待關聯既有單位）、`incomplete`（聯絡方式或備註缺漏）篩選 |
| POST | `/caregivers` | staff, admin | 新增照護人員，姓名與類型（`case_manager`＝個管／`specialist`＝專護）皆為必填；`status` 非 `active`／`inactive` 一律預設 `active` |
| GET | `/caregivers/template` | viewer, staff, admin | 下載批次匯入用 Excel 範本 |
| POST | `/caregivers/import` | staff, admin | 批次匯入照護人員 Excel（僅支援 .xlsx）；姓名或類型缺漏（或類型不是個管／專護）略過，單位比對不到或聯絡方式／備註缺漏仍建立資料並附警告 |
| PATCH | `/caregivers/:id` | staff, admin | |
| DELETE | `/caregivers/:id` | admin | |
| PUT | `/caregivers/:id/site` | staff, admin | 將單位待關聯的照護人員連結至既有單位，並清空原始單位名稱 |

## 自助身分資訊與密碼

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/auth/me` | authenticated | 取得目前 JWT actor、built-in role 與 resolved permission |
| POST | `/auth/change-password` | authenticated | 目前登入者變更自己的密碼；需依 Supabase Auth provider 流程驗證舊密碼 |

## 角色身分管理 `roleH`

角色資料落在 `roles` 表（`identity` 模組），非 Supabase 端資料，`is_system` 系統角色（`admin`/`dispatcher`/`staff`/`driver`/`viewer`）不可刪除且權限矩陣不可覆寫成別的 `base_role`。

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/roles` | admin | 角色清單（裸陣列，非 `Paged`），含各角色實際使用者數 |
| GET | `/roles/:id` | admin | |
| POST | `/roles` | admin | 新增自訂角色；`key` 未提供時由 `name` 產生 slug |
| PATCH | `/roles/:id` | admin | 系統角色不可修改（`ErrSystemRoleImmutable`） |
| DELETE | `/roles/:id` | admin | 系統角色或仍有使用者的角色不可刪除（`ErrSystemRoleImmutable`／`ErrRoleInUse`） |

目前 role route 也走 `RequirePermission("settings_roles", action)`；`is_system` 角色不可刪除，細節由 identity app service 與 DB contract 決定。仍需補 self-role、last-admin 與跨 instance permission cache 的安全規則，詳見 review report。

## 使用者帳號管理 `identityH`

底層是 Supabase Auth Admin API，需要 `SUPABASE_SERVICE_ROLE_KEY` 才能運作；**金鑰未設定時所有端點一律回 `503`（`CodeServiceUnavailable`），不會退化成假資料**。角色一律寫入 JWT 的 `app_metadata.role`（依 [jwt-role-metadata-precedence.md](../decisions/jwt-role-metadata-precedence.md)），`user_metadata` 只放 `displayName`/`phone`/`status` 等非授權資料。

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/users` | admin | 使用者清單（裸陣列），支援 `keyword`／`role` 篩選（app 層過濾；目前沒有完整 server-side pagination） |
| GET | `/users/:id` | admin | |
| POST | `/users` | admin | 建立使用者，`role` 須存在於 `roles` 表 |
| PATCH | `/users/:id` | admin | |
| PUT | `/users/:id/permissions` | admin | 覆寫個人自訂權限（存於 `app_metadata.custom_permissions`） |
| POST | `/users/:id/reset-password` | admin | 管理員重設指定使用者密碼 |
| DELETE | `/users/:id` | admin | 不可刪除自己（`ErrCannotDeleteSelf`，403） |
| POST | `/auth/change-password` | authenticated | 任何已登入者可改自己的密碼；後端先以舊密碼呼叫 Supabase `grant_type=password` 驗證通過才允許改新密碼 |
