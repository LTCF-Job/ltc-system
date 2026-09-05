# 後端核心業務流程

這份文件拆解幾條橫跨多個 handler／service／domain 套件的完整流程，比單看某一個檔案更容易看懂系統在幹嘛。分層架構背景見 [backend-framework.md](backend-framework.md)，逐支端點清單見 [backend-api-reference.md](backend-api-reference.md)。

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
   依明確白名單判斷「有坐／有搭乘」或「沒坐／沒有坐／未搭乘／沒有搭乘」；其他文字（含空白）視為未回報直接跳過。
5. 若個案排班是四趟制，匯報表上的「第 1 趟」「第 2 趟」要展開成資料庫的四趟（1→1,3；2→2,4），
   這是四趟展開規則的實作位置（`expandLegSeqs`）。
6. 每個展開後的趟次都會 `InsertRideSource` 存一筆來源紀錄，然後呼叫 `recalculateRideRecord`
   讀回該 slot 的全部來源列跑 `domain/merge` 演算法，用「同車取最新、跨車 OR」規則重算
   `ride_records` 的最終狀態——已經被人工更正過的紀錄不會被自動覆蓋。

未指定 `yearMonth` 時，單列日期無法解析只略過該列並回報原因，其餘日期照常寫入；指定
`yearMonth` 代表整月覆蓋，任何阻斷性解析錯誤都會在清除前拒絕整份匯入，避免舊資料被清掉。

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
- 出勤月報只讀取資料庫中所有未刪除且啟用的司機；司機清單或假日資料查詢失敗時直接回傳錯誤，
  不使用假資料，也不以固定筆數上限截斷結果。

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

- `ListMissingReports(ctx, targetDate, region)`：拿 `domain/calendar.CalculateExpectedRides` 算出「這天應該有的搭乘」，跟實際 `ride_records` 比對，供前端「未回報清單」頁查詢，純查詢不觸發通知。
- `CheckMissingReports(ctx, targetDate, region)`：同樣比對應搭與實際回報，但只由明確的後台任務入口呼叫，並觸發告警通知。
- `MonthEndReminder(ctx, year, month)`：每月 26 日跑，彙整檢核結果並發信提醒。
- 這兩支都各自有 `POST /tasks/*` 端點，正式環境由 Cloud Scheduler 定期打；本機要測試就直接手動 curl 這兩支。

## 4. 政府申報匯出（`PrecheckService` + `GovClaimService`）

一個個案一個月產一份 `.xlsx`，欄位比照政府範本的 33 欄（`domain/govform.Headers33`，工作表名「工作表1」）。

1. 使用者在「政府申報匯出」頁選申報年月、申報地區、申報個案（可多選）與匯出檔案模式（直接下載／壓縮檔）。
2. `GET/POST /exports/precheck` 以年月、地區與選取的 `caseIds` 建立 `ClaimScope` 後跑 `PrecheckService.RunPrecheck`，回傳 `PrecheckReport`；資料庫查詢失敗或有任何 error（包含未裁決混車衝突）就擋住匯出，前端列出 issue 讓使用者回去修。
   `POST /exports` 會以同一組年月、地區與 `caseIds` 建立 scope 並查詢申報來源，避免前置檢核與實際匯出檢查不同資料集。
3. `POST /exports` 交給 `GovClaimService.CreateGovClaimJob` 同步產檔（專案沒有背景 worker）：
   - `GovClaimRepository.QueryGovClaimSources` 一次撈齊該月 `effective_status = 'boarded'` 且沒有未裁決衝突的趟次，join `cases`／`case_schedules`／`schedule_legs`／`sites`／`vehicles`／`drivers`。
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

個案與照護人員匯入目前只接受 `.xlsx`。每筆預覽列都產生 `rowId = sheetName:rowIndex`；正式 commit 的 duplicate selection 使用 `rowId`，`rowIndex` 僅供畫面顯示，避免多工作表同列號碰撞。Excel reader 會在 parser 前檢查 ZIP 項目數、解壓總量、worksheet XML 大小與壓縮倍率，超過限制即拒絕。

## 6. 通知（`NotificationService`）

`SendNotification` 依 topic（例如未回報告警、月底提醒）撈出啟用中的收件人清單逐一寄信，寄送介面是 `EmailSender`，正式環境用 Resend（`RESEND_API_KEY`），本機未設定時才使用 `LogEmailSender`（只印 log，不真的寄信）。`SendResult` 會區分實際成功與失敗數量；任一 provider failure 會回傳 error，不再記成永遠成功。收件人管理走 `settings/notification-recipients` 系列端點。

`GET /rides/missing` 只呼叫 `TaskService.ListMissingReports` 查詢資料，不會觸發通知；只有明確執行檢核的 `POST /tasks/check-missing-reports` 才呼叫 `CheckMissingReports` 並派送告警。

個案改名會在同一個 use case 內同步更新 `name` 與 `name_normalized`；明確提交空白姓名會被拒絕。

## 7. 匯報表範本下載

`GET /driver-reports/:id/template` 依該車目前已對應的個案趟次欄產生空白 `.xlsx`：
表頭為「民國日期、駕駛人、各個案趟次欄、備註」，**刻意不放示範資料列**——匯入是逐列
讀到檔案結尾，任何非空的示範列都會被當成真實匯報寫進搭乘紀錄。填寫說明改掛在表頭
儲存格的註解上。

## 8. 稽核留痕（`middleware.RecordAuditLog`）

凡是會動到個資或關鍵狀態的操作（新增、修改、reveal PII、更正搭乘紀錄、裁決衝突、匯出、設定變更、匯入）都會呼叫 `RecordAuditLog` 寫一筆 `audit_log`，記錄操作者、角色、動作類型、異動前後的資料快照。`reveal_pii` 必須先成功寫入 durable audit，audit DB 不可用時回 503，不能先解密再 best-effort 留痕；匯入略過列的 audit 會遮罩身分證、姓名、地址與聯絡資訊。`GET /audit` 只有 `admin` 能查，是唯一的稽核紀錄查詢入口。

報表與查詢的月份解析使用共用 strict parser；非法月份不會 fallback 到目前月份。`/api/livez` 只檢查 process，`/api/readyz` 與相容的 `/api/health` 會在 DB 不可用時回 503。
