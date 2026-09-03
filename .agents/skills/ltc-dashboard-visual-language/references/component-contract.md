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
- 操作欄按鈕 hover 的位移／陰影動畫已透過 `element-overrides.scss` 的全域規則 `.el-table .cell:has(.table-row-actions) { overflow: visible; }` 放行，不必每個頁面另外處理——Element Plus 的 `.cell` 預設 `overflow: hidden`（給文字省略號截斷用），操作欄按鈕不需要這個效果，卻一樣被裁切，導致 hover 動畫在列高邊界被切掉、看起來沒有完整顯示。新增操作欄時只要照規範包 `<TableRowActions>`，不需要自己補這條 CSS；若動畫仍被裁切，先確認是不是欄位額外加了 `show-overflow-tooltip` 或自訂 `overflow: hidden`，那些情境會蓋掉這條全域規則。
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
--app-font-xs..3xl    /* 13/13/14/16/20/24/34px */
--app-label-size / --app-label-tracking   /* 11px / 0.08em，大寫微標籤 */
--app-radius-xs/sm/md/lg/full
--app-header-height   /* 56px */
--app-aside-width / -collapsed
--app-nav-*           /* 側欄淺色面板 */
--app-bar-track / --app-bar-height        /* 容量長條 */
```

新增樣式前先看這份清單有沒有合用的值，不要另外寫 magic number。

## 營運面板樣式（`element-overrides.scss`）

儀表板與清單頁的面板共用這組 class，不要在頁面裡重刻一份。

| Class | 用途 |
|---|---|
| `.app-micro-label` | 大寫微標籤。KPI 標籤與側欄分組標題共用，色為 `--app-text-secondary`（11px 小字用 `--app-text-muted` 只有 3.05:1，不符 WCAG AA） |
| `.app-metric-value` | KPI 數值。一張卡只放一個，是卡片裡唯一的強元素 |
| `.app-capacity-row` | 容量列外框，`grid` 三欄：標籤 / 軌道 / 右對齊數值 |
| `.app-capacity-track` + `.app-capacity-fill` | 長條軌道與填色。兩者都已設 `display: block`，用 `<span>` 承載時不必另外處理 |
| `.app-panel-footer` + `.app-panel-footer-link` | 面板底部行動（如「查看全部」）。置中、大寫、與內容以細線分隔 |

規則：

- 長條填色寬度用 inline `style="width: N%"` 綁定，比例基準要在程式碼註解裡寫清楚是「相對什麼」。目前儀表板車輛趟數是相對最忙的一台車，不是達成率——右側數值才是實際值。
- 填色預設 `--app-primary`。趟數、件數這類非風險語意的量值一律留在藍色；只有真正代表風險分級時才加 `.is-high` / `.is-mid` / `.is-low`。
- 「查看全部」放面板底部，不放標題列右上。

## 共用元件

| 元件 | 用途 |
|---|---|
| `PageHeader.vue` | 頁面標題區，props `title`／`description`，slot `actions` |
| `TableRowActions.vue` | 包住表格列的多顆操作按鈕，統一間距 |
| `DialogFooter.vue` | 對話框底部按鈕，props `confirmText`／`cancelText`／`confirmType`／`loading`，emits `confirm`／`cancel` |
| `StatusTag.vue` | 見上一節 |
| `DataTablePage.vue` | 列表頁版面骨架。新增了 `title`／`description` props 與 `#header` slot（內部渲染 `PageHeader`），`pageSizes` 可覆寫（預設 `[10,20,50,100]`）。`.filter-card` / `.table-card` class 名稱不可改，e2e 測試依賴它們。欄位少、內容短的列表頁可傳 `max-width`（純數字 px＝頁面實際欄寬總和含操作欄 + 約 30-60px 緩衝）讓版面不硬撐滿；`el-table` 本身仍維持 `style="width: 100%"` 不要拿掉，不要改用 `width: fit-content`（會讓 `fixed="right"` 操作欄被裁切，已實測過）。矩陣型表格或欄位內容本身需要大量空間的頁面不套用 `max-width`。**改動既有頁面的欄位（新增／移除欄、加寬既有欄）時，若該頁已經有 `max-width`，要重新核算欄寬總和並更新這個數字**——它不會隨新增欄位自動放寬。2026-09-03 在 `SiteListView.vue`／`CaregiverListView.vue` 加入啟用停用狀態欄後忘記回頭調整，`max-width="940"`／`"990"` 小於新的欄寬總和，即使視窗還有空間，表格仍被鎖死的容器逼出橫向卷軸；當下曾誤判成「不該限制寬度」直接拿掉 `max-width`，結果表格改成填滿整個視窗寬度，跟「欄位少、內容短維持既有 `max-width`」的設計初衷相反——正確處理是重算總和（改完後兩頁分別是 `1020`／`1070`），不是移除。 |

## 表格欄位

