---
doc_type: flow
covers:
  - apps/api/internal/modules/driverreport/
  - apps/api/internal/modules/ride/app/ride_service.go
  - apps/api/internal/modules/ride/infra/ride_repo.go
  - apps/api/internal/modules/ops/app/attendance_service.go
  - apps/api/internal/modules/ops/infra/attendance_repo.go
  - apps/api/internal/modules/ops/transport/attendance_handler.go
  - apps/web/src/api/driverReports.ts
  - apps/web/src/api/attendance.ts
  - apps/web/src/views/driverReports/
---

# 司機接送匯報 .xlsx 匯入

一份匯報表對應一台車（`uq_driver_report_forms_vehicle`），管理員上傳司機填好的 `.xlsx`，
系統把每日每個個案的搭乘回報展開成 `ride_records`。逐步敘述與四趟展開、混車合併的細節見
[backend-flows.md 第 1 節](../tech/backend-flows.md)。

## Trigger

- `POST /api/v1/driver-reports/:id/import`，角色限 `staff`、`admin`。
- `dryRun=true`（預設）回傳預覽不寫入；`dryRun=false` 正式寫入。
- `yearMonth`（`YYYY-MM`）選填，宣告這次上傳屬於哪一個月；檔案內落在該月以外的日期會標成錯誤列。
- 前端上傳入口為 `DriverReportImportView.vue`（`/driver-reports/import`，`/driver-reports/batch-import`
  與舊路徑 `/driver-reports/mappings` 皆重導向於此），用頁籤分成「批次上傳」與「待維護資料」：
  - 上傳頁籤採上下堆疊版面（不用彈窗），整塊限寬 1100px 靠左不撐滿頁面：上方是拖放區，下方是
    主色「選擇檔案」按鈕，再往下是重複上傳提示與每個檔案一列的表格。拖曳或選取多個 `.xlsx` 檔案，
    每個檔案是表格中的一列，不需先選月份，也沒有送出按鈕——整批解析完就自動匯入。
  - 每個檔案一加入就自動 dry run（不帶 `yearMonth`）取得預覽，前端由預覽列的 `serviceDate` 推導
    該檔案涵蓋的月份（可能不只一個月），顯示在該列的「涵蓋月份」欄；涵蓋月份已有資料時不做任何
    攔截，解析完就直接匯入——逐列比對本來就不覆蓋，值不同的會進待維護等裁決。
  - 匯入結果由 `importSummary.ts` 的 `describeImportResult` 把後端回傳的
    `importedRows`／`rideRecordRows`／`reaffirmedRows`／`pendingConflictRows`／`backfilledRows`
    組成一行說明。重傳同一份檔案時 `rideRecordRows` 為 0、`reaffirmedRows` 有數字，畫面顯示
    「內容與既有資料完全相同」，不能顯示成「沒有可寫入的搭乘資料」。
  - 欄位對應不再要求使用者逐欄確認才能匯入：有系統推薦個案的欄位自動視為已對應，完全比對不到
    個案的欄位維持 `pending`，兩者都直接跟著這次 commit 一起送出。
  - commit 時針對推導出的每個月份各自呼叫一次、各帶對應的 `yearMonth`；後端逐列比對既有資料，
    不再整月覆蓋。
  - 匯入完成後若有欄位進入待維護、或有資料與既有資料衝突，跳出確認視窗詢問是否立即前往待維護
    頁籤（同樣比照個案管理匯入完成後的提示模式），選「稍後再說」則留在上傳頁籤查看結果。
  - 待維護頁籤：`GET /driver-reports/submissions/review` 以「匯報表列」（一筆 `form_submissions`）為
    單位彙整目前尚待處理的問題——同一列可能同時有個案欄位比對不到（`caseIssues`）與駕駛人比對不到
    （`driverIssue`）。主表格每列只顯示問題總數，展開才看到每一項具體問題與操作：
    - 個案欄位：可連結既有個案（`PATCH /driver-reports/columns/:id/mapping`）或用
      `CaseCreateDialog.vue`（跟個案清單頁「新增個案基本資料」共用同一個元件與 `POST /cases`）
      建立新個案並直接綁定；新建個案後另外呼叫 `GET /driver-reports/columns/name-matches?name=`
      掃描目前其他待維護欄位裡姓名相符（含近似，沿用 `namenorm.ScoreCandidate`）的項目，詢問
      使用者是否一併連結到同一個案。
    - 駕駛人：可連結既有司機或用 `DriverCreateDialog.vue`（跟司機管理頁「新增司機」共用同一個元件
      與 `POST /drivers`）建立新司機，兩種情況都呼叫 `POST /driver-reports/drivers/bind`
      （見下方「司機回填」段落），一次處理掉所有姓名正規化後相符的既有回報，不需要另外掃描其他列。
    - 同一頁籤下方另有獨立的「出勤待維護」區塊（`GET /attendance/conflicts`），列出比對到司機、
      但當天人工出勤登記跟匯入判斷不一致的衝突，可選「保留人工登記」或「改採匯入結果」
      （`POST /attendance/conflicts/:id/resolve`）；這是 ops 模組的資料（`attendance_records`／
      `attendance_import_conflicts`），跟上面個案／司機待維護是兩個獨立資料模型，只是共用同一個
      頁籤呈現。見下方「匯入自動同步出勤」段落。
