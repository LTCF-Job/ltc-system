import { http, HttpResponse } from 'msw'
import {
  mockCases,
  mockSites,
  mockVehicles,
  mockDrivers,
  mockForms,
  mockFormColumns,
  mockPrecheckResult,
  mockExportJobs,
  mockDashboardStats
} from './data/mockData'

export const handlers = [
  // 1. 個案與排班
  http.get('/api/v1/cases', ({ request }) => {
    const url = new URL(request.url)
    const q = url.searchParams.get('q')?.toLowerCase()
    const region = url.searchParams.get('region')
    const status = url.searchParams.get('status')

    let filtered = [...mockCases]
    if (q) {
      filtered = filtered.filter(
        (c) => c.name.toLowerCase().includes(q) || c.code.toLowerCase().includes(q) || c.nationalId.includes(q)
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
      nationalId: `${body.nationalId.slice(0, 3)}***${body.nationalId.slice(-4)}`,
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
  }),

  // 2. 主檔：據點、車輛、司機
  http.get('/api/v1/sites', () => {
    return HttpResponse.json({ data: mockSites, meta: { total: mockSites.length } })
  }),

  http.post('/api/v1/sites', async ({ request }) => {
    const body = (await request.json()) as any
    const newSite = { id: `site_${Date.now()}`, ...body, createdAt: '2026-08-25' }
    mockSites.push(newSite)
    return HttpResponse.json(newSite)
  }),

  http.patch('/api/v1/sites/:id', async ({ params, request }) => {
    const s = mockSites.find((item) => item.id === params.id)
    if (!s) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    Object.assign(s, body)
    return HttpResponse.json(s)
  }),

  http.delete('/api/v1/sites/:id', ({ params }) => {
    const idx = mockSites.findIndex((item) => item.id === params.id)
    if (idx !== -1) mockSites.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  http.get('/api/v1/vehicles', () => {
    return HttpResponse.json({ data: mockVehicles, meta: { total: mockVehicles.length } })
  }),

  http.post('/api/v1/vehicles', async ({ request }) => {
    const body = (await request.json()) as any
    const newV = { id: `veh_${Date.now()}`, ...body, createdAt: '2026-08-25' }
    mockVehicles.push(newV)
    return HttpResponse.json(newV)
  }),

  http.patch('/api/v1/vehicles/:id', async ({ params, request }) => {
    const v = mockVehicles.find((item) => item.id === params.id)
    if (!v) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    Object.assign(v, body)
    return HttpResponse.json(v)
  }),

  http.get('/api/v1/drivers', () => {
    return HttpResponse.json({ data: mockDrivers, meta: { total: mockDrivers.length } })
  }),

  http.post('/api/v1/drivers', async ({ request }) => {
    const body = (await request.json()) as any
    const newD = { id: `drv_${Date.now()}`, ...body, createdAt: '2026-08-25' }
    mockDrivers.push(newD)
    return HttpResponse.json(newD)
  }),

  http.patch('/api/v1/drivers/:id', async ({ params, request }) => {
    const d = mockDrivers.find((item) => item.id === params.id)
    if (!d) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    Object.assign(d, body)
    return HttpResponse.json(d)
  }),

  http.post('/api/v1/drivers/:id/reveal', ({ params }) => {
    const plainMap: Record<string, string> = {
      drv_1: 'G121806465',
      drv_2: 'J120011223',
      drv_3: 'K120098177'
    }
    return HttpResponse.json({ nationalId: plainMap[params.id as string] || 'G123456789' })
  }),

  http.post('/api/v1/drivers/:id/assignments', async ({ params, request }) => {
    const d = mockDrivers.find((item) => item.id === params.id)
    if (!d) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    const veh = mockVehicles.find((v) => v.id === body.vehicleId)
    d.assignments = [
      {
        id: `asgn_${Date.now()}`,
        driverId: d.id,
        vehicleId: body.vehicleId,
        vehicleName: veh?.displayName,
        startDate: body.startDate,
        endDate: body.endDate,
        isPrimary: body.isPrimary
      }
    ]
    return HttpResponse.json({ success: true })
  }),

  // 3. 表單管理與對應
  http.get('/api/v1/forms', () => {
    return HttpResponse.json(mockForms)
  }),

  http.post('/api/v1/forms/:id/sync', () => {
    return HttpResponse.json({ syncedRows: 24, newColumns: 1 })
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
  }),

  // 4. 搭乘月曆矩陣
  http.get('/api/v1/rides/calendar', () => {
    // 構造 7 月份 1~31 日測試矩陣資料
    const rows = mockCases.map((c) => {
      const days: Record<string, any> = {}
      for (let day = 1; day <= 31; day++) {
        const dayStr = String(day).padStart(2, '0')
        const dateKey = `2026-07-${dayStr}`
        const dateObj = new Date(2026, 6, day)
        const dayOfWeek = dateObj.getDay() === 0 ? 7 : dateObj.getDay()
        const isExpected = (c.activeSchedule?.weekdays || []).includes(dayOfWeek)

        if (isExpected) {
          const isConflict = day === 20 && c.id === 'case_2'
          const isCorrected = day === 10 && c.id === 'case_1'
          const isAbsent = day === 15 && c.id === 'case_1'

          days[dateKey] = {
            date: dateKey,
            dayOfWeek,
            isExpected: true,
            records: (c.activeSchedule?.legs || []).map((leg) => ({
              id: `ride_${c.id}_${day}_${leg.legSeq}`,
              caseId: c.id,
              caseName: c.name,
              serviceDate: dateKey,
              legSeq: leg.legSeq,
              direction: leg.direction,
              mergedStatus: isAbsent ? 'absent' : 'boarded',
              effectiveStatus: isAbsent ? 'absent' : 'boarded',
              hasConflict: isConflict,
              vehicleId: leg.vehicleId,
              vehicleName: leg.vehicleName,
              driverName: '郭澤威',
              scheduledDepartTime: leg.departTime,
              scheduledDurationMin: 10,
              departTimeOverride: isCorrected ? '10:05' : null,
              durationMinOverride: null,
              notClaimedAa09: false,
              correctedAt: isCorrected ? '2026-07-11 09:30' : undefined,
              correctedByName: isCorrected ? '行政承辦' : undefined,
              correctionReason: isCorrected ? '司機填錯時間' : undefined,
              sources: [
                {
                  id: `src_${c.id}_${day}_1`,
                  submissionId: 'sub_1',
                  vehicleName: leg.vehicleName,
                  driverName: '郭澤威',
                  reported: isAbsent ? 'absent' : 'boarded',
                  submittedAt: `${dateKey} 17:30`
                },
                ...(isConflict
                  ? [
                      {
                        id: `src_${c.id}_${day}_2`,
                        submissionId: 'sub_2',
                        vehicleName: '竹北二車',
                        driverName: '林志豪',
                        reported: 'boarded' as const,
                        submittedAt: `${dateKey} 17:35`
                      }
                    ]
                  : [])
              ]
            }))
          }
        } else {
          days[dateKey] = {
            date: dateKey,
            dayOfWeek,
            isExpected: false,
            records: []
          }
        }
      }

      return {
        caseId: c.id,
        caseCode: c.code,
        caseName: c.name,
        region: c.region,
        tripPattern: c.activeSchedule?.tripPattern || 2,
        days
      }
    })

    return HttpResponse.json({
      month: '115-07',
      totalCases: rows.length,
      daysInMonth: 31,
      cases: rows
    })
  }),

  http.patch('/api/v1/rides/:id', async ({ params, request }) => {
    const body = (await request.json()) as any
    return HttpResponse.json({
      id: params.id,
      effectiveStatus: body.effectiveStatus,
      vehicleId: body.vehicleId,
      driverId: body.driverId,
      departTimeOverride: body.departTimeOverride,
      correctedAt: new Date().toISOString(),
      correctedByName: '當前使用者',
      correctionReason: body.reason
    })
  }),

  http.post('/api/v1/rides/:id/resolve-conflict', async ({ params, request }) => {
    const body = (await request.json()) as any
    return HttpResponse.json({
      id: params.id,
      hasConflict: false,
      vehicleId: body.vehicleId,
      driverId: body.driverId,
      effectiveStatus: 'boarded'
    })
  }),

  http.get('/api/v1/rides/issues', ({ request }) => {
    const url = new URL(request.url)
    const type = url.searchParams.get('issueType') || 'conflict'

    if (type === 'conflict') {
      return HttpResponse.json({
        data: [
          {
            id: 'ride_conflict_1',
            caseId: 'case_2',
            caseName: '葉秀珍',
            serviceDate: '2026-07-20',
            legSeq: 1,
            issueType: 'conflict',
            hasConflict: true,
            description: '竹北一車與竹北二車皆回報「有坐」，需指定正確承載車輛',
            vehicles: ['竹北一車', '竹北二車']
          }
        ],
        meta: { total: 1 }
      })
    } else if (type === 'unreported') {
      return HttpResponse.json({
        data: [
          {
            id: 'ride_unrep_1',
            caseId: 'case_1',
            caseName: '蔡曾切',
            serviceDate: '2026-07-15',
            legSeq: 2,
            issueType: 'unreported',
            hasConflict: false,
            description: '07/15 第 2 趟（回程）司機尚未提交表單回覆'
          }
        ],
        meta: { total: 1 }
      })
    }

    return HttpResponse.json({
      data: [
        {
          id: 'err_1',
          caseId: 'case_unknown',
          caseName: '去程到07/21',
          serviceDate: '2026-07-21',
          legSeq: 1,
          issueType: 'import_error',
          hasConflict: false,
          description: '搭乘欄填寫非標準字串「去程到07/21」，無法自動解析為有坐/沒坐'
        }
      ],
      meta: { total: 1 }
    })
  }),

  // 5. 匯出與前置檢核
  http.post('/api/v1/exports/precheck', () => {
    return HttpResponse.json(mockPrecheckResult)
  }),

  http.post('/api/v1/exports', async ({ request }) => {
    const body = (await request.json()) as any
    const newJob = {
      id: `job_${Date.now()}`,
      jobType: body.jobType,
      periodYm: body.periodYm,
      region: body.region,
      mode: body.mode,
      status: 'running' as const,
      totalCases: 42,
      totalRows: 380,
      createdAt: new Date().toISOString()
    }
    mockExportJobs.unshift(newJob as any)
    return HttpResponse.json(newJob)
  }),

  http.get('/api/v1/exports/:id', ({ params }) => {
    const job = mockExportJobs.find((j) => j.id === params.id) || {
      id: params.id,
      jobType: 'gov_claim',
      periodYm: '11507',
      status: 'succeeded',
      totalCases: 42,
      totalRows: 380,
      fileName: 'gov-claim-11507.xlsx',
      downloadUrl: 'https://placeholder-download.supabase.co/gov-claim-11507.xlsx'
    }
    return HttpResponse.json(job)
  }),

  http.get('/api/v1/exports', () => {
    return HttpResponse.json({ data: mockExportJobs, meta: { total: mockExportJobs.length } })
  }),

  // 6. 儀表板
  http.get('/api/v1/dashboard/stats', () => {
    return HttpResponse.json(mockDashboardStats)
  })
]
