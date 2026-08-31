# Component Contract

apps/web 的具體規範表，照抄即可。改任何 `.vue` 前先查這份，這裡沒有的值才討論擴充；不要在頁面裡另外硬寫一組顏色、間距或按鈕組合。

Token 定義在 `apps/web/src/styles/tokens.scss`；共用元件在 `apps/web/src/components/`；狀態色 preset 在 `apps/web/src/lib/statusPresets.ts`。

## 按鈕

| 情境 | 寫法 |
|---|---|
| 頁面主要動作（新增） | `type="primary"` + `:icon="Plus"` |
| 頁面次要動作（匯入／匯出／下載範本／列印） | `plain`（不帶 type） |
| 破壞性批次動作 | `type="danger" plain` |
| 篩選列查詢 | `type="primary"`，無 icon |
| 篩選列重設 | 不帶 type，無 icon |
| 表格列查看／導航類操作（如「查看排班」「前往回報」） | `link type="info" size="small"`，無 icon |
| 表格列編輯／設定／指派／新增子項類操作（如「編輯」「指派車輛」「設定權限」「新增據點」） | `link type="primary" size="small"`，無 icon |
| 表格列破壞性操作（如「刪除」） | `link type="danger" size="small"`，無 icon |
| 表格列可逆狀態轉換操作（如「停用」，非破壞性但會改變資料狀態） | `link type="warning" size="small"`，無 icon |
| 對話框取消 | 不帶 type |
| 對話框確認 | `type="primary"` |
| 對話框破壞性確認 | `type="danger"` |

規則：

- 「編輯」不用 `type="success"`。綠色只留給真正的成功狀態。
- 屬性順序：`link → type → size → :icon`。
- 表格列操作按鈕（`link` + `size="small"`）的按鈕感視覺（淡底色 + 實色邊框，靜態即可見、非僅 hover 才出現 + hover 陰影／位移加強）由 `element-overrides.scss` 的 `.table-row-actions .el-button.is-link` 統一提供，只要包在 `<TableRowActions>` 裡就會生效，不要在頁面裡另外寫背景色。這組樣式的左右 `padding`（10px）／`border-width` 維持不變，因為操作欄寬度是依按鈕數固定的（見下方表格欄位規則），加寬按鈕本身會讓既有頁面的操作欄溢出裁切；上下 `padding` 為改善擁擠感已從 2px 調整為 5px，只影響列高不影響欄寬，可再視需要調整。
- 表格列操作按鈕的顏色**依 `type` 區分**（查看=info 灰藍／編輯=primary 藍／可逆狀態切換=warning 黃／刪除=danger 紅），不同功能一律用不同顏色辨識，不要收斂成單一中性色（曾試過全部改中性灰，使用者回饋「無法識別不同功能」而打回）。新頁面若有表格列操作按鈕，一律包在 `<TableRowActions>` 裡並依語意選對應 `type`，不要手刻背景色或跳過這個共用元件。
- icon 一律用綁定寫法 `:icon="Edit"` + 具名 import，不用字串 prop、不靠全域註冊。
- `size` 只有兩種：表格列與 chip 用 `small`，其餘不寫。`large` 只留登入頁主按鈕。
- **帶文字的按鈕不放 icon**，唯一例外是頁面主要「新增」按鈕的 `Plus`。

## 狀態呈現：`<StatusTag>`

```vue
<StatusTag :status="row.status" preset="caseStatus" />
<StatusTag :status="row.status" preset="activeState" variant="dot" />
```

- `variant="tag"`（預設）：`el-tag`，用於表格狀態欄與計數。
- `variant="dot"`：圓點 + 文字，用於可點擊切換的行內狀態。
- `variant="chip"`：圓角淡色底＋小圓點，`<span>` 自組樣式、不使用 `el-tag`。用於表格內需要語意色分級、但欄位同時疊了其他 class（如自訂欄寬／對齊）的情境——`element-overrides.scss` 對 `el-tag` 的覆寫是單一 class 選擇器（如 `.el-tag--info`），優先權低於 Element Plus 內建 light effect 的雙 class 疊加選擇器，疊加情境下會蓋不掉，改用 `chip` 可完全繞開這個問題（曾在系統操作紀錄「動作」欄踩到）。
- 顏色一律來自 `src/lib/statusPresets.ts` 的 preset map。新增狀態值前先確認語意屬於哪個既有 preset，找不到情境才新增一組；不要在頁面裡寫三元式或硬寫色碼。
- 已知的 preset：`caseStatus`、`activeState`、`employmentState`、`driverReportImportStatus`、`fieldMappingStatus`、`completionStatus`、`role`、`auditAction`。去程／回程等非狀態語意的方向欄用純文字，不套 `StatusTag`。
- 不使用 `effect="plain"` 的 `el-tag`（不吃全域色覆寫，會跟實心色系不一致）。
- 分級用的 preset（如 `auditAction`）色階以 3 色為上限：新增類（success）／高風險類（danger）／其餘一般操作全部收斂成單一 neutral。曾試過另外配一個 info 色給「一般修改類」，但淡藍字跟灰字在小字級下辨識不出差異，等於沒收斂——語意相近的分級寧可合併，不要為了「看起來更細緻」硬拆出容易混淆的顏色。