- `DriverReportStatusView.vue`（`/driver-reports/status`，`/driver-reports` 重導向於此）是唯讀總覽，
  只顯示每台車已有資料的月份與天數，不提供任何上傳或編輯動作。
- `GET /api/v1/driver-reports/imported-months` 供總覽頁與匯入頁判斷某台車某個月是否已有資料；
  月份由 `form_submissions.service_date` 分組推得，不落地成欄位。
- `GET /api/v1/driver-reports/:id/months/:yearMonth` 供總覽頁鑽取單一月份：點月份標籤（表格欄或
  展開列皆可）開啟彈窗，以「逐日回報明細」（`form_submissions` 原始 payload）與「逐個案搭乘紀錄」
  （`ride_sources` 展開後的結果，含個案、司機姓名）兩個頁籤呈現，不需重新開啟原始檔案；`yearMonth`
  需符合 `YYYY-MM`，格式不符直接回 400。

## Steps

```
上傳 .xlsx
  > ParseDriverReport（認表頭、欄位對照、逐列解析民國日期與司機）
  > 宣告月份時檢查所有有效列都落在該月，不符即整份拒絕
  > [dryRun] 回傳預覽，使用者就地確認未對應欄位
  > CommitDriverReport（宣告月份時先確認預覽 `CanCommit`；以下寫入全部在同一個 pgxdb.TxRunner 交易內）
      > persistColumnDecisions 寫回 form_columns（以表頭文字為鍵），
        並收集這次真正 pending -> mapped 的欄位供稍後回填
      > collectImportableRows 挑出可寫入的列，其餘記入 SkippedRows
      > 逐列 ride.IngestSubmission（不再先清除既有資料，每次上傳是獨立事件）
          > SaveFormSubmission：一車一天一筆，同一天原地更新 payload／submitted_at，
            driver_id 用 COALESCE 保留既有值，不被這次沒解析出的司機蓋成 NULL
          > 司機比對不到司機主檔時，這一列完全不展開成搭乘來源（留在 payload 待補綁定）
          > 逐欄呼叫 reconcileRideSource（見下方「同車同個案逐列比對」）
      > 逐個剛完成對應的欄位呼叫 ride.BackfillColumn 補寫先前月份，計入 backfilledRows
        （必須排在上面的逐列寫入之後，理由見下方「匯入時的欄位回填」）
      > MarkImported
  > writeImportAudit（交易外，失敗只記錄不推翻匯入）
```

