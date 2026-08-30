# 後端核心演算法與驗證規則

`docs/tech/backend-flows.md` 講的是「資料怎麼流過哪些模組」，這份文件講的是那些模組**裡面實際的判斷邏輯**——條件式怎麼寫的、順序為什麼是這樣、邊界情況怎麼處理。改這些邏輯前一定要先讀對應的單元測試（`*_test.go`），這些規則大多有明確的規格書條號依據（程式碼註解會標，例如「規格書 4.6」），改動要連測試一起改。

## 混車合併演算法（`domain/merge.MergeRideSources`）

問題背景：同一趟服務可能有多個司機／車輛回報（例如個案臨時換車），系統要從多筆來源推導出「這趟到底有沒有搭到、是哪台車載的」單一結論。演算法：

1. **依 `vehicle_id` 分組，同組取 `submitted_at` 最新的一筆**當作該車的有效回報（同一台車先前回報「沒坐」後來改回「有坐」，以後者為準）。
2. 統計「有效回報」中有幾台車回報「有坐」（`boardedSources`）。
3. **決定 `mergedStatus`**：只要有任一台車回報「有坐」→ `boarded`；沒有任何回報「有坐」但至少有一筆有效回報 → `absent`；完全沒有回報 → `unreported`。
4. **判斷跨車衝突**：兩台以上「不同車輛」都回報「有坐」→ `hasConflict = true`（這是「異常集中處理」頁面資料的來源）。
5. **決定最終承載車輛／司機**（優先順序，前面命中就不看後面）：
   1. 已有人工裁決（`ConflictResolvedAt` 有值）→ 用裁決指定的車輛／司機，**不會被新回報覆蓋**。
   2. 已有人工更正過車輛（`CorrectedAt` 且 `CorrectedVehicle` 有值）→ 用更正指定的車輛／司機。
   3. 有回報「有坐」→ 取這些回報中 `submitted_at` **最早**的那一筆的車輛／司機（不是最新，是最早——最早回報視為第一線真實情況）。
   4. 都沒有 → 退回排班表的預設車輛／司機。預設司機由「該車在服務日生效的司機」推導（`DriverRepository.ListDriversForVehicleOnDate`）：**一台車可以有多位司機**，只有剛好一位時才自動帶入，多位時 `driverId` 留空由人工指定，不臆測是誰出車。
6. **決定 `effectiveStatus`**（畫面實際顯示、拿去算應搭日曆比對的狀態）：如果這筆紀錄之前被人工更正過（`CorrectedAt` 有值），維持人工指定的狀態不變；但如果這次重算出來的 `mergedStatus` 跟人工指定的不一樣，會標記 `SourceChanged = true`，前端要能提示「來源資料已變更但人工判定維持不變」，避免使用者誤以為系統又自動改掉了人工結果。

**保護規則的核心精神**：任何人工介入（裁決衝突、手動更正）之後，新進來的 Google 表單回報都只會被「記錄」（寫進 `ride_source`），不會反過來覆蓋人工結果；只會透過 `SourceChanged` 提示有落差。

## 車輛與司機指派關係（`driver_assignments`）

- 一位司機同一期間只會有一台車；一台車同一期間可以有多位司機（輪班或共同駕駛）。
- 資料庫用 `no_overlapping_driver_assignment`（`driver_id` + `effective_range` 的 GiST EXCLUDE）強制前者；車輛端沒有唯一限制，所以後者成立。
- 沒有「主要司機」的概念，`is_primary` 已於 migration `000006` 移除——司機唯一的那台車本來就是他的車。
- 從車輛端整批設定司機（`PUT /vehicles/:id/drivers`）時，被加入本車的司機在其他車上尚未結束的指派，會從生效日起被收掉（`effective_range` 上界收到生效日，尚未生效的指派直接刪除），以維持上面第一條規則。

## 應搭日曆計算（`domain/calendar.CalculateExpectedRides`）

用來回答「這個個案這個月哪幾天、哪幾趟應該要搭車」，是「未回報偵測」跟異常比對的比對基準。對月份內每一天依序檢查（**任一條件不成立就整天跳過，不看後面的條件**）：

1. 當天星期幾同時在「個案排班星期」跟「據點開放星期」的交集內（兩邊都要有才算，只要一邊沒開就跳過）。週日在這裡編碼為 `7`，不是 `0`。
2. 當天要在「個案申報起訖日」區間內（`claim_start_date` ~ `claim_end_date`，對應規格書 R8）。
3. 當天要在「排班本身的有效區間」內（`effective_from` ~ `effective_to`——排班可以中途換過，新排班生效前用舊排班）。
4. 當天不是國定假日（`holidays` map 命中就跳過）。
5. 通過以上全部條件的日子，把這筆排班底下**所有 leg**（去程／回程等時段定義）都展開成一筆 `ExpectedRide`。

## 四趟展開規則（在 `RideService.IngestWebhook` 裡）