## 語意色 token（`tokens.scss`）

```css
--app-status-success-bg / -fg
--app-status-warning-bg / -fg
--app-status-danger-bg  / -fg
--app-status-info-bg    / -fg
--app-status-neutral-bg / -fg
```

角色色：`--app-role-{admin,dispatcher,staff,driver,viewer}-fg` 與 `-dot`。

主要色：`--app-primary` / `-dark` / `-light`（舊名 `--app-orange*` 為相容別名，新程式碼一律用 `--app-primary*`）。

## 間距 / 字級 / 圓角 token

```css
--app-space-1..8      /* 4/8/12/16/24/32px */
--app-font-xs..2xl    /* 12/13/14/16/20/24px */
--app-radius-xs/sm/md/lg/full
--app-header-height   /* 56px */
--app-aside-width / -collapsed
```

新增樣式前先看這份清單有沒有合用的值，不要另外寫 magic number。

## 共用元件

| 元件 | 用途 |
|---|---|
| `PageHeader.vue` | 頁面標題區，props `title`／`description`，slot `actions` |
| `TableRowActions.vue` | 包住表格列的多顆操作按鈕，統一間距 |
| `DialogFooter.vue` | 對話框底部按鈕，props `confirmText`／`cancelText`／`confirmType`／`loading`，emits `confirm`／`cancel` |
| `StatusTag.vue` | 見上一節 |
| `DataTablePage.vue` | 列表頁版面骨架。新增了 `title`／`description` props 與 `#header` slot（內部渲染 `PageHeader`），`pageSizes` 可覆寫（預設 `[10,20,50,100]`）。`.filter-card` / `.table-card` class 名稱不可改，e2e 測試依賴它們。欄位少、內容短的列表頁可傳 `max-width`（純數字 px＝頁面實際欄寬總和含操作欄 + 約 30-60px 緩衝）讓版面不硬撐滿；`el-table` 本身仍維持 `style="width: 100%"` 不要拿掉，不要改用 `width: fit-content`（會讓 `fixed="right"` 操作欄被裁切，已實測過）。矩陣型表格或欄位內容本身需要大量空間的頁面不套用 `max-width`。 |

## 表格欄位

- 視窗寬度足夠時，欄位內容一律單行完整顯示，不主動換行或提前截斷；欄寬不夠但頁面還有空間，應該讓表格本身變寬去容納內容，而不是用固定寬度把內容擠成多行或提前用省略號蓋掉——只有視窗真的不夠寬時，才輪到 `show-overflow-tooltip`（省略號＋hover）或水平捲動接手。
- 需要「表格寬度依內容縮起、不撐滿頁面，但視窗夠寬時不截斷內容」的頁面（例如欄位長度變化大、不適合套用固定 `max-width` 的矩陣型表格），用這組寫法：`<el-table table-layout="auto">` + scoped `:deep(.el-table) { width: max-content; }`。`.el-table` 本體是 `div`，內建 `width: 100%`，光拿掉 inline `style="width:100%"` 沒用（`div` 的 `width: auto` 預設仍撐滿容器），必須顯式蓋成 `max-content` 才會縮到「各欄寬度加總」。想自然伸展的那一欄不要加 `show-overflow-tooltip`（底層是 `overflow: hidden`，`table-layout="auto"` 量測欄寬時會把它當成可裁切內容直接給極小寬度），改用 `class-name` 搭配 scoped `:deep(.xxx-col .cell) { white-space: nowrap; }` 只鎖不換行；欄位加總寬度超過版面時交給 `DataTablePage` 既有的 `.table-container { overflow-x: auto }` 處理水平捲動，不做裁切省略。
- `#table` slot 內若放 `<el-row :gutter="16">` 統計卡片（如加油總筆數／總花費金額），`DataTablePage` 的 `.table-container` 是 `overflow-x: auto`，`el-row` 的負 margin（`-8px` 兩側）會撐出捲軸，且不論 `max-width` 開多大都會出現——這是負 margin 溢出，不是欄寬不足，調大 `max-width` 沒用。要幫該 `el-row` 加一個 class 把左右 margin 蓋回 `0 !important`（`el-col` 的 padding 仍在，卡片間距不受影響）。
- 文字型欄位（姓名、地址、備註、信箱、名稱、說明、錯誤訊息）用 `min-width` + `show-overflow-tooltip`，不設固定 `width`。
- 固定 `width` 只給編號、日期、狀態、數值、操作欄。
- 操作欄一律 `fixed="right"`，寬度依按鈕數：1 顆 100px、2 顆 140px、3 顆 200px。這組數字是抓 2 字按鈕文案（如「編輯」「刪除」）的估值；文案較長（如「前往回報」「查看排班」這種 4 字操作句）時，先用實際文案量測需要的寬度再加寬，不要硬套 100/140/200 導致按鈕被裁切。
- 例外：欄位內容本身就需要換行呈現（例如異動前／異動後 diff）或欄位是結構化內容（tag／select 組合，非純文字）時，不套用 `show-overflow-tooltip`——它只會把內容截斷成一行，反而變差。
- 欄位內容是「原始名稱文字 + 下拉選單 + 按鈕」這種複合結構（例如據點/車輛待關聯的行內編輯列）時，外層 flex 容器要 `flex-wrap: nowrap`（而非 `wrap`，否則欄寬不夠時會把選單/按鈕擠到下一行，即使頁面還有空間也一樣），且容器內每個子元素都要 `flex-shrink: 0`；其中純文字的 span（如「原始名稱：xxx」）額外要鎖 `white-space: nowrap`——`flex-wrap: nowrap` 只防止「元素之間」互相擠壓換行，沒有固定寬度的文字 span 在 flexbox 預設 shrink 行為下，文字本身還是會自己斷行，兩層都要鎖才會真的整行單行顯示。
- 每個列表表格都要有分頁，用 `DataTablePage` 內建的 `el-pagination`，不要自刻。

