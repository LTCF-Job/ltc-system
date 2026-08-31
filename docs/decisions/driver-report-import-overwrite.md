---
doc_type: decision
covers:
  - apps/api/internal/modules/driverreport/app/commit.go
  - apps/api/internal/modules/ride/app/ride_service.go
---

# 司機接送匯報匯入採先刪後寫的覆蓋語意

## Context

匯報表以車輛為單位，每台車每個月一份 `.xlsx`。司機補填、管理員改錯字都會需要重傳同一個月。

原本的匯入是純累加：`submitted_at` 每次取當下時間，`uq_form_submission (form_id, service_date,
submitted_at)` 永遠不衝突；`ride_sources` 是純 INSERT 且表上沒有唯一鍵。重傳一次，該月的來源列
就整包多一份，混車合併演算法讀到重複來源後會算出錯誤的狀態與衝突判定。

批次上傳（一次選多台車多個月的檔案）會讓重傳成為常態：整批中有一個檔失敗，管理員多半整批再傳一次。

## Decision

匯入前先刪掉這份匯報表在本次涵蓋日期的 `form_submissions`，再寫入新資料，整段包在同一個交易內。

清除範圍由選填的 `yearMonth` 決定：有宣告就清整個月，沒宣告就只清檔案實際涵蓋的日期。宣告月份時，
檔案內出現該月以外的有效日期即整份拒絕。

批次上傳頁一律宣告月份，單車上傳對話框維持不宣告；兩種模式的殘留行為見
[driver-report-import.md](../flows/driver-report-import.md)。

## Alternatives

- **`ride_sources` 加唯一鍵改成 upsert。** 需要 migration，而且找不到穩定的自然鍵：同一 slot 本來就
  允許多列來源（混車情境下不同車輛各報一次），加唯一鍵會把合法的多來源擋掉。
- **以檔案日期的 min..max 當清除區間。** 檔案只填了月初與月底兩天時，會誤刪中間那些前次匯入寫進去、
  這次檔案沒涵蓋的日期。改用明確的日期集合避開這個問題。
- **每列各自一個交易（比照 `caseimport`）。** 逐列失敗只影響該列，但先刪後寫的「刪」若已完成而後續
  寫入中斷，該月資料會直接消失。覆蓋語意需要刪與寫同生共死。
- **有效列為零時仍清空該月。** 語意一致（宣告了就清），但傳錯一個空檔就會清掉整個月。誤刪的代價高於
  「無法用匯入清空一個月」這個做不到的功能。

## Consequences

- 資料庫層級的失敗從「略過該列」變成「整份回滾」。解析層級的失敗（日期打錯、欄位缺失）維持逐列略過。
- `RideRepository` 與 `DriverReportRepository` 的每個方法都要用 `pgxdb.FromContext` 取用外層交易，
  否則寫入會落在交易外而失去原子性。`pgxdb.Querier` 因此補上 `SendBatch`。
- 來源被清空的 `ride_records` 會被刪除，但帶有 `corrected_at`、`conflict_resolved_at` 或
  `not_claimed_aa09` 的紀錄保留。新增其他人工介入欄位時，要一併加進 `DeleteDerivedRideRecord`
  的保護條件，否則人工成果會在下一次重匯時消失。
- 月份不落地成資料庫欄位，一律由 `form_submissions.service_date` 推得。
- 清除限定 `source = 'import'`，人工補登的 `form_submissions` 不會被重匯抹掉。`ListRideSourceSlotsForForm`
  與 `DeleteFormSubmissions` 的篩選條件必須永遠一致，否則會有 slot 被刪了來源卻沒被重算。
- `RideService.recalculateRideRecord` 不再吞掉讀寫錯誤。單一交易內任一語句失敗會讓後續語句全部失效，
  吞掉錯誤只會讓最後爆出的訊息與真正的根因無關。
