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
- 表格列操作按鈕（`link` + `size="small"`）的按鈕感視覺（淡底色 + 實色邊框，靜態即可見、非僅 hover 才出現 + hover 陰影／位移加強）由 `element-overrides.scss` 的 `.table-row-actions .el-button.is-link` 統一提供，只要包在 `<TableRowActions>` 裡就會生效，不要在頁面裡另外寫背景色。這組樣式的 `padding`／`border-width` 刻意維持原尺寸不變（只加深邊框飽和度與 hover 陰影），因為操作欄寬度是依按鈕數固定的（見下方表格欄位規則），放大按鈕本身會讓既有頁面的操作欄溢出裁切。
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
- 顏色一律來自 `src/lib/statusPresets.ts` 的 preset map。新增狀態值前先確認語意屬於哪個既有 preset，找不到情境才新增一組；不要在頁面裡寫三元式或硬寫色碼。
- 已知的 preset：`caseStatus`、`activeState`、`employmentState`、`batchImportStatus`、`fieldMappingStatus`、`completionStatus`、`role`。去程／回程等非狀態語意的方向欄用純文字，不套 `StatusTag`。
- 不使用 `effect="plain"` 的 `el-tag`（不吃全域色覆寫，會跟實心色系不一致）。

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
| `DataTablePage.vue` | 列表頁版面骨架。新增了 `title`／`description` props 與 `#header` slot（內部渲染 `PageHeader`），`pageSizes` 可覆寫（預設 `[10,20,50,100]`）。`.filter-card` / `.table-card` class 名稱不可改，e2e 測試依賴它們。欄位少、內容短的列表頁可傳 `max-width`（純數字 px＝頁面實際欄寬總和含操作欄 + 約 30-60px 緩衝）讓版面不硬撐滿；`el-table` 本身仍維持 `style="width: 100%"` 不要拿掉，不要改用 `width: fit-content`（會讓 `fixed="right"` 操作欄被裁切，已實測過）。矩陣型表格或欄位內容本身需要大量空間的頁面不套用 |

## 表格欄位

- 文字型欄位（姓名、地址、備註、信箱、名稱、說明、錯誤訊息）用 `min-width` + `show-overflow-tooltip`，不設固定 `width`。
- 固定 `width` 只給編號、日期、狀態、數值、操作欄。
- 操作欄一律 `fixed="right"`，寬度依按鈕數：1 顆 100px、2 顆 140px、3 顆 200px。
- 例外：欄位內容本身就需要換行呈現（例如異動前／異動後 diff）或欄位是結構化內容（tag／select 組合，非純文字）時，不套用 `show-overflow-tooltip`——它只會把內容截斷成一行，反而變差。
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
- 表格列有 2 顆以上操作按鈕時包一層 `<TableRowActions>`（`apps/web/src/components/TableRowActions.vue`），統一按鈕間距，按鈕本身仍照上面按鈕規範表寫。
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
