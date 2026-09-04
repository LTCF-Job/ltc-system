# 後端核心業務流程

這份文件拆解幾條橫跨多個 handler／service／domain 套件的目前流程，比單看某一個檔案更容易看懂系統在幹嘛。分層架構背景見 [backend-framework.md](backend-framework.md)，逐支端點清單見 [backend-api-reference.md](backend-api-reference.md)。流程中的「靜態已確認」不代表已用真實 DB／Supabase／外部 provider 執行驗證；目前已知落差集中在 [2026-09-04 Full-stack Review](../reviews/2026-09-04-full-stack-review.md)。

## 1. 司機接送匯報匯入 → 搭乘紀錄（`DriverReportService.CommitDriverReport` → `RideService.IngestSubmission`）

這是整個系統最核心的資料流。一份匯報表對應一台車，操作人員上傳司機填好的 `.xlsx`：

1. `POST /api/v1/driver-reports/:id/import?dryRun=true` 先解析不寫入：`ParseDriverReport` 認表頭
   （民國日期、駕駛人、各個案趟次欄、備註），把每個個案欄位與既有 `form_columns` 對照，
   沒對應過的欄位附上以姓名相似度算出的推薦個案與由 `[去程]／[回程]` 推得的推薦趟次。
2. 使用者在預覽畫面就地確認欄位對應後，改打 `?dryRun=false`，並以 form field `columnDecisions`
   帶回確認結果。`CommitDriverReport` 先把欄位對應寫回 `form_columns`（以表頭文字為鍵，
   個案增減造成的欄號位移不會錯配），清掉本次要覆蓋的既有匯入資料，再逐列交給 ride 模組。
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

`CommitDriverReport` 逐列呼叫 `RideService.IngestSubmission` 之後，若這一列比對到司機
（`driver_id` 有值），會在同一個交易內接著呼叫 `AttendanceService.SyncFromImport`（透過
`AttendanceRegistrar` port，adapter 見 `cmd/server/module_adapters.go` 的
`driverReportAttendanceRegistrar`）同步該司機當天的出勤月曆：

- 當天沒有出勤紀錄：直接寫入出勤（`work`），`attendance_records.source = 'import'`。
- 當天既有紀錄本身也是上次匯入寫入的（`source = 'import'`）：直接刷新，不算衝突。
- 當天既有人工登記（`source = 'manual'`）且狀態剛好也是出勤：不需處理。
- 當天既有人工登記且狀態不同：**不覆蓋**，改在 `attendance_import_conflicts` 記一筆待維護
  衝突（`UNIQUE(driver_id, record_date)`），交給司機接送匯報「待維護資料」頁籤的「出勤待維護」
  區塊處理——使用者可選擇 `keep_manual`（保留人工登記，不動 `attendance_records`）或
  `use_import`（改採匯入判斷覆蓋，`source` 一併改回 `import`）。已解決且人工狀態未再變動時
  維持已解決，不會因為重匯同一批資料反覆跳出來打擾使用者。比對不到司機的列走另一條既有的
  「司機待維護」流程（見下方司機回填段落），不會產生出勤衝突。
- `GET /api/v1/attendance/conflicts` 查詢目前待處理清單；
  `POST /api/v1/attendance/conflicts/:id/resolve` 提交使用者的選擇。

### 覆蓋語意與交易邊界

匯入是覆蓋不是疊加：重匯同一份檔案的結果與只匯一次相同。`clearPreviousImport` 在寫入前呼叫
`RideService.ClearImportedDates` 刪掉這份匯報表在本次涵蓋日期的 `form_submissions`，`ride_sources`
由 `submission_id` 的 `ON DELETE CASCADE` 連帶清除；只刪本匯報表的提交，其他車輛對同一 slot
的混車來源不受影響。

清除範圍由選填的 `yearMonth`（`YYYY-MM`）query param 決定：

- 有帶：整個月都被這份檔案覆蓋，且檔案內任一有效列落在該月之外就整份拒絕（dry run 階段就擋）。
- 未帶：只覆蓋檔案實際涵蓋的日期。
- 檔案沒有任何可寫入的列時不執行清除，避免傳錯空檔清空整月資料。