匯入不再有檔案層級的重複判斷：`driver_report_imports` 與 `ClaimDriverReportImport` 已於
migration `000036` 移除，冪等單獨由 `reconcileRideSource` 保證。`LockDriverReportImport`
（advisory lock）與前端 `commitMonthOnce` 的本地去重都保留，防的是併發與重複送出。

模組交界在 `RideIngestor` port，由 `cmd/server/module_adapters.go` 的
`driverReportRideIngestor` 銜接 driverreport 與 ride。

### 同車同個案逐列比對（reconcileRideSource）

`IngestSubmission`、`BackfillColumn`、`BackfillDriver` 三個進入點都透過同一個
`RideService.reconcileRideSource` 決定每一格要直接寫入、視為無變化的重複回報，還是進待維護；
設計理由見 [driver-report-import-overwrite.md](../decisions/driver-report-import-overwrite.md)。

```
reconcileRideSource(vehicleId, caseId, serviceDate, legSeq, reported, driverId, ...)
  > ListRideSourcesForSlot 取這個 slot 全部來源，找這台車目前最新的一筆
  > 沒有 > InsertRideSource + recalculateRideRecord（直接寫入）
  > 有，且 reported 與 driver_id 都相同 > 不動作（重複回報）
  > 有，且任一不同 > UpsertRideSourceRowConflict 暫存衝突，既有來源不動
      > 同一 slot 同時只保留一筆未解決的衝突（partial unique index），
        再次衝突只更新新值，previous_* 保持不變
```

待維護欄位補綁定的回填走另一條較短的路徑，一樣經過 `reconcileRideSource`：

```
PATCH /driver-reports/columns/:id/mapping（status=mapped）
  > DriverReportService.UpdateColumnMapping（同一個 pgxdb.TxRunner 交易內）
      > FormStore.UpdateColumnMappingByID（RETURNING 更新前狀態、form_id、column_header、column_index）
      > 只有「更新前不是 mapped」才繼續；重複對已是 mapped 的欄位送出不會再回填
      > RideIngestor.BackfillColumn
          > ListSubmissionAnswersForColumn 讀 form_submissions.payload->'answers'->>header，
            含 driver_id：司機仍待維護的答案跳過，等司機也綁定後才由 BackfillDriver 補寫
          > 逐筆 reconcileRideSource
  > 回應帶回 backfilledRows
```

待維護資料頁籤的「匯報表列」清單彙整走一條讀取路徑，不寫入任何狀態：

```
GET /driver-reports/submissions/review
  > DriverReportService.ListSubmissionReview
      > FormStore.ListColumnsWithMapping(formId="", mappingStatus="pending") 依 formId 分組
      > RideIngestor.ListSubmissionsForForms(這些 formId)
          > 讀 form_submissions.payload->'answers' 完整 map，逐列比對每個 pending 欄位
            是否用 merge.ParseReportedValue 判斷為「有回報」，有就算這一列的一個 caseIssue
      > RideIngestor.ListUnmatchedDriverSubmissions()
          > 讀 form_submissions WHERE driver_id IS NULL AND driver_name_raw <> ''，
            合併進同一個 submissionId 的 driverIssue
      > RideIngestor.ListRowConflicts()
          > 讀 ride_source_row_conflicts WHERE resolved_at IS NULL，
            合併進 new_submission_id 對應的同一筆匯報表列
      > 三者皆空的列不列入清單
```

司機回填跟欄位回填分屬不同模組（`form_submissions.driver_id` 由 `ride` 模組持有），比對邏輯
是精確正規化相等（沿用既有 `namenorm.Normalize`，不像個案有分數式模糊比對）：

