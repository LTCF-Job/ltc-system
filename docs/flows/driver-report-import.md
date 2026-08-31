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
- 前端主要入口為 `DriverReportBatchImportView.vue`（上傳接送匯報，`/driver-reports/batch-import`）：
  - 進入頁面預設帶入當前月份，表格每列為「一輛車 × 一個月」。
  - 逐列各自送出 dry run 與 commit，兩者都帶該列的 `yearMonth`。
  - 共用 `DriverReportColumnMappingTable.vue` 進行欄位對應。
- `GET /api/v1/driver-reports/imported-months` 供批次頁判斷某台車某個月是否為重傳；月份由
  `form_submissions.service_date` 分組推得，不落地成欄位。

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

單車上傳對話框不帶 `yearMonth`，清除範圍只涵蓋檔案實際有的日期，因此會有一種殘留：某列的
日期從 `1150302` 改成 `1150402` 後重傳，`2026-03-02` 不在本次涵蓋日期內，舊資料留著不會被
清掉。批次上傳頁一律宣告月份，不會有這個殘留。

### 批次頁的逐列失敗

逐列各自請求，單列失敗只停在那一列：建表失敗、表頭不符、月份不符都只讓該列標記失敗，
其他列照常試算與匯入。有未對應欄位的列不得被納入確認匯入，直到使用者在展開的對應表格處理完。
並發上限為 3，避免一次選十幾列時打出過多同時請求。

## Unverified

- 交易回滾行為只以 fake `TxRunner` 的單元測試覆蓋，未在真實 PostgreSQL 上驗證回滾與
  `ON DELETE CASCADE` 的實際連帶效果。`caseimport` 有 `commit_integration_test.go` 的前例
  （`//go:build integration` 搭配 `DATABASE_URL`），本次未比照建立。
- `pgxdb.Querier` 新增的 `SendBatch` 在真實交易內的批次寫入行為未經 runtime 驗證。
- 同一份匯報表併發重匯的鎖競爭行為未驗證，現行的 fake 也覆蓋不到。批次頁的並發上限只限制
  單一瀏覽器分頁，兩位管理員同時對同一台車同一個月匯入仍會競爭。
- `imported-months` 的 SQL 分組（`to_char(service_date, 'YYYY-MM')` 與 `source = 'import'` 篩選）
  只以 app 層的 fake 與 handler 測試覆蓋，未在真實 PostgreSQL 上驗證。