## 對話框與表單

- 對話框寬度分 4 級，一律用 responsive 寫法，不寫死 px：

  ```
  小型表單   width="min(480px, calc(100vw - 32px))"
  一般表單   width="min(600px, calc(100vw - 32px))"
  含表格     width="min(820px, calc(100vw - 32px))"
  匯入預覽   width="min(960px, calc(100vw - 32px))"
  ```

  `el-drawer` 用 `size="min(Npx, 92vw)"` 同樣道理（drawer 不吃 `element-overrides.scss` 裡 ≤720px 的對話框寬度補救規則，必須自己處理）。
- 對話框／抽屜的取消／確認鈕一律用 `<DialogFooter>`（`apps/web/src/components/DialogFooter.vue`），不要手刻 `<el-button>` 配對。支援 `confirmText`／`cancelText`／`confirmType`／`loading`／`confirmDisabled`／`showConfirm`（唯讀模式時設 `false` 隱藏確認鈕，例如已過編輯期限的表單）。
- 表格列操作按鈕一律包一層 `<TableRowActions>`（`apps/web/src/components/TableRowActions.vue`），即使只有 1 顆也要包——`.table-row-actions .el-button.is-link` 的可見按鈕樣式（淡底色＋邊框）依附在這層 class 上，沒包就會退化成純文字連結，讀不出是按鈕。按鈕本身仍照上面按鈕規範表寫。
- icon 元件一律具名 import + 綁定寫法，`main.ts` 已移除全域註冊 Element Plus icon；新頁面若用到 `<el-icon><Xxx /></el-icon>` 或 `:icon="Xxx"`／`:prefix-icon="Xxx"`，必須在該檔案自己 `import { Xxx } from '@element-plus/icons-vue'`，否則畫面會顯示不出圖示（不是編譯錯誤，是執行期才會發現，型別檢查抓不到）。
- 內容可能超過視窗高度的對話框，表單或內容容器加 `class="dialog-scroll-form"`（定義在 `tokens.scss`，`max-height: min(560px, 70vh); overflow-y: auto`），不要自己重寫一份。
- `label-width` 收斂成 3 級：短表單 90px、一般 110px、長標籤 140px。
- `el-row` 的 `gutter` 統一為 16。
- 篩選列的關鍵字搜尋輸入框統一 `style="width: 240px"`，不要每頁自己挑一個數字。

## 確認文案

| 情境 | 文字 |
|---|---|
| 新增 | 新增 |
| 修改既有資料 | 儲存 |
| 送出不可逆操作（匯入、覆蓋、裁決） | 確認送出 |
| 取消 | 取消 |

`ElMessageBox.confirm` 的刪除確認統一用「刪除」+ `confirmButtonClass: 'el-button--danger'`。

改文案時同步檢查 `apps/web/tests/e2e/*.spec.ts` 有沒有用 `getByRole('button', { name: ... })` 依賴舊文字。
