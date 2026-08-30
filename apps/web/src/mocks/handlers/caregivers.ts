import { http, HttpResponse } from 'msw'
import { mockCaregivers, mockSites } from '../data/mockData'
import { createMockExcelBlob } from '../utils/mockExcel'

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
    return new HttpResponse(createMockExcelBlob(), {
      headers: {
        'Content-Type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'Content-Disposition': 'attachment; filename="caregiver_template.xlsx"'
      }
    })
  }),

  http.post('/api/v1/caregivers/import', ({ request }) => {
    const url = new URL(request.url)
    const isDryRun = url.searchParams.get('dryRun') === 'true'

    if (isDryRun) {
      return HttpResponse.json({
        totalRows: 4,
        validRows: 2,
        errorRows: 2,
        warningRows: 1,
        previewRows: [
          { rowIndex: 2, siteName: '查無此據點', name: '張小芳', type: '個管', contact: '0911-222-333', notes: '', __hasWarning: true },
          { rowIndex: 3, siteName: '竹北日照中心', name: '林小美', type: '專護', contact: '0922-333-444', notes: '個性溫和' }
        ],
        errors: [
          { rowIndex: 4, field: '姓名', message: '姓名：未填寫，本列已略過' },
          { rowIndex: 5, field: '類型', message: '類型：未填寫或不是「個管」／「專護」，本列已略過' }
        ],
        warnings: [
          { rowIndex: 2, name: '張小芳', field: 'site', message: '單位「查無此據點」未於據點管理中找到，已建立資料並保留原始名稱待人工關聯' }
        ]
      })
    }

    return HttpResponse.json({
      importedCount: 2,
      skippedRows: [
        { rowIndex: 4, name: '', reasons: ['姓名：未填寫，本列已略過'] },
        { rowIndex: 5, name: '', reasons: ['類型：未填寫或不是「個管」／「專護」，本列已略過'] }
      ],
      warnings: [
        { rowIndex: 2, name: '張小芳', field: 'site', message: '單位「查無此據點」未於據點管理中找到，已建立資料並保留原始名稱待人工關聯' }
      ]
    })
  })
]