```
POST /driver-reports/drivers/bind { driverNameRaw, driverId }
  > DriverReportService.BindPendingDriver
      > RideIngestor.BackfillDriver
          > namenorm.Normalize(driverNameRaw) 取正規化姓名
          > ListUnmatchedDriverSubmissions() 撈出所有 driver_id IS NULL 的既有回報，
            含 vehicle_id、submitted_at 與完整原始答案，在應用層過濾出正規化姓名相符的
            （天然涵蓋「其他待維護列同一人」的狀況，不需要額外的模糊比對或使用者確認）
          > UpdateSubmissionDriverID 回填提交紀錄的司機
          > 讀該表單已對應的欄位，逐欄從這筆提交的原始答案呼叫 reconcileRideSource——
            司機比對不到時原本就沒有展開成搭乘來源，回填是從頭比對寫入，不是更新既有來源
  > 回應帶回 affectedCount（實際回填的提交筆數）
```

同車同個案衝突的裁決：

```
POST /driver-reports/row-conflicts/:id/resolve { useNew }
  > DriverReportService.ResolveRowConflict（同一個 pgxdb.TxRunner 交易內）
      > RideIngestor.ResolveRowConflict
          > 條件式 UPDATE 標記已解決（resolved_at IS NULL 才會成功，避免覆寫他人已做的裁決）
          > useNew=true > 重放 InsertRideSource + recalculateRideRecord 寫入新值；
            useNew=false > 只標記已解決，既有來源與搭乘紀錄不動
      > useNew 套用的新值有司機時，同一交易內呼叫 AttendanceRegistrar.SyncFromImport
        同步出勤，比照初次匯入與司機補綁定的既有流程
  > writeRowConflictResolutionAudit（交易外，失敗只記錄不推翻裁決結果）
  > 回應 { success: true }
```

匯入時比對到司機的列，`CommitDriverReport` 逐列在寫入搭乘紀錄後接著同步該司機當天的出勤：

```
CommitDriverReport（逐列，緊接在 IngestSubmission 之後、同一個交易內）
  > 這一列比對到司機（driverId 有值）才觸發，否則交給既有的司機待維護流程
  > AttendanceRegistrar.SyncFromImport（cmd/server 的 driverReportAttendanceRegistrar 轉呼叫
    AttendanceService.SyncFromImport）
      > AttendanceStore.GetOne 查當天既有出勤紀錄
      > 沒有紀錄，或既有紀錄本身就是上次匯入寫入的（source=import）
          > Upsert(status=work, source=import)
      > 既有人工登記（source=manual）且狀態剛好也是出勤 > 不動作
      > 既有人工登記且狀態不同 > UpsertConflict 記一筆待維護（不覆蓋人工判斷）
```

```
GET /attendance/conflicts > AttendanceService.ListConflicts（只回傳 status=pending）
POST /attendance/conflicts/:id/resolve { choice }
  > AttendanceService.ResolveConflict
      > choice=use_import 時先 Upsert(status=importedStatus, source=import) 覆蓋人工登記
      > 兩種 choice 都呼叫 AttendanceStore.ResolveConflict 標記 status=resolved
      > 稽核寫入 attendance_import_conflicts 的 resolve 動作
```

`UpsertConflict` 用 `ON CONFLICT (driver_id, record_date) DO UPDATE` 一次處理「已有待處理衝突」
與「重新開啟已解決衝突」兩種情況：已解決且 `resolved_choice = keep_manual`、既有人工狀態跟上次
解決時完全相同，維持 `resolved`（重匯同一批資料不會反覆打擾使用者）；除此之外一律變回
`pending`（含尚未處理，或人工狀態在上次解決後又被改過）。

## Failure modes

- **重複匯入**：每次上傳是獨立事件，逐列比對既有資料——值相同視為重複回報，不重新寫入；值不同
  暫存衝突讓使用者選擇；未被本次上傳觸及的既有資料完全不受影響。沒有檔案層級的重複判斷，重傳
  同一份檔案照樣完整跑一次比對，結果會是 `rideRecordRows: 0` 但 `reaffirmedRows` 有數字。
  決策與替代方案見
  [driver-report-import-overwrite.md](../decisions/driver-report-import-overwrite.md)。
