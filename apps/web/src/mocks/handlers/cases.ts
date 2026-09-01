import { http, HttpResponse } from 'msw'
import { mockCases, mockSites, mockVehicles } from '../data/mockData'
import { createCaseImportTemplateExcelBlob, createCaseProfileExcelBlob } from '../utils/mockExcel'
import { readXlsxRows } from '../utils/parseImportFile'
import { findCaseHeader, parseCaseBirthDate } from '../utils/caseImportRules'
import { isValidTaiwanNationalId, maskNationalId } from '../utils/nationalId'

export const casesHandlers = [
  http.get('/api/v1/cases', ({ request }) => {
    const url = new URL(request.url)
    const q = url.searchParams.get('q')?.toLowerCase()
    const region = url.searchParams.get('region')
    const status = url.searchParams.get('status')
    const unresolvedLink = url.searchParams.get('unresolvedLink') === 'true'
    const excludePending = url.searchParams.get('excludePending') === 'true'

    let filtered = [...mockCases]
    if (unresolvedLink) {
      filtered = filtered.filter((c) => c.siteNameRaw || c.outboundVehicleNameRaw || c.inboundVehicleNameRaw)
    }
    if (excludePending) {
      filtered = filtered.filter((c) => !c.siteNameRaw && !c.outboundVehicleNameRaw && !c.inboundVehicleNameRaw)
    }
    if (q) {
      const keyword = q.trim().toLowerCase()
      filtered = filtered.filter(
        (c) =>
          c.name.toLowerCase().includes(keyword) ||
          c.code.toLowerCase().includes(keyword) ||
          (c.nationalId ?? '').toLowerCase().includes(keyword) ||
          (c.homeAddress && c.homeAddress.toLowerCase().includes(keyword))
      )
    }
    if (region) {
      filtered = filtered.filter((c) => c.region === region)
    }
    if (status) {
      filtered = filtered.filter((c) => c.status === status)
    }

    return HttpResponse.json({
      data: filtered,
      meta: {
        page: 1,
        pageSize: 20,
        total: filtered.length,
        totalPages: 1
      }
    })
  }),

  http.get('/api/v1/cases/template', () => {
    const excelBlob = createCaseImportTemplateExcelBlob()
    return new HttpResponse(excelBlob, {
      headers: {
        'Content-Type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'Content-Disposition': 'attachment; filename="case_template.xlsx"'
      }
    })
  }),

  http.get('/api/v1/cases/export', ({ request }) => {
    const url = new URL(request.url)
    const caseIdsParam = url.searchParams.get('caseIds')
    const caseIds = caseIdsParam ? caseIdsParam.split(',').map((id) => id.trim()) : []
    const targetCases = caseIds.length > 0 ? mockCases.filter((c) => caseIds.includes(c.id)) : mockCases
    const excelBlob = createCaseProfileExcelBlob(targetCases)
    return new HttpResponse(excelBlob, {
      headers: {
        'Content-Type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'Content-Disposition': 'attachment; filename="case_profiles.xlsx"'
      }
    })
  }),

  http.get('/api/v1/cases/:id', ({ params }) => {
    const c = mockCases.find((item) => item.id === params.id)
    if (!c) return new HttpResponse(null, { status: 404 })
    return HttpResponse.json(c)
  }),

  http.post('/api/v1/cases', async ({ request }) => {
    const body = (await request.json()) as any
    const newCase = {
      id: `case_${Date.now()}`,
      code: `C00${mockCases.length + 1}`,
      name: body.name,
      nationalId: body.nationalId,
      homeAddress: body.homeAddress,
      region: body.region,
      serviceCategory: body.serviceCategory,
      serviceUsageType: body.serviceUsageType,
      status: body.status || 'active',
      remarks: body.remarks,
      createdAt: new Date().toISOString().split('T')[0],
      updatedAt: new Date().toISOString().split('T')[0]
    }
    mockCases.unshift(newCase)
    return HttpResponse.json(newCase)
  }),

  http.patch('/api/v1/cases/:id', async ({ params, request }) => {
    const c = mockCases.find((item) => item.id === params.id)
    if (!c) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    Object.assign(c, body, { updatedAt: new Date().toISOString().split('T')[0] })
    return HttpResponse.json(c)
  }),

  // 三欄位皆選填：僅覆寫請求中實際帶入的欄位，並清空對應的 *_name_raw（比對到主檔後不再是待補建關聯狀態）
  http.put('/api/v1/cases/:id/transport-preference', async ({ params, request }) => {
    const c = mockCases.find((item) => item.id === params.id)
    if (!c) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    if (body.siteId) {
      c.siteId = body.siteId
      c.siteNameRaw = undefined
    }
    if (body.outboundVehicleId) {
      c.outboundVehicleId = body.outboundVehicleId
      c.outboundVehicleNameRaw = undefined
    }
    if (body.inboundVehicleId) {
      c.inboundVehicleId = body.inboundVehicleId
      c.inboundVehicleNameRaw = undefined
    }
    c.updatedAt = new Date().toISOString().split('T')[0]
    return HttpResponse.json(c)
  }),

  http.delete('/api/v1/cases/:id', ({ params }) => {
    const idx = mockCases.findIndex((item) => item.id === params.id)
    if (idx === -1) return new HttpResponse(null, { status: 404 })
    mockCases.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  http.post('/api/v1/cases/:id/reveal', ({ params }) => {
    const plainMap: Record<string, string> = {
      case_1: 'A202559750',
      case_2: 'J220123456',
      case_3: 'H229876543',
      case_4: 'O201122334'
    }
    return HttpResponse.json({ nationalId: plainMap[params.id as string] || 'A123456789' })
  }),

  http.get('/api/v1/cases/:id/schedule', ({ params }) => {
    const c = mockCases.find((item) => item.id === params.id)
    return HttpResponse.json(c?.activeSchedule || null)
  }),

  http.put('/api/v1/cases/:id/schedule', async ({ params, request }) => {
    const c = mockCases.find((item) => item.id === params.id)
    if (!c) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    c.activeSchedule = {
      id: `sch_${Date.now()}`,
      caseId: c.id,
      ...body
    }
    return HttpResponse.json(c.activeSchedule)
  }),

  // 依實際上傳的 .xlsx 內容解析，邏輯對齊後端 caseimport/app/parse.go 與 commit.go：
  // 生日／身分證字號格式錯誤整列略過；疑似重複個案預設略過（需勾選才匯入）；單位與去回
  // 程車輛比對不到既有主檔仍建立個案並保留原始名稱待人工關聯。
  http.post('/api/v1/cases/import', async ({ request }) => {
    const url = new URL(request.url)
    const isDryRun = url.searchParams.get('dryRun') === 'true'

    const formData = await request.formData()
    const file = formData.get('file') as File | null
    if (!file) {
      return HttpResponse.json({ error: { code: 'VALIDATION_FAILED', message: '未提供上傳檔案' } }, { status: 400 })
    }

    const allRows = await readXlsxRows(file)
    const { headerRowIdx, colMap, caseNameIdx, careContactNameIdx } = findCaseHeader(allRows)

    interface ParsedRow {
      rowIndex: number
      name: string
      nationalId: string
      householdType: string
      gender: string
      birthDate: string
      siteName: string
      outboundVehicle: string
      inboundVehicle: string
      careContactRole: string
      careContactName: string
      registeredAddress: string
      homeAddress: string
      remarks: string
      isDuplicate: boolean
      duplicateCode?: string
      duplicateName?: string
      hasError: boolean
      rowErrors: string[]
    }

    const results: ParsedRow[] = []
    const errorsList: any[] = []
    const warningsList: any[] = []
    const previewRows: any[] = []
    let totalRows = 0
    let validRows = 0
    let warningRows = 0

    if (caseNameIdx >= 0) {
      const getVal = (row: string[], key: string) => {
        const idx = colMap[key]
        return idx !== undefined && idx < row.length ? (row[idx] || '').trim() : ''
      }
      const getIdxVal = (row: string[], idx: number) => (idx >= 0 && idx < row.length ? (row[idx] || '').trim() : '')

      for (let rIdx = headerRowIdx + 1; rIdx < allRows.length; rIdx++) {
        const row = allRows[rIdx]
        if (!row || row.length === 0 || (row.length === 1 && !row[0].trim())) continue

        const name = getIdxVal(row, caseNameIdx)
        if (!name || name.startsWith('例:') || name.startsWith('例：')) continue

        totalRows++
        const rowIndex = rIdx + 1

        const nationalId = getVal(row, '身分證字號')
        const householdType = getVal(row, '戶別')
        const gender = getVal(row, '性別')
        const rawBirth = getVal(row, '生日')
        const birthDate = parseCaseBirthDate(rawBirth)
        // 舊版範本的欄位標題是「據點」，與後端 parse.go 一樣保留相容讀取
        const siteName = getVal(row, '單位') || getVal(row, '據點')
        const outboundVehicle = getVal(row, '接送車輛(去)')
        const inboundVehicle = getVal(row, '接送車輛(回)')
        const careContactRole = getVal(row, '個管or照專')
        const careContactName = getIdxVal(row, careContactNameIdx)
        const registeredAddress = getVal(row, '戶籍')
        const homeAddress = getVal(row, '居住地')
        const remarks = getVal(row, '備註') || getVal(row, 'REMARK')

        const rowErrors: string[] = []
        if (rawBirth && !birthDate) {
          const message = '生日：格式錯誤'
          rowErrors.push(message)
          errorsList.push({ rowIndex, caseName: name, field: '生日', message })
        }

        const normalizedNationalId = nationalId.toUpperCase()
        if (normalizedNationalId && !isValidTaiwanNationalId(normalizedNationalId)) {
          const message = '身分證字號：格式錯誤'
          rowErrors.push(message)
          errorsList.push({ rowIndex, caseName: name, field: '身分證字號', message })
        }

        const hasError = rowErrors.length > 0
        let isDuplicate = false
        let duplicateCode: string | undefined
        let duplicateName: string | undefined

        if (!hasError) {
          const dup = normalizedNationalId
            ? mockCases.find((c) => (c.nationalId || '').toUpperCase() === normalizedNationalId)
            : mockCases.find((c) => c.name === name)
          if (dup) {
            isDuplicate = true
            duplicateCode = dup.code
            duplicateName = dup.name
            warningsList.push({
              rowIndex,
              caseName: name,
              field: '重複個案',
              message: `疑似重複個案（既有個案代碼 ${dup.code}），預設略過，需勾選才會匯入`
            })
          }
        }

        if (hasError) {
          // 錯誤列不計入合法筆數
        } else {
          validRows++
          if (isDuplicate) warningRows++
        }

        results.push({
          rowIndex, name, nationalId, householdType, gender, birthDate, siteName, outboundVehicle, inboundVehicle,
          careContactRole, careContactName, registeredAddress, homeAddress, remarks,
          isDuplicate, duplicateCode, duplicateName, hasError, rowErrors
        })

        previewRows.push({
          rowIndex,
          name,
          nationalId: maskNationalId(nationalId),
          householdType,
          gender,
          birthDate,
          siteName,
          outboundVehicle,
          inboundVehicle,
          careContactRole,
          careContactName,
          registeredAddress,
          homeAddress,
          remarks,
          isDuplicate,
          ...(isDuplicate ? { duplicateOf: { code: duplicateCode, name: duplicateName } } : {}),
          __hasError: hasError,
          __hasWarning: isDuplicate
        })
      }
    }

    if (isDryRun) {
      return HttpResponse.json({
        totalRows,
        validRows,
        errorRows: totalRows - validRows,
        warningRows,
        previewRows,
        errors: errorsList,
        warnings: warningsList
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

    let importedCount = 0
    const skippedRows: any[] = []
    const warnings: any[] = []

    for (const row of results) {
      if (row.hasError) {
        skippedRows.push({ rowIndex: row.rowIndex, caseName: row.name, reasons: row.rowErrors })
        continue
      }
      if (row.isDuplicate && !includeSet.has(row.rowIndex)) {
        skippedRows.push({ rowIndex: row.rowIndex, caseName: row.name, reasons: ['偵測為重複個案，未勾選匯入'] })
        continue
      }

      const matchedSite = mockSites.find((s) => s.name === row.siteName)
      const matchedOutbound = mockVehicles.find((v) => v.displayName === row.outboundVehicle)
      const matchedInbound = mockVehicles.find((v) => v.displayName === row.inboundVehicle)

      const newCase: any = {
        id: `case_${Date.now()}_${importedCount}`,
        code: `IMP-${Math.random().toString(36).slice(2, 10).toUpperCase()}`,
        name: row.name,
        nationalId: row.nationalId || undefined,
        householdType: row.householdType || undefined,
        gender: row.gender || undefined,
        birthDate: row.birthDate || undefined,
        careContactRole: row.careContactRole || undefined,
        careContactName: row.careContactName || undefined,
        registeredAddress: row.registeredAddress || undefined,
        homeAddress: row.homeAddress || undefined,
        status: 'active',
        serviceCategory: 1,
        serviceUsageType: 2,
        remarks: row.remarks || undefined,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString()
      }

      if (matchedSite) {
        newCase.siteId = matchedSite.id
        newCase.siteName = matchedSite.name
      } else if (row.siteName) {
        newCase.siteNameRaw = row.siteName
        warnings.push({
          rowIndex: row.rowIndex,
          caseName: row.name,
          message: `單位「${row.siteName}」未於車輛/單位管理中找到，已建立個案並保留原始名稱待人工關聯`
        })
      }
      if (matchedOutbound) {
        newCase.outboundVehicleId = matchedOutbound.id
        newCase.outboundVehicle = matchedOutbound.displayName
      } else if (row.outboundVehicle) {
        newCase.outboundVehicleNameRaw = row.outboundVehicle
        warnings.push({
          rowIndex: row.rowIndex,
          caseName: row.name,
          message: `接送車輛(去)『${row.outboundVehicle}』未於車輛/單位管理中找到，已建立個案並保留原始名稱待人工關聯`
        })
      }
      if (matchedInbound) {
        newCase.inboundVehicleId = matchedInbound.id
        newCase.inboundVehicle = matchedInbound.displayName
      } else if (row.inboundVehicle) {
        newCase.inboundVehicleNameRaw = row.inboundVehicle
        warnings.push({
          rowIndex: row.rowIndex,
          caseName: row.name,
          message: `接送車輛(回)『${row.inboundVehicle}』未於車輛/單位管理中找到，已建立個案並保留原始名稱待人工關聯`
        })
      }

      mockCases.push(newCase)
      importedCount++
    }

    return HttpResponse.json({ importedCount, skippedRows, warnings })
  })
]
