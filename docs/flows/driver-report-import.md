---
doc_type: flow
covers:
  - apps/api/internal/modules/driverreport/
  - apps/api/internal/modules/ride/app/ride_service.go
  - apps/api/internal/modules/ride/infra/ride_repo.go
  - apps/web/src/api/driverReports.ts
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
  - 上傳頁籤採左右分欄版面（不用彈窗）：左側固定卡片放拖放上傳區、待處理檔案數量與送出按鈕，
    右側是每個檔案一張卡片的網格，即時顯示解析狀態。拖曳或選取多個 `.xlsx` 檔案，每個檔案是
    右側一張卡片，不需先選月份。
  - 每個檔案一加入就自動 dry run（不帶 `yearMonth`）取得預覽，前端由預覽列的 `serviceDate` 推導
    該檔案涵蓋的月份（可能不只一個月），就地顯示在卡片上；若涵蓋月份已有資料，左側跳出覆蓋警示
    （比照舊版 Google 表單同步「此月份已同步過」的提醒），需勾選「我已確認風險」才能送出，而不是
    送出後才用彈出視窗攔截。
  - 欄位對應不再要求使用者逐欄確認才能匯入：有系統推薦個案的欄位自動視為已對應，完全比對不到
    個案的欄位維持 `pending`，兩者都直接跟著這次 commit 一起送出。
  - commit 時針對推導出的每個月份各自呼叫一次、各帶對應的 `yearMonth`，沿用整月覆蓋語意。
  - 匯入完成後若有欄位進入待維護，跳出確認視窗詢問是否立即前往待維護頁籤（同樣比照個案管理
    匯入完成後的提示模式），選「稍後再說」則留在上傳頁籤查看結果。
  - 待維護頁籤：`GET /driver-reports/columns?mappingStatus=pending`（不帶 `formId`，回傳所有匯報表
    的待對應欄位）列出完全比對不到個案的欄位，可連結既有個案（`PATCH /driver-reports/columns/:id/mapping`）
    或建立新個案並直接綁定（`POST /cases` 帶入欄位解析出的姓名，成功後再呼叫同一支綁定 API）。
- `DriverReportStatusView.vue`（`/driver-reports/status`，`/driver-reports` 重導向於此）是唯讀總覽，
  只顯示每台車已有資料的月份與天數，不提供任何上傳或編輯動作。
- `GET /api/v1/driver-reports/imported-months` 供總覽頁與匯入頁判斷某台車某個月是否已有資料；
  月份由 `form_submissions.service_date` 分組推得，不落地成欄位。

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

## Failure modes

- **重複匯入**：覆蓋而非疊加，重匯同一份檔案的結果與只匯一次相同。決策與替代方案見
  [driver-report-import-overwrite.md](../decisions/driver-report-import-overwrite.md)。
- **解析層級失敗**（日期打錯、欄位缺失）：逐列略過並記入 `SkippedRows`，其餘日期照常寫入。
- **資料庫層級失敗**：整份回滾，`last_imported_at` 不更新。先刪後寫若不回滾，該月資料會消失。
- **月份不符**：宣告 `yearMonth` 後檔案內出現該月以外的有效日期即整份拒絕，dry run 階段就擋。
  放行等於讓傳錯檔案清空另一個月。
- **空檔**：沒有任何可寫入的列時不執行清除，避免傳錯空檔清空整月資料。
- **混車**：只刪本匯報表的 `form_submissions`，其他車輛對同一 slot 的來源保留並參與重算。
- **人工成果**：帶 `corrected_at`、`conflict_resolved_at` 或 `not_claimed_aa09` 的 `ride_records`
  不會被覆蓋式重匯刪除。
- **稽核寫入失敗**：只記 server log，不推翻已完成的匯入。

### 未宣告月份時的殘留行為

dry run 階段不帶 `yearMonth`，清除範圍只涵蓋檔案實際有的日期，因此會有一種殘留：某列的
日期從 `1150302` 改成 `1150402` 後重傳，`2026-03-02` 不在本次涵蓋日期內，舊資料留著不會被
清掉。上傳頁的 commit 階段會針對前端自動推導出的每個月份各自宣告 `yearMonth`，因此
「同一份檔案內、同一個月份」的重傳仍是整月覆蓋，不會有這個殘留；殘留只會發生在使用者把某天
的日期改到檔案完全沒涵蓋到的另一個月份時。

### 上傳頁的逐檔失敗

逐檔各自請求，單一檔案失敗只停在那一列：建表失敗、表頭不符、車輛比對失敗都只讓該檔標記失敗，
其他檔案照常試算與匯入。並發處理維護上限（3），避免一次拖入十幾個檔案時打出過多同時請求。

### 欄位自動對應與待維護

commit 前不再要求使用者逐欄確認：有系統推薦個案（`suggestedCaseId`/`suggestedLegSeq`）的欄位
直接視為 `mapped` 送出；完全沒有推薦的欄位維持 `pending`，寫入 `form_columns` 供待維護頁籤查詢。
若整份檔案的欄位都沒有任何推薦（`mapped.size === 0`），後端會拒絕整份 commit（見
`persistColumnDecisions` 的必要條件），該檔案在上傳頁顯示失敗，需要先手動到待維護頁籤補建個案。

## Unverified

- 交易回滾行為只以 fake `TxRunner` 的單元測試覆蓋，未在真實 PostgreSQL 上驗證回滾與
  `ON DELETE CASCADE` 的實際連帶效果。`caseimport` 有 `commit_integration_test.go` 的前例
  （`//go:build integration` 搭配 `DATABASE_URL`），本次未比照建立。
- `pgxdb.Querier` 新增的 `SendBatch` 在真實交易內的批次寫入行為未經 runtime 驗證。
- 同一份匯報表併發重匯的鎖競爭行為未驗證，現行的 fake 也覆蓋不到。批次頁的並發上限只限制
  單一瀏覽器分頁，兩位管理員同時對同一台車同一個月匯入仍會競爭。
- `imported-months` 的 SQL 分組（`to_char(service_date, 'YYYY-MM')` 與 `source = 'import'` 篩選）
  只以 app 層的 fake 與 handler 測試覆蓋，未在真實 PostgreSQL 上驗證。