來源被清空的 slot 不能靠重算修正，會由 `DeleteDerivedRideRecord` 刪除；帶有 `corrected_at`、
`conflict_resolved_at` 或 `not_claimed_aa09` 的紀錄一律保留，人工成果不被覆蓋式重匯抹掉。

清除與重寫落在同一個 `pgxdb.TxRunner` 交易內。解析層級的失敗仍逐列略過，但資料庫層級的失敗
會整份回滾，`last_imported_at` 不更新——先刪後寫若中途失敗而不回滾，該月資料會直接消失。
`RideRepository` 與 `DriverReportRepository` 因此都改用 `pgxdb.FromContext` 取用外層交易。

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
ClearImportedDates（刪本表本月 form_submissions，cascade 清 ride_sources）
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

- `CheckMissingReports(ctx, targetDate, region)`：拿 `domain/calendar.CalculateExpectedRides` 算出「這天應該有的搭乘」，跟實際 `ride_records` 比對，抓出應搭但沒回報的趟次，並可能觸發告警通知。
- `MonthEndReminder(ctx, year, month)`：每月 26 日跑，彙整檢核結果並發信提醒。
- 這兩支都各自有 `POST /tasks/*` 端點，正式環境由 Cloud Scheduler 定期打；本機要測試就直接手動 curl 這兩支。

目前另有 `GET /api/v1/rides/missing` 接到 `taskH.GetMissingReports`，前端把它當成可分頁、可篩選的查詢頁使用，但 handler 只解析少數 query，且呼叫的 task path 具有 notification-capable 行為。這是目前的 query／command 邊界缺陷：在修正前不要把該 GET 當成純讀取 API，也不要以頁面顯示結果推論通知已成功送達。

## 4. 政府申報匯出（`PrecheckService` + `GovClaimService`）

一個個案一個月產一份 `.xlsx`，欄位比照政府範本的 33 欄（`domain/govform.Headers33`，工作表名「工作表1」）。

1. 使用者在「政府申報匯出」頁選申報年月、申報地區、申報個案（可多選）與匯出檔案模式（直接下載／壓縮檔）。
2. `GET/POST /exports/precheck` 跑 `PrecheckService.RunPrecheck`，回傳 `PrecheckReport`；有 error 就擋住匯出，前端列出 issue 讓使用者回去修。
3. `POST /exports` 交給 `GovClaimService.CreateGovClaimJob` 同步產檔（專案沒有背景 worker）：
   - `GovClaimRepository.QueryGovClaimSources` 一次撈齊該月 `effective_status = 'boarded'` 的趟次，join `cases`／`case_schedules`／`schedule_legs`／`sites`／`vehicles`／`drivers`。
   - 逐筆驗證後呼叫 `domain/govform.BuildClaimRow` 組出 33 欄，再用 `SortClaimRows` 排成「leg1 整月 → leg2 整月」。缺排班趟次、缺司機、缺出發時間等資料的趟次計入 `skipped` 並回報，不套用預設值硬湊。
   - `ExcelRenderer.RenderGovClaim` 產出每個個案的工作簿位元組；壓縮檔模式再由 `ZipArchiver.BuildZip` 打包。
   - 單一交易寫入 `export_lines`（申報列快照）、`export_job_files`（逐案檔案中繼資料）與 `export_jobs` 狀態。
4. 下載時不從物件儲存讀檔（專案沒有 storage adapter），改由 `export_lines.raw_payload` 快照重繪。快照的第 1 欄與第 7 欄（個案／服務人員身分證）一律留空，只存 `driverId`，重繪時才由密文解密補回，明文身分證不落資料庫。
5. `GET /exports/:id/files/:caseId/download` 取單一個案的 `.xlsx`；`GET /exports/:id/download` 只服務壓縮檔模式的工作。歷史紀錄頁不提供下載，只能用 `GET /exports/:id` 查看該次匯出包含哪些個案。

