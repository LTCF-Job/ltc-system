---
doc_type: flow
covers:
  - tests/qa-crud/
---

# 前端 CRUD 全功能測試流程

以真實使用者操作為準的黑箱測試流程：用 Playwright 驅動瀏覽器點畫面、記錄前端實際送出的 request 與後端回應、再回 PostgreSQL 驗證資料。指令與 suite 清單見 [tests/qa-crud/README.md](../../tests/qa-crud/README.md)，本文件只定義流程與判準。

與現有測試的分工：`apps/api` 的 `go test` 驗單元邏輯，`npm run type-check` 驗型別，這套驗的是「使用者在畫面上做得到什麼、做完之後資料庫變成什麼樣」——這兩者中間的契約漂移只有這一層抓得到。

## 五道驗證層級

一次操作要同時看完這五層才算驗過，只看畫面成功訊息不算：

| 層級 | 看什麼 | 怎麼取得 |
|---|---|---|
| 1. 前端表單驗證 | 必填、格式、範圍是否在送出前擋下 | `L.submit()` 回傳的 `errors`（`.el-form-item__error`） |
| 2. 實際送出的 payload | 前端把畫面上的值序列化成什麼 | `net.calls` 的 `phase: 'req'` 與 `body` |
| 3. 後端回應 | status、錯誤碼、`details` 是否足以讓使用者知道錯在哪 | `net.calls` 的 `phase: 'res'` |
| 4. 畫面回饋 | toast 文案、對話框是否關閉、清單是否更新 | `toasts`、`dialogOpen`、`tableRows()` |
| 5. 資料庫 | 欄位值、NULL 與空字串、軟刪除標記、關聯表 | `docker exec ltc-postgres psql` |

第 4 與第 5 層一起看才抓得到契約漂移：API 回 200、DB 也寫進去了，但清單顯示空白，代表回應欄位命名與前端預期不符。

## 每個模組的固定測試矩陣

新模組照抄這份矩陣即可，缺哪一項就補哪一項：

1. **空白送出**：所有欄位留白按儲存，確認前端擋下且不發出 request。
2. **只填必填**：確認選填欄位送 `null` 或空字串時後端接受，且 DB 存的是預期的 NULL／空值。
3. **全欄位填滿**：每個欄位都給值，逐欄比對 DB。
4. **欄位組合**：A 填 B 不填、B 填 A 不填，特別針對互相依賴的欄位（例如據點名稱 + 所屬區域）。
5. **邊界值**：純空白字元、超長字串（300 字）、負數、未來日期、不合理年份、HTML／`javascript:` 字串。
6. **唯一鍵衝突**：用同一組值再新增一次，確認回 409 且錯誤訊息指得出是哪個欄位。
7. **下拉選項對照後端值域**：把選項全部列出來（`L.selectOptions()`），逐一比對 DB 的 CHECK constraint 與 enum；前端提供得出、後端存不進去的選項一律是 bug。
8. **編輯**：先 dump 表單載入值確認舊值有帶入，改幾個欄位存檔，確認沒被帶到的欄位不會被清空。
9. **刪除**：分別測「沒有被引用」與「已被其他資料引用」兩種，後者應該擋下而不是留下孤兒資料。
10. **權限**：以 `viewer` 帳號登入（`QA_LOGIN_EMAIL` 帶 `viewer` 字樣），確認唯讀角色看不到寫入按鈕。

## 匯入匯出的測試流程

匯入匯出照這個順序測，任何一步斷掉整條路就是不可用：

1. **從畫面下載範本**，記錄檔名與檔案大小。
2. **原封不動上傳剛下載的範本**。這一步在問兩件事：範本能不能被系統自己接受；範本內附的範例列會不會被當成真實資料寫進去。
3. **填入組合測試資料再上傳**：必填缺漏、格式錯誤、值域外的值、主檔比對得到與比對不到的名稱、完全重複的兩列。
4. **比對預覽與實際寫入**：`dryRun=true` 的 `totalRows` 要等於檔案實際資料列數，少掉的列一定要有對應的錯誤或警告訊息；靜默丟列是 bug。
5. **重複上傳同一份檔案**：確認冪等，已匯入的列應被辨識為「已匯入」而不是變成待裁決或重複建立。
6. **回 DB 逐欄驗證**：民國日期換算、姓名 trim、加密欄位遮罩、關聯是否正確接上（`site_id` / `vehicle_id` 有值 vs 只留 `*_name_raw`）。
7. **匯出檔**：下載後用 `xlsx.py dump` 檢查表頭與內容，確認欄位順序與筆數符合預期。

`.xlsx` 的讀寫一律用 `tests/qa-crud/xlsx.py`（Python + openpyxl），不引入新的 Node 相依：

```bash
python tests/qa-crud/xlsx.py dump <file.xlsx> [max_rows]
python tests/qa-crud/xlsx.py fill <範本.xlsx> <輸出.xlsx> <rows.json> [起始列] [工作表]
python tests/qa-crud/xlsx.py create <輸出.xlsx> <rows.json> [工作表名稱]
```

## 測試資料的標記與清除

每次執行會產生一個 `QA` + 時間戳的標記（`L.TAG`），所有測試資料的名稱都帶這個前綴，讓測試資料在畫面上一眼可辨、也讓重跑不會撞唯一鍵。跑完用 `tests/qa-crud/cleanup.sql` 清除。

會破壞既有 demo 資料的檢查（刪除已有搭乘紀錄的車輛、刪除被引用的據點、刪除有出勤紀錄的司機）預設不執行，要跑得帶 `QA_DESTRUCTIVE=1`，跑完自行復原。

## 撰寫 suite 時的已知陷阱

這些是實際踩過、會讓測試假失敗的坑：

- **不要用 `Escape` 關閉 el-select 下拉或 el-date-picker 面板**：事件會冒泡把整個 `el-dialog` 一起關掉。改成再點一次觸發器（`L.selectOptions`）或點對話框標題列（`L.pickDate`）。
- **不要用 `document.querySelector(...).remove()` 清除 toast**：那些節點屬於 Vue，硬移除會弄壞 renderer，之後所有互動都會靜默失效。`L.clearToasts` 改成等它自己消失。
- **列定位要用整格完全相符**：`hasText: '甲'` 會同時命中「甲改」。用 `L.resolveRow()`，它先試 `td :text-is()` 再退回子字串。
- **確認對話框的按鈕文案各頁不同**（確定／刪除／確認發送），`L.confirmBox()` 一律點最後一顆按鈕。
- **建立司機／個案會做加解密，回應可能超過 1.5 秒**，`L.submit()` 已放寬到 2.6 秒。

## 判讀結果

`tests/qa-crud/reports/report-<timestamp>.json` 是每次執行的完整紀錄。優先看這幾個欄位：

- `suites[].httpErrors`：非 2xx 的回應。500 一律是 bug；400／409 要看訊息是否足以讓使用者自己修正。
- `pageErrors`：未被攔截的前端例外。
- `steps[].calls` 的 `req.body`：確認前端送出的值與畫面一致，沒有被靜默改寫或補上使用者沒選的預設值。
- `steps[].stepError`：該步驟本身失敗，先確認是測試腳本定位問題還是畫面真的壞掉。
