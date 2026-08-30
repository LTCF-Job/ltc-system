# 後端核心業務流程

這份文件拆解幾條橫跨多個 handler／service／domain 套件的完整流程，比單看某一個檔案更容易看懂系統在幹嘛。分層架構背景見 [backend-framework.md](backend-framework.md)，逐支端點清單見 [backend-api-reference.md](backend-api-reference.md)。

## 1. 司機接送匯報匯入 → 搭乘紀錄（`DriverReportService.CommitDriverReport` → `RideService.IngestSubmission`）

這是整個系統最核心的資料流。一份匯報表對應一台車，操作人員上傳司機填好的 `.xlsx`：

1. `POST /api/v1/driver-reports/:id/import?dryRun=true` 先解析不寫入：`ParseDriverReport` 認表頭
   （民國日期、駕駛人、各個案趟次欄、備註），把每個個案欄位與既有 `form_columns` 對照，
   沒對應過的欄位附上以姓名相似度算出的推薦個案與由 `[去程]／[回程]` 推得的推薦趟次。
2. 使用者在預覽畫面就地確認欄位對應後，改打 `?dryRun=false`，並以 form field `columnDecisions`
   帶回確認結果。`CommitDriverReport` 先把欄位對應寫回 `form_columns`（以表頭文字為鍵，
   個案增減造成的欄號位移不會錯配），再逐列交給 ride 模組。
3. `RideService.IngestSubmission` 把該列原封不動存一筆 `form_submissions`（`source = 'import'`，
   raw payload 方便日後追查），並用 `domain/namenorm.Normalize` 正規化司機姓名比對司機主檔抓
   `driver_id`（配不到就先留空，並在預覽階段以警告提示）。
4. 逐一走過已設定對應（`mapping_status = mapped`）的欄位，用 `domain/merge.ParseReportedValue`
   判斷每欄值是「有坐」還是「沒坐」，其他文字（含空白）視為未回報直接跳過。
5. 若個案排班是四趟制，匯報表上的「第 1 趟」「第 2 趟」要展開成資料庫的四趟（1→1,3；2→2,4），
   這是四趟展開規則的實作位置（`expandLegSeqs`）。
6. 每個展開後的趟次都會 `InsertRideSource` 存一筆來源紀錄，然後呼叫 `recalculateRideRecord`
   讀回該 slot 的全部來源列跑 `domain/merge` 演算法，用「同車取最新、跨車 OR」規則重算
   `ride_records` 的最終狀態——已經被人工更正過的紀錄不會被自動覆蓋。

單列日期無法解析時只略過該列並回報原因，其餘日期照常寫入，不會讓一整個月的匯報因為
一列打錯而全部匯不進來。

```
使用者上傳匯報表 .xlsx
   │
   ▼
POST /driver-reports/:id/import?dryRun=true → 欄位對應與每日匯報預覽
   │
   ▼（就地確認未對應欄位）
POST /driver-reports/:id/import?dryRun=false + columnDecisions
   │
   ▼
寫回 form_columns（以表頭文字為鍵）
   │
   ▼
存 form_submissions（原始 payload，source = import）
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

## 7. 匯報表範本下載

`GET /driver-reports/:id/template` 依該車目前已對應的個案趟次欄產生空白 `.xlsx`：
表頭為「民國日期、駕駛人、各個案趟次欄、備註」，**刻意不放示範資料列**——匯入是逐列
讀到檔案結尾，任何非空的示範列都會被當成真實匯報寫進搭乘紀錄。填寫說明改掛在表頭
儲存格的註解上。

## 8. 稽核留痕（`middleware.RecordAuditLog`）

凡是會動到個資或關鍵狀態的操作（新增、修改、reveal PII、更正搭乘紀錄、裁決衝突、匯出、設定變更、匯入）都會呼叫 `RecordAuditLog` 寫一筆 `audit_log`，記錄操作者、角色、動作類型、異動前後的資料快照。`GET /audit` 只有 `admin` 能查，是唯一的稽核紀錄查詢入口。
