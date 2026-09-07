---
doc_type: decision
covers:
  - apps/web/src/views/driverReports/DriverReportImportView.vue
  - apps/web/src/views/driverReports/DriverReportStatusView.vue
  - apps/api/internal/modules/driverreport/transport/driver_report_handler.go
---

# 批次上傳採前端逐檔請求＋自動判斷車輛與月份，後端只多一支唯讀查詢

## Context

管理員每個月要處理多台車、多個月的匯報檔。最早只有一個逐車開對話框的入口：一次只能一台車，
對話框沒有月份輸入。第一版批次頁改成表格每列「一輛車 × 一個月」，但要求使用者先用月份選擇器
挑出要處理的月份、逐列選檔案再逐列試算比對，操作步驟仍多：使用者得先猜對月份，選錯月份或忘記
補選新月份的車輛就完全不會出現在表格裡。

匯報表是管理員手動逐車建立的（`uq_driver_report_forms_vehicle` 限制一台車一份），沒建表的車無法匯入。
另外沒有任何 API 能回答「哪台車哪個月已經匯入過」，`driver_report_forms.last_imported_at` 只有單一時間戳。

## Decision

表格每一列是「一個上傳檔案」，不再要求使用者預先選月份：

- 車輛由檔名自動比對車輛顯示名稱；唯一命中才視為判斷成功，命中零筆或多筆都要求使用者手動選擇，
  避免猜錯車輛而覆蓋到別台車的資料。
- 月份由試算結果自動推導：dry run 呼叫 `ParseDriverReport` 時不帶 `yearMonth`，讓後端依內容逐日
  解析；前端再從預覽列的 `serviceDate` 取出所有出現過的 `YYYY-MM`，顯示成月份標籤。
- commit 時針對推導出的每個月份各自呼叫一次、各帶對應的 `yearMonth`；絕大多數檔案只涵蓋一個月，
  只有橫跨月份的補傳檔案才會送出多次請求。後端不再整月覆蓋，`yearMonth` 只用來判斷檔案內落在
  該月以外的日期要標成錯誤列，寫入語意見
  [driver-report-import-overwrite.md](driver-report-import-overwrite.md)。
- 上傳與「哪些車哪些月份已有資料」的檢視拆成兩個頁面：`DriverReportImportView.vue`
  （`/driver-reports/import`）用頁籤分「批次上傳」與「待維護資料」，`DriverReportStatusView.vue`
  （`/driver-reports/status`）是純唯讀總覽，只列出各車已有資料的月份與天數，不放任何上傳或編輯動作。
- 「批次上傳」頁籤採上下堆疊版面，不用彈窗，整塊限寬 1100px 靠左：上方是拖放區，下方是主色
  「選擇檔案」按鈕，再往下是覆蓋警示與每個檔案一列的表格，欄位為
  檔案名稱／車輛／涵蓋月份／狀態／說明／操作，即時顯示比對到的車輛與解析狀態。表格用
  `table-layout="auto"`，每欄各自鎖 `white-space: nowrap` 與 `min-width`，內容不換行；超寬時由
  `.file-panel` 的 `overflow-x: auto` 接手水平捲動。檔案一加入即自動
  dry run 顯示涵蓋月份。頁面沒有送出按鈕：同一批拖入的檔案全部解析完（`analyzePending` 歸零）就
  自動匯入，避免使用者選完檔還要多按一次。涵蓋月份已有資料時不做任何攔截，選完檔案直接上傳——
  逐列比對本來就不覆蓋既有資料，值不同的會進待維護等使用者裁決，這道確認手續沒有防護意義
  （原本的「我已確認」核取方塊已移除，見
  [driver-report-import-overwrite.md](driver-report-import-overwrite.md) 的後續修訂）。
- 沒有匯報表的車輛在第一次上傳時，由前端先呼叫既有的 `POST /driver-reports` 建表再匯入。
- 已匯入月份由唯讀端點 `GET /driver-reports/imported-months` 提供，以
  `form_submissions.service_date` 分組統計，不新增資料表或欄位。
- 欄位對應不再要求使用者於上傳當下逐欄確認：有系統推薦個案的欄位自動視為已對應直接送出，
  完全沒有推薦的欄位維持 `pending`，寫入 `form_columns` 後交由「待維護資料」頁籤事後處理——
  可連結既有個案，或建立新個案並直接綁定（帶入欄位解析出的姓名）。原本獨立的
  `FieldMappingView.vue`（`/driver-reports/mappings`）已併入這個頁籤，該路徑改為重導向。

`yearMonth` 由前端自動推導取代手動輸入；後端寫入邏輯後續改為逐列比對，不再整月覆蓋，見
[driver-report-import-overwrite.md](driver-report-import-overwrite.md)。

## Alternatives

- **新增一支批次匯入 API。** 後端要處理部分成功、逐檔回報與檔案打包，而覆蓋語意需要刪與寫同生共死，
  一個交易涵蓋多台車會讓任一台失敗就整批回滾。逐檔各自一個交易才符合現有的失敗邊界。
- **後端自動建表。** 匯入端點在找不到匯報表時自己建一份，前端少一次請求。但那會讓「一台車一份匯報表」
  這條規則有兩個入口，且匯入端點從此帶有寫主檔的副作用。
- **把已匯入月份落地成欄位或獨立資料表。** 查詢快，但要跟每一次匯入、清除、人工補登同步，
  一旦漏更新就會與實際資料不符。由 `service_date` 推導不可能不同步。
- **保留月份選擇器，只是把它變成可選填的篩選。** 改動最小，但仍需要使用者先做一次判斷；
  拖曳上傳＋完全自動判斷才是「不用手動比對日期」這個需求要的結果。
- **狀態總覽與上傳合併成單一頁面。** 少一次導頁，但上傳頁的檔案列表、試算結果與可展開的欄位
  對應已經佔滿版面；總覽需要的是「一眼看出哪些月份缺資料」，兩者混在同一張表格會互相干擾。

## Consequences

- `POST /driver-reports` 對已有匯報表的車輛改為回傳既有那一份。原本 `ON CONFLICT` 保留舊 ID，
  服務層卻拿新產生的 ID 去查而查不到，回傳一個沒有原因的 500；批次頁需要「沒有就建、有就用既有那份」
  這個語意才能安全重試。
- 一個檔案涵蓋 N 個月就會送出 N 次 commit 請求；試算與匯入各自維護並發上限 3，逐檔顯示進度與結果。
- 新車第一次匯入會多一次建表請求；同一台車的多個檔案共用同一次建表，各自建會撞唯一索引。
  該次建表失敗只讓該檔標記失敗。
- 上傳頁不再讀 `imported-months`：移除重複月份攔截後這份快照在該頁沒有用途，也連帶消除了
  「兩位管理員同時操作同一台車時快照過期」這個窗口。總覽頁 `DriverReportStatusView.vue` 仍在使用
  該端點，後端 `MarkImported` 與 `GET /driver-reports/imported-months` 都保留。
- `imported-months` 只統計 `source = 'import'`；畫面顯示的「已有資料」純粹是統計用途，不代表
  後端會用任何方式覆蓋這些既有資料。
- 全數未對應的檔案會保存 pending 欄位供待維護流程處理，但不清除或寫入任何搭乘紀錄；完成對應後須重新匯入。
- 檔名比對不到車輛（或比對到多輛）時，該檔案停在「待選車輛」狀態，不會用猜測值送出請求。
- 月份不再寫進 route query（已無月份選擇器可分享），改由總覽頁 `/driver-reports/status`
  承擔「哪幾個月已有資料」的可視化與分享需求。
