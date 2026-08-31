---
doc_type: decision
covers:
  - apps/web/src/views/driverReports/DriverReportBatchImportView.vue
  - apps/api/internal/modules/driverreport/transport/driver_report_handler.go
---

# 批次上傳採前端逐列請求，後端只多一支唯讀查詢

## Context

管理員每個月要處理多台車、多個月的匯報檔。原本只有 `DriverReportListView.vue` 逐車開對話框這一條路：
一次只能一台車，而且對話框沒有月份輸入，`yearMonth` 這個已實作的整月覆蓋參數沒有任何呼叫端。

匯報表是管理員手動逐車建立的（`uq_driver_report_forms_vehicle` 限制一台車一份），沒建表的車無法匯入。
另外沒有任何 API 能回答「哪台車哪個月已經匯入過」，`driver_report_forms.last_imported_at` 只有單一時間戳。

## Decision

表格每一列是「一輛車 × 一個月」，逐列各自送出 dry run 與 commit，兩者都帶該列的 `yearMonth`。

- 沒有匯報表的車輛在第一次上傳時，由前端先呼叫既有的 `POST /driver-reports` 建表再匯入。
- 已匯入月份由新的唯讀端點 `GET /driver-reports/imported-months` 提供，以
  `form_submissions.service_date` 分組統計，不新增資料表或欄位。
- 欄位對應 UI 從 `DriverReportImportDialog.vue` 抽成 `DriverReportColumnMappingTable.vue`，兩個入口共用。

後端的匯入邏輯、覆蓋語意與交易邊界完全沒有改動。

## Alternatives

- **新增一支批次匯入 API。** 後端要處理部分成功、逐列回報與檔案打包，而覆蓋語意需要刪與寫同生共死，
  一個交易涵蓋多台車會讓任一台失敗就整批回滾。逐列各自一個交易才符合現有的失敗邊界。
- **後端自動建表。** 匯入端點在找不到匯報表時自己建一份，前端少一次請求。但那會讓「一台車一份匯報表」
  這條規則有兩個入口，且匯入端點從此帶有寫主檔的副作用。
- **把已匯入月份落地成欄位或獨立資料表。** 查詢快，但要跟每一次匯入、清除、人工補登同步，
  一旦漏更新就會與實際資料不符。由 `service_date` 推導不可能不同步。
- **沿用對話框、只加上月份輸入。** 改動最小，但一次仍只能一台車；表格、逐列試算結果與可展開的
  欄位對應需要的空間，對話框放不下，也無法深連結到「這幾個月要補傳」。

## Consequences

- `POST /driver-reports` 對已有匯報表的車輛改為回傳既有那一份。原本 `ON CONFLICT` 保留舊 ID，
  服務層卻拿新產生的 ID 去查而查不到，回傳一個沒有原因的 500；批次頁需要「沒有就建、有就用既有那份」
  這個語意才能安全重試。
- N 台車 N 個月會產生 N 次請求。畫面以並發上限 3 送出，並逐列顯示進度與結果。
- 新車第一次匯入會多一次建表請求；同一台車的多個月份共用同一次建表，逐列各自建會撞唯一索引。
  該次建表失敗只讓該列標記失敗。
- 覆蓋確認依據的是前端的已匯入統計快照。快照在建表後與每次匯入後重讀，但兩位管理員同時操作
  同一台車同一個月時，仍可能有一方看不到對方剛寫入的資料。後端沒有樂觀鎖，這個窗口關不掉。
- `imported-months` 只統計 `source = 'import'`，與 `DeleteFormSubmissions` 的覆蓋範圍一致；
  兩邊的篩選條件必須永遠相同，否則畫面會顯示一個重匯其實不會覆蓋的月份。
- 未對應欄位的列不得被納入確認匯入。放行等於讓那些欄位靜默不寫入搭乘紀錄。
- 選定月份寫進 route query，讓「哪幾個月要補傳」可以直接分享連結。
