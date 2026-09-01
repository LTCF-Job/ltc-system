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
   - **批次上傳**頁籤採上下堆疊版面（不用彈窗），整塊限寬 1100px 靠左不撐滿頁面：上方是拖放區，
     下方是主色「選擇檔案」按鈕，再往下是覆蓋警示與每個檔案一列的表格（檔案名稱／車輛／涵蓋月份／
     狀態／說明／操作），即時顯示比對到的車輛／月份與解析狀態。頁面沒有送出按鈕，同一批檔案全部
     解析完就自動匯入。
     表格用 `table-layout="auto"`，每欄各自鎖 `white-space: nowrap` 與 `min-width` 讓內容不換行，
     超寬時由 `.file-panel` 的 `overflow-x: auto` 接手水平捲動。
     逐檔依檔名自動比對車輛（比對不到或有多個候選時才要求手動選擇），不需要使用者先選月份。
     每個檔案一加入就自動以 `?dryRun=true`（不帶 `yearMonth`）取得預覽，讓後端依內容逐日解析；
     前端從預覽列的 `serviceDate` 推導這份檔案實際涵蓋的月份，顯示在該列的「涵蓋月份」欄。命中
     已有資料的月份時，表格上方跳出覆蓋警示（沿用舊版 Google 表單同步「此月份已同步過」的提醒
     寫法），自動匯入停在這裡等使用者勾選「我已確認風險」，勾完才續跑，不是匯入後才用彈出視窗攔截。
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

`ExportView` 的設定只有四個輸入：申報年月、申報地區、申報個案（`CaseSelectDialog` 多選視窗）、匯出檔案模式（直接下載／壓縮檔）。操作順序是強制的：

1. 選申報年月、申報地區與申報個案。沒選個案就按匯出會被擋下。
2. 按「執行前置檢核」打 `GET/POST /exports/precheck`。
3. 檢核有阻斷性錯誤（畫面會列出每一項未過原因）就不能匯出，回去處理資料；只有警告時會跳確認框。
4. `POST /exports` 同步回傳結果，不需輪詢。

卡片由上而下是「匯出設定」→「本次匯出結果」→「前置檢核報告」→「歷史匯出紀錄」。匯出結果排在檢核報告之上，因為匯出完成後使用者要做的下一件事是下載檔案，不是回頭看檢核內容。

匯出結果的呈現依模式而定：

- 直接下載：表格逐案列出檔案，每列一顆下載鈕，由使用者自行點選。不自動連續觸發下載，否則會被瀏覽器擋掉。單案走 `GET /exports/:id/files/:caseId/download`。
- 壓縮檔：單一「下載壓縮檔」按鈕，走 `GET /exports/:id/download`。

前置檢核報告（`PrecheckResult.vue`）每一筆明細右側都有導向該問題處理入口的按鈕，依 `item.code` 決定去處：`UNREPORTED_EXPECTED_RIDES` 的處理入口是未回報清單，按「查看未回報」導到 `/rides/missing?q=<個案姓名>`，`MissingRidesView` 進頁時把 `q` 預填進搜尋框，清單直接只剩該個案；其餘項目維持「檢視個案」導到個案詳情。

資料不完整而未納入申報的趟次會以警告列出（`skipped`），這份統計只在本次匯出的回應中有值。

歷史匯出紀錄不提供下載，只有「檢視個案」按鈕，用 `GET /exports/:id` 顯示該次匯出包含哪些個案。儀表板的「最近申報匯出紀錄」同樣只呈現摘要，不再有下載入口，避免出現第二條下載路徑。

前端不會自己組 Excel，實際產檔在後端（見 [backend-flows.md](backend-flows.md) 的政府申報匯出流程）。

展示模式（MSW）沒有 `ride_records` 資料表，搭乘紀錄由 `mocks/utils/demoRides.ts` 的 `buildDemoCaseMonth` 依個案排班逐日展開後套上示範例外與使用者的人工更正產生。搭乘月曆與申報匯出**讀同一份**：匯出走 `listDemoBoardedRides`，只取 `effectiveStatus === 'boarded'`，與後端的判準一致。因此展示資料裡的請假、缺席、未回報都不會進申報檔，加開趟次、人工更正過的出發時間與「不申報 AA09」標記則會反映。`11-exports.spec.ts` 的「申報列只取月曆上實際成行的趟次」用月曆 API 交叉比對列數守住這件事。

## 主檔批次匯入

`CaseListView` 有「匯入 Excel」入口，先 `GET /cases/template` 讓使用者下載範本，填好後上傳走 `POST /cases/import`，後端回傳每列的解析結果（成功／失敗與原因），前端要把失敗列的錯誤原因列出來讓使用者修正後重傳，不是整批全有效才能匯入。
