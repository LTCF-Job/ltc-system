import { http, HttpResponse } from 'msw'
import { mockForms, mockFormColumns } from '../data/mockData'

export const formsHandlers = [
  http.get('/api/v1/forms/google-drive-files', () => {
    return HttpResponse.json([
      { id: '1A2B3C4D5E6F7G8H9I0J_zhubei1', name: '竹北一車每日接送回報 (回覆)' },
      { id: '1A2B3C4D5E6F7G8H9I0J_zhubei2', name: '竹北二車每日接送回報 (回覆)' },
      { id: '1A2B3C4D5E6F7G8H9I0J_zhunan1', name: '竹南1車每日接送回報 (回覆)' },
      { id: '1A2B3C4D5E6F7G8H9I0J_zhunan2', name: '竹南2車每日接送回報 (回覆)' }
    ])
  }),

  http.post('/api/v1/forms/inspect-sheet', async ({ request }) => {
    const body = (await request.json()) as any
    const sheetId = body.spreadsheetId || body.sheetUrl || 'demo_sheet_id'
    return HttpResponse.json({
      spreadsheetId: sheetId,
      title: body.sheetUrl?.includes('zhunan') ? '竹南1車每日接送回報 (回覆)' : '竹北一車每日接送回報 (回覆)',
      sheetTabs: ['8月回報', '7月回報', '表單回覆 1'],
      previewHeaders: ['時間戳記', '今天日期', '今日駕駛人', '蔡曾切（去）', '蔡曾切（回）', '問題回報']
    })
  }),

  http.get('/api/v1/forms', ({ request }) => {
    const url = new URL(request.url)
    const q = url.searchParams.get('q')?.trim().toLowerCase()
    let list = [...mockForms]
    if (q) {
      list = list.filter(
        (f) =>
          f.title.toLowerCase().includes(q) ||
          f.formId.toLowerCase().includes(q)
      )
    }
    return HttpResponse.json(list)
  }),

  http.post('/api/v1/forms', async ({ request }) => {
    const body = (await request.json()) as any
    const newForm = {
      id: `form_${Date.now()}`,
      formId: `form_${Date.now()}`,
      title: body.title,
      sheetUrl: body.sheetUrl,
      vehicleId: body.vehicleId || 'veh_1',
      vehicleName: body.vehicleName || '竹北三車',
      region: body.region || 'hsinchu',
      sheetTabs: body.sheetTabs || ['工作表1'],
      activeTab: body.activeTab || '工作表1',
      syncedMonths: [],
      lastSyncedAt: undefined,
      totalColumns: 40,
      pendingColumns: 2,
      hasSyncAlert: false
    }
    mockForms.unshift(newForm)
    return HttpResponse.json(newForm)
  }),

  http.delete('/api/v1/forms/:id', ({ params }) => {
    const idx = mockForms.findIndex((f) => f.id === params.id)
    if (idx !== -1) {
      mockForms.splice(idx, 1)
    }
    return HttpResponse.json({ success: true })
  }),

  http.post('/api/v1/forms/:id/sync', async ({ params, request }) => {
    let body: any = {}
    try {
      body = (await request.json()) as any
    } catch {
      // no body
    }
    const form = mockForms.find((f) => f.id === params.id)
    if (form) {
      form.lastSyncedAt = new Date().toISOString().replace('T', ' ').substring(0, 16)
      form.hasSyncAlert = false
      if (body.month) {
        if (!form.syncedMonths) form.syncedMonths = []
        if (!form.syncedMonths.includes(body.month)) {
          form.syncedMonths.push(body.month)
        }
      }
      if (body.sheetTab) {
        form.activeTab = body.sheetTab
      }
    }
    return HttpResponse.json({
      syncedRows: 24,
      newColumns: 1,
      month: body.month || '2026-08',
      sheetTab: body.sheetTab || '8月回報'
    })
  }),

  http.get('/api/v1/forms/columns', ({ request }) => {
    const url = new URL(request.url)
    const status = url.searchParams.get('mappingStatus')
    let cols = [...mockFormColumns]
    if (status) {
      cols = cols.filter((c) => c.mappingStatus === status)
    }
    return HttpResponse.json(cols)
  }),

  http.patch('/api/v1/forms/columns/:id/mapping', async ({ params, request }) => {
    const col = mockFormColumns.find((c) => c.id === params.id)
    if (!col) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    Object.assign(col, body)
    return HttpResponse.json(col)
  }),

  http.post('/api/v1/forms/columns/batch-mapping', async ({ request }) => {
    const body = (await request.json()) as any
    return HttpResponse.json({ updatedCount: body.mappings.length })
  })
]
