# 前端核心功能流程

幾個橫跨多個元件、多支 API 的操作流程，從前端視角描述使用者實際會怎麼操作、畫面之間怎麼串。對應的後端邏輯見 [backend-flows.md](backend-flows.md)；頁面與路由對照見 [frontend-pages.md](frontend-pages.md)。

## 搭乘月曆與人工補登

`RideCalendarView` 打 `GET /rides/calendar` 拿一整月的矩陣資料（個案 × 日期 × 趟次）。點空白格會開 `RideManualEntryDialog`，送出時打 `POST /rides/manual-report`，成功後重新整理該日資料（不整月重抓）。這個流程對應後端 `RideService.ManualReportRide`，繞過匯報檔匯入直接寫入回報。

## 異常集中處理

`RideIssuesView` 打 `GET /rides/issues` 列出後端混車合併演算法判定為衝突或待確認的紀錄，點一筆會開 `RideCorrectionDrawer`：

- 純更正（單一來源但狀態錯誤）→ `PATCH /rides/:id`。
- 多來源衝突（同一趟不同司機回報不一致）→ `POST /rides/:id/resolve-conflict`，選擇要採信哪一筆來源。

兩種操作後端都會寫 audit log，前端目前不直接顯示 audit 內容——要查歷史要另外去 `/audit` 頁，且僅 admin 可見。

## 未回報清單

`MissingRidesView` 打 `GET /rides/missing`，資料來源是後端「未回報偵測」批次（拿應搭日曆跟實際回報比對），純唯讀清單，處理方式是回到搭乘月曆手動補登，或去確認司機是否漏填當日匯報。

## 司機接送匯報與欄位對應

一台車一份匯報表，頁面分成總覽與批次上傳兩支：

1. `DriverReportStatusView`（`/driver-reports/status`，`/driver-reports` 重導向於此）是唯讀總覽，
   只列出每台車的匯報表、已有資料的月份與天數（展開列或標籤呈現）、最後匯入時間，不放任何上傳
   或編輯動作；也能在此建立新車輛的匯報表（`POST /driver-reports`）。
2. `DriverReportImportView`（`/driver-reports/import`；舊路徑 `/driver-reports/batch-import`、
   `/driver-reports/mappings` 皆重導向於此）用頁籤分成「批次上傳」與「待維護資料」：
   - **批次上傳**頁籤採左右分欄版面（不用彈窗）：左側固定卡片是拖放上傳區、待處理檔案數量、
     覆蓋警示、送出按鈕；右側是每個檔案一張卡片的網格，即時顯示比對到的車輛／月份與解析狀態。
     逐檔依檔名自動比對車輛（比對不到或有多個候選時才要求手動選擇），不需要使用者先選月份。
     每個檔案一加入就自動以 `?dryRun=true`（不帶 `yearMonth`）取得預覽，讓後端依內容逐日解析；
     前端從預覽列的 `serviceDate` 推導這份檔案實際涵蓋的月份，就地顯示在卡片上。命中已有資料的
     月份時，左側跳出覆蓋警示（沿用舊版 Google 表單同步「此月份已同步過」的提醒寫法），需勾選
     「我已確認風險」才能送出，不是送出後才用彈出視窗攔截。
     欄位對應不需使用者當下確認：有系統推薦個案（依姓名相似度與 `[去程]／[回程]` 判斷）的欄位
     自動視為已對應，完全沒有推薦的欄位維持待對應狀態，兩者一起隨 commit 送出。確認匯入時，
     針對同一個檔案推導出的每個月份各自呼叫一次 `?dryRun=false&yearMonth=YYYY-MM`，沿用既有的
     整月覆蓋語意；多數檔案只涵蓋一個月，只有橫跨月份的補傳檔案才會送出多次請求。匯入完成後若
     有欄位進入待維護，跳出確認視窗詢問是否立即前往「待維護資料」頁籤。
   - **待維護資料**頁籤：`GET /driver-reports/columns?mappingStatus=pending`（不帶 `formId`）列出
     所有匯報表中完全比對不到個案的欄位，可連結既有個案（`PATCH /driver-reports/columns/:id/mapping`），
     或建立新個案並直接綁定——新增表單會帶入欄位解析出的原始姓名，只需補趟次即可送出
     （`POST /cases` 再接同一支綁定 API）。

沒設定對應（`mapping_status != mapped`）的欄位，匯入時不會處理成搭乘紀錄。匯入失敗時，
API client 會把後端 `details` 裡的具體原因（例如表頭不符）以通知條列出來，不要只顯示通用訊息。

## 政府申報匯出

`ExportView` 的操作順序是強制的：

1. 選期別／地區。
2. 按「檢核」打 `GET/POST /exports/precheck`。
3. 檢核沒過（畫面會列出每一項未過原因）就不能按「匯出」，回去處理資料。
4. 檢核過了才能 `POST /exports` 建立匯出工作，之後靠 `GET /exports/:id` 輪詢狀態拿下載連結。

前端不會自己組 Excel，實際產檔在後端 Cloud Run Job（見 [backend-flows.md](backend-flows.md) 的政府申報匯出流程）。

## 主檔批次匯入

`CaseListView` 有「匯入 Excel」入口，先 `GET /cases/template` 讓使用者下載範本，填好後上傳走 `POST /cases/import`，後端回傳每列的解析結果（成功／失敗與原因），前端要把失敗列的錯誤原因列出來讓使用者修正後重傳，不是整批全有效才能匯入。
