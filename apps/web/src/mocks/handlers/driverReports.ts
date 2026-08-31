import { http, HttpResponse } from 'msw'
import {
  mockDriverReportForms,
  mockDriverReportColumns,
  mockDriverReportImportedDates,
  mockDriverReportLastImportByMonth,
  mockCases,
  mockDrivers,
  mockVehicles
} from '../data/mockData'
import { saveRideOverride } from './rides'
import { readXlsxRows } from '../utils/parseImportFile'
import { createDriverReportTemplateExcelBlob } from '../utils/mockExcel'
import type {
  DriverReportColumnDTO,
  DriverReportImportedMonthDTO,
  DriverReportPreviewColumn,
  DriverReportPreviewRow
} from '@/types/api'

// 以下解析規則與 apps/api 的 driverreport/app/parse.go 對齊：欄位順序、日期寫法、
// 「有坐／沒坐」判斷與訊息文字都逐字相同，讓展示模式看到的檢核結果與正式環境一致。

const HEADER_REPORT_DATE = '民國日期'
const HEADER_REMARK = '備註'

// findReportHeader 找出表頭列，回傳日期欄與備註欄的 0-based 位置（駕駛人固定緊接在日期欄後）。
// 日期欄以「含『日期』字樣、右鄰欄含『駕駛』字樣」定位，而非固定在第 0 欄，讓 Google
// 表單原始匯出檔（開頭多一欄「時間戳記」）也能正確辨識；備註欄同樣以內容搜尋定位，不要求
// 它是整列最後一格，備註欄之後不論還有多少殘留欄位一律略過不匯入。
function findReportHeader(
  rows: string[][]
): { headerRowIdx: number; dateIdx: number; remarkIdx: number } | string {
  for (let idx = 0; idx < rows.length; idx++) {
    const header = trimTrailingEmpty(rows[idx])
    let dateIdx = -1
    for (let i = 0; i + 1 < header.length; i++) {
      if (header[i].includes('日期') && header[i + 1].includes('駕駛')) {
        dateIdx = i
        break
      }
    }
    if (dateIdx < 0) continue

    let remarkIdx = -1
    for (let i = dateIdx + 2; i < header.length; i++) {
      const cell = header[i].trim()
      if (cell === HEADER_REMARK || cell.includes('問題回報')) {
        remarkIdx = i
        break
      }
    }
    if (remarkIdx < 0) return `找不到「${HEADER_REMARK}」欄`
    return { headerRowIdx: idx, dateIdx, remarkIdx }
  }
  return `找不到匯報表表頭，應有一欄含「日期」且右鄰欄含「駕駛」`
}

function trimTrailingEmpty(row: string[]): string[] {
  let end = row.length
  while (end > 0 && (row[end - 1] ?? '').trim() === '') end--
  return row.slice(0, end)
}

// 接受 1150302、115/3/2、115-03-02，並保留西元 2026-03-02 作為後備
function parseReportDate(raw: string): string | null {
  const value = (raw ?? '').trim()
  if (!value) return null

  if (/^\d{6,7}$/.test(value)) {
    return fromRoc(Number(value.slice(0, value.length - 4)), Number(value.slice(-4, -2)), Number(value.slice(-2)))
  }

  const parts = value.split(/[/.\-]/)
  if (parts.length !== 3) return null
  const [y, m, d] = parts.map((p) => Number(p.trim()))
  if (!y || !m || !d) return null
  return y < 1000 ? fromRoc(y, m, d) : fromRoc(y - 1911, m, d)
}

function fromRoc(rocYear: number, month: number, day: number): string | null {
  if (rocYear < 1 || month < 1 || month > 12 || day < 1 || day > 31) return null
  const year = rocYear + 1911
  const date = new Date(Date.UTC(year, month - 1, day))
  if (date.getUTCMonth() + 1 !== month || date.getUTCDate() !== day) return null
  return `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`
}

function parseReportedValue(raw: string): 'boarded' | 'absent' | null {
  const value = (raw ?? '').trim()
  if (value.includes('有坐')) return 'boarded'
  if (value.includes('沒坐')) return 'absent'
  return null
}

