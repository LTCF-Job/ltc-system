---
doc_type: api
covers: ["apps/api/cmd/server/routes.go"]
---

# 後端 API 路由總覽

Base path：`/api/v1`，全部要帶 JWT（`auth.Middleware`），除了 `/api/health`（不驗證）跟 `/api/v1/ingest/google-form`（走 `X-Ingest-Token`）。實作對應各能力模組的 `internal/modules/<capability>/transport/*.go`。路由表以 `apps/api/cmd/server/routes.go` 為唯一事實來源，改路由記得同步更新這份文件。

下表「角色」欄列的是**目前系統五個內建角色（viewer/dispatcher/staff/driver/admin）實際能通過的結果**，不是授權機制本身：所有 API 路由都透過 `auth.RequirePermission(module, action)` 查角色的模組權限矩陣（`roles.permissions`，可在「角色身分管理」頁調整，自訂角色的實際存取範圍以矩陣為準，不受下表侷限）。機制細節見 [role-permission-api-authorization.md](../decisions/role-permission-api-authorization.md)。

架構背景見 [backend-framework.md](backend-framework.md)，每支端點背後的業務流程見 [backend-flows.md](backend-flows.md)。

## 個案主檔與排班 `caseH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/cases` | viewer, staff, admin | 個案清單（回傳遮罩身分證）。**待維護個案預設不回傳**：`unresolvedLink=true` 只取待維護，`includePending=true` 取全部；`region` 依已關聯據點的區域篩選（對 `sites.region` 模糊比對） |
| POST | `/cases` | staff, admin | 新增個案；`siteId`（所屬據點）與 `caregiverId`（照護人員）為必填 |
| GET | `/cases/template` | viewer, staff, admin | 下載批次匯入用 Excel 範本；欄位版面與 `/cases/export` 共用同一組 A~K 11 欄（見系統邏輯規格書準則三.二） |
| GET | `/cases/:id` | viewer, staff, admin | |
| PATCH | `/cases/:id` | staff, admin | |
| DELETE | `/cases/:id` | admin | 軟刪除（`deleted_at`/`deleted_by`），同交易內收斂生效中排班 |
| POST | `/cases/:id/reveal` | staff, admin | 明文顯示身分證字號（會寫 audit log 的 `reveal_pii`） |
| GET | `/cases/:id/schedule` | viewer, staff, admin | 取得排班（星期、時段、四趟制設定） |
| PUT | `/cases/:id/schedule` | staff, admin | 覆寫排班 |
| POST | `/cases/schedules` | staff, admin | 批次建立排班 |
| POST | `/cases/import` | staff, admin | 批次匯入個案 Excel；疑似重複個案不建立個案，改建立為待裁決暫存列。「個管or照專」旁的姓名會比對 `caregivers` 主檔寫入 `caregiver_id`，比不到則落入待維護 |
| POST | `/masters/import` | staff, admin | 同上，走另一條相容路徑（歷史因素，實際都打 `caseH.ImportExcel`） |
| GET | `/cases/export?caseIds=` | viewer, staff, admin | 匯出個案彙整表；`caseIds` 為逗號分隔的個案 ID，省略則匯出全部個案。「個管or照專」與其右方姓名取自關聯的照護人員主檔 |
| PUT | `/cases/:id/transport-preference` | staff, admin | 更新個案交通偏好設定（去/回程車輛，完整替換語意：`outboundVehicleId`／`inboundVehicleId` 與對應的 `outboundVehicleNameRaw`／`inboundVehicleNameRaw` 未帶上即視為清空，尚未完成關聯的匯入原始名稱必須原樣回送）。**據點不在此端點設定**，個案的據點改由 `PATCH /cases/:id` 的 `siteId` 更新 |
| GET | `/cases/import/duplicates` | viewer, staff, admin | 列出待裁決的疑似重複個案暫存列 |
| POST | `/cases/import/duplicates/:id/reveal` | staff, admin | 解密單筆暫存列身分證字號供裁決比對（會寫 audit log 的 `reveal_pii`） |
| POST | `/cases/import/duplicates/:id/resolve` | staff, admin | 裁決疑似重複個案（`confirmed_new` 建立新個案／`merged_existing` 合併進既有個案） |
| DELETE | `/cases/import/duplicates/:id` | admin | 忽略疑似重複個案：刪除暫存列，不建立也不合併個案。重新匯入同一份檔案時該筆會再次出現 |

