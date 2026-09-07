---
doc_type: module
covers: ["apps/web/src/views/rides/RideCalendarView.vue", "apps/web/src/lib/rideCalendarDisplay.ts"]
---

# 搭乘月曆表

## Responsibility

顯示指定月份的個案排班與搭乘紀錄矩陣，提供月份、個案關鍵字篩選，以及搭乘紀錄更正與未回報補登入口。
本頁只呈現 API 回傳的排班與紀錄，不將實際搭乘筆數推算成排班趟次。

## Entrypoints

- 前端路由：`/rides`
- 頁面載入、月份切換或查詢：呼叫 `rides/calendar`，並並行載入該月份的 `holidays` 輔助標記。
- 搭乘紀錄狀態點擊：開啟 `RideCorrectionDrawer`。
- 應搭但尚無紀錄的格位：開啟 `RideManualEntryDialog`。

## Flow

`月份選擇／查詢 > 西元月份轉民國月份 > 取得月曆矩陣與假日 > 產生日期欄位與個案列 > 依每日應搭趟次呈現格位 > 更正或補登搭乘紀錄`

沒有排班但有搭乘紀錄的個案由 API 以 `tripPattern: 0` 標示；前端顯示「未設定排班」，不顯示為「0 趟」。

## Shared state

- `rides/calendar` 回應的 `month`、`daysInMonth`、`cases` 與每日 `days` 矩陣。
- `holidays` 回應建立的月份假日標記。
- `RideCorrectionDrawer` 與 `RideManualEntryDialog` 更新完成後會重新查詢月曆矩陣。

## Invariants and gotchas

- 預設查詢月份是瀏覽器本地目前月份；若本機資料沒有該月份的排班或搭乘紀錄，表格應顯示明確的空狀態，不能自動填入其他月份或假資料。
- `tripPattern: 0` 是無排班資料的 API 佔位值，不代表個案預期搭乘 0 趟，也不能用當月實際紀錄筆數取代排班趟次。
- 月曆日期欄位依查詢月份的實際天數產生，月份轉換只影響 API 查詢參數，不改變畫面上的西元月份選擇值。

## Unverified

none
