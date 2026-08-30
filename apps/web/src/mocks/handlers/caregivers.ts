import { http, HttpResponse } from 'msw'
import { mockCaregivers, mockSites } from '../data/mockData'
import { createCaregiverImportTemplateExcelBlob } from '../utils/mockExcel'
import { readXlsxRows, buildColumnIndex } from '../utils/parseImportFile'

// 表頭關鍵字對應，比對規則與後端 caregiver_import.go 的 caregiverColumns 一致
const CAREGIVER_COLUMNS: Record<string, string[]> = {
  type: ['類型'],
  site: ['單位'],
  name: ['姓名'],
  contact: ['聯絡方式'],
  notes: ['備註']
}

const CAREGIVER_TYPE_CODE_BY_LABEL: Record<string, string> = {
  個管: 'case_manager',
  專護: 'specialist'
}

// 照護人員主檔與批次匯入。dry-run/commit 契約比照 /cases/import：warnings 的 field
// 區分 "site"（單位待關聯）與 "contact"／"notes"（資料待補齊）。
export const caregiversHandlers = [
  http.get('/api/v1/caregivers', ({ request }) => {
    const url = new URL(request.url)
    const q = url.searchParams.get('q')?.trim().toLowerCase()
    const unresolvedLink = url.searchParams.get('unresolvedLink') === 'true'
    const incomplete = url.searchParams.get('incomplete') === 'true'

    let filtered = [...mockCaregivers]
    if (q) {
      filtered = filtered.filter((c) => c.name.toLowerCase().includes(q))
    }
    if (unresolvedLink) {
      filtered = filtered.filter((c) => !c.siteId && !!c.siteNameRaw)
    }
    if (incomplete) {
      filtered = filtered.filter((c) => !c.contact || !c.notes)
    }

    return HttpResponse.json({ data: filtered, meta: { total: filtered.length } })
  }),

  http.post('/api/v1/caregivers', async ({ request }) => {
    const body = (await request.json()) as any
    const site = mockSites.find((s) => s.id === body.siteId)
    const newCaregiver = {
      id: `caregiver_${Date.now()}`,
      siteId: body.siteId,
      siteName: site?.name,
      name: body.name,
      type: body.type,
      contact: body.contact || '',
      notes: body.notes || '',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    }
    mockCaregivers.push(newCaregiver)
    return HttpResponse.json({ data: newCaregiver }, { status: 201 })
  }),

  http.patch('/api/v1/caregivers/:id', async ({ params, request }) => {
    const c = mockCaregivers.find((item) => item.id === params.id)
    if (!c) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    if (body.siteId !== undefined) {
      c.siteId = body.siteId
      c.siteName = mockSites.find((s) => s.id === body.siteId)?.name
      c.siteNameRaw = undefined
    }
    if (body.name !== undefined) c.name = body.name
    if (body.type !== undefined) c.type = body.type
    if (body.contact !== undefined) c.contact = body.contact
    if (body.notes !== undefined) c.notes = body.notes
    c.updatedAt = new Date().toISOString()
    return HttpResponse.json({ data: c })
  }),

  http.delete('/api/v1/caregivers/:id', ({ params }) => {
    const idx = mockCaregivers.findIndex((item) => item.id === params.id)
    if (idx !== -1) mockCaregivers.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  http.put('/api/v1/caregivers/:id/site', async ({ params, request }) => {
    const c = mockCaregivers.find((item) => item.id === params.id)
    if (!c) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    c.siteId = body.siteId
    c.siteName = mockSites.find((s) => s.id === body.siteId)?.name
    c.siteNameRaw = undefined
    c.updatedAt = new Date().toISOString()
    return HttpResponse.json({ data: c })
  }),

  http.get('/api/v1/caregivers/template', () => {
    return new HttpResponse(createCaregiverImportTemplateExcelBlob(), {
      headers: {
        'Content-Type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'Content-Disposition': 'attachment; filename="caregiver_template.xlsx"'
      }
    })
  }),

  // 依實際上傳的 .xlsx 內容解析，邏輯對齊後端 caregiver_import.go：姓名或類型缺漏／類型
  // 不是個管／專護的列整列略過；單位比對不到既有據點、聯絡方式或備註缺漏仍建立資料並附警告。
  http.post('/api/v1/caregivers/import', async ({ request }) => {
    const url = new URL(request.url)
    const isDryRun = url.searchParams.get('dryRun') === 'true'

    const formData = await request.formData()
    const file = formData.get('file') as File | null
    if (!file) {
      return HttpResponse.json({ error: { code: 'VALIDATION_FAILED', message: '未提供上傳檔案' } }, { status: 400 })
    }

    const rows = await readXlsxRows(file)
    const colIndex = buildColumnIndex(rows[0] ?? [], CAREGIVER_COLUMNS)
    const getVal = (row: string[], field: string) => (colIndex[field] !== undefined ? (row[colIndex[field]] || '').trim() : '')

    const previewRows: any[] = []
    const errors: any[] = []
    const validRows: Array<{
      rowIndex: number
      siteName: string
      name: string
      type: string
      contact: string
      notes: string
      isDuplicate: boolean
    }> = []

    for (let r = 1; r < rows.length; r++) {
      const row = rows[r]
      if (!row || row.every((cell) => !cell)) continue

      const siteName = getVal(row, 'site')
      const name = getVal(row, 'name')
      const typeLabel = getVal(row, 'type')
      const contact = getVal(row, 'contact')
      const notes = getVal(row, 'notes')
      if (!siteName && !name && !typeLabel && !contact && !notes) continue

      const rowIndex = r + 1
      if (!name) {
        errors.push({ rowIndex, field: '姓名', message: '姓名：未填寫，本列已略過' })
        previewRows.push({ rowIndex, siteName, name, type: typeLabel, contact, notes, __hasError: true })
        continue
      }
      const type = CAREGIVER_TYPE_CODE_BY_LABEL[typeLabel]
      if (!type) {
        errors.push({ rowIndex, name, field: '類型', message: '類型：未填寫或不是「個管」／「專護」，本列已略過' })
        previewRows.push({ rowIndex, siteName, name, type: typeLabel, contact, notes, __hasError: true })
        continue
      }

      const dup = mockCaregivers.find((c) => c.name === name)
      const isDuplicate = !!dup
      const matchedSite = mockSites.find((s) => s.name === siteName)
      const hasWarning = (!!siteName && !matchedSite) || !contact || !notes || isDuplicate
      previewRows.push({
        rowIndex,
        siteName,
        name,
        type: typeLabel,
        contact,
        notes,
        isDuplicate,
        ...(isDuplicate ? { duplicateOf: { name: dup!.name } } : {}),
        __hasWarning: hasWarning
      })
      validRows.push({ rowIndex, siteName, name, type, contact, notes, isDuplicate })
    }

    if (isDryRun) {
      return HttpResponse.json({
        totalRows: previewRows.length,
        validRows: validRows.length,
        errorRows: errors.length,
        warningRows: previewRows.filter((p) => p.__hasWarning).length,
        previewRows,
        errors,
        warnings: []
      })
    }

    let includeDuplicateRows: number[] = []
    try {
      const raw = formData.get('includeDuplicateRows')
      includeDuplicateRows = raw ? JSON.parse(String(raw)) : []
    } catch {
      includeDuplicateRows = []
    }
    const includeSet = new Set(includeDuplicateRows)

    const warnings: any[] = []
    const duplicateSkipped: any[] = []
    let importedCount = 0
    for (const row of validRows) {
      if (row.isDuplicate && !includeSet.has(row.rowIndex)) {
        duplicateSkipped.push({ rowIndex: row.rowIndex, name: row.name, reasons: ['偵測為重複人員，未勾選匯入'] })
        continue
      }

      const matchedSite = mockSites.find((s) => s.name === row.siteName)
      const newCaregiver: any = {
        id: `caregiver_${Date.now()}_${importedCount}`,
        name: row.name,
        type: row.type,
        contact: row.contact,
        notes: row.notes,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString()
      }
      if (matchedSite) {
        newCaregiver.siteId = matchedSite.id
        newCaregiver.siteName = matchedSite.name
      } else if (row.siteName) {
        newCaregiver.siteNameRaw = row.siteName
        warnings.push({
          rowIndex: row.rowIndex,
          name: row.name,
          field: 'site',
          message: `單位「${row.siteName}」未於據點管理中找到，已建立資料並保留原始名稱待人工關聯`
        })
      }
      if (!row.contact) {
        warnings.push({ rowIndex: row.rowIndex, name: row.name, field: 'contact', message: '聯絡方式未填寫，已建立資料待後續補齊' })
      }
      if (!row.notes) {
        warnings.push({ rowIndex: row.rowIndex, name: row.name, field: 'notes', message: '備註未填寫，已建立資料待後續補齊' })
      }
      mockCaregivers.push(newCaregiver)
      importedCount++
    }

    return HttpResponse.json({
      importedCount,
      skippedRows: [...errors.map((e) => ({ rowIndex: e.rowIndex, name: e.name || '', reasons: [e.message] })), ...duplicateSkipped],
      warnings
    })
  })
]