- **裁決後重傳**：使用者對某筆衝突裁決「保留原資料」後，重傳同一份（與既有資料仍不同的）檔案
  會**重新產生一筆未解決衝突**，因為 `uq_ride_source_row_conflict_open` 只涵蓋
  `resolved_at IS NULL`。這是刻意行為：每次上傳都重新判斷，不讓比對不上的資料被系統自行吞掉。
- **解析層級失敗**：未宣告月份時，日期打錯的列逐列略過並記入 `SkippedRows`；宣告整月時，日期無法解析是
  blocking error，整份拒絕不寫入任何資料。
- **資料庫層級失敗**：整份回滾，`last_imported_at` 不更新。
- **月份不符**：宣告 `yearMonth` 後，檔案內落在該月以外的有效日期僅該列標記為錯誤、記入
  `SkippedRows`，不中斷整份解析，其餘列照常產生預覽並可正常寫入；commit 時這些列一併略過。
  上傳頁針對每個自動推導出的月份各自宣告一次 `yearMonth`，因此「同一份檔案橫跨多個月份」是預期
  情境，不屬於此列表示的月份不符——這些列會在其所屬月份的那一輪 commit 正常匯入，提示訊息只說明
  「這一輪略過、另行處理」，不是要求使用者重新確認上傳檔案。
- **空檔**：沒有任何可寫入的列時整份 commit 直接視為未匯入，不更新 `last_imported_at`。
- **檔案格式與規模**：API 只接受 `.xlsx`；共用 reader 會先檢查 XLSX ZIP 項目數、解壓後總量、worksheet XML 大小與壓縮倍率，超過限制時在 parser 前拒絕。
- **混車**：跨車衝突（同一 slot 有 ≥2 台不同車輛都回報「有坐」）由 `merge.MergeRideSources` 判斷，
  走既有的 `ride_records.has_conflict`／`RideIssuesView.vue`「混車衝突待裁決」流程，與這次新增的
  同車同個案衝突是兩個獨立機制，互不影響彼此的判斷。
- **同車同個案衝突**：值不同時不寫入 `ride_sources`，暫存進 `ride_source_row_conflicts` 待使用者
  裁決；裁決前既有搭乘紀錄與月曆顯示維持既有值不動。同一 slot 同時只有一筆未解決的衝突，再次上傳
  到同一個未解決的衝突只更新其新值。
- **人工成果**：帶 `corrected_at`、`conflict_resolved_at` 或 `not_claimed_aa09` 的 `ride_records`
  只會被人工更正／裁決／不申報標記改變，不會被匯入或補綁定流程覆蓋。
- **稽核寫入失敗**：只記 server log，不推翻已完成的匯入或裁決。
- **出勤同步失敗**：`AttendanceRegistrar.SyncFromImport` 回傳錯誤會讓整筆匯入或裁決回滾（跟
  `IngestSubmission`／`ResolveRowConflict` 失敗同一等級），不會出現「搭乘紀錄寫成功、出勤沒同步」
  的半套結果。

### 上傳頁的逐檔失敗

逐檔各自請求，單一檔案失敗只停在那一列：建表失敗、表頭不符、車輛比對失敗都只讓該檔標記失敗，
其他檔案照常試算與匯入。並發處理維護上限（3），避免一次拖入十幾個檔案時打出過多同時請求。

### 情境資料為空清單時的前端防呆與提示

