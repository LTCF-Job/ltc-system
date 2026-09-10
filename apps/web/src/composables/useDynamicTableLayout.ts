import { ref, onMounted, onBeforeUnmount, nextTick, type Ref, unref } from 'vue'

export interface DynamicTableLayoutOptions {
  /** 安全寬度額外緩衝（像素），預設 10 */
  buffer?: number
  /** 當完成計算與套用後的回呼 */
  onCalculated?: (result: {
    requiredWidth: number
    containerWidth: number
    windowWidth: number
    isExpanded: boolean
  }) => void
}

export interface ColumnWidthInfo {
  label: string
  finalWidth: number
}

/**
 * 測量單元格實際內容所需寬度（排除區塊容器自身寬度干擾）
 *
 * 關鍵技術原理：
 * 瀏覽器中區塊元素（如 div.cell）未溢出時，scrollWidth 等於 clientWidth，
 * 若直接讀取 scrollWidth 會把前次展開的寬度當成內容寬度，導致數值無限膨脹放大。
 * 故採用 Range 取得字元 glyphs 的精確物理渲染邊界；動作按鈕與藥丸標籤則以其真實 bounding rect 測量。
 */
function measureCellContentWidth(cell: HTMLElement, range: Range): number {
  // 1. 操作按鈕組（固定操作區塊）
  const actions = cell.querySelector<HTMLElement>('.table-row-actions')
  if (actions) {
    return Math.ceil(actions.getBoundingClientRect().width) + 24
  }

  // 2. 狀態切換膠囊按鈕（啟用／停用狀態）
  const pill = cell.querySelector<HTMLElement>('.status-toggle-pill')
  if (pill) {
    return Math.ceil(pill.getBoundingClientRect().width) + 24
  }

  // 3. 行內下拉選取器（InlineOptionPicker，如單位、駕駛司機、指派車輛）
  // 關鍵：使用 scrollWidth 獲取文字真實完整寬度，並補足按鈕 padding(14px) + border(2px) + gap(6px) + icon(12px) + cell padding(24px)
  // 杜絕文字被截斷為省略號 (...) 的問題！
  const picker = cell.querySelector<HTMLElement>('.inline-picker-trigger')
  if (picker) {
    const val = picker.querySelector<HTMLElement>('.inline-picker-value')
    const textWidth = val ? Math.ceil(val.scrollWidth) : Math.ceil(picker.getBoundingClientRect().width)
    return textWidth + 62
  }

  // 4. 一般 Element Plus 元件（按鈕、標籤、下拉選單）
  const widget = cell.querySelector<HTMLElement>('.el-button, .el-tag, .el-dropdown')
  if (widget) {
    return Math.ceil(widget.getBoundingClientRect().width) + 24
  }

  // 5. 純文字內容：透過 TreeWalker 尋找實際具備非空白字元的首尾文字節點
  // 避免選取到 block 容器（例如 Element Plus type="index" 產生的 <div>1</div>），
  // 導致 Range 框選到佔滿寬度的 block 元素而誤判為極大寬度。
  if (typeof document !== 'undefined' && typeof document.createTreeWalker === 'function') {
    const walker = document.createTreeWalker(cell, NodeFilter.SHOW_TEXT, null)
    let firstText: Node | null = null
    let lastText: Node | null = null
    let node: Node | null
    while ((node = walker.nextNode())) {
      if (node.textContent && node.textContent.trim().length > 0) {
        if (!firstText) firstText = node
        lastText = node
      }
    }

    if (firstText && lastText) {
      range.setStart(firstText, 0)
      range.setEnd(lastText, lastText.textContent?.length || 0)
      return Math.ceil(range.getBoundingClientRect().width) + 24
    }
  }

  range.selectNodeContents(cell)
  const fallback = range.getBoundingClientRect().width
  return fallback > 0 ? Math.ceil(fallback) + 24 : 0
}

/**
 * 動態計算表格欄位自適應內容寬度 Composable
 *
 * 核心保證：
 * 1. 欄位內容太寬時（如長司機姓名、長地址）：自動展開欄位寬度，絕不截斷 (...)、絕不折行。
 * 2. 欄位內容較短時：維持自然緊湊總寬度，不硬性拉伸至 100% 網頁寬度，杜絕欄位間大量空白，操作按鈕緊鄰內容。
 * 3. 容器寬度小於總所需寬度時：自動啟動橫向捲軸，標題與資料單行完整呈現。
 */
