import { http, HttpResponse } from 'msw'
import { mockCases, mockMissingRides, mockIssueRides, mockVehicles, mockDrivers, mockAuditLogs } from '../data/mockData'
import { buildDemoCaseMonth } from '../utils/demoRides'
import { mockRideOverrides, parseRideId, saveRideOverride, findRideOverride } from '../utils/rideOverrides'
import type { RideOverride } from '../utils/rideOverrides'

// 其他 handler 與 demoStore 沿用既有的匯入路徑，實作已搬到 utils/rideOverrides.ts
export { mockRideOverrides, parseRideId, saveRideOverride, findRideOverride } from '../utils/rideOverrides'
export type { RideOverride } from '../utils/rideOverrides'

export const ridesHandlers = [
  // 搭乘月曆表
  http.get('/api/v1/rides/calendar', ({ request }) => {
    const url = new URL(request.url)
    const monthParam = url.searchParams.get('month') || '2026-07'
    const q = url.searchParams.get('q')?.trim().toLowerCase()
    const region = url.searchParams.get('region')

    let targetYear = 2026
    let targetMonth = 7

    if (monthParam) {
      const parts = monthParam.split('-')
      if (parts.length === 2) {
        const p1 = parseInt(parts[0], 10)
        const p2 = parseInt(parts[1], 10)
        if (p1 < 1000) {
          targetYear = p1 + 1911
        } else {
          targetYear = p1
        }
        targetMonth = p2
      }
    }

    const totalDays = new Date(targetYear, targetMonth, 0).getDate()

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
      const { days, scheduledTripCounts } = buildDemoCaseMonth(c, targetYear, targetMonth)

      let rowTripPattern: any = c.activeSchedule?.tripPattern || 2
      let rowTripPatternText: string | undefined = undefined

      if (scheduledTripCounts.size > 1) {
        rowTripPattern = 'custom'
        rowTripPatternText = '自訂'
      } else if (scheduledTripCounts.size === 1) {
        rowTripPattern = Array.from(scheduledTripCounts)[0]
        rowTripPatternText = `${rowTripPattern} 趟`
      }

      return {
        caseId: c.id,
        caseCode: c.code,
        caseName: c.name,
        region: c.region,
        tripPattern: rowTripPattern,
        tripPatternText: rowTripPatternText,
        days
      }
    })

    return HttpResponse.json({
      month: monthParam,
      totalCases: rows.length,
      daysInMonth: totalDays,
      cases: rows
    })
  }),

  // 具體路徑須排在 /api/v1/rides/:id 之前，否則 issues/missing 會被當成 :id 參數攔截，回傳單筆假資料而非清單
  http.get('/api/v1/rides/issues', ({ request }) => {
    const url = new URL(request.url)
    const type = url.searchParams.get('issueType') || 'conflict'
    const q = url.searchParams.get('q')?.trim().toLowerCase()

    let conflictList = mockIssueRides
      .filter((item) => item.issueType === 'conflict')
      .filter((item) => {
        const override = findRideOverride(item.caseId, item.serviceDate, item.legSeq, item.id)
        return !override || override.hasConflict !== false
      })

    let unreportedList = mockIssueRides
      .filter((item) => item.issueType === 'unreported')
      .filter((item) => {
        const override = findRideOverride(item.caseId, item.serviceDate, item.legSeq, item.id)
        return !override || override.effectiveStatus === 'unreported'
      })

    let errorList = mockIssueRides.filter((item) => item.issueType === 'import_error')

    if (q) {
      const matchItem = (item: any) =>
        item.caseName.toLowerCase().includes(q) ||
        item.description.toLowerCase().includes(q) ||
        (item.vehicles && item.vehicles.some((v: string) => v.toLowerCase().includes(q)))

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
  }),

  http.get('/api/v1/rides/:id', ({ params }) => {
    const rideId = params.id as string
    const override = mockRideOverrides[rideId]
    if (override) {
      return HttpResponse.json(override)
    }

    const parsed = parseRideId(rideId)
    const caseId = parsed.caseId || 'case_1'
    const serviceDate = parsed.serviceDate || '2026-07-10'
    const legSeq = parsed.legSeq || 1

    const c = mockCases.find((x) => x.id === caseId)
    const leg = c?.activeSchedule?.legs?.find((l) => l.legSeq === legSeq)

    return HttpResponse.json({
      id: rideId,
      caseId: caseId,
      caseName: c?.name || '個案姓名',
      serviceDate: serviceDate,
      legSeq: legSeq,
      direction: legSeq % 2 === 1 ? 'outbound' : 'inbound',
      effectiveStatus: 'boarded',
      mergedStatus: 'boarded',
      hasConflict: false,
      vehicleId: leg?.vehicleId || 'veh_4',
      vehicleName: leg?.vehicleName || '竹南2車',
      driverId: 'drv_1',
      driverName: '郭澤威'
    })
  }),

  http.patch('/api/v1/rides/:id', async ({ params, request }) => {
    const body = (await request.json()) as any
    const rideId = params.id as string
    const parsed = parseRideId(rideId)

    const caseId = body.caseId || parsed.caseId || 'case_1'
    const serviceDate = body.serviceDate || parsed.serviceDate || '2026-07-10'
    const legSeq = body.legSeq || parsed.legSeq || 1
    const isAbsent = body.effectiveStatus === 'absent'

    const veh = body.vehicleId ? mockVehicles.find((v) => v.id === body.vehicleId) : undefined
    const drv = body.driverId ? mockDrivers.find((d) => d.id === body.driverId) : undefined

    const now = new Date()
    const formattedNow = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')} ${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`

    const override: RideOverride = {
      id: rideId,
      caseId: caseId,
      serviceDate: serviceDate,
      legSeq: legSeq,
      effectiveStatus: body.effectiveStatus,
      mergedStatus: body.effectiveStatus,
      vehicleId: isAbsent ? undefined : (body.vehicleId || undefined),
      vehicleName: isAbsent ? undefined : (veh?.displayName || (body.vehicleId ? '指定車輛' : undefined)),
      driverId: isAbsent ? undefined : (body.driverId || undefined),
      driverName: isAbsent ? undefined : (drv?.name || (body.driverId ? '指定司機' : undefined)),
      departTimeOverride: isAbsent ? null : body.departTimeOverride,
      durationMinOverride: isAbsent ? null : body.durationMinOverride,
      notClaimedAa09: isAbsent ? false : (body.notClaimedAa09 || false),
      hasConflict: false,
      correctedAt: formattedNow,
      correctedByName: '當前使用者',
      correctionReason: body.reason || (isAbsent ? '標記沒坐' : '人工更正')
    }

    saveRideOverride(override, {
      rideId,
      caseId,
      serviceDate,
      legSeq
    })

    mockAuditLogs.unshift({
      id: `audit_${Date.now()}`,
      actorId: 'usr_staff',
      actorName: '當前使用者',
      actorRole: 'staff',
      action: 'correct',
      entityType: 'ride_records',
      entityId: rideId,
      entityName: `搭乘紀錄更正 (${rideId})`,
      beforeData: undefined,
      afterData: override,
      ipAddress: '127.0.0.1',
      userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
      createdAt: formattedNow
    })

    return HttpResponse.json(override)
  }),

  // 人工補登回報
  http.post('/api/v1/rides/manual-report', async ({ request }) => {
    const body = (await request.json()) as any
    const rideId = body.id || `ride_${body.caseId}_${body.serviceDate}_${body.legSeq}`
    const isAbsent = body.effectiveStatus === 'absent'

    const veh = body.vehicleId ? mockVehicles.find((v) => v.id === body.vehicleId) : undefined
    const drv = body.driverId ? mockDrivers.find((d) => d.id === body.driverId) : undefined

    const now = new Date()
    const formattedNow = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')} ${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`

    const override: RideOverride = {
      id: rideId,
      caseId: body.caseId,
      serviceDate: body.serviceDate,
      legSeq: body.legSeq,
      effectiveStatus: body.effectiveStatus,
      mergedStatus: body.effectiveStatus,
      vehicleId: isAbsent ? undefined : (body.vehicleId || undefined),
      vehicleName: isAbsent ? undefined : (veh?.displayName || (body.vehicleId ? '指定車輛' : undefined)),
      driverId: isAbsent ? undefined : (body.driverId || undefined),
      driverName: isAbsent ? undefined : (drv?.name || (body.driverId ? '指定司機' : undefined)),
      departTimeOverride: isAbsent ? null : body.departTimeOverride,
      durationMinOverride: isAbsent ? null : body.durationMinOverride,
      notClaimedAa09: isAbsent ? false : (body.notClaimedAa09 || false),
      hasConflict: false,
      correctedAt: formattedNow,
      correctedByName: '當前使用者',
      correctionReason: body.reason || (isAbsent ? '補登沒坐' : '人工補登回報')
    }

    saveRideOverride(override, {
      rideId,
      caseId: body.caseId,
      serviceDate: body.serviceDate,
      legSeq: body.legSeq
    })

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

    const targetCase = mockCases.find((c) => c.id === body.caseId)
    const legText = body.legSeq === 1 ? '去程' : body.legSeq === 2 ? '回程' : `第 ${body.legSeq} 趟`
    const caseDisplayName = targetCase ? targetCase.name : '搭乘個案'

    mockAuditLogs.unshift({
      id: `audit_${Date.now()}`,
      actorId: 'usr_staff',
      actorName: '當前使用者',
      actorRole: 'staff',
      action: 'manual_report',
      entityType: 'ride_records',
      entityId: rideId,
      entityName: `${caseDisplayName} (${body.serviceDate} ${legText})`,
      beforeData: undefined,
      afterData: {
        ...override,
        caseName: caseDisplayName
      },
      ipAddress: '127.0.0.1',
      userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
      createdAt: formattedNow
    })

    return HttpResponse.json({
      data: {
        id: rideId,
        caseId: body.caseId,
        serviceDate: body.serviceDate,
        legSeq: body.legSeq,
        effectiveStatus: body.effectiveStatus,
        mergedStatus: body.effectiveStatus,
        vehicleId: override.vehicleId,
        vehicleName: override.vehicleName,
        driverId: override.driverId,
        driverName: override.driverName,
        departTimeOverride: override.departTimeOverride,
        durationMinOverride: override.durationMinOverride,
        notClaimedAa09: override.notClaimedAa09,
        correctedAt: formattedNow,
        correctedByName: '當前使用者',
        correctionReason: body.reason,
        sources: []
      }
    })
  }),

  http.post('/api/v1/rides/:id/resolve-conflict', async ({ params, request }) => {
    const body = (await request.json()) as any
    const rideId = params.id as string
    const parsed = parseRideId(rideId)

    const veh = body.vehicleId ? mockVehicles.find((v) => v.id === body.vehicleId) : undefined
    const drv = body.driverId ? mockDrivers.find((d) => d.id === body.driverId) : undefined

    const now = new Date()
    const formattedNow = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')} ${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`

    const override: RideOverride = {
      id: rideId,
      caseId: parsed.caseId,
      serviceDate: parsed.serviceDate,
      legSeq: parsed.legSeq,
      effectiveStatus: 'boarded',
      mergedStatus: 'boarded',
      hasConflict: false,
      vehicleId: body.vehicleId,
      vehicleName: veh?.displayName || (body.vehicleId ? '指定車輛' : undefined),
      driverId: body.driverId,
      driverName: drv?.name || (body.driverId ? '指定司機' : undefined),
      correctedAt: formattedNow,
      correctedByName: '當前使用者',
      correctionReason: body.reason || '混車衝突裁決'
    }

    saveRideOverride(override, {
      rideId,
      caseId: parsed.caseId,
      serviceDate: parsed.serviceDate,
      legSeq: parsed.legSeq
    })

    return HttpResponse.json(override)
  })
]