個案排班若設定 `TripPattern == 4`（四趟制，例如去程分成「上車」「下車」兩個表單欄位），Google 表單上填的「第 1 趟」「第 2 趟」要展開成資料庫實際的四趟紀錄：表單第 1 趟 → 資料庫 leg 1、leg 3；表單第 2 趟 → leg 2、leg 4。非四趟制的排班則表單填第幾趟就對應資料庫第幾趟，不展開。

## 姓名／欄位正規化與比對（`domain/namenorm`）

司機在 Google 表單填的姓名常常有全形半形混用、多打空白、簡繁或異體字（例如「淑」vs「俶」），要先正規化才能拿去比對司機主檔：

1. `Normalize(s)`：Unicode NFKC 正規化（全形轉半形）→ 去除所有空白字元 → 依 `variants.go` 的異體字對照表把常見異體字／俗體字換成標準字 → 全部轉小寫。
2. `ParseColumnHeader(header)`：解析表單欄名（例如「1. 王小明 (去程)[備註]」），依序：去掉 Google 重複欄位自動加的 `" 2"`／`" 3"` 後綴 → 抓最後方括號內容判斷「去程／回程」方向 → 去掉開頭數字序號 → 去掉 `*` 之後的備註文字 → 去掉所有括號內容 → 剩下的字串跑 `Normalize` 得到乾淨姓名。
3. `ScoreCandidate(cleanedName, targetNormalized)`：兩字串完全相同給 1.0 分；Levenshtein 編輯距離 ≤ 1（差一個字或差一個字的順序）給 0.6 分（模糊比對，允許打錯一個字）；其他一律 0 分，不做更寬鬆的模糊比對。這個分數目前用在表單欄位對應建議（人工還是要確認，不是全自動套用）。

## 身分證驗證與加密（`domain/crypto`）

- `ValidateNationalID`：檢查長度必須是 10 碼；第 1 碼是英文字母，換算成兩位數代碼（`letterCodeMap`，A=10、B=11...W=32 這種對照表，不是連續編號）；第 2 碼（性別碼）本國人限 `1`／`2`，外來人口居留證限 `8`／`9`；第 3~10 碼必須是數字；最後依身分證檢查碼公式加權加總（權重 `8,7,6,5,4,3,2,1` 對應第 2~9 碼，代碼十位數 + 個位數×9 為基底）驗算總和是否為 10 的倍數。
- 身分證明文**不落地存資料庫**：`Encrypt`／`Decrypt` 用 AES-256-GCM（32 bytes key，隨機 12 bytes nonce），`Index` 另外算一組 HMAC-SHA256 當作可查詢比對的索引（因為密文本身無法直接 `WHERE =` 查重複），`Mask` 給列表畫面顯示用（`A20***9750`，前 3 碼＋`***`＋後 4 碼，不足 10 碼就整串打星號）。清單 API 預設回傳遮罩版本，要看明文得走 `POST /cases/:id/reveal`（會寫 audit log）。

## 匯出前置檢核（`PrecheckService.RunPrecheck`）

⚠️ **目前只有三項檢核**，規格書列的其他檢核項目（例如個案配給額度）**尚未實作**，不要誤以為 precheck 通過就代表資料完全沒問題：

| 順序 | Severity | Code | 判斷條件 |
|---|---|---|---|
| 1 | `info`（固定出現） | `QUOTA_CHECK_SKIPPED` | 恆常提示：配給額度檢查未執行（規則尚未取得，不影響 `Passed`） |
| 2 | `error` | `MISSING_CASE_PROFILE` | 該地區有效個案缺身分證、住家地址或服務使用類型任一欄位 |
| 3 | `warning` | `UNRESOLVED_CONFLICT` | 該地區有 `ride_records` 存在未裁決的混車衝突（`resolve-conflict` 還沒處理） |

`Passed = (errorCount == 0)`——只有 `error` 等級的項目會擋匯出，`warning`／`info` 只是提示不會擋。

## 政府申報表排序規則（`domain/govform.SortClaimRows`）

匯出的 Excel 資料列排序，依序比較（前面相等才看下一條）：

1. 若為「單檔多案模式」（一份檔案裝多個個案），先依身分證字號字典序排。
2. 依趟次序號（`LegSeq`）升冪：第 1 趟全月列完才排第 2 趟，以此類推。
3. 同趟次內，去程（`outbound`）排在回程（`inbound`）之前。
4. 最後依服務日期升冪排序。

## 統一錯誤碼

見 [backend-framework.md](backend-framework.md) 的「Response 格式」一節；`DUPLICATE_NATIONAL_ID` 是新增個案時身分證 HMAC 索引查到重複觸發（`internal/modules/casemgmt/transport/case_handler.go`）。`ASSIGNMENT_OVERLAP` 目前只有常數定義，程式碼裡沒有任何地方實際觸發它，屬於預留但未串接的錯誤碼。其他錯誤碼對應到哪種業務規則違反，要看實際呼叫端 `RespondError` 的 handler 程式碼，這裡不重複列。
