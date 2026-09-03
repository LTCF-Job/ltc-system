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
- `yearMonth`（`YYYY-MM`）選填，宣告這次要覆蓋哪一個月。
- 前端上傳入口為 `DriverReportImportView.vue`（`/driver-reports/import`，`/driver-reports/batch-import`
  與舊路徑 `/driver-reports/mappings` 皆重導向於此），用頁籤分成「批次上傳」與「待維護資料」：
  - 上傳頁籤採上下堆疊版面（不用彈窗），整塊限寬 1100px 靠左不撐滿頁面：上方是拖放區，下方是
    主色「選擇檔案」按鈕，再往下是覆蓋警示與每個檔案一列的表格。拖曳或選取多個 `.xlsx` 檔案，
    每個檔案是表格中的一列，不需先選月份，也沒有送出按鈕——整批解析完就自動匯入。
  - 每個檔案一加入就自動 dry run（不帶 `yearMonth`）取得預覽，前端由預覽列的 `serviceDate` 推導
    該檔案涵蓋的月份（可能不只一個月），顯示在該列的「涵蓋月份」欄；若涵蓋月份已有資料，表格上方
    跳出覆蓋警示（比照舊版 Google 表單同步「此月份已同步過」的提醒），自動匯入停在這裡等使用者
    勾選「我已確認風險」，勾完才續跑，而不是匯入後才用彈出視窗攔截。
  - 欄位對應不再要求使用者逐欄確認才能匯入：有系統推薦個案的欄位自動視為已對應，完全比對不到
    個案的欄位維持 `pending`，兩者都直接跟著這次 commit 一起送出。
  - commit 時針對推導出的每個月份各自呼叫一次、各帶對應的 `yearMonth`，沿用整月覆蓋語意。
  - 匯入完成後若有欄位進入待維護，跳出確認視窗詢問是否立即前往待維護頁籤（同樣比照個案管理
    匯入完成後的提示模式），選「稍後再說」則留在上傳頁籤查看結果。
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
  > CommitDriverReport（以下全部在同一個 pgxdb.TxRunner 交易內）
      > persistColumnDecisions 寫回 form_columns（以表頭文字為鍵）
      > collectImportableRows 挑出可寫入的列，其餘記入 SkippedRows
      > clearPreviousImport > ride.ClearImportedDates
          > 先收集受影響 slot（刪除後就查不到）
          > 刪 form_submissions，ride_sources 由 CASCADE 連帶清除
          > 每個 slot 還有來源就重算，全空就 DeleteDerivedRideRecord
      > 逐列 ride.IngestSubmission > InsertRideSource > recalculateRideRecord
      > MarkImported
  > writeImportAudit（交易外，失敗只記錄不推翻匯入）
```

模組交界在 `RideIngestor` port，由 `cmd/server/module_adapters.go` 的
`driverReportRideIngestor` 銜接 driverreport 與 ride。

待維護欄位補綁定的回填走另一條較短的路徑：

```
PATCH /driver-reports/columns/:id/mapping（status=mapped）
  > DriverReportService.UpdateColumnMapping（同一個 pgxdb.TxRunner 交易內）
      > FormStore.UpdateColumnMappingByID（RETURNING 更新前狀態、form_id、column_header、column_index）
      > 只有「更新前不是 mapped」才繼續；重複對已是 mapped 的欄位送出不會再回填
      > RideIngestor.BackfillColumn
          > ListSubmissionAnswersForColumn 讀 form_submissions.payload->'answers'->>header
          > 逐筆 InsertRideSource > recalculateRideRecord
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
      > 兩者皆空的列不列入清單