// 表頭 → 乾淨姓名與去回程，對齊 domain/namenorm.ParseColumnHeader
function parseColumnHeader(header: string): { cleanedName: string; direction?: 'outbound' | 'inbound' } {
  let text = header.trim()
  let direction: 'outbound' | 'inbound' | undefined
  const bracket = text.match(/\[([^\]]+)\]\s*$/)
  if (bracket) {
    if (bracket[1].includes('去程')) direction = 'outbound'
    else if (bracket[1].includes('回程')) direction = 'inbound'
    text = text.replace(/\[([^\]]+)\]\s*$/, '')
  }
  text = text.replace(/^\s*\d+[.、．\s]+/, '')
  const starIdx = text.indexOf('*')
  if (starIdx !== -1) text = text.slice(0, starIdx)
  text = text.replace(/\([^)]*\)|（[^）]*）/g, '')
  return { cleanedName: text.replace(/\s/g, ''), direction }
}

function legSeqForDirection(direction?: 'outbound' | 'inbound'): number | undefined {
  if (direction === 'outbound') return 1
  if (direction === 'inbound') return 2
  return undefined
}

function knownColumnsFor(formId: string): Map<string, DriverReportColumnDTO> {
  return new Map(
    mockDriverReportColumns.filter((c) => c.formId === formId).map((c) => [c.columnHeader, c])
  )
}

// persistColumnRecords 對應後端 CommitDriverReport 的 persistColumnDecisions：正式匯入時把檔案
// 出現過的所有欄位登錄成 form_columns（首次出現即 pending），讓「待維護資料」查得到完全比對不到
// 個案的欄位；已對應的欄位覆寫成 mapped，其餘沿用既有狀態。
function persistColumnRecords(
  formId: string,
  vehicleName: string,
  formTitle: string,
  columns: DriverReportPreviewColumn[],
  mapped: Map<string, { caseId: string; legSeq: number }>
) {
  const known = knownColumnsFor(formId)
  for (const col of columns) {
    const target = mapped.get(col.columnHeader)
    const existing = known.get(col.columnHeader)
    if (existing) {
      if (target) {
        existing.mappingStatus = 'mapped'
        existing.caseId = target.caseId
        existing.caseName = mockCases.find((c) => c.id === target.caseId)?.name ?? existing.caseName
        existing.legSeq = target.legSeq
      }
      continue
    }
    mockDriverReportColumns.push({
      id: `col_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`,
      formId,
      formTitle,
      vehicleName,
      columnIndex: col.columnIndex,
      columnHeader: col.columnHeader,
      cleanedName: col.cleanedName,
      kind: 'ride',
      mappingStatus: target ? 'mapped' : 'pending',
      caseId: target?.caseId ?? null,
      caseName: target ? mockCases.find((c) => c.id === target.caseId)?.name ?? null : null,
      legSeq: target?.legSeq ?? null,
      suggestedCaseId: col.suggestedCaseId ?? null,
      suggestedCaseName: col.suggestedCaseName ?? null,
      suggestedLegSeq: col.suggestedLegSeq ?? null,
      suggestionScore: col.suggestionScore ?? 0
    })
  }
}

// colOffset 是 caseHeaders[0] 在原始列中的 0-based 位置，用來還原每欄實際的 columnIndex。
function buildColumnPreviews(
  formId: string,
  caseHeaders: string[],
  colOffset: number
): DriverReportPreviewColumn[] {
  const known = knownColumnsFor(formId)

  return caseHeaders.map((header, i) => {
    const parsed = parseColumnHeader(header)
    const prev = known.get(header)
    const column: DriverReportPreviewColumn = {
      columnId: prev?.id,
      columnIndex: colOffset + i + 1,
      columnHeader: header,
      cleanedName: parsed.cleanedName,
      direction: parsed.direction,
      mappingStatus: prev?.mappingStatus ?? 'pending',
      caseId: prev?.caseId ?? undefined,
      caseName: prev?.caseName ?? undefined,
      legSeq: prev?.legSeq ?? undefined,
      suggestionScore: 0,
      boardedCount: 0,
      absentCount: 0
    }

    if (column.mappingStatus === 'pending') {
      const match = mockCases.find((c) => c.name === parsed.cleanedName)
      if (match) {
        column.suggestedCaseId = match.id
        column.suggestedCaseName = match.name
        column.suggestionScore = 1
      }
      column.suggestedLegSeq = legSeqForDirection(parsed.direction)
    }
    return column
  })
}

async function readUploadedRows(request: Request): Promise<string[][] | null> {
  const formData = await request.formData()
  const file = formData.get('file')
  if (!(file instanceof File)) return null
  return readXlsxRows(file)
}

