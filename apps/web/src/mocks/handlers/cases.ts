import { http, HttpResponse } from 'msw'
import { mockCases } from '../data/mockData'
import { createMockExcelBlob, createCaseImportTemplateExcelBlob } from '../utils/mockExcel'

export const casesHandlers = [
  http.get('/api/v1/cases', ({ request }) => {
    const url = new URL(request.url)
    const q = url.searchParams.get('q')?.toLowerCase()
    const region = url.searchParams.get('region')
    const status = url.searchParams.get('status')
    const unresolvedLink = url.searchParams.get('unresolvedLink') === 'true'

    let filtered = [...mockCases]
    if (unresolvedLink) {
      filtered = filtered.filter((c) => c.siteNameRaw || c.outboundVehicleNameRaw || c.inboundVehicleNameRaw)
    }
    if (q) {
      const keyword = q.trim().toLowerCase()
      filtered = filtered.filter(
        (c) =>
          c.name.toLowerCase().includes(keyword) ||
          c.code.toLowerCase().includes(keyword) ||
          (c.nationalId ?? '').toLowerCase().includes(keyword) ||
          (c.phone && (
            c.phone.toLowerCase().includes(keyword) ||
            c.phone.replace(/[-\s]/g, '').includes(keyword.replace(/[-\s]/g, ''))
          )) ||
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

  http.get('/api/v1/cases/template', ({ request }) => {
    const url = new URL(request.url)
    const format = (url.searchParams.get('format') || 'xlsx').toLowerCase()

    if (format === 'csv') {
      const csvContent =
        '﻿姓名*,戶別,身分證字號*,性別(男/女),生日(YYYY-MM-DD),據點,去程車,回程車,個管or照專,個管姓名,戶籍,居住地,備註\r\n' +
        '張曾阿妹,與子女同住,A202559750,女,1948-03-12,竹南日照據點,竹南2車,竹南2車,個管,蔡怡君,苗栗縣竹南鎮大營路123號,苗栗縣竹南鎮大營路123號,\r\n' +
        '李國盛,獨居,J123458899,男,1952-01-05,竹北日照中心,竹北一車,竹北一車,照專,林小華,新竹縣竹北市文興路一段200號,新竹縣竹北市文興路一段200號,\r\n'
      const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8' })
      return new HttpResponse(blob, {
        headers: {
          'Content-Type': 'text/csv;charset=utf-8',
          'Content-Disposition': 'attachment; filename="case_template.csv"'
        }
      })
    }

    const excelBlob = createCaseImportTemplateExcelBlob()
    return new HttpResponse(excelBlob, {
      headers: {
        'Content-Type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'Content-Disposition': 'attachment; filename="case_template.xlsx"'
      }
    })
  }),

  http.get('/api/v1/cases/export', () => {
    const excelBlob = createMockExcelBlob()
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
      claimStartDate: body.claimStartDate,
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

  http.post('/api/v1/cases/import', ({ request }) => {
    const url = new URL(request.url)
    const isDryRun = url.searchParams.get('dryRun') === 'true'

    if (isDryRun) {
      return HttpResponse.json({
        totalRows: 3,
        validRows: 3,
        errorRows: 0,
        warningRows: 2,
        previewRows: [
          { rowIndex: 2, name: '張曾阿妹', householdType: '與子女同住', nationalId: 'A2****9750', gender: '女', birthDate: '1948-03-12', siteName: '竹南日照據點', outboundVehicle: '竹南2車', inboundVehicle: '竹南2車', careContactRole: '個管', careContactName: '蔡怡君', registeredAddress: '苗栗縣竹南鎮大營路123號', homeAddress: '苗栗縣竹南鎮大營路123號', remarks: '' },
          { rowIndex: 3, name: '李國盛', householdType: '獨居', nationalId: 'J1****8899', gender: '男', birthDate: '1952-01-05', siteName: '竹北日照中心', outboundVehicle: '竹北一車', inboundVehicle: '竹北一車', careContactRole: '照專', careContactName: '林小華', registeredAddress: '新竹縣竹北市文興路一段200號', homeAddress: '新竹縣竹北市文興路一段200號', remarks: '', __hasWarning: true, isDuplicate: true, duplicateOf: { code: 'C0005', name: '李國盛' } },
          { rowIndex: 4, name: '邱美玲', householdType: '獨居', nationalId: 'K2****7654', gender: '女', birthDate: '1950-05-20', siteName: '', outboundVehicle: '', inboundVehicle: '', careContactRole: '個管', careContactName: '邱志明', registeredAddress: '苗栗縣頭份市中華路50號', homeAddress: '苗栗縣頭份市中華路50號', remarks: '需輪椅接送', __hasWarning: true, siteNameRaw: '頭份日照中心（新）', outboundVehicleNameRaw: '頭份1號車' }
        ],
        errors: [],
        warnings: [
          { rowIndex: 3, caseName: '李國盛', field: '身分證字號', message: '疑似與既有個案「李國盛」(C0005) 重複，預設略過，請勾選後確認匯入' },
          { rowIndex: 4, caseName: '邱美玲', field: '據點/接送車輛', message: '據點「頭份日照中心（新）」、去程車輛「頭份1號車」未於主檔中找到，將保留原始名稱待人工關聯' }
        ]
      })
    }

    return HttpResponse.json({
      importedCount: 2,
      skippedRows: [
        { rowIndex: 3, caseName: '李國盛', reasons: ['偵測為重複個案，未勾選匯入'] }
      ],
      warnings: [
        { rowIndex: 4, caseName: '邱美玲', field: '據點', message: '據點「頭份日照中心（新）」未於車輛/據點管理中找到，已建立個案並保留原始名稱待人工關聯' },
        { rowIndex: 4, caseName: '邱美玲', field: '接送車輛(去)', message: '去程車輛「頭份1號車」未於車輛/據點管理中找到，已建立個案並保留原始名稱待人工關聯' }
      ]
    })
  })
]