## 據點主檔 `siteH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/sites` | viewer, staff, admin | 支援 `q`、`region`、`status`（`active`／`inactive`）篩選；`region` 為使用者自由填寫的文字，以模糊比對；包含 `remarks` 備註欄位 |
| POST | `/sites` | staff, admin | 僅 `name` 為必填，`region`（自由文字）、`address` 與 `remarks` 為選填 |
| PATCH | `/sites/:id` | staff, admin | 整筆覆寫，必填與選填欄位同 POST |
| DELETE | `/sites/:id` | admin | 刪除據點 |

## 車輛主檔 `vehicleH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/vehicles` | viewer, staff, admin | 支援 `q`、`status`（`active`／`inactive`）篩選（`siteId`／`region` 篩選與唯讀 `region` 欄位已隨車輛與據點主檔解耦移除）；每筆帶 `drivers`（該車今日生效的司機，一台車可有多位）、車輛自己的自由文字 `siteName`（不關聯據點主檔）、`remarks`（備註），以及四項證件註記 `hasVehicleLicense`（行照）、`hasPurchaseContract`（汽車買賣合約書）、`hasPlateRegistration`（領牌登記書）、`hasTransferRegistration`（異動登記書） |
| POST | `/vehicles` | staff, admin | `plateNo` 與 `displayName`（車別）為必填，`siteName`（自由文字，不驗證關聯）、`remarks` 與其餘車籍欄位皆為選填（支援 `null` 與空值）；四項證件註記未提供時一律為 `false`；`status` 非 `active`／`inactive` 一律預設 `active`；車號或車別重複時回 409 並帶 `details` |
| PATCH | `/vehicles/:id` | staff, admin | 整筆覆寫，必填欄位同 POST；車號或車別重複時回 409 並帶 `details` |
| DELETE | `/vehicles/:id` | admin | 軟刪除（僅標記 `deleted_at`，不影響 `status` 啟用/停用狀態）；仍有生效中司機指派或排班趟次綁定時回 409（`CodeResourceInUse`） |
| PUT | `/vehicles/:id/drivers` | staff, admin | 整批設定本車司機：`{ driverIds: string[], effectiveFrom?: date }`；`driverIds` 為空代表清空 |

## 司機主檔 `driverH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/drivers` | viewer, staff, admin | 支援 `q`、`status`（`active`／`inactive`）篩選；回傳欄位含 `gender`、`birthDate`、`hasProfessionalLicense`、`employmentDate`、`hasTransferCert`、`remarks`、`licenseClass`、`licenseExpiryDate` |
| POST | `/drivers` | staff, admin | `name` 與 `nationalId` 為必填，擴充欄位（`gender`, `birthDate`, `hasProfessionalLicense`, `employmentDate`, `hasTransferCert`, `remarks`, `licenseClass`, `licenseExpiryDate`）與 `vehicleId`（指派車輛 ID）皆為選填，提供 `vehicleId` 時於同一交易內建立指派紀錄，新增一律為 `active` |
| PATCH | `/drivers/:id` | staff, admin | 欄位未提供代表不變更；日期欄位明確給 `null` 才會清空；`status` 非 `active`／`inactive` 時保留原值不變更。可帶 `nationalId` 變更身分證：會重新驗證檢查碼並同步重算密文、HMAC 索引與遮罩值，檢查碼錯誤回 400、與其他司機重複回 409，皆帶 `details` |
| DELETE | `/drivers/:id` | admin | 軟刪除（僅標記 `deleted_at`，不影響 `status` 啟用/停用狀態），同交易內收斂生效中的司機指派區間 |
| POST | `/drivers/:id/reveal` | staff, admin | 明文顯示司機個資 |
| POST | `/drivers/:id/assignments` | staff, admin | 指派車輛給司機；body 只有 `vehicleId`，一律自今日起生效且不設結束日。一位司機同期只會有一台車，指派新車會先收斂原本的指派 |

## 司機接送匯報與欄位對應 `driverReportH`