```
使用者選年月/地區/個案/模式
   │
   ▼
GET/POST /exports/precheck ──未通過──► 前端列出 issue，回去修資料
   │ 通過
   ▼
POST /exports（同步產檔）
   │  QueryGovClaimSources → BuildClaimRow → SortClaimRows → RenderGovClaim（→ BuildZip）
   │  單一交易寫入 export_lines + export_job_files + export_jobs
   ▼
逐案下載 GET /exports/:id/files/:caseId/download（由快照重繪）
或整包下載 GET /exports/:id/download（僅壓縮檔模式）
```

目前系統沒有資料、一律留白的欄位：備註、服務人員身分證 2-5、訪視未遇、C 碼五欄、OT01 餐別、出發地／目的地經緯度。里程數目前取自排班層級的 `case_schedules.distance_km`，去回程會是同一個數字。

## 5. 主檔批次匯入（`ImportService`）

`POST /cases/import`（或相容路徑 `/masters/import`）吃使用者上傳的 Excel，`ImportService.ParseCasesFromExcel` 逐列解析、驗證欄位（含 `ParseWeekdays` 解析「每週單位開放時間」這種自由文字格式），回傳每列的解析結果與統計，成功的列才會實際寫入個案主檔；另外 `ParseScheduleWorkbook` 專門解析「(參考用) 交通車接送班表」這份既有 Excel，抽出單位跟司機資訊。

## 6. 通知（`NotificationService`）

`SendNotification` 依 topic（例如未回報告警、月底提醒）撈出啟用中的收件人清單逐一呼叫 `EmailSender`，收件人管理走 `settings/notification-recipients` 系列端點。目前 `cmd/server/main.go` 傳入 nil sender，service 會使用 `LogEmailSender`，只印 simulated email log；`RESEND_API_KEY` 與 `NOTIFY_FROM` 雖存在設定中，尚未接上可證明 delivery 的 Resend adapter。完成 adapter、provider credentials、retry／delivery status 與 runtime check 前，不可把通知成功說成 email 已送達。

## 7. 匯報表範本下載

`GET /driver-reports/:id/template` 依該車目前已對應的個案趟次欄產生空白 `.xlsx`：
表頭為「民國日期、駕駛人、各個案趟次欄、備註」，**刻意不放示範資料列**——匯入是逐列
讀到檔案結尾，任何非空的示範列都會被當成真實匯報寫進搭乘紀錄。填寫說明改掛在表頭
儲存格的註解上。

## 8. 稽核留痕（`middleware.RecordAuditLog`）

設計上，會動到個資或關鍵狀態的操作（新增、修改、reveal PII、更正搭乘紀錄、裁決衝突、匯出、設定變更、匯入）應呼叫 audit writer，記錄操作者、角色、動作類型與異動前後快照。靜態 review 發現部分 service 沒有注入 audit、部分 write error 被忽略，且 change-password 的 actor role 可能為空，因此不能把這段設計描述當成所有 mutation 都已留下可靠 audit。`GET /audit` 是現行稽核查詢 route，但完整 coverage 仍待逐 route runtime／DB 驗證。

## Failure modes

- 匯入解析錯誤可逐列略過；但 migration、transaction、DB write error 應使整份 operation 回滾，不能混成成功結果。
- 缺漏檢查若被 GET 呼叫，會把查詢失敗與通知副作用混在同一個 user action；在 command/query 拆分前應視為高風險流程。
- export query 若部分 vehicle／case 查詢失敗，不能只回傳剩餘資料並標示成功；應明確回報 partial／failed state。
- notification sender 使用 LogEmailSender 時，log 只代表 application 呼叫 sender，不代表外部信件 delivery。
- local 沒有 DB 時，部分 module 可能回空資料或假成功，部分 repository 可能失敗；offline 啟動不代表本文件的 DB 流程已驗證。

## Unverified

- 真實 PostgreSQL 上 migration、seed、transaction rollback、soft-delete scope、lock 與 row-level error 行為。
- 真實 Supabase Auth／Admin API 的 JWT role metadata、user list pagination、password change、permission cache 與 logout 行為。
- Cloud Scheduler 的 retry／duplicate trigger、notification delivery、政府 holiday provider 與正式 export download。
- 大資料量下的 report／missing query latency、N+1、前端 stale response 與 timezone 邊界。
