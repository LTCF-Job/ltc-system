---
name: domain-data-guidelines
description: Use when changing business dates, ROC dates, time zones, schedules, rides, holidays, status values, merge semantics, array filters, or other domain data rules.
---

# Domain data guidelines

## 語意來源

- 對 blank、zero、unknown、未提供、無限期與非法值分別定義語意；只有契約明定的未提供值才使用預設值。
- 業務日期、民國日期、時區、business date、排班、趟次、假日、搭乘狀態與合併規則集中在 domain parser／rule table，不散落在 handler、頁面或 SQL 字串中。
- 狀態轉換與清除語意寫成明確的 domain rule；非法狀態、互斥選項與不存在的關聯回傳可區分的 validation／conflict 結果。
- PostgreSQL array filter 明確指定資料型別與序列化協定；driver、vehicle 等 UUID 陣列同時驗證空陣列、單值、多值與資料庫 driver 行為。

## 驗證

- 使用 table-driven boundary cases 覆蓋西元／民國日期、跨日與時區、空值與零值、假日排班、清除欄位、狀態轉換與合併衝突。
- 從 HTTP 輸入一路驗證到 domain、persistence 與回應，確認格式化與儲存不改變原始業務語意。
- UI 顯示時間統一使用 `@/utils/formatters`，只顯示到秒數：`YYYY-MM-DD HH:mm:ss` 或 `HH:mm:ss`。
