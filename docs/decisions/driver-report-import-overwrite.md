---
doc_type: decision
covers:
  - apps/api/internal/modules/driverreport/app/commit.go
  - apps/api/internal/modules/driverreport/infra/driver_report_repo.go
  - apps/api/internal/modules/ride/app/ride_service.go
  - apps/web/src/views/driverReports/importSummary.ts
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

## 後續修訂：移除檔案雜湊冪等鍵，冪等單獨由逐列比對保證

### Context

上面的逐列比對上線後，`CommitDriverReport` 仍保留了更早期加入的第二層防護：以
`(form_id, year_month, file_hash)` 為鍵的 `driver_report_imports` claim（migration `000030`）。
命中時整份 commit 直接短路 return，欄位對應不登記、一列都不寫，回傳
`status: already_imported`、`importedRows: 0`。

實際使用時這道防護造成誤判：使用者把一份檔案關聯到某台車上傳，畫面顯示「已匯入、共 0 天、
沒有可寫入的搭乘資料」，卻查不出原因——因為那台車的表單在同樣月份早已存在同樣雜湊的 claim。
系統無法區分「真的是重複上傳」與「使用者選錯車輛、恰好撞到不相干的既有 claim」。

而這道防護與逐列比對功能重疊：檔案內容完全相同時，`reconcileRideSource` 對每一格的判定
必然是「值相同 → 不動作」，本身就是冪等的。上層那道只省下一次計算，卻引入了 false negative。

### Decision

- 移除 `ClaimDriverReportImport`、`DriverReportImportIdempotencyStore` 與
  `CommitResult.FileHash`／`AlreadyImported`；`driver_report_imports` 表由 migration `000036` 移除。
  冪等完全交給 `reconcileRideSource`。`LockDriverReportImport`（advisory lock）與前端
  的本地去重（見下方「單次全檔匯入」修訂，該去重現已改成以表單為單位）都保留，兩者防的是
  併發與重複送出，不是內容重複。
- **已裁決「保留原資料」後重傳同一份檔案，會重新產生一筆未解決衝突，這是刻意行為。**
  `uq_ride_source_row_conflict_open` 只涵蓋 `resolved_at IS NULL`，已裁決的列不擋新的 INSERT。
  使用者要求每次上傳都重新判斷，不希望任何比對不上的資料被系統自行吞掉；代價是重傳同一份
  與既有資料不同的檔案會反覆要求裁決。先前這條路被檔案雜湊鍵擋住而從未觸發。
- **匯入路徑與待維護頁手動綁定共用同一套回填觸發條件。** `UpdateColumnMappingByHeader` 改為
  比照 `UpdateColumnMappingByID` 回傳更新前狀態，`persistColumnDecisions` 收集這次真正
  `pending → mapped` 的欄位，於逐列寫入後呼叫 `BackfillColumn`，筆數回傳為 `backfilledRows`。
- **`BackfillColumn` 必須排在逐列 `IngestSubmission` 之後**，因為 `SaveFormSubmission` 是
  `(form_id, service_date)` 原地更新：先補寫會讀到這幾天更新前的舊答案寫進去，再被本次的新值
  比出一筆並不存在的衝突。放在之後，本月各天走 Reaffirmed 跳過，只有先前月份真的被補寫。
- **回填要排除這份檔案涵蓋的所有服務日期**。`ListSubmissionAnswersForColumn` 沒有日期條件，
  回填範圍是整份表單的全部歷史；補寫會讀到這份檔案剛寫進去的新答案，不排除就會比出一批假衝突。
  `BackfillColumn` 因此新增 `skipDates`，由 `collectFileServiceDates` 從預覽列取出整份檔案的
  日期傳入；待維護頁的手動綁定沒有這種日期，傳 `nil`。
- 前端移除「這幾個月份已有資料，請勾選確認」的攔截，選完檔案直接上傳；改以
  `importSummary.ts` 把 `rideRecordRows`／`reaffirmedRows`／`pendingConflictRows`／
  `backfilledRows` 呈現成一行說明，重傳同一份檔案會明講「內容與既有資料完全相同」。

### Alternatives

- **保留雜湊鍵，只是把命中時的回應改成明確提示。** 使用者仍然無法重傳，且「選錯車輛撞到
  別人的 claim」這個根本問題沒解決，只是把靜默失敗換成看得懂的失敗。
- **雜湊鍵改帶車輛以外的維度（例如檔名）。** 治標：只要維度沒涵蓋「使用者其實想重新比對」
  這個意圖，就仍然會擋掉合法的重傳。
- **匯入時自動對應的欄位不回填，讓它留在待維護。** 欄位既然已經對應到個案，留在待維護是
  假的待辦；而且待維護清單是 `mapping_status = 'pending'` 與 payload 的交叉查詢，欄位一旦
  變 mapped 就查不到，等於資料還在 payload 裡卻永遠沒有入口撈得出來。
- **已裁決 kept_previous 後重傳不再跳出（把已裁決的新舊值組合納入比對）。** 噪音較低，但
  使用者明確要求每次都重新判斷，不接受系統代為省略。

### Consequences

- 重傳同一份檔案不再被擋，會完整跑一次逐列比對；值相同時 `rideRecordRows` 為 0 但
  `reaffirmedRows` 有數字，前端據此顯示「內容與既有資料完全相同」而不是「沒有可寫入的資料」。
- `writeImportAudit` 原本被雜湊鍵擋住的重複匯入現在都會留痕，稽核表成長變快；每次上傳本就是
  獨立事件，留痕正確，量大時再另評估保留策略。檔案雜湊仍算在稽核快照裡（`AuditSnapshot` 的
  `fileHash` 參數），純粹供事後追溯某筆搭乘來源出自哪一次上傳，不再參與任何重複判斷。