async function readColumnDecisions(request: Request) {
  const formData = await request.formData()
  const raw = formData.get('columnDecisions')
  const file = formData.get('file')
  const rows = file instanceof File ? await readXlsxRows(file) : null
  let decisions: Array<{ columnHeader: string; mappingStatus: string; caseId?: string | null; legSeq?: number | null }> = []
  if (typeof raw === 'string' && raw.trim()) {
    try {
      decisions = JSON.parse(raw)
    } catch {
      decisions = []
    }
  }
  return { rows, decisions }
}

// mockDriverReportImportedDates／mockDriverReportLastImportByMonth（見 mockData.ts）記錄每份
// 匯報表已匯入的服務日期與各月份最後匯入時間，讓 mock 與後端一樣是覆蓋而非疊加：重匯同一份
// 檔案時 submissionCount 不會翻倍。ride 資料本身由 saveRideOverride 以 (caseId, serviceDate,
// legSeq) 為鍵覆寫，本來就冪等。用純物件（而非 Map/Set）儲存是為了讓 demoStore 的 JSON 快照／
// 還原機制能正確持久化，重新整理頁面後不遺失。

function getImportedDates(formId: string): Set<string> {
  return new Set(mockDriverReportImportedDates[formId] ?? [])
}

function setImportedDates(formId: string, dates: Set<string>) {
  mockDriverReportImportedDates[formId] = [...dates]
}

function formMonthKey(formId: string, yearMonth: string): string {
  return `${formId}::${yearMonth}`
}

// forgetImportedDates 對應 form_submissions.form_id 的 ON DELETE CASCADE：
// 匯報表被刪除時，它的已匯入月份也要一起消失
function forgetImportedDates(formId: string) {
  delete mockDriverReportImportedDates[formId]
  for (const key of Object.keys(mockDriverReportLastImportByMonth)) {
    if (key.startsWith(`${formId}::`)) delete mockDriverReportLastImportByMonth[key]
  }
}

// clearImportedDates 對應後端的 RideService.ClearImportedDates：宣告月份時清整個月，
// 未宣告時只清本次檔案涵蓋的日期。
function clearImportedDates(formId: string, dates: string[], yearMonth: string): Set<string> {
  const known = getImportedDates(formId)
  if (yearMonth) {
    for (const date of [...known]) {
      if (date.startsWith(yearMonth)) known.delete(date)
    }
  } else {
    for (const date of dates) known.delete(date)
  }
  setImportedDates(formId, known)
  return known
}

// monthRejection 把宣告月份格式錯誤的訊息包成與後端一致的錯誤回應。
// 個別列的日期不屬於宣告月份不在此列——那與單列日期打錯一樣，由 buildPreview
// 標記成該列的 errorMessage，讓使用者在預覽畫面自行決定是否忽略或調整。
function monthRejection(message: string) {
  return HttpResponse.json(
    {
      error: {
        code: 'DRIVER_REPORT_IMPORT_FAILED',
        message,
        details: [{ field: 'file', reason: message }]
      }
    },
    { status: 400 }
  )
}

