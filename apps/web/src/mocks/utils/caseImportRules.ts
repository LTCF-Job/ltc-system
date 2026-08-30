/**
 * 個案批次匯入的表頭偵測與生日格式解析，邏輯對齊後端
 * apps/api/internal/modules/caseimport/app/parse.go 的 findHeader／parseProfileBirthDate，
 * 讓 mock 模式下的匯入預覽依實際上傳內容運作，而非固定示範資料。
 */

export interface CaseHeaderResult {
  headerRowIdx: number
  colMap: Record<string, number>
  caseNameIdx: number
  careContactNameIdx: number
}

/**
 * 表頭「姓名」出現兩次（個案姓名在前、個管/照專姓名在「個管or照專」欄之後），
 * 依欄位出現順序區分，不可用一般 map 覆寫造成後者蓋掉前者。
 */
export function findCaseHeader(rows: string[][]): CaseHeaderResult {
  for (let r = 0; r < Math.min(3, rows.length); r++) {
    const row = rows[r] ?? []
    const rowText = row.join(',')
    if (!rowText.includes('姓名')) continue

    const colMap: Record<string, number> = {}
    const cols: Array<{ name: string; idx: number }> = []

    row.forEach((cell, c) => {
      let cleanName = cell.replace(/\*/g, '').trim()
      if (cleanName.includes('接送車輛(去)') || cleanName.includes('接送車輛（去）')) {
        colMap['接送車輛(去)'] = c
        return
      }
      if (cleanName.includes('接送車輛(回)') || cleanName.includes('接送車輛（回）')) {
        colMap['接送車輛(回)'] = c
        return
      }
      cleanName = cleanName.split('(')[0].split('（')[0].trim()
      if (cleanName) cols.push({ name: cleanName, idx: c })
    })

    let careRoleIdx = -1
    for (const hc of cols) {
      if (hc.name === '個管or照專' || hc.name === '個管／照專' || hc.name === '個管/照專') {
        careRoleIdx = hc.idx
      }
    }

    let caseNameIdx = -1
    let careContactNameIdx = -1
    for (const hc of cols) {
      if (hc.name !== '姓名') {
        if (colMap[hc.name] === undefined) colMap[hc.name] = hc.idx
        continue
      }
      if (careRoleIdx >= 0 && hc.idx > careRoleIdx) {
        if (careContactNameIdx === -1) careContactNameIdx = hc.idx
      } else if (caseNameIdx === -1) {
        caseNameIdx = hc.idx
      }
    }

    if (caseNameIdx >= 0) {
      return { headerRowIdx: r, colMap, caseNameIdx, careContactNameIdx }
    }
  }
  return { headerRowIdx: 0, colMap: {}, caseNameIdx: -1, careContactNameIdx: -1 }
}

/** 支援 YYYY-MM-DD 與 YYYY/MM/DD 等分隔格式；西元年小於 1911 視為民國年並自動轉換。 */
export function parseCaseBirthDate(raw: string): string {
  const value = raw.trim()
  if (!value) return ''

  const parts = value.split(/[/\-.]/)
  if (parts.length !== 3) return ''

  let year = Number(parts[0])
  const month = Number(parts[1])
  const day = Number(parts[2])
  if (!Number.isInteger(year) || !Number.isInteger(month) || !Number.isInteger(day)) return ''
  if (year < 1911) year += 1911

  const date = new Date(Date.UTC(year, month - 1, day))
  if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) return ''

  return `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`
}
