---
doc_type: api
covers: ["apps/api/cmd/server/routes.go"]
---

# 後端 API 路由總覽

Base path：`/api/v1`，全部要帶 JWT（`auth.Middleware`），除了 `/api/health`（不驗證）跟 `/api/v1/ingest/google-form`（走 `X-Ingest-Token`）。角色欄是 `auth.RequireRoles(...)` 白名單，實作對應各能力模組的 `internal/modules/<capability>/transport/*.go`。路由表以 `apps/api/cmd/server/routes.go` 為唯一事實來源，改路由記得同步更新這份文件。

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
| POST | `/cases/:id/reveal` | staff, admin | 明文顯示身分證字號（會寫 audit log 的 `reveal_pii`） |
| GET | `/cases/:id/schedule` | viewer, staff, admin | 取得排班（星期、時段、四趟制設定） |
| PUT | `/cases/:id/schedule` | staff, admin | 覆寫排班 |
| POST | `/cases/schedules` | staff, admin | 批次建立排班 |
| POST | `/cases/import` | staff, admin | 批次匯入個案 Excel |
| POST | `/masters/import` | staff, admin | 同上，走另一條相容路徑（歷史因素，實際都打 `caseH.ImportExcel`） |
| GET | `/cases/export` | viewer, staff, admin | 匯出個案彙整表 |
| PUT | `/cases/:id/transport-preference` | staff, admin | 更新個案交通偏好設定 |

## 據點主檔 `siteH`

| Method | Path | 角色 |
|---|---|---|
| GET | `/sites` | viewer, staff, admin |
| POST | `/sites` | staff, admin |
| PATCH | `/sites/:id` | staff, admin |
| DELETE | `/sites/:id` | admin |

## 車輛主檔 `vehicleH`

| Method | Path | 角色 |
|---|---|---|
| GET | `/vehicles` | viewer, staff, admin |
| POST | `/vehicles` | staff, admin |
| PATCH | `/vehicles/:id` | staff, admin |

## 司機主檔 `driverH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/drivers` | viewer, staff, admin | |
| POST | `/drivers` | staff, admin | |
| PATCH | `/drivers/:id` | staff, admin | |
| POST | `/drivers/:id/reveal` | staff, admin | 明文顯示司機個資 |
| POST | `/drivers/:id/assignments` | staff, admin | 指派車輛給司機 |

## 表單同步與欄位對應 `formH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/forms` | viewer, staff, admin | 已串接的 Google 表單清單 |
| POST | `/forms/:id/sync` | staff, admin | 觸發表單欄位同步（重新讀取 Google Sheet 欄名） |
| GET | `/forms/columns` | viewer, staff, admin | 表單欄位清單與對應狀態 |
| PATCH | `/forms/columns/:id/mapping` | staff, admin | 設定單一欄位對應到哪個個案的哪一趟 |
| POST | `/forms/columns/batch-mapping` | staff, admin | 批次設定欄位對應 |
| GET | `/forms/google-drive-files` | staff, admin | 列出 Google Drive 可串接的表單檔案 |
| POST | `/forms/inspect-sheet` | staff, admin | 讀取指定 Google Sheet 的欄名（串接前預覽） |

## 搭乘月曆、異常與更正 `rideH` / `taskH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/rides/calendar` | viewer, staff, admin | 月曆矩陣視圖（個案 × 日期 × 趟次的搭乘狀態） |
| GET | `/rides/issues` | viewer, staff, admin | 異常搭乘集中清單（衝突、待確認等） |
| GET | `/rides/missing` | viewer, staff, admin | 未回報清單（`taskH.GetMissingReports`） |
| GET | `/rides/:id` | viewer, staff, admin | 單筆搭乘紀錄詳情 |
| PATCH | `/rides/:id` | staff, admin | 人工更正搭乘紀錄（寫 audit log） |
| POST | `/rides/manual-report` | staff, admin | 人工補登整筆回報（月曆空白格填寫） |
| POST | `/rides/:id/resolve-conflict` | staff, admin | 裁決同車衝突回報 |
| POST | `/api/v1/ingest/google-form` | 無（`X-Ingest-Token`） | Google 表單 Webhook 接收端點 |

## 匯出前置檢核與工作管理 `exportH`

| Method | Path | 角色 | 說明 |
|---|---|---|---|
| GET | `/exports/precheck` | staff, admin | 執行匯出前置檢核 |
| POST | `/exports/precheck` | staff, admin | 同上 |
| GET | `/exports` | viewer, staff, admin | 匯出工作歷史清單 |
| POST | `/exports` | staff, admin | 建立匯出工作 |
| GET | `/exports/:id` | viewer, staff, admin | 單筆匯出工作詳情（狀態、下載連結） |

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
| POST | `/settings/notification-recipients` | admin | 新增收件人 |
| PATCH | `/settings/notification-recipients/:id` | admin | |
| DELETE | `/settings/notification-recipients/:id` | admin | |
| GET | `/notifications/logs` | viewer, staff, admin | 通知發送歷史 |

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
| GET | `/fuel-logs` | viewer, staff, admin |
| POST | `/fuel-logs` | staff, admin |
| PATCH | `/fuel-logs/:id` | staff, admin |
| DELETE | `/fuel-logs/:id` | staff, admin |

## 儀表板 `dashboardH`

| Method | Path | 角色 |
|---|---|---|
| GET | `/dashboard/metrics` | viewer, staff, admin |
| GET | `/dashboard/stats` | viewer, staff, admin |

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
| GET | `/caregivers` | viewer, staff, admin | 支援 `q`、`unresolvedLink`（單位待關聯既有據點）、`incomplete`（聯絡方式或備註缺漏）篩選 |
| POST | `/caregivers` | staff, admin | 新增照護人員，僅姓名必填 |
| GET | `/caregivers/template` | viewer, staff, admin | 下載批次匯入用 Excel 範本 |
| POST | `/caregivers/import` | staff, admin | 批次匯入照護人員 Excel／CSV；姓名缺漏略過，單位比對不到或聯絡方式／備註缺漏仍建立資料並附警告 |
| PATCH | `/caregivers/:id` | staff, admin | |
| DELETE | `/caregivers/:id` | admin | |
| PUT | `/caregivers/:id/site` | staff, admin | 將單位待關聯的照護人員連結至既有據點，並清空原始單位名稱 |

## ⚠️ 前端已預留但後端尚未實作

前端 `src/api/users.ts`、`src/api/roles.ts` 已經寫好 `/users`、`/roles`、`/auth/change-password` 這幾支的呼叫（型別也定義好了），對應「使用者管理」「角色身分管理」頁面。**這幾支路由目前不存在於 `cmd/server/main.go`**，只有 MSW mock（`apps/web/src/mocks/handlers/`）在假裝有這個 API。

要串接時：新增 `UserHandler`／`RoleHandler`（連同對應 `service`／`repository`），路由掛在 `apiV1` 群組下，權限比照現有 `settings/*` 系列多半是 `admin` only。串接前先確認前端 `apps/web/src/types/api.d.ts` 裡 `UserDTO`／`RoleDTO`／`CreateUserRequest` 等型別跟要設計的後端 DTO 是否一致。