一台車一份匯報表；資料來源是使用者上傳的 `.xlsx`，沒有任何 Google 串接。

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/driver-reports` | viewer, staff, admin | 各車匯報表清單與欄位對應進度 |
| POST | `/driver-reports` | staff, admin | 為一台車建立匯報表；該車已有匯報表時只更新名稱並回傳既有那一份 |
| DELETE | `/driver-reports/:id` | staff, admin | 刪除匯報表（欄位對應與匯報紀錄一併移除） |
| GET | `/driver-reports/:id/template` | staff, admin | 下載該車空白匯報範本（`.xlsx`，只有表頭） |
| POST | `/driver-reports/:id/import?dryRun=&yearMonth=` | staff, admin | 上傳匯報檔；`dryRun=true`（預設）回傳預覽，`dryRun=false` 正式寫入，逐列比對既有資料（沒問題直接寫入、值不同進待維護）。`yearMonth`（`YYYY-MM`）選填，宣告後拒收該月以外的日期 |
| GET | `/driver-reports/imported-months` | viewer, staff, admin | 每份匯報表各月份已匯入的筆數與最後匯入時間 |
| GET | `/driver-reports/columns` | viewer, staff, admin | 欄位清單與對應狀態（可帶 `formId`、`mappingStatus`） |
| GET | `/driver-reports/columns/name-matches?name=` | viewer, staff, admin | 找出目前待維護欄位中姓名與傳入姓名相符（含近似）的欄位 |
| PATCH | `/driver-reports/columns/:id/mapping` | staff, admin | 設定單一欄位對應到哪個個案的哪一趟；剛從待維護變成已對應時同一交易內立即回填搭乘紀錄 |
| POST | `/driver-reports/columns/batch-mapping` | staff, admin | 批次設定欄位對應 |
| GET | `/driver-reports/submissions/review` | viewer, staff, admin | 以匯報表列（一天一筆提交）為單位列出待維護資料，一列可能同時有個案欄位與駕駛人兩種問題 |
| POST | `/driver-reports/drivers/bind` | staff, admin | 把某個比對不到司機主檔的原始姓名綁定到指定司機，立即回填所有正規化姓名相符的既有回報 |
| POST | `/driver-reports/row-conflicts/:id/resolve` | staff, admin | 裁決一筆「同車同個案」衝突；body `{useNew}`，`true` 採用這次上傳的新值並重算搭乘紀錄，`false` 保留既有資料 |
| DELETE | `/driver-reports/columns/:id` | staff, admin | 忽略一筆欄位對應待維護資料，直接刪除該列 |
| DELETE | `/driver-reports/row-conflicts/:id` | staff, admin | 忽略一筆「同車同個案」衝突，直接刪除該衝突列；既有搭乘資料維持原值 |
| DELETE | `/driver-reports/submissions/:id` | staff, admin | 忽略一筆駕駛人未比對到司機主檔的匯報列；刪除提交紀錄並重算受連帶刪除的搭乘來源所影響的搭乘紀錄 |

匯入檔的欄位順序固定為：民國日期、駕駛人、各個案趟次欄、備註。個案趟次欄只接受
「有坐」「沒坐」，其餘（含空白）視為未回報不建立紀錄。`dryRun=false` 時可另外以
form field `columnDecisions` 帶入預覽畫面就地確認的欄位對應（JSON 陣列，元素為
`{columnHeader, mappingStatus, caseId, legSeq}`）。

## 搭乘月曆、異常與更正 `rideH` / `taskH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/rides/calendar` | viewer, staff, admin | 月曆矩陣視圖（個案 × 日期 × 趟次的搭乘狀態） |
| GET | `/rides/issues` | viewer, staff, admin | 異常搭乘集中清單（衝突、待確認等） |
| GET | `/rides/missing` | viewer, staff, admin | 未回報清單（`taskH.GetMissingReports`） |
| GET | `/rides/:id` | viewer, staff, admin | 單筆搭乘紀錄詳情 |
| PATCH | `/rides/:id` | staff, admin | 人工更正搭乘紀錄（寫 audit log） |
| POST | `/rides/manual-report` | staff, admin | 人工補登整筆回報（月曆空白格填寫） |
| POST | `/rides/:id/resolve-conflict` | staff, admin | 裁決同車衝突回報：`{vehicleId, driverId, reason}`，寫入 `conflict_resolution_note` 並記稽核（`resolve_conflict`） |

`GET /rides/issues` 支援 `issueType=conflict\|unreported\|import_error`（三擇一）、`month`（`YYYY-MM`，省略則預設當月）、`keyword`、`page`、`pageSize`。三種類型的資料路徑完全不同：`conflict` 讀 `ride_records` 聚合 `ride_sources` 車輛陣列；`unreported` 重用 `task/app.TaskService.ListMissingReportsForMonth` 的整月查詢（不觸發催報通知）；`import_error` 讀 `form_submissions.anomaly_flags` 非空的列。

## 匯出前置檢核與工作管理 `exportH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/exports/precheck` | staff, admin | 依 `periodYm` 與可選 `caseIds` 執行匯出前置檢核 |
| POST | `/exports/precheck` | staff, admin | 同上；body 可帶 `periodYm`、`caseIds` |
| GET | `/exports` | viewer, staff, admin | 匯出工作歷史清單（不含檔案明細與下載連結） |
| POST | `/exports` | staff, admin | 建立政府申報匯出工作並同步產檔；body 需帶 `periodYm`(民國 5 碼)、`mode`(`direct`\|`zip`)、`caseIds`(至少一筆) |
| GET | `/exports/:id` | viewer, staff, admin | 單筆匯出工作詳情，含逐案檔案清單 `files` |
| GET | `/exports/:id/files/:caseId/download` | viewer, staff, admin | 下載單一個案的申報 `.xlsx` |
| GET | `/exports/:id/download` | viewer, staff, admin | 下載整包 `.zip`；非壓縮檔模式回 400 |
| POST | `/exports/by-region` | staff, admin | 依區域批次匯出：`{regions[], periodYms[]}`，逐月各建立一個工作 |
| GET | `/exports/batch-download` | viewer, staff, admin | 合併多筆工作的檔案成單一 `.zip`；`?jobIds=a&jobIds=b` |
| POST | `/exports/site-trip-summary` | viewer, staff, admin | 據點趟數彙總表：`{siteIds[], periodYms[]}`，直接回 `.xlsx` 位元組 |

`/exports/precheck` 的 body 可另外帶 `periodYms`（民國 5 碼陣列，與 `periodYm` 擇一）與
`regions`（與 `caseIds` 擇一；區域會先展開成個案再檢核），三個分頁籤共用同一支端點。

`POST /exports/by-region` 一律以壓縮檔模式產出，`periodYms` 上限 12 個月，並掛上與匯報表匯入
相同的延長逾時 middleware（一次可能產出數百份檔案並逐一上傳 object storage）。回應為
`{jobs[], batchDownloadUrl, batchFileName, totalFiles}`，**一個月一筆工作**——`export_job_files`
有 `UNIQUE(job_id, case_id)`，同一位個案的不同月份無法共存於同一個工作。

`POST /exports/site-trip-summary` **不建立匯出工作**：趟數彙總表是管理用統計而非申報檔，
不進 `export_jobs` 的不可變快照與稽核軌跡，因此也不會出現在歷史匯出紀錄。

以上三支與 `POST /exports` 在查無任何可申報資料時一律回 **422 `NO_EXPORT_DATA`**（訊息
「指定條件下沒有可申報的資料」），不再靜默回成功並產出 0 份檔案。

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

`notification_recipients` 除 `email` 型別外，資料庫已加 `recipient_type`/`target_role`/`user_id` 欄位（`role`／`user` 型別），但目前沒有任何前端頁面會建立這兩種型別；寄送時若收件人未能解析出 email 會略過並記 log，不讓整批通知失敗。

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
| DELETE | `/attendance/conflicts/:id` | staff, admin |
| GET | `/fuel-logs` | viewer, staff, admin |
| POST | `/fuel-logs` | staff, admin |
| PATCH | `/fuel-logs/:id` | staff, admin |
| DELETE | `/fuel-logs/:id` | staff, admin |

## 儀表板 `dashboardH`

| Method | Path | 角色 |
|---|---|---|
| GET | `/dashboard/metrics` | viewer, staff, admin |
| GET | `/dashboard/stats` | viewer, staff, admin | 回 `recentExports`：最近 5 筆申報匯出工作（重用 `exportH` 的 `ExportJobDTO` 形狀），其餘欄位見 [integration-contract.md](integration-contract.md) |

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
| GET | `/caregivers` | viewer, staff, admin | 支援 `q`、`status`（`active`／`inactive`）篩選。**待維護資料（姓名或類型未填寫）預設不回傳**：`pending=true` 只取待維護，`includePending=true` 取全部 |
| POST | `/caregivers` | staff, admin | 新增照護人員，姓名與類型（`case_manager`＝個管／`specialist`＝照專）皆為必填；`status` 非 `active`／`inactive` 一律預設 `active` |
| GET | `/caregivers/template` | viewer, staff, admin | 下載批次匯入用 Excel 範本 |
| POST | `/caregivers/import` | staff, admin | 批次匯入照護人員 Excel（僅支援 .xlsx，表頭欄位為：類型*、單位、姓名*、聯絡方式、備註；向前相容舊表頭「據點」）；姓名或類型缺漏（或類型不是個管／照專，向後相容專護）改以空白建立並列入待維護，單位（`siteName`）為自由文字、不比對據點主檔，缺漏直接留白、不列入待維護，聯絡方式與備註缺漏不再產生警告 |
| PATCH | `/caregivers/:id` | staff, admin | |
| DELETE | `/caregivers/:id` | admin | 刪除照護人員；待維護清單的「忽略此筆」也走這支 |

照護人員所屬「單位」（欄位名稱仍為 `site_name`，UI 與範本已由「據點」改為「單位」）為自由文字，不關聯據點主檔。個案主檔則新增關聯 `caregiver_id`（外鍵關聯照護人員主檔），新增個案時必須選擇照護人員。

## 角色身分管理 `roleH`

角色資料落在 `roles` 表（`identity` 模組），非 Supabase 端資料，`is_system` 系統角色（`admin`/`dispatcher`/`staff`/`driver`/`viewer`）不可刪除且權限矩陣不可覆寫成別的 `base_role`。

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/roles` | admin | 角色清單（裸陣列，非 `Paged`），含各角色實際使用者數 |
| GET | `/roles/:id` | admin | |
| POST | `/roles` | admin | 新增自訂角色；`key` 未提供時由 `name` 產生 slug |
| PATCH | `/roles/:id` | admin | 系統角色不可修改（`ErrSystemRoleImmutable`） |
| DELETE | `/roles/:id` | admin | 系統角色或仍有使用者的角色不可刪除（`ErrSystemRoleImmutable`／`ErrRoleInUse`） |