```

司機回填跟欄位回填分屬不同模組（`form_submissions.driver_id` 由 `ride` 模組持有），比對邏輯
是精確正規化相等（沿用既有 `namenorm.Normalize`，不像個案有分數式模糊比對）：

```
POST /driver-reports/drivers/bind { driverNameRaw, driverId }
  > DriverReportService.BindPendingDriver
      > RideIngestor.BackfillDriver
          > namenorm.Normalize(driverNameRaw) 取正規化姓名
          > ListUnmatchedDriverSubmissions() 撈出所有 driver_id IS NULL 的既有回報，
            在應用層過濾出正規化姓名相符的（天然涵蓋「其他待維護列同一人」的狀況，
            不需要額外的模糊比對或使用者確認）
          > 逐筆 UpdateSubmissionDriverID + 對該 submission 底下既有的 ride_sources
            逐筆 UpdateRideSourceDriverID > recalculateRideRecord
  > 回應帶回 affectedCount（實際回填的提交筆數）
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

- **重複匯入**：覆蓋而非疊加，重匯同一份檔案的結果與只匯一次相同。決策與替代方案見
  [driver-report-import-overwrite.md](../decisions/driver-report-import-overwrite.md)。
- **解析層級失敗**（日期打錯、欄位缺失）：逐列略過並記入 `SkippedRows`，其餘日期照常寫入。
- **資料庫層級失敗**：整份回滾，`last_imported_at` 不更新。先刪後寫若不回滾，該月資料會消失。
- **月份不符**：宣告 `yearMonth` 後，檔案內落在該月以外的有效日期僅該列標記為錯誤、記入
  `SkippedRows`，不中斷整份解析，其餘列照常產生預覽並可正常寫入；commit 時這些列一併略過，
  不會被計入清除範圍。上傳頁針對每個自動推導出的月份各自宣告一次 `yearMonth`，因此「同一份
  檔案橫跨多個月份」是預期情境，不屬於此列表示的月份不符——這些列會在其所屬月份的那一輪
  commit 正常匯入，提示訊息只說明「這一輪略過、另行處理」，不是要求使用者重新確認上傳檔案。
- **空檔**：沒有任何可寫入的列時不執行清除，避免傳錯空檔清空整月資料。
- **混車**：只刪本匯報表的 `form_submissions`，其他車輛對同一 slot 的來源保留並參與重算。
- **人工成果**：帶 `corrected_at`、`conflict_resolved_at` 或 `not_claimed_aa09` 的 `ride_records`
  不會被覆蓋式重匯刪除。
- **稽核寫入失敗**：只記 server log，不推翻已完成的匯入。
- **出勤同步失敗**：`AttendanceRegistrar.SyncFromImport` 回傳錯誤會讓整筆匯入回滾（跟
  `IngestSubmission` 失敗同一等級），不會出現「搭乘紀錄寫成功、出勤沒同步」的半套結果。

### 未宣告月份時的殘留行為

dry run 階段不帶 `yearMonth`，清除範圍只涵蓋檔案實際有的日期，因此會有一種殘留：某列的
日期從 `1150302` 改成 `1150402` 後重傳，`2026-03-02` 不在本次涵蓋日期內，舊資料留著不會被
清掉。上傳頁的 commit 階段會針對前端自動推導出的每個月份各自宣告 `yearMonth`，因此
「同一份檔案內、同一個月份」的重傳仍是整月覆蓋，不會有這個殘留；殘留只會發生在使用者把某天
的日期改到檔案完全沒涵蓋到的另一個月份時。

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
是否已對應個案；只有 `mapped` 的欄位會在當次 commit 展開成 `ride_sources`／`ride_records`。
匯入只會使用本次檔案出現的 mapped 欄位，不能沿用舊檔已對應、但本次未出現的欄位。

尚未對應個案的欄位不會因此卡住：待維護頁籤把某欄從 `pending` 改成 `mapped`
（`PATCH /driver-reports/columns/:id/mapping`）時，`DriverReportService.UpdateColumnMapping`
會在同一個交易內呼叫 `RideService.BackfillColumn`，直接讀取這個表單既有
`form_submissions.payload.answers` 裡這一欄留下的原始文字，逐筆展開成 `ride_sources` 並重算
`ride_records`——不需要使用者重新上傳原始檔案，回應會帶回本次實際補寫的筆數
（`backfilledRows`）。只有「這一次是從非 mapped 變成 mapped」才會觸發回填，重複對已經是
mapped 的欄位送出同樣的更新不會再次回填，避免疊加出重複的搭乘來源。

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