- 匯入現在可能寫入本次宣告月份以外的資料：剛完成對應的欄位會補寫**不在這份檔案裡**的先前月份，
  `backfilledRows` 是唯一能看出這件事的數字。檔案自己涵蓋的月份由單次 commit 一起寫入（見下方
  「單次全檔匯入」修訂）。
- `expandLegSeqs` 依當下排班展開：排班改過後重傳同一份檔案，舊 legSeq 的來源留著、新 legSeq
  走 Inserted，會出現「重傳卻有新增」，屬預期。
- migration `000036` 直接 `DROP TABLE driver_report_imports`，down migration 只還原結構、
  不還原資料；正式環境執行前需備份資料庫（比照 `000034` 的既有要求）。

## 後續修訂：單次全檔匯入取代逐月拆送，並以逐列 savepoint 取代整份回滾

### Context

跨月匯報表（一份 `.xlsx` 涵蓋多個月份）原本由前端 `computeMonths()` 先算出檔案涵蓋的月份，
再對每個月份各帶一次 `yearMonth` 呼叫 `CommitDriverReport`。`ParseDriverReport` 對每一輪
commit 都會把「這輪以外月份的列」標成 `ErrorMessage`（`不屬於本次宣告匯入的 YYYY-MM`），這些
本是設計本身的排除，卻被前端一律當成錯誤累積進說明清單——5 個月的檔案會產生「5 × 其他 4 個月
的列數」則假錯誤，使用者無從分辨真假。

同一時間，`CommitDriverReport` 逐列呼叫 `IngestSubmission`／`SyncFromImport` 只要有一列失敗
就整份交易回滾：跨月檔案改成單次 commit 後，這個代價會被放大——一台已停用的個案、一筆格式異常
的列，就足以擋掉整份檔案裡其他全部正確的資料，與 `docs/tech/system-logic-specification.md`
準則三的「非阻擋原則」（部分列有瑕疵不得中斷整份檔案）直接衝突。

### Decision

- **前端不再逐月拆送。**`ParseDriverReport` 解析時把所有列的 `service_date` 去重收進
  `PreviewResult.CoveredMonths`（`CommitResult` 也回同一份），前端 dry-run 與 commit 都直接
  讀這個欄位顯示涵蓋月份，不再自己從 `previewRows` 算。`processRow` 改成單次呼叫
  `commitImportDriverReport(formId, file, payload)`，不帶 `yearMonth`；`computeMonths`／
  `commitMonthOnce`／`activeMonthImports`／`monthImportLocks` 全數移除，改成以 `formId` 為
  鍵的單一 in-flight guard（`activeImports`／`formImportLocks`）。`yearMonth` 這個 query
  參數與 `parse.go` 的跨月排除分支本身不動——後端仍支援帶月份的整月覆蓋語意，只是前端不再使用。
- **逐列寫入改用 savepoint 隔離失敗。**`pgxdb.TxRunner` 新增 `WithSavepoint`：在既有事務內
  呼叫 `pgx.Tx.Begin` 開的是 SAVEPOINT 而非新交易，失敗時只回滾這個 savepoint。
  `CommitDriverReport` 對每一列的 `IngestSubmission` + `SyncFromImport` 呼叫
  `SavepointRunner.WithSavepoint`：失敗記入 `SkippedRows`（原因為原始錯誤訊息）並跳到下一列，
  不再中止整份交易。`persistColumnDecisions`、`LockDriverReportImport`、`BackfillColumn`
  仍在交易層級，失敗照樣整份回滾——那些不是單列資料問題。
  `DriverReportService.txRunner` 以 `.(SavepointRunner)` 型別斷言取得能力，未實作時退回
  「一列失敗即整份回滾」的舊行為，供沒有這項能力的測試替身使用；正式環境注入的
  `*pgxdb.TxRunner` 一律具備此能力。

### Alternatives

- **繼續逐月拆送，只是把跨月排除訊息標成 warning 而不是 error。** 治標：訊息數量沒變，使用者
  仍要在幾百則提示裡分辨哪些是真正需要處理的，也沒解決「一份檔案要發 N 次請求」的效能問題。
- **單次 commit，但保留整份回滾。** 符合「不逐月拆送」，但沒解決非阻擋原則要求的「部分列有
  瑕疵不得拖累整份」；且單次 commit 後失敗半徑從一個月放大到整份檔案（可能十幾個月），風險
  比逐月拆送時更高，必須同時解決。
- **應用層手動記錄「要跳過的列」再重試整份，不用資料庫 savepoint。** 需要自己實作補償邏輯與
  部分寫入的追蹤，複雜度高於直接借助 `pgx.Tx.Begin` 既有的 savepoint 語意。

### Consequences

- 說明清單不再出現「不屬於本次宣告匯入」的假錯誤；`SkippedRows` 現在同時可能來自解析層級
  （跨月、格式錯誤）與資料庫層級（個案停用、外鍵衝突等）失敗，前端統一顯示，不用分開處理。
- 「僅重試失敗月份」功能移除：失敗改成逐列跳過並記錄在 `SkippedRows`，不會再有「這個月成功、
  那個月失敗」的部分完成狀態需要重試；主表的「重試」按鈕保留，但意義變成整份請求層級的失敗
  （網路錯誤、鎖定失敗、欄位對應資料格式錯誤）才需要重試。
- 單一交易一次寫入整份檔案，交易與鎖持有時間比逐月長；以單一表單、月級資料量可接受。若日後
  單檔規模大幅成長出現逾時，應在後端切批並共用同一把表單鎖，而不是退回前端逐月上傳。
