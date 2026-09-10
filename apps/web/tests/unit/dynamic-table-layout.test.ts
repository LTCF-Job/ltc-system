import assert from 'node:assert/strict'
import test from 'node:test'

/**
 * 模擬 useDynamicTableLayout 的計算核心
 */
function evaluateTableLayout(
  containerWidth: number,
  columns: Array<{
    width?: number
    minWidth?: number
    headerWidth?: number
    maxCellWidth?: number
  }>,
  buffer = 6,
  hasRows = true,
) {
  let totalWidth = 0
  const colWidths: number[] = []

  for (const col of columns) {
    const declaredWidth = col.width && col.width > 0 ? col.width : 0
    const declaredMinWidth = col.minWidth && col.minWidth > 0 ? col.minWidth : 0
    const headerWidth = col.headerWidth && col.headerWidth > 0 ? col.headerWidth : 0
    const maxCellWidth = col.maxCellWidth && col.maxCellWidth > 0 ? col.maxCellWidth : 0

    const contentNeeded = Math.max(headerWidth, maxCellWidth) + buffer
    let finalWidth = 0
    if (declaredWidth > 0) {
      finalWidth = Math.max(declaredWidth, contentNeeded)
    } else if (!hasRows) {
      finalWidth = Math.max(headerWidth + buffer, declaredMinWidth, 60)
    } else {
      finalWidth = Math.max(contentNeeded, 60)
    }

    colWidths.push(finalWidth)
    totalWidth += finalWidth
  }

  const isExpanded = containerWidth >= totalWidth

  return {
    requiredWidth: totalWidth,
    containerWidth,
    colWidths,
    isExpanded,
    isScrollable: !isExpanded,
    styleWidth: `${totalWidth}px`,
    styleMinWidth: `${totalWidth}px`,
    styleMaxWidth: isExpanded ? '100%' : 'none',
  }
}

test('dynamic table layout compacts table width when content is short without excessive whitespace from declaredMinWidth', () => {
  // 模擬照護人員管理表格：
  // 過去因模板人為宣告 min-width: 220 (單位), 180 (備註), 140 (聯絡), 90 (類型) 造成巨大空白。
  // 現在依真實內容感知自適應，緊湊收縮至最適單行寬度。
  const columns = [
    { minWidth: 90, headerWidth: 48, maxCellWidth: 50 },   // 類型 (個管)
    { minWidth: 220, headerWidth: 48, maxCellWidth: 111 }, // 單位 (QA300581單位)
    { minWidth: 120, headerWidth: 48, maxCellWidth: 132 }, // 姓名 (QA300581照護A)
    { minWidth: 140, headerWidth: 72, maxCellWidth: 106 }, // 聯絡方式
    { minWidth: 180, headerWidth: 48, maxCellWidth: 128 }, // 備註
    { minWidth: 120, headerWidth: 48, maxCellWidth: 96 },  // 狀態
    { minWidth: 140, headerWidth: 48, maxCellWidth: 140 }, // 操作欄
  ]

  // 大螢幕寬度 1400px，表格應維持緊湊的約 850px，不被 220/180 等過大 min-width 或全螢幕寬度撐開
  const result = evaluateTableLayout(1400, columns)

  assert.equal(result.isExpanded, true)
  assert.equal(result.isScrollable, false)
  // 單位欄位不再是 220px，而是縮到 117px (111+6)
  assert.equal(result.colWidths[1], 117)
  // 備註欄位不再是 180px，而是縮到 134px (128+6)
  assert.equal(result.colWidths[4], 134)
  // 類型欄位不再是 90px，而是縮到 60px (底限保證)
  assert.equal(result.colWidths[0], 60)
  // 總寬度大幅收斂，杜絕欄位間無意義的空洞
  assert.ok(result.requiredWidth < 870)
})

test('dynamic table layout automatically expands column width when cell content is longer than declared min-width', () => {
  // 模擬司機管理：司機姓名宣告 min-width 110，但實際長姓名需要 151px
  const columns = [
    { minWidth: 110, headerWidth: 72, maxCellWidth: 151 }, // 司機姓名
    { minWidth: 140, headerWidth: 84, maxCellWidth: 97 },  // 身分證字號
    { minWidth: 70, headerWidth: 48, maxCellWidth: 29 },   // 性別
  ]

  const result = evaluateTableLayout(1200, columns)

  // 司機姓名欄位應自動「展開」至 157px（151 + 6 buffer），絕不被 110px 壓縮或截斷出 ...
  assert.equal(result.colWidths[0], 157)
  assert.ok(result.colWidths[0] > 110)
  assert.equal(result.isExpanded, true)
})

test('dynamic table layout compacts index and ID columns without block container expansion', () => {
  // 模擬車輛管理「編號」與司機管理「身分證字號」：
  // 編號 (header 48px, cell 39px) -> 54px, 依底限 60px 呈現，不再被 block div 膨脹至 112px
  const columns = [
    { minWidth: 70, headerWidth: 48, maxCellWidth: 39 },  // 編號
    { minWidth: 140, headerWidth: 84, maxCellWidth: 97 }, // 身分證字號
  ]

  const result = evaluateTableLayout(1200, columns)

  assert.equal(result.colWidths[0], 60)
  assert.equal(result.colWidths[1], 103)
})

test('dynamic table layout locks min-width and enables horizontal scroll when container is narrow', () => {
  // 模擬系統操作紀錄表格：欄位需求加總
  const columns = [
    { minWidth: 170, headerWidth: 150, maxCellWidth: 140 },
    { minWidth: 110, headerWidth: 90, maxCellWidth: 90 },
    { minWidth: 150, headerWidth: 80, maxCellWidth: 130 },
    { minWidth: 110, headerWidth: 100, maxCellWidth: 100 },
    { minWidth: 160, headerWidth: 110, maxCellWidth: 120 },
    { minWidth: 130, headerWidth: 90, maxCellWidth: 80 },
    { minWidth: 120, headerWidth: 80, maxCellWidth: 90 },
  ]

  // 容器寬度只有 750px
  const result = evaluateTableLayout(750, columns)

  assert.equal(result.isExpanded, false)
  assert.equal(result.isScrollable, true)
  assert.ok(result.requiredWidth > 750)
})

test('dynamic table layout falls back to declared min-width when table has no data rows', () => {
  // 表格尚無資料列時，應保留表頭與 declaredMinWidth 作為基線
  const columns = [
    { minWidth: 180, headerWidth: 50 },
  ]

  const result = evaluateTableLayout(1200, columns, 6, false)

  assert.equal(result.colWidths[0], 180)
})