export function useDynamicTableLayout(
  containerRef: Ref<HTMLElement | null | undefined>,
  options: DynamicTableLayoutOptions = {},
) {
  const isExpanded = ref(true)
  const isScrollable = ref(false)
  const requiredWidth = ref(0)
  const currentContainerWidth = ref(0)

  let resizeObserver: ResizeObserver | null = null
  let mutationObserver: MutationObserver | null = null
  let resizeTimer: number | null = null
  let isUpdating = false
  let lastAppliedKey = ''

  function getTableEl(): HTMLElement | null {
    const container = unref(containerRef)
    if (!container) return null
    if (container.classList.contains('el-table')) return container
    return container.querySelector<HTMLElement>('.el-table')
  }

  function getTableComponent(tableEl: HTMLElement): any {
    const comp = (tableEl as any).__vueParentComponent
    return comp?.exposed || comp?.proxy || comp?.ctx || null
  }

  /** 計算並套用各欄位依內容感知之最適寬度 */
  function applyLayout() {
    if (isUpdating) return
    const container = unref(containerRef)
    if (!container) return

    const tableEl = getTableEl()
    if (!tableEl) return

    const containerWidth = container.clientWidth
    if (containerWidth <= 0) return

    const tableComp = getTableComponent(tableEl)
    const rawColumns: any[] = tableComp?.store?.states?.columns?.value || []
    const flattenColumns = rawColumns.filter((c: any) => !c.isColumnGroup)

    const headerWrapper = tableEl.querySelector<HTMLElement>('.el-table__header-wrapper') || tableEl
    const headerCells = Array.from(
      headerWrapper.querySelectorAll<HTMLElement>('thead tr:first-child th.el-table__cell'),
    )

    const rows = Array.from(
      tableEl.querySelectorAll<HTMLElement>('.el-table__body-wrapper tbody tr.el-table__row'),
    ).slice(0, 30)

    const colCount = Math.max(flattenColumns.length, headerCells.length)
    if (colCount === 0) return

    const range = document.createRange()
    const optimalWidths: number[] = []
    let totalWidth = 0

    for (let i = 0; i < colCount; i++) {
      const col = flattenColumns[i]
      const th = headerCells[i]

      // 1. 測量表頭單元格所需寬度（文字 + 排序 caret + 篩選 trigger + padding）
      let headerWidth = 0
      if (th) {
        const thCell = th.querySelector<HTMLElement>('.cell') || th
        let textWidth = 0
        if (typeof document !== 'undefined' && typeof document.createTreeWalker === 'function') {
          const walker = document.createTreeWalker(thCell, NodeFilter.SHOW_TEXT, null)
          let firstText: Node | null = null
          let lastText: Node | null = null
          let node: Node | null
          while ((node = walker.nextNode())) {
            if (node.textContent && node.textContent.trim().length > 0) {
              if (!firstText) firstText = node
              lastText = node
            }
          }
          if (firstText && lastText) {
            range.setStart(firstText, 0)
            range.setEnd(lastText, lastText.textContent?.length || 0)
            textWidth = Math.ceil(range.getBoundingClientRect().width)
          }
        }
        if (textWidth === 0) {
          range.selectNodeContents(thCell)
          textWidth = Math.ceil(range.getBoundingClientRect().width)
        }
        const hasSort = !!th.querySelector('.caret-wrapper')
        const hasFilter = !!th.querySelector('.el-table__column-filter-trigger')
        headerWidth = textWidth + 24 + (hasSort ? 24 : 0) + (hasFilter ? 20 : 0)
      }

      // 2. 抽樣前 30 列單元格文字/元件的真實渲染寬度，取最大值
      let maxCellWidth = 0
      for (const row of rows) {
        const td = row.children[i] as HTMLElement | undefined
        if (!td) continue
        const cell = td.querySelector<HTMLElement>('.cell') || td
        const measured = measureCellContentWidth(cell, range)
        if (measured > maxCellWidth) {
          maxCellWidth = measured
        }
      }

      if ((col as any)._originalDeclaredWidth === undefined) {
        ;(col as any)._originalDeclaredWidth =
          typeof col?.rawColumn?.width === 'number' && col.rawColumn.width > 0
            ? col.rawColumn.width
            : typeof col?.width === 'number' && col.width > 0
              ? col.width
              : 0
        ;(col as any)._originalDeclaredMinWidth =
          typeof col?.rawColumn?.minWidth === 'number' && col.rawColumn.minWidth > 0
            ? col.rawColumn.minWidth
            : typeof col?.minWidth === 'number' && col.minWidth > 0
              ? col.minWidth
              : 0
      }
      const declaredWidth = (col as any)._originalDeclaredWidth || 0
      const declaredMinWidth = (col as any)._originalDeclaredMinWidth || 0

      // 3. 欄位寬度裁決（內容感知自適應，徹底解決留白過多與壓縮截斷問題）：
      const buffer = options.buffer ?? 6
      const contentNeeded = Math.max(headerWidth, maxCellWidth) + buffer
      let finalWidth = 0
      if (declaredWidth > 0) {
        finalWidth = Math.max(declaredWidth, contentNeeded)
      } else if (rows.length === 0) {
        finalWidth = Math.max(headerWidth + buffer, declaredMinWidth, 60)
      } else {
        finalWidth = Math.max(contentNeeded, 60)
      }

      optimalWidths.push(finalWidth)
      totalWidth += finalWidth

      // 同步寫入 Element Plus 欄位狀態與 DOM colgroup
      if (col) {
        col.minWidth = finalWidth
        col.width = finalWidth
      }
    }

    const currentKey = `${containerWidth}:${totalWidth}:${optimalWidths.join(',')}`
    if (currentKey === lastAppliedKey) {
      return
    }
    lastAppliedKey = currentKey

    isUpdating = true
    try {
      // 同步更新 DOM 中所有 <colgroup><col> 寬度屬性
      const colElements = Array.from(tableEl.querySelectorAll<HTMLElement>('colgroup col'))
      for (let i = 0; i < colElements.length; i++) {
        const colIdx = i % colCount
        const w = optimalWidths[colIdx]
        if (w) {
          colElements[i].setAttribute('width', String(w))
          colElements[i].style.width = `${w}px`
        }
      }

      requiredWidth.value = totalWidth
      currentContainerWidth.value = containerWidth

      tableEl.style.width = `${totalWidth}px`
      tableEl.style.minWidth = `${totalWidth}px`

      if (containerWidth >= totalWidth) {
        isExpanded.value = true
        isScrollable.value = false
        tableEl.style.maxWidth = '100%'

        container.classList.add('table-compact')
        container.classList.remove('table-scrollable')
        container.style.overflowX = 'visible'
      } else {
        isExpanded.value = false
        isScrollable.value = true
        tableEl.style.maxWidth = 'none'

        container.classList.add('table-scrollable')
        container.classList.remove('table-compact')
        container.style.overflowX = 'auto'
      }

      if (tableComp && typeof tableComp.doLayout === 'function') {
        tableComp.doLayout()
      }

      options.onCalculated?.({
        requiredWidth: totalWidth,
        containerWidth,
        windowWidth: window.innerWidth,
        isExpanded: isExpanded.value,
      })
    } finally {
      isUpdating = false
    }
  }

  function scheduleLayout() {
    if (resizeTimer !== null) {
      window.cancelAnimationFrame(resizeTimer)
    }
    resizeTimer = window.requestAnimationFrame(() => {
      applyLayout()
      resizeTimer = null
    })
  }

  onMounted(() => {
    nextTick(() => {
      applyLayout()

      const container = unref(containerRef)
      if (container) {
        resizeObserver = new ResizeObserver(() => {
          scheduleLayout()
        })
        resizeObserver.observe(container)

        // 監聽表格資料更新（如非同步載入、分頁切換）
        mutationObserver = new MutationObserver((mutations) => {
          // 只在非屬性變更（例如資料列新增/刪除）時觸發，避免 doLayout 更新屬性導致遞迴
          const hasStructuralChange = mutations.some((m) => m.type === 'childList')
          if (hasStructuralChange) {
            scheduleLayout()
          }
        })
        mutationObserver.observe(container, {
          childList: true,
          subtree: true,
        })
      }

      window.addEventListener('resize', scheduleLayout, { passive: true })
    })
  })

  onBeforeUnmount(() => {
    if (resizeTimer !== null) {
      window.cancelAnimationFrame(resizeTimer)
      resizeTimer = null
    }
    if (resizeObserver) {
      resizeObserver.disconnect()
      resizeObserver = null
    }
    if (mutationObserver) {
      mutationObserver.disconnect()
      mutationObserver = null
    }
    window.removeEventListener('resize', scheduleLayout)
  })

  return {
    isExpanded,
    isScrollable,
    requiredWidth,
    currentContainerWidth,
    applyLayout,
  }
}
