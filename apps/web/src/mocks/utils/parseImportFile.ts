import * as XLSX from 'xlsx'

/**
 * 將使用者上傳的 .xlsx／.xls 檔案讀取為逐列字串陣列，讓 mock/demo 模式下的批次匯入
 * 依實際上傳內容運作，而非回傳固定示範資料。僅取第一個工作表。
 */
export async function readXlsxRows(file: File): Promise<string[][]> {
  const buffer = await file.arrayBuffer()
  const workbook = XLSX.read(buffer, { type: 'array' })
  const sheetName = workbook.SheetNames[0]
  if (!sheetName) return []
  const sheet = workbook.Sheets[sheetName]
  const rows = XLSX.utils.sheet_to_json<unknown[]>(sheet, { header: 1, raw: false, defval: '' })
  return rows.map((row) => row.map((cell) => String(cell ?? '').trim()))
}

/**
 * 依表頭關鍵字對應出每個欄位在標題列中的欄位索引，比對前會去除儲存格內的 "*" 必填標記。
 */
export function buildColumnIndex(headerRow: string[], columns: Record<string, string[]>): Record<string, number> {
  const index: Record<string, number> = {}
  headerRow.forEach((cell, i) => {
    const clean = cell.replace(/\*/g, '').trim()
    for (const [field, keywords] of Object.entries(columns)) {
      if (keywords.includes(clean) && index[field] === undefined) {
        index[field] = i
      }
    }
  })
  return index
}