function buildPreview(formId: string, rows: string[][], yearMonth: string) {
  const found = findReportHeader(rows)
  if (typeof found === 'string') return found

  const { headerRowIdx, dateIdx, remarkIdx } = found
  const driverIdx = dateIdx + 1
  const header = trimTrailingEmpty(rows[headerRowIdx])
  const columns = buildColumnPreviews(formId, header.slice(driverIdx + 1, remarkIdx), driverIdx + 1)
  const form = mockDriverReportForms.find((f) => f.id === formId)

  const previewRows: DriverReportPreviewRow[] = []
  const errors: Array<{ rowIndex: number; field?: string; message: string }> = []
  const warnings: Array<{ rowIndex: number; message: string }> = []
  let totalRows = 0
  let validRows = 0
  let errorRows = 0
  let warningRows = 0

  for (let i = headerRowIdx + 1; i < rows.length; i++) {
    const rowNum = i + 1
    const dateRaw = (rows[i][dateIdx] ?? '').trim()
    if (!dateRaw) continue

    totalRows++
    const serviceDate = parseReportDate(dateRaw)
    if (!serviceDate) {
      const message = `日期格式無法解析（${dateRaw}），請填寫民國日期如 1150302`
      errors.push({ rowIndex: rowNum, field: HEADER_REPORT_DATE, message })
      errorRows++
      previewRows.push({
        rowIndex: rowNum,
        reportDate: dateRaw,
        serviceDate: '',
        driverRaw: '',
        boardedCount: 0,
        absentCount: 0,
        errorMessage: message
      })
      continue
    }

    if (yearMonth && !serviceDate.startsWith(yearMonth)) {
      const message = `日期 ${serviceDate} 不屬於宣告匯入的 ${yearMonth}，請確認上傳的是該月份的檔案`
      errors.push({ rowIndex: rowNum, field: HEADER_REPORT_DATE, message })
      errorRows++
      previewRows.push({
        rowIndex: rowNum,
        reportDate: dateRaw,
        serviceDate,
        driverRaw: '',
        boardedCount: 0,
        absentCount: 0,
        errorMessage: message
      })
      continue
    }

    const driverRaw = (rows[i][driverIdx] ?? '').trim()
    const driver = mockDriverReportDriver(driverRaw)
    let warningMessage = ''
    if (!driverRaw) {
      warningMessage = '未填寫駕駛人，該日搭乘紀錄將沒有司機'
    } else if (!driver) {
      warningMessage = `駕駛人「${driverRaw}」未比對到司機主檔`
    }

    let boardedCount = 0
    let absentCount = 0
    columns.forEach((column, ci) => {
      const reported = parseReportedValue(rows[i][column.columnIndex - 1] ?? '')
      if (!reported) return
      if (reported === 'boarded') {
        boardedCount++
        columns[ci].boardedCount++
      } else {
        absentCount++
        columns[ci].absentCount++
      }
    })

    if (warningMessage) {
      warnings.push({ rowIndex: rowNum, message: warningMessage })
      warningRows++
    }
    validRows++
    previewRows.push({
      rowIndex: rowNum,
      reportDate: dateRaw,
      serviceDate,
      driverRaw,
      driverId: driver?.id,
      driverName: driver?.name,
      remark: (rows[i][remarkIdx] ?? '').trim(),
      boardedCount,
      absentCount,
      warningMessage: warningMessage || undefined
    })
  }

  return {
    formId,
    vehicleId: form?.vehicleId ?? '',
    vehicleName: form?.vehicleName ?? '',
    totalRows,
    validRows,
    errorRows,
    warningRows,
    unmappedColumns: columns.filter((c) => c.mappingStatus === 'pending').length,
    columns,
    previewRows,
    errors,
    warnings
  }
}

// 展示模式的司機主檔比對：以姓名完全相符為準，與後端的 name_normalized 查詢等價
function mockDriverReportDriver(name: string) {
  if (!name) return undefined
  return mockDrivers.find((d) => d.name === name)
}

