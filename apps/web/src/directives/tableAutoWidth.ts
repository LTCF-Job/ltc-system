import type { Directive, DirectiveBinding } from 'vue'

interface TableAutoWidthState {
  resizeObserver: ResizeObserver | null
  mutationObserver: MutationObserver | null
  resizeTimer: number | null
  cleanup: () => void
}

const stateMap = new WeakMap<HTMLElement, TableAutoWidthState>()

/**
 * 測量單元格實際內容所需寬度（排除區塊容器自身寬度干擾）
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
 * v-table-auto-width 自訂指令
 * 可套用於 .el-table 元素或其父容器
 * 支援欄位內容自動展開（內容過寬不截斷 (...)、不折行）與緊湊縮放（不硬拉滿網頁寬度）
 */
export const vTableAutoWidth: Directive = {
  mounted(el: HTMLElement, binding: DirectiveBinding) {
    const buffer = typeof binding.value === 'number' ? binding.value : 6
    let lastKey = ''
    let isUpdating = false

    function getElements(): { container: HTMLElement; tableEl: HTMLElement } | null {
      if (el.classList.contains('el-table')) {
        const container = el.parentElement || el
        return { container, tableEl: el }
      }
      const tableEl = el.querySelector<HTMLElement>('.el-table')
      if (tableEl) {
        return { container: el, tableEl }
      }
      return null
    }

    function update() {
      if (isUpdating) return
      const targets = getElements()
      if (!targets) return

      const { container, tableEl } = targets
      const containerWidth = container.clientWidth
      if (containerWidth <= 0) return

      const comp = (tableEl as any).__vueParentComponent
      const tableComp = comp?.exposed || comp?.proxy || comp?.ctx || null
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

        const contentNeeded = Math.max(headerWidth, maxCellWidth) + buffer
        let finalWidth = 0
        if (declaredWidth > 0) {
          finalWidth = Math.max(declaredWidth, contentNeeded)
        } else if (rows.length === 0) {
          finalWidth = Math.max(headerWidth + buffer, declaredMinWidth, 60)
        } else {
          // 即使量測到內容寬度，仍須以作者宣告的 min-width 為底線，
          // 避免多元件並排（如多個標籤）只量到其中一個導致欄位被壓縮、內容被裁切
          finalWidth = Math.max(contentNeeded, declaredMinWidth, 60)
        }

        optimalWidths.push(finalWidth)
        totalWidth += finalWidth

        if (col) {
          col.minWidth = finalWidth
          col.width = finalWidth
        }
      }

      const key = `${containerWidth}:${totalWidth}:${optimalWidths.join(',')}`
      if (key === lastKey) return
      lastKey = key

      isUpdating = true
      try {
        const colElements = Array.from(tableEl.querySelectorAll<HTMLElement>('colgroup col'))
        for (let i = 0; i < colElements.length; i++) {
          const colIdx = i % colCount
          const w = optimalWidths[colIdx]
          if (w) {
            colElements[i].setAttribute('width', String(w))
            colElements[i].style.width = `${w}px`
          }
        }

        tableEl.style.width = `${totalWidth}px`
        tableEl.style.minWidth = `${totalWidth}px`

        if (containerWidth >= totalWidth) {
          tableEl.style.maxWidth = '100%'
          container.classList.add('table-compact')
          container.classList.remove('table-scrollable')
          container.style.overflowX = 'visible'
        } else {
          tableEl.style.maxWidth = 'none'
          container.classList.add('table-scrollable')
          container.classList.remove('table-compact')
          container.style.overflowX = 'auto'
        }

        if (tableComp && typeof tableComp.doLayout === 'function') {
          tableComp.doLayout()
        }
      } finally {
        isUpdating = false
      }
    }

    let resizeTimer: number | null = null
    function scheduleUpdate() {
      if (resizeTimer !== null) {
        window.cancelAnimationFrame(resizeTimer)
      }
      resizeTimer = window.requestAnimationFrame(() => {
        update()
        resizeTimer = null
      })
    }

    setTimeout(scheduleUpdate, 50)

    const targets = getElements()
    const observeTarget = targets?.container || el

    const resizeObserver = new ResizeObserver(() => {
      scheduleUpdate()
    })
    resizeObserver.observe(observeTarget)

    const mutationObserver = new MutationObserver((mutations) => {
      const hasStructuralChange = mutations.some((m) => m.type === 'childList')
      if (hasStructuralChange) {
        scheduleUpdate()
      }
    })
    mutationObserver.observe(observeTarget, {
      childList: true,
      subtree: true,
    })

    window.addEventListener('resize', scheduleUpdate, { passive: true })

    stateMap.set(el, {
      resizeObserver,
      mutationObserver,
      resizeTimer,
      cleanup: () => {
        window.removeEventListener('resize', scheduleUpdate)
      },
    })
  },

  unmounted(el: HTMLElement) {
    const state = stateMap.get(el)
    if (state) {
      if (state.resizeTimer !== null) {
        window.cancelAnimationFrame(state.resizeTimer)
      }
      state.resizeObserver?.disconnect()
      state.mutationObserver?.disconnect()
      state.cleanup()
      stateMap.delete(el)
    }
  },
}