所有路由（包含 `/users`、`/roles`、`/auth/change-password`、`/tasks/*` 與 `/holidays*`）都使用同一套 effective permission resolver；`auth.RequireRoles` 已移除，自訂角色不再被寫死的角色字串額外擋下。

## 使用者帳號管理 `identityH`

底層是 Supabase Auth Admin API，需要 `SUPABASE_SERVICE_ROLE_KEY` 才能運作；**金鑰未設定時所有端點一律回 `503`（`CodeServiceUnavailable`），不會退化成假資料**。角色一律寫入 JWT 的 `app_metadata.role`（依 [jwt-role-metadata-precedence.md](../decisions/jwt-role-metadata-precedence.md)），`user_metadata` 只放 `displayName`/`phone`/`status` 等非授權資料。

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/users` | admin | 使用者清單（`{data,meta}`），支援 `q`／`role`／`page`／`pageSize`，由本地 PostgreSQL projection 執行搜尋與分頁 |
| GET | `/users/:id` | admin | |
| POST | `/users` | admin | 建立使用者，`role` 須存在於 `roles` 表 |
| PATCH | `/users/:id` | admin | |
| PUT | `/users/:id/permissions` | admin | 覆寫個人自訂權限（存於 `app_metadata.custom_permissions`） |
| DELETE | `/users/:id` | admin | 不可刪除自己（`ErrCannotDeleteSelf`，403） |
| POST | `/auth/change-password` | viewer, staff, admin | 任何已登入者可改自己的密碼；後端先以舊密碼呼叫 Supabase `grant_type=password` 驗證通過才允許改新密碼 |
