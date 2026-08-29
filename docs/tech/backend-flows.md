# 後端核心業務流程

這份文件拆解幾條橫跨多個 handler／service／domain 套件的完整流程，比單看某一個檔案更容易看懂系統在幹嘛。分層架構背景見 [backend-framework.md](backend-framework.md)，逐支端點清單見 [backend-api-reference.md](backend-api-reference.md)。

## 1. Google 表單回報 → 搭乘紀錄（`RideService.IngestWebhook`）

這是整個系統最核心的資料流，一次表單提交會走過：

1. Google Apps Script（不在本 repo）在表單提交時打 `POST /api/v1/ingest/google-form`，帶 `X-Ingest-Token` 跟整列答案（`answers` 是欄位 index/表頭 → 值的 map）。
2. 依 token 找出對應的 `forms` 記錄跟預設車輛，把這次提交原封不動存一筆 `form_submissions`（raw payload，方便日後追查原始回報內容）。
3. 用 `domain/namenorm.Normalize` 正規化司機姓名，比對司機主檔抓 `driver_id`（配不到就先留空，之後可以在「表單同步管理」頁手動對應）。
4. 逐一走過這個表單已設定好對應（`mapping_status = mapped`）的欄位，判斷每欄值是「有坐」還是「沒坐」，其他文字視為非明確標記直接跳過。
5. 若個案排班是四趟制，表單上的「第 1 趟」「第 2 趟」要展開成資料庫的四趟（1→1,3；2→2,4），這是四趟展開規則的實作位置。
6. 每個展開後的趟次都會 `InsertRideSource` 存一筆來源紀錄，然後呼叫 `recalculateRideRecord` 重新跑 `domain/merge` 演算法，用「同車取最新、跨車 OR」規則重算 `ride_records` 的最終狀態——已經被人工更正過的紀錄不會被自動覆蓋。

```
Google 表單提交
   │
   ▼
POST /api/v1/ingest/google-form (X-Ingest-Token)
   │
   ▼
存 form_submissions（原始 payload）
   │
   ▼
namenorm.Normalize(driverRaw) → 配對司機主檔
   │
   ▼
逐欄位判斷「有坐/沒坐」 → 四趟展開（若排班為四趟制）
   │
   ▼
InsertRideSource（來源紀錄）
   │
   ▼
merge.MergeRideSources（同車取最新、跨車 OR）
   │
   ▼
更新 ride_records（人工更正過的紀錄不覆蓋）
```

## 2. 搭乘紀錄更正與衝突處理

- 前端「異常集中處理」頁列出 `domain/merge` 判定為衝突或待確認的紀錄，operator 可以：
  - `PATCH /rides/:id`（`rideH.Correct`）：直接改成正確狀態，寫入 audit log（action = `correct`）。
  - `POST /rides/:id/resolve-conflict`：在多個來源互相矛盾時選擇要採信哪一筆。
- 月曆矩陣頁點空白格可以 `POST /rides/manual-report` 直接補登整筆回報，等同司機事後補交表單（`RideService.ManualReportRide`）。

## 3. 未回報偵測與月底提醒（`TaskService`）

- `CheckMissingReports(ctx, targetDate, region)`：拿 `domain/calendar.CalculateExpectedRides` 算出「這天應該有的搭乘」，跟實際 `ride_records` 比對，抓出應搭但沒回報的趟次，回傳給前端「未回報清單」頁，並觸發告警通知。
- `MonthEndReminder(ctx, year, month)`：每月 26 日跑，彙整檢核結果並發信提醒。
- 這兩支都各自有 `POST /tasks/*` 端點，正式環境由 Cloud Scheduler 定期打；本機要測試就直接手動 curl 這兩支。

## 4. 政府申報匯出（`PrecheckService` + `cmd/exporter`）

1. 使用者在「政府申報匯出」頁選期別／地區，先打 `GET/POST /exports/precheck` 跑 `PrecheckService.RunPrecheck`：檢查這個月這個地區的資料是否完整（有沒有未結案的異常、必要欄位是否齊全），回傳 `PrecheckReport`（含每一項檢核的 severity）。
2. 檢核沒過（`Passed=false`）就不能真的匯出，前端要把 issue 列出來讓使用者回去處理。
3. 檢核通過後才建立匯出工作（`POST /exports`），實際產檔是由獨立的 Cloud Run Job（`cmd/exporter`）執行：一樣先跑一次 precheck，通過才呼叫 `domain/govform.BuildClaimRow` 組出符合政府 33 欄規格的資料列，`export.GenerateGovClaimExcel` 用 `excelize` 產出 `.xlsx`，並計算 SHA-256 checksum 供事後驗證檔案未被竄改。

```
使用者選期別/地區
   │
   ▼
GET/POST /exports/precheck ──未通過──► 前端列出 issue，回去修資料
   │ 通過
   ▼
POST /exports（建立匯出工作）
   │
   ▼
Cloud Run Job cmd/exporter：precheck → govform.BuildClaimRow → excelize 產出 .xlsx → SHA-256 checksum
   │
   ▼
GET /exports/:id 輪詢狀態拿下載連結
```

## 5. 主檔批次匯入（`ImportService`）

`POST /cases/import`（或相容路徑 `/masters/import`）吃使用者上傳的 Excel，`ImportService.ParseCasesFromExcel` 逐列解析、驗證欄位（含 `ParseWeekdays` 解析「每週據點開放時間」這種自由文字格式），回傳每列的解析結果與統計，成功的列才會實際寫入個案主檔；另外 `ParseScheduleWorkbook` 專門解析「(參考用) 交通車接送班表」這份既有 Excel，抽出據點跟司機資訊。

## 6. 通知（`NotificationService`）

`SendNotification` 依 topic（例如未回報告警、月底提醒）撈出啟用中的收件人清單逐一寄信，寄送介面是 `EmailSender`，正式環境用 Resend（`RESEND_API_KEY`），本機沒設定時退化成 `LogEmailSender`（只印 log，不真的寄信）。收件人管理走 `settings/notification-recipients` 系列端點。

## 7. 表單來源設定（Google Drive 檔案列表與 Sheet 欄名預覽）

串新表單前，`GET /forms/google-drive-files` 列出 Google Drive 上的候選試算表，`POST /forms/inspect-sheet` 預覽指定 Sheet 的欄名，兩支都呼叫 `FormService` 底層的 Google API client（`internal/modules/formsync/app/form_service.go`）。**沒有設定 Google 服務帳號憑證（`GOOGLE_SA_JSON`）時，這兩支會直接回 `FORM_SOURCE_FAILED` 錯誤，不會回傳假資料**——這是刻意修正過的行為，避免本機或測試環境誤以為串接成功。

## 8. 稽核留痕（`middleware.RecordAuditLog`）

凡是會動到個資或關鍵狀態的操作（新增、修改、reveal PII、更正搭乘紀錄、裁決衝突、匯出、設定變更、匯入）都會呼叫 `RecordAuditLog` 寫一筆 `audit_log`，記錄操作者、角色、動作類型、異動前後的資料快照。`GET /audit` 只有 `admin` 能查，是唯一的稽核紀錄查詢入口。