`vehicles`／`forms`／`importedMonths`／`cases` 進入頁面前一次載入的四份清單，一律預設空陣列：分頁端點
（`GET /vehicles`、`GET /cases`）在 0 筆結果時後端回傳 `data: null`（nil slice 序列化行為），若前端直接
指派而不做 `?? []`，後續 `detectVehicle` 等處對 `null` 呼叫 `.filter`／`.map` 會丟出未捕捉例外，發生在
選檔案加入列表之前，使用者會看到選了檔案卻毫無反應。載入這四份資料本身失敗時顯示非技術性錯誤訊息
並讓拖放區維持停用；資料載入成功但車輛清單為空時，另外顯示「尚未建立車輛」提示並附前往車輛管理的
捷徑，不讓使用者對著一個選不出任何選項的下拉選單卡住。

### 欄位自動對應與待維護

commit 前不再要求使用者逐欄確認：有系統推薦個案（`suggestedCaseId`/`suggestedLegSeq`）的欄位
直接視為 `mapped` 送出；完全沒有推薦的欄位維持 `pending`，寫入 `form_columns` 供待維護頁籤查詢。
`form_submissions.payload.answers` 一律保存這一列「所有」欄位的原始儲存格文字，不論該欄當時
是否已對應個案；只有「欄位已 `mapped` 且司機已比對到司機主檔」兩個條件同時成立，才會在當次
commit 展開成 `ride_sources`／`ride_records`——任一條件不成立時，這一列只出現在待維護頁籤，
不會出現在司機日曆等其他頁面，直到使用者完成對應或綁定。匯入只會使用本次檔案出現的 mapped 欄位，
不能沿用舊檔已對應、但本次未出現的欄位。

尚未對應個案的欄位不會因此卡住：待維護頁籤把某欄從 `pending` 改成 `mapped`
（`PATCH /driver-reports/columns/:id/mapping`）時，`DriverReportService.UpdateColumnMapping`
會在同一個交易內呼叫 `RideService.BackfillColumn`，直接讀取這個表單既有
`form_submissions.payload.answers` 裡這一欄留下的原始文字，逐筆展開成 `ride_sources` 並重算
`ride_records`——不需要使用者重新上傳原始檔案，回應會帶回本次實際補寫的筆數
（`backfilledRows`）。只有「這一次是從非 mapped 變成 mapped」才會觸發回填，重複對已經是
mapped 的欄位送出同樣的更新不會再次回填，避免疊加出重複的搭乘來源。

### 匯入時的欄位回填

匯入路徑走的是同一套觸發條件。某個欄位先前比對不到個案而留在 `pending`，之後個案建好了、
下一次上傳時 `bestCaseMatch` 比對到並自動送出 `mapped`，`persistColumnDecisions` 會從
`UpdateColumnMappingByHeader` 回傳的更新前狀態判斷這是「這次才從 `pending` 變 `mapped`」，
收集起來在逐列寫入後呼叫 `BackfillColumn`，把先前月份留在 `form_submissions.payload` 的原始值
補寫進去，筆數回傳為 `backfilledRows`。

少了這一步會靜默掉資料：待維護清單是 `mapping_status = 'pending'` 與 payload 的交叉查詢，
欄位一旦變 `mapped` 就查不到，先前月份的值還在 payload 裡卻再也沒有任何入口撈得出來——待維護
不顯示、匯入不回頭補、手動綁定也因為 `previousStatus` 已是 `mapped` 而不觸發。

這條路徑有兩個一起才成立的保護，少任何一個都會製造假衝突：

1. **`BackfillColumn` 必須排在逐列 `IngestSubmission` 之後**：`SaveFormSubmission` 以
   `(form_id, service_date)` 原地更新，先回填會讀到本次涵蓋日期更新前的舊答案並寫入，接著本次的
   新值再比對一次，就會憑空產生一筆使用者其實沒遇到的衝突。排在之後，本次月份各天都走
   Reaffirmed 跳過。由 `TestCommitDriverReport_BackfillRunsAfterIngest` 鎖住。
