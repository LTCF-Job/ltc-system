# 待維護資料的可見範圍與「忽略此筆」語意

決策日期：2026-09-08

## 背景

「待維護」原本只有 `CaseRepository.List` 的 WHERE 子句認得，其餘讀取個案的查詢（排班日曆、月結、
趟次統計、新竹接送時刻表、儀表板在案數、個案 Excel 匯出、政府申報匯出、司機匯報姓名自動比對）
各自繞過，導致資料不完整的個案仍會出現在選單、報表與匯出結果中。

同時，各處待維護清單只有「補齊」與「裁決」兩種出口，使用者判斷某筆本來就不該匯入時沒有任何
方式讓它消失，重新匯入同一份檔案還會再出現一次。

## 決策

### 1. 待維護資料預設對全站隱藏

判定下沉到資料庫成為單一來源：

- `cases.profile_pending`（生日或身分證字號格式錯誤）
- `case_transport_preferences.link_pending`（單位／去程車輛／回程車輛比對不到主檔）
- `case_pending_status` view 合併上述兩者成單一 `is_pending`
- `caregivers.is_pending`（姓名或類型未填寫）

所有讀取個案的查詢一律 `JOIN case_pending_status` 並排除 `is_pending`。新增查詢時沿用這個 view，
不要複製條件——條件散落各處正是這次要修掉的問題。

HTTP 契約同步改為預設安全：`GET /cases` 與 `GET /caregivers` 預設排除待維護資料，呼叫端要明確
帶 `unresolvedLink`／`pending`（只取待維護）或 `includePending`（取全部）才拿得到。原本的
`excludePending` 參數移除。這樣未來新增的呼叫端預設就是安全的，不必記得加參數。

### 2. 政府申報匯出一併排除待維護個案

**這推翻了 2026-09-07「缺欄位一律留白仍匯出」的決定。** 待維護個案的搭乘紀錄不再進入申報匯出。

配套：申報前置檢查（`FindIncompleteActiveCases`）同步把 `is_pending` 納入「資料不完整」的判定，
否則會出現「前置檢查說沒問題、匯出卻少了人」的落差。另外，待維護個案的未裁決衝突不再擋住匯出，
因為它的搭乘紀錄本來就不會被匯出。

### 3.「忽略此筆」＝ 從系統移除

全站統一成「忽略此筆」，點選後直接刪除該筆資料。**不使用 `discarded` / `ignored` 之類的終結狀態
來消音**：使用者要的是把資料從系統移除，若下次重新匯入同一份檔案該筆再次出現，屬預期行為。

| 待維護型態 | 刪除對象 |
|---|---|
| 照護人員 | `caregivers` 該列（沿用既有 `DELETE /caregivers/:id`，硬刪） |
| 個案 | `cases` 該列（沿用既有 `DELETE /cases/:id`，軟刪除並收斂排班） |
| 疑似重複個案 | `case_import_duplicate_rows` 該列 |
| 欄位對應 | `form_columns` 該列 |
| 駕駛人未綁定的匯報列 | `form_submissions` 該列 |
| 同車同個案值衝突 | `ride_source_row_conflicts` 該列 |
| 出勤匯入衝突 | `attendance_import_conflicts` 該列 |

`form_columns` 原有的 `mapping_status='ignored'` 保留為 enum 值但不再有 UI 入口。

### 例外：`form_submissions` 不能直接 DELETE

`ride_sources.submission_id` 與 `ride_source_row_conflicts.new_submission_id` 都是 `ON DELETE CASCADE`，
直接刪會連帶砍掉搭乘來源，而 `ride_records` 只在寫入路徑重算，會留下對不上任何來源的搭乘紀錄。
因此 `DELETE /driver-reports/submissions/:id` 在同一交易內先取得該提交展開出的 slot 清單，刪除後
逐一重算搭乘紀錄。實務上待維護的匯報列 `driver_id IS NULL`，`IngestSubmission` 對這類提交提早返回、
不展開搭乘來源，所以 slot 清單通常是空的；重算路徑存在是為了不依賴這個假設。

## 權限

- 個案與照護人員的忽略沿用各自的 `delete` 軸（`masters_cases` / `masters_caregivers`，僅 admin）。
- 司機匯報的欄位對應、值衝突、匯報列忽略沿用 `driver_report_mappings` 的 **`edit`** 軸——該模組的
  `delete` 軸在權限矩陣中對所有角色皆為 false（見 migration 000018），用 delete 會讓所有人都看不到按鈕。
- 出勤衝突用 `attendance_fuel` 的 `delete` 軸（該模組的 delete 與 edit 同層級）。

## 已知取捨

- 忽略是不可復原的刪除，暫存列刪除後只能靠 audit log 回溯，誤刪只能重新匯入原始檔案。
- 同車同個案值衝突的判斷基準是 `ride_sources` 的現值而非衝突表，因此重傳且值仍不同時會重新產生
  一筆衝突。這與「每次上傳都重新判斷、不記住裁決來消音」的既有設計一致，不是本次要改的行為。
- 既有 `site_name_raw` 有值的照護人員在新規則下不再是待維護，會回到主清單。欄位與資料保留不動，
  不做一次性清理。
