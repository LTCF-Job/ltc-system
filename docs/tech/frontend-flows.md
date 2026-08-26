# 前端核心功能流程

幾個橫跨多個元件、多支 API 的操作流程，從前端視角描述使用者實際會怎麼操作、畫面之間怎麼串。對應的後端邏輯見 [backend-flows.md](backend-flows.md)；頁面與路由對照見 [frontend-pages.md](frontend-pages.md)。

## 搭乘月曆與人工補登

`RideCalendarView` 打 `GET /rides/calendar` 拿一整月的矩陣資料（個案 × 日期 × 趟次）。點空白格會開 `RideManualEntryDialog`，送出時打 `POST /rides/manual-report`，成功後重新整理該日資料（不整月重抓）。這個流程對應後端 `RideService.ManualReportRide`，繞過 Google 表單直接寫入回報。

## 異常集中處理

`RideIssuesView` 打 `GET /rides/issues` 列出後端混車合併演算法判定為衝突或待確認的紀錄，點一筆會開 `RideCorrectionDrawer`：

- 純更正（單一來源但狀態錯誤）→ `PATCH /rides/:id`。
- 多來源衝突（同一趟不同司機回報不一致）→ `POST /rides/:id/resolve-conflict`，選擇要採信哪一筆來源。

兩種操作後端都會寫 audit log，前端目前不直接顯示 audit 內容——要查歷史要另外去 `/audit` 頁，且僅 admin 可見。

## 未回報清單

`MissingRidesView` 打 `GET /rides/missing`，資料來源是後端「未回報偵測」批次（拿應搭日曆跟實際回報比對），純唯讀清單，處理方式是回到搭乘月曆手動補登，或去確認司機是否忘記填表單。

## 表單同步與欄位對應

新串一份 Google 表單時的操作順序：

1. `FormListView` 建立表單設定。
2. 打 `POST /forms/:id/sync` 讓後端重新抓一次 Google Sheet 的欄名。
3. 到 `FieldMappingView` 把每個欄位對應到「哪個個案的哪一趟」（`PATCH /forms/columns/:id/mapping` 單筆，或 `POST /forms/columns/batch-mapping` 批次）。

沒設定對應（`mapping_status != mapped`）的欄位，Webhook 收到答案也不會處理成搭乘紀錄。

## 政府申報匯出

`ExportView` 的操作順序是強制的：

1. 選期別／地區。
2. 按「檢核」打 `GET/POST /exports/precheck`。
3. 檢核沒過（畫面會列出每一項未過原因）就不能按「匯出」，回去處理資料。
4. 檢核過了才能 `POST /exports` 建立匯出工作，之後靠 `GET /exports/:id` 輪詢狀態拿下載連結。

前端不會自己組 Excel，實際產檔在後端 Cloud Run Job（見 [backend-flows.md](backend-flows.md) 的政府申報匯出流程）。

## 主檔批次匯入

`CaseListView` 有「匯入 Excel」入口，先 `GET /cases/template` 讓使用者下載範本，填好後上傳走 `POST /cases/import`，後端回傳每列的解析結果（成功／失敗與原因），前端要把失敗列的錯誤原因列出來讓使用者修正後重傳，不是整批全有效才能匯入。
