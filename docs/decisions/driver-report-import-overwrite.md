---
doc_type: decision
covers:
  - apps/api/internal/modules/driverreport/app/commit.go
  - apps/api/internal/modules/ride/app/ride_service.go
---

# 司機接送匯報匯入改採逐列比對，取代先刪後寫的整段覆蓋

## Context

原本的匯入採「先刪後寫」：匯入前先刪掉這份匯報表在本次涵蓋日期的 `form_submissions`（`ride_sources`
隨 CASCADE 連帶清除），再整批寫入新資料。這個設計解決了純累加造成的重複來源問題，但代價是每次重傳
都會不分青紅皂白清空整段日期範圍——即使檔案裡大多數列跟既有資料完全相同，也會整批砍掉重建；同一台車
同一個個案的資料若前後兩次上傳的值不同，也只會被後面那次靜默覆蓋，使用者完全看不到發生過變更。

需求變成：上傳是彼此獨立的事件，不應該互相覆蓋；同一台車同一個個案的資料若與既有資料衝突，要讓使用者
自己選擇要保留哪一筆，而不是自動覆蓋或自動忽略。

## Decision

移除整段清除（`clearPreviousImport`／`RideService.ClearImportedDates`，兩者皆已刪除），改成
`RideService.IngestSubmission` 逐格呼叫 `reconcileRideSource` 比對「這台車在這個 slot
（`case_id`, `service_date`, `leg_seq`, `vehicle_id`）目前最新的一筆來源」：

- 這台車在此 slot 從未有資料 → 直接寫入。
- 既有資料與新資料完全相同（`reported` 與 `driver_id` 皆同）→ 視為重複回報，不動作。
- 任一不同 → 不寫入 `ride_sources`，改成在 `ride_source_row_conflicts`（migration `000035`）暫存
  一筆待處理衝突，保留新舊兩筆值，等使用者透過 `POST /driver-reports/row-conflicts/:id/resolve`
  選擇要採用新資料還是保留原資料。

`BackfillColumn`（欄位補綁定）與 `BackfillDriver`（司機補綁定）都改成呼叫同一個 `reconcileRideSource`，
不再各自直接呼叫 `InsertRideSource`／更新既有列，確保三個進入點的比對邏輯一致。司機比對不到司機主檔時，
`IngestSubmission` 完全不展開這一列成搭乘來源（只留在 `form_submissions.payload`），司機補綁定後才由
`BackfillDriver` 讀取表單既有欄位對應與這筆提交的原始答案，逐欄重新跑一次 `reconcileRideSource`。

`form_submissions` 的唯一鍵從 `(form_id, service_date, submitted_at)` 改成 `(form_id, service_date)`
（migration `000034`），讓「一車一天一筆」的語意真正由資料庫保證：同一天重傳時原地更新該筆的
`submitted_at`／`payload`，不再疊加出多筆。`driver_id` 用 `COALESCE(EXCLUDED.driver_id,
form_submissions.driver_id)`，避免這次沒解析出司機（`NULL`）覆蓋掉先前已比對成功或人工綁定的司機。

`ride_sources` 新增自己的 `submitted_at` 欄位（原本借用 `form_submissions.submitted_at`）：既然
`form_submissions` 現在會被同一天的後續上傳原地更新，繼續借用會讓同一天較早寫入、本次未變動的來源，
因為同一天稍後的另一次上傳而被誤判成剛剛才寫入，進而在跨車衝突判斷（`merge.MergeRideSources`）中
贏過真正較新的來源。

## Alternatives

- **維持先刪後寫，只是把清除範圍縮小到真的有變動的列。** 仍然需要先算出「這次哪些列跟舊資料不同」才
  知道要清什麼，等於還是要做逐列比對；比對完之後直接決定寫入或暫存衝突，比比對完再回頭刪除更直接。
- **`ride_sources` 加唯一鍵改成 upsert。** 舊決策已經否決過這個方向：同一 slot 本來就允許多列來源
  （混車情境下不同車輛各報一次），加唯一鍵會把合法的多來源擋掉。這次的暫存衝突表刻意獨立建表，識別鍵
  多帶 `vehicle_id`，不落在 `ride_sources` 本身，避免這個問題。
- **衝突當下就自動採用最新上傳的值。** 不需要暫存表，實作最簡單，但等於默默恢復了舊版的覆蓋行為，
  只是換了個名字；使用者仍然看不到、也無法選擇要保留哪一筆，不符合需求。
- **司機或個案任一有變動就整批標成衝突。** 會把單純的「駕駛人姓名後來被綁定成功」這種良性差異也當
  成需要人工裁決的衝突，噪音過多；但使用者已明確要求回報值或司機任一不同都算衝突，因此仍採此設計，
  只是這裡記錄下曾考慮過的折衷方案。

## Consequences

- 上傳結果從單一「寫入筆數」拆成 `importedRows`（處理的匯報列數）、`rideRecordRows`（新增的搭乘來源）、
  `reaffirmedRows`（無變化的重複回報）、`pendingConflictRows`（進待維護的筆數）四個數字，前端需要
  分開顯示，不能再假設「匯入成功」等於「資料已經完全生效」。
- 同一 slot 同時只允許一筆未解決的衝突（`ride_source_row_conflicts` 的 partial unique index）；
  第三次上傳到同一個未解決的衝突只更新其新值，`previous_*` 保持第一次偵測到衝突時的既有資料不動，
  對應「待維護若是同一筆以最新上傳為準」的需求。
- 「採用新資料」裁決會重放寫入一筆新的 `ride_sources`，既有那筆不會被刪除——同一 vehicle+slot
  可能因此累積多筆歷史來源列，混車合併只取每台車最新一筆，功能上無影響，但長期會有資料量成長，
  之後若需要可另外評估是否要清理已解決衝突的舊來源列。
- 稽核策略沿用 `docs/decisions/mutation-audit-policy.md` 既有的 `conflict_resolve` action，
  裁決結果的稽核寫入失敗只記 server log，不推翻已完成的裁決（比照既有匯入稽核策略）。
- `form_submissions` 唯一鍵變更需要先清理既有重複資料（同一台車同一天若過去被重複上傳、留下多筆
  歷史紀錄，只保留最新一筆），此步驟不可逆，正式環境執行 migration `000034` 前需先備份資料庫。
- `ListRideSourceSlotsForForm`／`DeleteFormSubmissions`／`ListRideSourcesForSubmission`／
  `UpdateRideSourceDriverID`／`ClearImportedDates` 與其對應的 `RideSlot`／`RideSourceForSubmission`
  型別已全數移除；未來若需要「刪除本月匯入」之類的管理功能，需要重新設計，不能沿用這些已刪除的方法。
