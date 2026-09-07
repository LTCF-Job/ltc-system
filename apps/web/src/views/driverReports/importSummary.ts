// 匯入結果的四個數字分開呈現：後端逐列比對後，「有處理」與「有新增」是兩件事，
// 只看單一數字會把「重傳同一份檔案、內容完全相同」誤讀成什麼都沒匯入。

export interface ImportResultCounts {
  importedDays: number
  rideRecords: number
  reaffirmed: number
  conflicts: number
  backfilled: number
  pendingColumns: number
}

// describeImportResult 把一次匯入的統計組成一行說明。
export function describeImportResult(counts: ImportResultCounts): string {
  const { importedDays, rideRecords, reaffirmed, conflicts, backfilled, pendingColumns } = counts

  if (importedDays === 0) {
    if (pendingColumns > 0) {
      return `已建立 ${pendingColumns} 個待維護欄位，完成個案連結後會自動補寫搭乘紀錄`
    }
    if (backfilled > 0) {
      return `補寫先前月份 ${backfilled} 筆`
    }
    return '沒有可寫入的搭乘資料'
  }

  const head = `已處理 ${importedDays} 天`
  // 值全部與既有相同是重傳同一份檔案的正常結果，要明講，否則會被當成匯入失敗
  if (reaffirmed > 0 && rideRecords === 0 && conflicts === 0 && backfilled === 0 && pendingColumns === 0) {
    return `${head}，內容與既有資料完全相同，沒有需要更新的地方`
  }

  const parts: string[] = []
  if (rideRecords > 0) parts.push(`新增 ${rideRecords} 筆`)
  if (reaffirmed > 0) parts.push(`與既有相同 ${reaffirmed} 筆`)
  if (conflicts > 0) parts.push(`待維護 ${conflicts} 筆`)
  if (backfilled > 0) parts.push(`另補寫先前月份 ${backfilled} 筆`)
  if (pendingColumns > 0) parts.push(`${pendingColumns} 欄待維護`)

  if (parts.length === 0) {
    return `${head}，但這些欄位都沒有可判讀的回報值`
  }
  return `${head}：${parts.join('、')}`
}

// hasPendingWork 判斷這次結果是否還有需要使用者處理的項目，供畫面決定要不要用警示色。
export function hasPendingWork(counts: Pick<ImportResultCounts, 'conflicts' | 'pendingColumns'>): boolean {
  return counts.conflicts > 0 || counts.pendingColumns > 0
}