2. **回填要排除這份檔案涵蓋的所有服務日期，不只本次宣告的月份**：`ListSubmissionAnswersForColumn`
   沒有日期條件，回填範圍是整份表單的全部歷史；而跨月檔案是逐月各送一次 commit，先 commit 的
   那個月執行回填時，其他月份的 payload 還是上一次上傳的舊值，補進去就會在下一輪被本次新值比出
   一批假衝突。因此 `commit.go` 用 `collectFileServiceDates` 從 `preview.PreviewRows` 取出檔案裡
   每一列的服務日期（含被標成「不屬於宣告月份」的列），整批當作 `BackfillColumn` 的 `skipDates`
   傳入——這些日期的權威值是使用者手上這份檔案，會在各自月份的 commit 正常寫入。由
   `TestCommitDriverReport_BackfillSkipsEveryDateInTheFile` 與
   `TestBackfillColumn_SkipsDatesOwnedByTheCaller` 鎖住。

待維護頁的手動綁定沒有這個問題，`skipDates` 傳 `nil`：那時沒有「另有來源」的日期，該欄位留下的
既有回報全部都要補寫。

## Unverified

- `ListSubmissionAnswersForColumn` 的 `payload->'answers'->>$2` 與 `payload->'answers' ? $2`
  JSONB 查詢只以 app 層的 fake 覆蓋，未在真實 PostgreSQL 上驗證 payload 實際落地格式與這兩個
  運算子的行為是否一致。
- 交易回滾行為只以 fake `TxRunner` 的單元測試覆蓋，未在真實 PostgreSQL 上驗證回滾與
  `ON DELETE CASCADE` 的實際連帶效果。`caseimport` 有 `commit_integration_test.go` 的前例
  （`//go:build integration` 搭配 `DATABASE_URL`），本次未比照建立。
- `pgxdb.Querier` 新增的 `SendBatch` 在真實交易內的批次寫入行為未經 runtime 驗證。
- 同一份匯報表併發重匯的鎖競爭行為未驗證，現行的 fake 也覆蓋不到。批次頁的並發上限只限制
  單一瀏覽器分頁，兩位管理員同時對同一台車同一個月匯入仍會競爭。
- `imported-months` 的 SQL 分組（`to_char(service_date, 'YYYY-MM')` 與 `source = 'import'` 篩選）
  只以 app 層的 fake 與 handler 測試覆蓋，未在真實 PostgreSQL 上驗證。
- 月份鑽取的兩支查詢（`ListSubmissionsForFormMonth` 的 `payload->'answers'` 解析、
  `ListRideEntriesForFormMonth` 的 `ride_sources` 與 `cases`／`drivers` 兩個 LEFT JOIN）只以 app 層
  的 fake 與 handler 測試覆蓋，未在真實 PostgreSQL 上驗證 JOIN 結果與空值處理。
- 待維護資料頁籤的三支新查詢（`ListSubmissionsForForms` 的 `payload->'answers'` 全量解析、
  `ListUnmatchedDriverSubmissions` 與 `driver_report_forms`／`vehicles` 的 LEFT JOIN、
  `BackfillDriver` 在應用層對所有 `driver_id IS NULL` 列做全表掃描比對正規化姓名）只以 app 層的
  fake 覆蓋，未在真實 PostgreSQL 上驗證，也未驗證「未比對司機的回報量變大後」全表掃描的效能。
- 前端「新增個案／司機並綁定」與「掃描其他待維護項目詢問是否一併連結」的完整互動流程
  （`DriverReportImportView.vue` 的 `promptRelatedCaseIssues`、`CaseCreateDialog.vue`／
  `DriverCreateDialog.vue`）只驗證了 `type-check`／`build`，未在瀏覽器對真實後端資料實測。
