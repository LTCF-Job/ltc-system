import { http, HttpResponse } from 'msw'
import { mockCases, mockMissingRides } from '../data/mockData'

export const ridesHandlers = [
  // 搭乘月曆矩陣
  http.get('/api/v1/rides/calendar', ({ request }) => {
    const url = new URL(request.url)
    const q = url.searchParams.get('q')?.trim().toLowerCase()
    const region = url.searchParams.get('region')

    let sourceCases = [...mockCases]
    if (q) {
      sourceCases = sourceCases.filter(
        (c) =>
          c.name.toLowerCase().includes(q) ||
          c.code.toLowerCase().includes(q) ||
          (c.nationalId && c.nationalId.toLowerCase().includes(q)) ||
          (c.homeAddress && c.homeAddress.toLowerCase().includes(q))
      )
    }
    if (region) {
      sourceCases = sourceCases.filter((c) => c.region === region)
    }

    const rows = sourceCases.map((c) => {
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

  // 人工補登回報
  http.post('/api/v1/rides/manual-report', async ({ request }) => {
    const body = (await request.json()) as any
    const rideId = body.id || `ride_${body.caseId}_${body.serviceDate}_${body.legSeq}`

    // 搭乘狀態已確認時自未回報清單中排除
    if (body.effectiveStatus === 'boarded' || body.effectiveStatus === 'absent') {
      const idx = mockMissingRides.findIndex(
        (r) =>
          (body.id && r.id === body.id) ||
          (r.caseId === body.caseId && r.serviceDate === body.serviceDate && r.legSeq === body.legSeq)
      )
      if (idx !== -1) {
        mockMissingRides.splice(idx, 1)
      }
    }

    return HttpResponse.json({
      data: {
        id: rideId,
        caseId: body.caseId,
        serviceDate: body.serviceDate,
        legSeq: body.legSeq,
        effectiveStatus: body.effectiveStatus,
        mergedStatus: body.effectiveStatus,
        vehicleId: body.vehicleId,
        driverId: body.driverId,
        departTimeOverride: body.departTimeOverride,
        durationMinOverride: body.durationMinOverride,
        notClaimedAa09: body.notClaimedAa09 || false,
        correctedAt: new Date().toISOString(),
        correctedByName: '當前使用者',
        correctionReason: body.reason,
        sources: []
      }
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
    const q = url.searchParams.get('q')?.trim().toLowerCase()

    let conflictList = [
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
    ]

    let unreportedList = [
      {
        id: 'ride_unrep_1',
        caseId: 'case_1',
        caseName: '蔡曾切',
        serviceDate: '2026-07-15',
        legSeq: 2,
        issueType: 'unreported',
        hasConflict: false,
        description: '07/15 第 2 趟（回程）司機尚未提交表單回覆',
        vehicles: ['苗栗一車']
      }
    ]

    let errorList = [
      {
        id: 'err_1',
        caseId: 'case_unknown',
        caseName: '去程到07/21',
        serviceDate: '2026-07-21',
        legSeq: 1,
        issueType: 'import_error',
        hasConflict: false,
        description: '搭乘欄填寫非標準字串「去程到07/21」，無法自動解析為有坐/沒坐',
        vehicles: []
      }
    ]

    if (q) {
      const matchItem = (item: any) =>
        item.caseName.toLowerCase().includes(q) ||
        item.description.toLowerCase().includes(q) ||
        item.vehicles.some((v: string) => v.toLowerCase().includes(q))

      conflictList = conflictList.filter(matchItem)
      unreportedList = unreportedList.filter(matchItem)
      errorList = errorList.filter(matchItem)
    }

    if (type === 'conflict') {
      return HttpResponse.json({
        data: conflictList,
        meta: { total: conflictList.length }
      })
    } else if (type === 'unreported') {
      return HttpResponse.json({
        data: unreportedList,
        meta: { total: unreportedList.length }
      })
    }

    return HttpResponse.json({
      data: errorList,
      meta: { total: errorList.length }
    })
  }),

  http.get('/api/v1/rides/missing', ({ request }) => {
    const url = new URL(request.url)
    const vehicleId = url.searchParams.get('vehicleId')
    const q = url.searchParams.get('q')?.trim().toLowerCase()

    let list = [...mockMissingRides]
    if (vehicleId) {
      list = list.filter((r) => r.vehicleId === vehicleId)
    }
    if (q) {
      list = list.filter(
        (r) =>
          r.caseName.toLowerCase().includes(q) ||
          (r.vehicleName && r.vehicleName.toLowerCase().includes(q)) ||
          (r.driverName && r.driverName.toLowerCase().includes(q))
      )
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
  })
]