export const driverReportsHandlers = [
  http.get('/api/v1/driver-reports', ({ request }) => {
    const q = new URL(request.url).searchParams.get('q')?.trim().toLowerCase()
    let list = [...mockDriverReportForms]
    if (q) {
      list = list.filter(
        (f) => f.title.toLowerCase().includes(q) || f.vehicleName.toLowerCase().includes(q)
      )
    }
    return HttpResponse.json(list)
  }),

  // 與後端一樣由已匯入的服務日期分組推導，不另外維護一份月份狀態
  http.get('/api/v1/driver-reports/imported-months', () => {
    const items: DriverReportImportedMonthDTO[] = []
    for (const [formId, dates] of Object.entries(mockDriverReportImportedDates)) {
      const countByMonth = new Map<string, number>()
      for (const date of dates) {
        const yearMonth = date.slice(0, 7)
        countByMonth.set(yearMonth, (countByMonth.get(yearMonth) ?? 0) + 1)
      }
      for (const [yearMonth, submissionCount] of countByMonth) {
        items.push({
          formId,
          yearMonth,
          submissionCount,
          lastImportedAt: mockDriverReportLastImportByMonth[formMonthKey(formId, yearMonth)] ?? ''
        })
      }
    }
    items.sort((a, b) => a.formId.localeCompare(b.formId) || b.yearMonth.localeCompare(a.yearMonth))
    return HttpResponse.json(items)
  }),

  // 比照後端的 ON CONFLICT (vehicle_id)：一台車一份匯報表，重複建立只更新名稱
  http.post('/api/v1/driver-reports', async ({ request }) => {
    const body = (await request.json()) as { vehicleId: string; title: string }
    const existing = mockDriverReportForms.find((f) => f.vehicleId === body.vehicleId)
    if (existing) {
      existing.title = body.title
      return HttpResponse.json(existing, { status: 201 })
    }
    const created = {
      id: `form_${Date.now()}`,
      vehicleId: body.vehicleId,
      vehicleName: mockVehicles.find((v) => v.id === body.vehicleId)?.displayName || '未知車輛',
      title: body.title,
      region: 'hsinchu' as const,
      lastImportedAt: null,
      totalColumns: 0,
      mappedColumns: 0,
      pendingColumns: 0,
      submissionCount: 0,
      status: 'active'
    }
    mockDriverReportForms.unshift(created)
    return HttpResponse.json(created, { status: 201 })
  }),

  http.delete('/api/v1/driver-reports/:id', ({ params }) => {
    const idx = mockDriverReportForms.findIndex((f) => f.id === params.id)
    if (idx !== -1) mockDriverReportForms.splice(idx, 1)
    forgetImportedDates(String(params.id))
    return HttpResponse.json({ success: true })
  }),

  http.get('/api/v1/driver-reports/:id/template', async ({ params }) => {
    const form = mockDriverReportForms.find((f) => f.id === params.id)
    const caseColumns = mockDriverReportColumns
      .filter((c) => c.formId === params.id && c.mappingStatus === 'mapped')
      .map((c) => c.columnHeader)

    const blob = createDriverReportTemplateExcelBlob(caseColumns)
    return new HttpResponse(await blob.arrayBuffer(), {
      headers: {
        'Content-Type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'Content-Disposition': `attachment; filename="driver_report_template.xlsx"; filename*=UTF-8''${encodeURIComponent(
          `${form?.vehicleName ?? '車輛'}接送匯報範本.xlsx`
        )}`
      }
    })
  }),

  http.post('/api/v1/driver-reports/:id/import', async ({ params, request }) => {
    const formId = String(params.id)
    const query = new URL(request.url).searchParams
    const dryRun = query.get('dryRun') !== 'false'
    const yearMonth = query.get('yearMonth') ?? ''
    if (yearMonth && !/^\d{4}-(0[1-9]|1[0-2])$/.test(yearMonth)) {
      return monthRejection(`匯入月份格式錯誤，請使用 YYYY-MM：${yearMonth}`)
    }

    if (dryRun) {
      const rows = await readUploadedRows(request)
      if (!rows) {
        return HttpResponse.json(
          { error: { code: 'VALIDATION_FAILED', message: '未提供上傳檔案' } },
          { status: 400 }
        )
      }
      const preview = buildPreview(formId, rows, yearMonth)
      if (typeof preview === 'string') {
        return HttpResponse.json(
          {
            error: {
              code: 'DRIVER_REPORT_IMPORT_FAILED',
              message: preview,
              details: [{ field: 'file', reason: preview }]
            }
          },
          { status: 400 }
        )
      }
      return HttpResponse.json(preview)
    }

    const { rows, decisions } = await readColumnDecisions(request)
    if (!rows) {
      return HttpResponse.json(
        { error: { code: 'VALIDATION_FAILED', message: '未提供上傳檔案' } },
        { status: 400 }
      )
    }

    const preview = buildPreview(formId, rows, yearMonth)
    if (typeof preview === 'string') {
      return HttpResponse.json(
        {
          error: {
            code: 'DRIVER_REPORT_IMPORT_FAILED',
            message: preview,
            details: [{ field: 'file', reason: preview }]
          }
        },
        { status: 400 }
      )
    }

    const mapped = new Map(
      decisions
        .filter((d) => d.mappingStatus === 'mapped' && d.caseId && d.legSeq)
        .map((d) => [d.columnHeader, { caseId: d.caseId as string, legSeq: d.legSeq as number }])
    )
    // 使用者未重新確認的欄位，沿用既有已對應設定
    preview.columns.forEach((c) => {
      if (!mapped.has(c.columnHeader) && c.mappingStatus === 'mapped' && c.caseId && c.legSeq) {
        mapped.set(c.columnHeader, { caseId: c.caseId, legSeq: c.legSeq })
      }
    })

    if (mapped.size === 0) {
      return HttpResponse.json(
        {
          error: {
            code: 'DRIVER_REPORT_IMPORT_FAILED',
            message: '尚未有任何欄位對應到個案，請先於預覽畫面完成對應'
          }
        },
        { status: 400 }
      )
    }

    const form = mockDriverReportForms.find((f) => f.id === formId)
    persistColumnRecords(formId, form?.vehicleName ?? '', form?.title ?? '', preview.columns, mapped)
    const coveredDates = preview.previewRows
      .filter((row) => !row.errorMessage && row.serviceDate)
      .map((row) => row.serviceDate)
    // 沒有任何可寫入的列時不清除，與後端 clearPreviousImport 一致
    const importedDates = coveredDates.length
      ? clearImportedDates(formId, coveredDates, yearMonth)
      : getImportedDates(formId)
    let rideRecordRows = 0
    const skippedRows: Array<{ rowIndex: number; reportDate: string; reasons: string[] }> = []
    let importedRows = 0

    for (const row of preview.previewRows) {
      if (row.errorMessage) {
        skippedRows.push({
          rowIndex: row.rowIndex,
          reportDate: row.reportDate,
          reasons: [row.errorMessage]
        })
        continue
      }

      for (const column of preview.columns) {
        const target = mapped.get(column.columnHeader)
        if (!target) continue
        const reported = parseReportedValue(rows[row.rowIndex - 1][column.columnIndex - 1] ?? '')
        if (!reported) continue

        for (const legSeq of expandLegSeqs(target.caseId, target.legSeq)) {
          saveRideOverride(
            {
              caseId: target.caseId,
              serviceDate: row.serviceDate,
              legSeq,
              mergedStatus: reported,
              effectiveStatus: reported,
              hasConflict: false,
              vehicleId: form?.vehicleId,
              vehicleName: form?.vehicleName,
              driverId: row.driverId,
              driverName: row.driverName
            },
            { caseId: target.caseId, serviceDate: row.serviceDate, legSeq }
          )
          rideRecordRows++
        }
      }
      importedDates.add(row.serviceDate)
      importedRows++
    }
    setImportedDates(formId, importedDates)

    // 全部列都被跳過（月份不符或格式錯誤）時代表沒有任何一列真的匯入，
    // 不更新最後匯入時間，避免使用者誤以為這次上傳已經成功，與後端 MarkImported 的守則一致。
    if (importedRows > 0) {
      const importedAt = new Date().toISOString().replace('T', ' ').substring(0, 19)
      for (const date of coveredDates) {
        mockDriverReportLastImportByMonth[formMonthKey(formId, date.slice(0, 7))] = importedAt
      }
      if (form) form.lastImportedAt = importedAt
    }

    if (form) {
      form.submissionCount = importedDates.size
      form.totalColumns = preview.columns.length
      form.mappedColumns = mapped.size
      form.pendingColumns = preview.columns.length - mapped.size
    }

    return HttpResponse.json({
      importedRows,
      rideRecordRows,
      mappedColumns: mapped.size,
      skippedRows,
      warnings: preview.warnings
    })
  }),

  http.get('/api/v1/driver-reports/columns', ({ request }) => {
    const url = new URL(request.url)
    const formId = url.searchParams.get('formId')
    const status = url.searchParams.get('mappingStatus')
    let cols = [...mockDriverReportColumns]
    if (formId) cols = cols.filter((c) => c.formId === formId)
    if (status) cols = cols.filter((c) => c.mappingStatus === status)
    return HttpResponse.json(cols)
  }),

  http.patch('/api/v1/driver-reports/columns/:id/mapping', async ({ params, request }) => {
    const col = mockDriverReportColumns.find((c) => c.id === params.id)
    if (!col) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as Partial<DriverReportColumnDTO>
    Object.assign(col, body)
    if (col.caseId) {
      col.caseName = mockCases.find((c) => c.id === col.caseId)?.name ?? col.caseName
    }
    return HttpResponse.json(col)
  }),

  http.post('/api/v1/driver-reports/columns/batch-mapping', async ({ request }) => {
    const body = (await request.json()) as {
      mappings: Array<{ columnId: string; caseId?: string; legSeq?: number; mappingStatus: string }>
    }
    body.mappings.forEach((m) => {
      const col = mockDriverReportColumns.find((c) => c.id === m.columnId)
      if (!col) return
      col.mappingStatus = m.mappingStatus as DriverReportColumnDTO['mappingStatus']
      col.caseId = m.caseId ?? null
      col.legSeq = m.legSeq ?? null
      col.caseName = mockCases.find((c) => c.id === m.caseId)?.name ?? null
    })
    return HttpResponse.json({ updatedCount: body.mappings.length })
  })
]

// 四趟個案的第 1／2 趟展開為 1、3 與 2、4，與 ride/app 的 expandLegSeqs 一致
function expandLegSeqs(caseId: string, legSeq: number): number[] {
  const target = mockCases.find((c) => c.id === caseId)
  if (target?.activeSchedule?.tripPattern !== 4) return [legSeq]
  if (legSeq === 1) return [1, 3]
  if (legSeq === 2) return [2, 4]
  return [legSeq]
}