- 視窗寬度足夠時，欄位內容一律單行完整顯示，不主動換行或提前截斷；欄寬不夠但頁面還有空間，應該讓表格本身變寬去容納內容，而不是用固定寬度把內容擠成多行或提前用省略號蓋掉——只有視窗真的不夠寬時，才輪到 `show-overflow-tooltip`（省略號＋hover）或水平捲動接手。
- **任何**套用 `<el-table table-layout="auto">` 的頁面（不限於下一條的「縮起不撐滿」pattern），只要欄位可能換行，都要同時做到兩件事，缺一都不會生效：① 固定 `width` 改成 `min-width`；② 依欄位有沒有自訂 `<template>#default` 補對應的 nowrap 鎖定——有自訂 template 就在裡面的 `<span>` 直接補 `white-space: nowrap`（例如 `.xxx-value { white-space: nowrap; }`）；沒有自訂 template（純 `<el-table-column prop="..." min-width="N" />`）就加 `class-name="xxx-col"`，並在同檔案 `<style scoped>` 補一條 `:deep(.xxx-col .cell) { white-space: nowrap; }`。`table-layout="auto"` 底下固定 `width` 只會鎖死欄寬讓文字換行、不會依內容自動撐開；`min-width` 欄位若沒鎖 nowrap，欄寬吃緊時一樣會被壓成逐字換行——這正是 2026-09-01 在 `CaregiverListView.vue` 踩到、後來排查全站 10 個頁面都中招的漏洞：只加了 `class-name` 卻忘記補對應的 `:deep()` CSS 規則。**改完後務必自我檢查**：對每個新加的 `class-name="xxx-col"`，同檔案要能 grep 到對應的 `:deep(.xxx-col .cell)` 規則；反之每個新加的 `:deep()` nowrap 規則，也要能在 template 找到對應的 `class-name`——兩邊對不上就是漏了一半，不算改完。想自然伸展的欄位另外要注意：不加 `show-overflow-tooltip`（底層是 `overflow: hidden`，`table-layout="auto"` 量測欄寬時會把它當成可裁切內容直接給極小寬度）。欄位加總寬度超過版面時交給 `DataTablePage` 既有的 `.table-container { overflow-x: auto }` 處理水平捲動，不做裁切省略。
- 需要「表格寬度依內容縮起、不撐滿頁面，但視窗夠寬時不截斷內容」的頁面（例如欄位長度變化大、不適合套用固定 `max-width` 的矩陣型表格），在上一條的基礎上再加：scoped `:deep(.el-table) { width: max-content; }`。`.el-table` 本體是 `div`，內建 `width: 100%`，光拿掉 inline `style="width:100%"` 沒用（`div` 的 `width: auto` 預設仍撐滿容器），必須顯式蓋成 `max-content` 才會縮到「各欄寬度加總」。**`<el-table>` 標籤上如果還留著 `style="width: 100%"`，這條 `:deep()` 規則會被 inline style 蓋過去、完全不生效**（inline style 優先權高於任何 class 選擇器）——套用這個 pattern 時務必把 inline `style="width: 100%"` 一併拿掉，不是「加規則就好」。這個 pattern 跟 `DataTablePage` 的 `:max-width` 數字上限是兩套不同機制，不要混用：`:max-width` 是開發者手動估算的欄寬總和，靠 `min-width: max-content` 防止被壓縮，並非「縮到內容」；只有這條 `width: max-content` scoped 規則才是真正依內容縮起。
- 這個 pattern 若用在**沒有包在 `DataTablePage` 裡**的表格（例如「待維護」頁籤常見的 `<div class="xxx-panel"><el-table>` 這種直接寫在頁面裡、不經過 `DataTablePage` 的區塊），該容器 div 要自己補 `overflow-x: auto`——`DataTablePage` 的 `.table-container` 內建這條規則，這種獨立面板沒有，內容超版面寬時沒有 `overflow-x: auto` 頂多是撐破面板讓整個頁面跳出橫向卷軸，而不是卷軸包在面板內；2026-09 的個案清單／照護人員管理待維護頁籤都踩過這個漏洞。
- **`min-width` prop 在這個 `width: max-content` pattern 底下不是真正的 CSS 下限，只是拿去算 Element Plus 內部「表格總寬度預算」的其中一個數字。** 實測（`element-plus/es/components/table/src/h-helper.mjs`）：`table-layout="auto"` 時，只有宣告 `width` 的欄位會被塞進 `<col style="width:Npx">`；只宣告 `min-width` 的欄位完全沒有任何 `<col>` inline 樣式，最終欄寬 100% 由瀏覽器原生 auto-layout 依「該欄實際內容」決定——欄位當筆內容比 `min-width` 短（例如原本有原始名稱要行內編輯、這筆卻已關聯完成顯示「-」）時，欄寬就會被壓到只剩內容本身那幾 px，跟同一張表其他撐開的欄位比例明顯不一致，這正是「這欄沒有展開」的成因。要讓 `min-width` 真的變成下限，一律照上一條的 class-name 做法，額外補一條 `:deep(.xxx-col .cell) { min-width: Npx; }`（跟 nowrap 那條共用同一個 class-name 即可，兩條規則疊在一起）；固定 `width` 欄（含操作欄）在這個 pattern 下同理也不是下限，一樣用 `class-name` + `:deep() { min-width: Npx; }` 鎖住，否則操作欄可能被壓到比 component-contract 規定的按鈕欄寬（100/140/200px）更窄。`show-overflow-tooltip` 欄位若也想让内容完整撐開而非截斷，除了拿掉 `show-overflow-tooltip` 本身，同樣要補這條 `min-width` CSS，否則 Element Plus 用來換算表格總寬度預算的還是 `min-width` prop 那個數字，內容一樣可能比預算更寬而被截斷或溢出。2026-09 的 `RideIssuesView.vue`（表單匯入異常頁籤的服務日期／錯誤訊息欄）與 `CaseListView.vue`／`CaregiverListView.vue` 待維護頁籤（回程車輛／單位欄）都踩過這個漏洞。**這條規則是逐欄位生效，不是逐表格生效**：同一張表格裡沒補這條 CSS 的欄位不會因為隔壁欄位補了就一起受惠，2026-09 這幾個頁面都是使用者一次抓一欄回報（個案編號、服務日期各自回報過一次）才補齊——套用 `width: max-content` pattern 時，**該表格內每一個 `<el-table-column>`（操作欄含在內；`show-overflow-tooltip` 欄位除外，它們的截斷是刻意設計）都要在改完當下就一次補齊 `class-name` + `:deep() min-width`**，不要等使用者一個一個欄位回報才補。
- 修完不該出現的兩種副作用：① 一般寬度視窗下跳出不必要的橫向卷軸（代表欄寬加總過頭或量測錯誤）；② 內容明明很短卻把表格拉到滿版寬度（代表不該加大 `min-width` 或不該拿掉既有 `max-width`）。只針對「視窗夠寬、內容卻被壓成多行」這個症狀處理，不要順手加大整體欄寬或移除合理的 `max-width`——欄位少、內容短的列表頁維持既有 `max-width` 是刻意設計，不是待修的問題。
- `#table` slot 內若放 `<el-row :gutter="16">` 統計卡片（如加油總筆數／總花費金額），`DataTablePage` 的 `.table-container` 是 `overflow-x: auto`，`el-row` 的負 margin（`-8px` 兩側）會撐出捲軸，且不論 `max-width` 開多大都會出現——這是負 margin 溢出，不是欄寬不足，調大 `max-width` 沒用。要幫該 `el-row` 加一個 class 把左右 margin 蓋回 `0 !important`（`el-col` 的 padding 仍在，卡片間距不受影響）。
- 文字型欄位（姓名、地址、備註、信箱、名稱、說明、錯誤訊息）用 `min-width` + `show-overflow-tooltip`，不設固定 `width`。
- 固定 `width` 只給編號、日期、狀態、數值、操作欄；套用 `table-layout="auto"` 的頁面例外，一律改 `min-width` + nowrap class（見上面「任何套用 `table-layout="auto"` 的頁面」那條）。
- 操作欄一律 `fixed="right"`，寬度依按鈕數：1 顆 100px、2 顆 140px、3 顆 200px。這組數字是抓 2 字按鈕文案（如「編輯」「刪除」）的估值；文案較長（如「前往回報」「查看排班」這種 4 字操作句）時，先用實際文案量測需要的寬度再加寬，不要硬套 100/140/200 導致按鈕被裁切。
- 例外：欄位內容本身就需要換行呈現（例如異動前／異動後 diff）或欄位是結構化內容（tag／select 組合，非純文字）時，不套用 `show-overflow-tooltip`——它只會把內容截斷成一行，反而變差。
- 欄位內容是「原始名稱文字 + 下拉選單 + 按鈕」這種複合結構（例如據點/車輛待關聯的行內編輯列）時，外層 flex 容器要 `flex-wrap: nowrap`（而非 `wrap`，否則欄寬不夠時會把選單/按鈕擠到下一行，即使頁面還有空間也一樣），且容器內每個子元素都要 `flex-shrink: 0`；其中純文字的 span（如「原始名稱：xxx」）額外要鎖 `white-space: nowrap`——`flex-wrap: nowrap` 只防止「元素之間」互相擠壓換行，沒有固定寬度的文字 span 在 flexbox 預設 shrink 行為下，文字本身還是會自己斷行，兩層都要鎖才會真的整行單行顯示。
- 每個列表表格都要有分頁，用 `DataTablePage` 內建的 `el-pagination`，不要自刻。
- 有啟用停用設計的主檔表格，狀態欄放在所有資料欄位之後、操作欄之前（即倒數第二欄）。地區／單位／車輛／司機／照護人員管理五個主檔清單頁都依此排列。

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
