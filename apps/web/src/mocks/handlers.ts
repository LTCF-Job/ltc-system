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
  mockDashboardStats,
  mockAuditLogs,
  mockNotificationRecipients,
  mockNotificationLogs,
  mockMissingRides,
  mockTripSummaryReport,
  mockMaintenanceLogs,
  mockAttendanceReport,
  mockFuelLogs,
  mockHsinchuScheduleReport,
  mockDashboardMetrics
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
    // 構造 7 月份 1~31 日測試矩陣資料，涵蓋所有狀態與異常案例
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
          const isCorrected = (day === 10 && c.id === 'case_1') || (day === 13 && c.id === 'case_5')
          const isAbsent = (day === 15 && c.id === 'case_1') || (day === 18 && c.id === 'case_7')
          const isUnreported = (day === 28 && c.id === 'case_1')
          const isNotClaimed = day === 9 && c.id === 'case_3'

          const status = isUnreported ? 'unreported' : (isAbsent ? 'absent' : 'boarded')

          days[dateKey] = {
            date: dateKey,
            dayOfWeek,
            isExpected: true,
            records: (c.activeSchedule?.legs || []).map((leg) => {
              const legUnreported = isUnreported || (day === 24 && c.id === 'case_2' && leg.legSeq === 4)
              const effectiveStatus = legUnreported ? 'unreported' : (isAbsent ? 'absent' : 'boarded')
              const legConflict = isConflict && leg.legSeq === 1

              return {
                id: `ride_${c.id}_${day}_${leg.legSeq}`,
                caseId: c.id,
                caseName: c.name,
                serviceDate: dateKey,
                legSeq: leg.legSeq,
                direction: leg.direction,
                mergedStatus: effectiveStatus,
                effectiveStatus: effectiveStatus,
                hasConflict: legConflict,
                vehicleId: leg.vehicleId,
                vehicleName: leg.vehicleName,
                driverName: leg.vehicleName?.includes('竹北一') ? '郭澤威' : (leg.vehicleName?.includes('竹北二') ? '林志豪' : (leg.vehicleName?.includes('竹南1') ? '曾建宏' : (leg.vehicleName?.includes('苗栗') ? '吳秀珠' : '陳國華'))),
                scheduledDepartTime: leg.departTime,
                scheduledDurationMin: c.activeSchedule?.serviceDurationMin || 10,
                departTimeOverride: isCorrected ? (c.id === 'case_1' ? '10:05' : '09:15') : null,
                durationMinOverride: null,
                notClaimedAa09: isNotClaimed,
                correctedAt: isCorrected ? (c.id === 'case_1' ? '2026-07-11 09:30' : '2026-07-14 14:00') : undefined,
                correctedByName: isCorrected ? '行政承辦' : undefined,
                correctionReason: isCorrected ? (c.id === 'case_1' ? '司機填錯時間' : '事後補報') : undefined,
                sources: legUnreported ? [] : [
                  {
                    id: `src_${c.id}_${day}_1`,
                    submissionId: `sub_${c.id}_${day}`,
                    vehicleName: leg.vehicleName,
                    driverName: leg.vehicleName?.includes('竹北一') ? '郭澤威' : '陳國華',
                    reported: isAbsent ? 'absent' as const : 'boarded' as const,
                    submittedAt: `${dateKey} 17:30`
                  },
                  ...(legConflict
                    ? [
                        {
                          id: `src_${c.id}_${day}_2`,
                          submissionId: `sub_${c.id}_${day}_conflict`,
                          vehicleName: '竹北二車',
                          driverName: '林志豪',
                          reported: 'boarded' as const,
                          submittedAt: `${dateKey} 17:35`
                        }
                      ]
                    : [])
                ]
              }
            })
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
  }),

  // 7. 未回報清單
  http.get('/api/v1/rides/missing', ({ request }) => {
    const url = new URL(request.url)
    const vehicleId = url.searchParams.get('vehicleId')
    let list = [...mockMissingRides]
    if (vehicleId) {
      list = list.filter((r) => r.vehicleId === vehicleId)
    }
    return HttpResponse.json({
      data: list,
      meta: {
        page: 1,
        pageSize: 20,
        total: list.length,
        totalPages: 1
      }
    })
  }),

  // 8. 車輛趟數表
  http.get('/api/v1/reports/trip-summary', ({ request }) => {
    const url = new URL(request.url)
    const vehicleId = url.searchParams.get('vehicleId')
    const report = { ...mockTripSummaryReport }
    if (vehicleId) {
      report.vehicles = report.vehicles.filter((v) => v.vehicleId === vehicleId)
    }
    return HttpResponse.json(report)
  }),

  http.get('/api/v1/reports/trip-summary/export', () => {
    const dummyBlob = new Blob(['mock-excel-binary-data'], {
      type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    })
    return new HttpResponse(dummyBlob, {
      headers: {
        'Content-Type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'Content-Disposition': 'attachment; filename="trip-summary.xlsx"'
      }
    })
  }),

  // 9. 系統稽核紀錄
  http.get('/api/v1/audit', ({ request }) => {
    const url = new URL(request.url)
    const action = url.searchParams.get('action')
    const entityType = url.searchParams.get('entityType')
    let list = [...mockAuditLogs]
    if (action) {
      list = list.filter((a) => a.action === action)
    }
    if (entityType) {
      list = list.filter((a) => a.entityType === entityType)
    }
    return HttpResponse.json({
      data: list,
      meta: {
        page: 1,
        pageSize: 20,
        total: list.length,
        totalPages: 1
      }
    })
  }),

  // 10. 通知收件人管理
  http.get('/api/v1/settings/notification-recipients', ({ request }) => {
    const url = new URL(request.url)
    const topic = url.searchParams.get('topic')
    let list = [...mockNotificationRecipients]
    if (topic) {
      list = list.filter((r) => r.topic === topic)
    }
    return HttpResponse.json(list)
  }),

  http.post('/api/v1/settings/notification-recipients', async ({ request }) => {
    const body = (await request.json()) as any
    const newRecipient = {
      id: `rec_${Date.now()}`,
      topic: body.topic,
      email: body.email,
      displayName: body.displayName || '',
      active: body.active !== undefined ? body.active : true,
      createdByName: '系統管理員',
      createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
    }
    mockNotificationRecipients.push(newRecipient)
    return HttpResponse.json(newRecipient)
  }),

  http.patch('/api/v1/settings/notification-recipients/:id', async ({ params, request }) => {
    const target = mockNotificationRecipients.find((r) => r.id === params.id)
    if (!target) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    Object.assign(target, body)
    return HttpResponse.json(target)
  }),

  http.delete('/api/v1/settings/notification-recipients/:id', ({ params }) => {
    const idx = mockNotificationRecipients.findIndex((r) => r.id === params.id)
    if (idx !== -1) mockNotificationRecipients.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // 11. 通知歷史紀錄
  http.get('/api/v1/notifications/logs', ({ request }) => {
    const url = new URL(request.url)
    const topic = url.searchParams.get('topic')
    let list = [...mockNotificationLogs]
    if (topic) {
      list = list.filter((l) => l.topic === topic)
    }
    return HttpResponse.json({
      data: list,
      meta: {
        page: 1,
        pageSize: 20,
        total: list.length,
        totalPages: 1
      }
    })
  }),

  // 12. 任務手動觸發
  http.post('/api/v1/tasks/check-missing-reports', () => {
    const newLog = {
      id: `nlog_${Date.now()}`,
      topic: 'missing_report' as const,
      channel: 'email' as const,
      recipientEmails: ['admin@ltc.example.com'],
      subject: `【長照交通系統】手動催報執行通知 (${new Date().toLocaleDateString()})`,
      contentSummary: `已發送未回報提醒，共計 ${mockMissingRides.length} 筆待回報項目。`,
      status: 'sent' as const,
      triggeredByName: '當前操作人員 (手動觸發)',
      sentAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
    }
    mockNotificationLogs.unshift(newLog)
    return HttpResponse.json({
      triggeredCount: mockMissingRides.length,
      message: `已成功執行未回報檢核，並發送催報通知至收件人信箱。`
    })
  }),

  // 13. 新竹接送時刻表 (B6.1 / W6.1)
  http.get('/api/v1/reports/hsinchu-schedule', () => {
    return HttpResponse.json(mockHsinchuScheduleReport)
  }),

  http.get('/api/v1/reports/hsinchu-schedule/export', () => {
    const dummyBlob = new Blob(['mock-hsinchu-schedule-binary'], {
      type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    })
    return new HttpResponse(dummyBlob, {
      headers: {
        'Content-Type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'Content-Disposition': 'attachment; filename="hsinchu-schedule.xlsx"'
      }
    })
  }),

  // 14. 車輛維修保養 (B6.2 / W6.2)
  http.get('/api/v1/vehicles/maintenance', ({ request }) => {
    const url = new URL(request.url)
    const vehicleId = url.searchParams.get('vehicleId')
    let list = [...mockMaintenanceLogs]
    if (vehicleId) {
      list = list.filter((m) => m.vehicleId === vehicleId)
    }
    return HttpResponse.json({
      data: list,
      meta: {
        page: 1,
        pageSize: 20,
        total: list.length,
        totalPages: 1
      }
    })
  }),

  http.post('/api/v1/vehicles/maintenance', async ({ request }) => {
    const body = (await request.json()) as any
    const veh = mockVehicles.find((v) => v.id === body.vehicleId)
    const newLog = {
      id: `maint_${Date.now()}`,
      vehicleId: body.vehicleId,
      vehicleName: veh?.displayName || '未知車輛',
      plateNo: veh?.plateNo || '',
      serviceDate: body.serviceDate,
      mileage: body.mileage,
      items: body.items,
      vendor: body.vendor,
      cost: body.cost,
      note: body.note,
      createdBy: '系統使用者',
      createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
    }
    mockMaintenanceLogs.unshift(newLog)
    return HttpResponse.json(newLog)
  }),

  http.patch('/api/v1/vehicles/maintenance/:id', async ({ params, request }) => {
    const target = mockMaintenanceLogs.find((m) => m.id === params.id)
    if (!target) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    Object.assign(target, body)
    return HttpResponse.json(target)
  }),

  http.delete('/api/v1/vehicles/maintenance/:id', ({ params }) => {
    const idx = mockMaintenanceLogs.findIndex((m) => m.id === params.id)
    if (idx !== -1) mockMaintenanceLogs.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  http.get('/api/v1/vehicles/maintenance/blank-template', () => {
    const dummyBlob = new Blob(['mock-blank-maintenance-excel'], {
      type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    })
    return new HttpResponse(dummyBlob, {
      headers: {
        'Content-Type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'Content-Disposition': 'attachment; filename="blank-maintenance-template.xlsx"'
      }
    })
  }),

  // 15. 司機出勤與請假 (B6.3 / W6.3)
  http.get('/api/v1/attendance', ({ request }) => {
    const url = new URL(request.url)
    const driverId = url.searchParams.get('driverId')
    const report = { ...mockAttendanceReport }
    if (driverId) {
      report.drivers = report.drivers.filter((d) => d.driverId === driverId)
    }
    return HttpResponse.json(report)
  }),

  http.post('/api/v1/attendance', async ({ request }) => {
    const body = (await request.json()) as any
    const driver = mockAttendanceReport.drivers.find((d) => d.driverId === body.driverId)
    if (driver) {
      driver.days[body.recordDate] = {
        date: body.recordDate,
        status: body.status,
        note: body.note
      }
    }
    return HttpResponse.json({ success: true })
  }),

  // 16. 車輛油資紀錄 (B6.3 / W6.3)
  http.get('/api/v1/fuel-logs', ({ request }) => {
    const url = new URL(request.url)
    const vehicleId = url.searchParams.get('vehicleId')
    let list = [...mockFuelLogs]
    if (vehicleId) {
      list = list.filter((f) => f.vehicleId === vehicleId)
    }
    return HttpResponse.json({
      data: list,
      meta: {
        page: 1,
        pageSize: 20,
        total: list.length,
        totalPages: 1
      }
    })
  }),

  http.post('/api/v1/fuel-logs', async ({ request }) => {
    const body = (await request.json()) as any
    const veh = mockVehicles.find((v) => v.id === body.vehicleId)
    const drv = mockDrivers.find((d) => d.id === body.driverId)
    const newLog = {
      id: `fuel_${Date.now()}`,
      vehicleId: body.vehicleId,
      vehicleName: veh?.displayName || '',
      plateNo: veh?.plateNo || '',
      driverId: body.driverId,
      driverName: drv?.name || '',
      fuelDate: body.fuelDate,
      liters: body.liters,
      cost: body.cost,
      createdBy: '系統使用者',
      createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
    }
    mockFuelLogs.unshift(newLog)
    return HttpResponse.json(newLog)
  }),

  http.patch('/api/v1/fuel-logs/:id', async ({ params, request }) => {
    const target = mockFuelLogs.find((f) => f.id === params.id)
    if (!target) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    Object.assign(target, body)
    return HttpResponse.json(target)
  }),

  http.delete('/api/v1/fuel-logs/:id', ({ params }) => {
    const idx = mockFuelLogs.findIndex((f) => f.id === params.id)
    if (idx !== -1) mockFuelLogs.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // 17. 完整版營運儀表板指標 (B6.4 / W6.4)
  http.get('/api/v1/dashboard/metrics', () => {
    return HttpResponse.json(mockDashboardMetrics)
  })
]