- 出勤自動同步與待維護衝突（`AttendanceService.SyncFromImport`／`UpsertConflict`／
  `ResolveConflict`）的四個分支只以 app 層 fake（`recordingAttendanceStore`）與
  `driverreport` 端的 `fakeAttendanceRegistrar` 驗證；`ON CONFLICT ... DO UPDATE` 的「已解決
  且人工狀態未變時維持已解決」CASE 邏輯未在真實 PostgreSQL 上以實際資料驗證。前端「出勤待維護」
  區塊只驗證了 `type-check`／`build`，以及對空清單（無司機、無出勤資料）情境下呼叫真實
  `GET /attendance/conflicts` 成功回應、無主控台錯誤，未實際造出一筆衝突並在瀏覽器完成
  「保留人工登記」／「改採匯入結果」兩種解決路徑的操作。
- 同車同個案逐列比對（`reconcileRideSource`、`ride_source_row_conflicts` 的
  `ON CONFLICT (vehicle_id, case_id, service_date, leg_seq) WHERE resolved_at IS NULL`
  partial unique index upsert、`ResolveRowConflict` 的條件式 `UPDATE ... WHERE resolved_at
  IS NULL`）只以 app 層 fake 覆蓋，未在真實 PostgreSQL 上驗證 partial unique index 的
  upsert 語意與併發裁決時的鎖行為。
- `form_submissions` 唯一鍵從 `(form_id, service_date, submitted_at)` 改成
  `(form_id, service_date)`（migration `000034`）的既有重複資料清理，只在 SQL 本身做了
  window function 去重，未在代表既有資料規模的 staging／production 資料庫上實際跑過這支
  migration 並驗證清理結果與效能。
- `ride_sources` 新增的 `submitted_at` 欄位（migration `000034`）從既有
  `form_submissions.submitted_at` 回填，只驗證了 SQL 語法本身，未在真實資料上確認回填後的
  混車合併排序結果與遷移前一致。
- 同一台車同一 slot 的併發上傳／裁決競爭（例如兩個瀏覽器分頁同時對同一天送出不同的值、或
  裁決同一筆衝突的同時又有新的上傳進來）未經真實資料庫的鎖與交易隔離層級驗證，現行的 fake
  也覆蓋不到。
- 前端待維護頁籤新增的「同車同個案衝突」子區塊（`DriverReportImportView.vue` 的
  `handleResolveRowConflict`）只驗證了 `type-check`／`build`，未在瀏覽器對真實後端資料造出
  一筆衝突並完成「採用新資料」／「保留原資料」兩種解決路徑的操作。

## 資料一致性防護規則

這些規則是匯入／匯出流程的長期契約，不能只依賴目前 Excel 欄位順序或 fake 測試：

- 每筆來源資料必須保留 raw headers／values 與不可變的 row identity。不可只用 `時間戳記` 作為 `source_key`；若來源沒有 immutable ID，遇到相同 timestamp 的多列必須明確回報衝突，不可靜默覆寫。
- 歷史資料匯出必須依該筆保存的欄位識別與 mapping version 解讀，不可用現在的 `form_columns` 位置回頭解讀舊 payload。
- 未知欄位可略過並設為 `null`，但 schema、連線、權限、migration 或其他 infrastructure error 不得偽裝成可接受的 unmatched；該列應 rollback 或明確標示不可寫入。
- 所有會寫入相同 ride slot 的 writer（import、webhook、manual correction 及其他來源）必須共用同一個 transaction + slot lock API。多 slot 操作要先收集、去重，再按 `(case_id, service_date, leg_seq)` 排序鎖定，必要時 retry deadlock。
- `form_columns` 的完整 metadata 應原子更新或版本化；transaction rollback 後的 counters、audit 與 mapping 狀態不得留下半套結果。
- 無時區來源時間必須明確套用來源時區（目前預期為 `Asia/Taipei`），並在 staging／production 以實際資料驗證。

## 驗證分層

```text
unit / frontend E2E
              !=
real PostgreSQL migration + transaction + concurrency
              !=
production import/export observation
```

回報時必須分開列出每一層證據。`go test`、type-check、build 或前端 E2E 通過，不代表 migration、rollback、duplicate identity 或跨程序 lock 已在真實 PostgreSQL 驗證。
