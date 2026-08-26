import { http, HttpResponse } from 'msw'
import { mockCases } from '../data/mockData'

export const casesHandlers = [
  http.get('/api/v1/cases', ({ request }) => {
    const url = new URL(request.url)
    const q = url.searchParams.get('q')?.toLowerCase()
    const region = url.searchParams.get('region')
    const status = url.searchParams.get('status')

    let filtered = [...mockCases]
    if (q) {
      const keyword = q.trim().toLowerCase()
      filtered = filtered.filter(
        (c) =>
          c.name.toLowerCase().includes(keyword) ||
          c.code.toLowerCase().includes(keyword) ||
          c.nationalId.toLowerCase().includes(keyword) ||
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

  http.get('/api/v1/cases/template', () => {
    const csvContent =
      '\uFEFF個案姓名*,身分證字號*,申報地區*(苗栗/新竹),住家地址*,開始申報日*(YYYY-MM-DD),服務類別*(1:補助/2:自費),服務使用類型*(1:社區長照/2:社區據點/3:輔具中心/4:身障日照),所屬據點*,每週搭乘日*(如 1,2,3,4,5),趟數型態*(1:單趟/2:來回/4:四趟),去程時間(HH:mm),回程時間(HH:mm),申報單價(元),單趟里程(公里),服務時長(分鐘)\r\n' +
      '張曾阿妹,A202559750,苗栗,苗栗縣竹南鎮大營路123號,2026-07-01,1,2,竹南日照據點,"1,2,3,4,5",2,09:00,16:00,115,5.0,10\r\n' +
      '李國盛,J123458899,新竹,新竹縣竹北市文興路一段200號,2026-07-01,2,1,竹北日照中心,"1,3,5",2,09:30,15:30,200,8.0,20\r\n'
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8' })
    return new HttpResponse(blob, {
      headers: {
        'Content-Type': 'text/csv;charset=utf-8',
        'Content-Disposition': 'attachment; filename="個案批次匯入範本.csv"'
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
        totalRows: 15,
        validRows: 14,
        errorRows: 1,
        warningRows: 2,
        previewRows: [
          { rowIndex: 2, name: '張曾阿妹', region: '苗栗', claimStartDate: '2026-07-01', siteName: '竹南日照據點', weekdays: '週一至週五', departTime: '09:00', returnTime: '16:00', tripPattern: 2 },
          { rowIndex: 3, name: '李國盛', region: '新竹', claimStartDate: '2026-07-01', siteName: '竹北日照中心', weekdays: '週二、週四', departTime: '09:30', returnTime: '15:30', tripPattern: 2, __hasWarning: true },
          { rowIndex: 4, name: '何阿財', region: '苗栗', claimStartDate: '2026-07-01', siteName: '未知據點', weekdays: '週一至週五', departTime: '09:00', returnTime: '16:00', tripPattern: 2, __hasError: true }
        ],
        errors: [
          { rowIndex: 4, caseName: '何阿財', field: '據點', message: '據點「未知據點」不存在於系統主檔中' }
        ],
        warnings: [
          { rowIndex: 3, caseName: '李國盛', field: '單價與里程', message: '單價使用預設值 115 元、時長預設 10 分鐘，請確認' }
        ]
      })
    }

    return HttpResponse.json({ count: 14 })
  })
]
