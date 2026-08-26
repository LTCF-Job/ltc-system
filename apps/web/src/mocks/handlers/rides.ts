import { http, HttpResponse } from 'msw'
import { mockCases, mockMissingRides, mockVehicles, mockDrivers, mockAuditLogs } from '../data/mockData'

// 記憶體內儲存更正或補登後的搭乘紀錄狀態
export interface RideOverride {
  id?: string
  caseId?: string
  serviceDate?: string
  legSeq?: number
  effectiveStatus?: 'boarded' | 'absent' | 'unreported'
  mergedStatus?: 'boarded' | 'absent' | 'unreported'
  vehicleId?: string
  vehicleName?: string
  driverId?: string
  driverName?: string
  departTimeOverride?: string | null
  durationMinOverride?: number | null
  notClaimedAa09?: boolean
  hasConflict?: boolean
  correctedAt?: string
  correctedByName?: string
  correctionReason?: string
  sources?: any[]
}

export const mockRideOverrides: Record<string, RideOverride> = {}

// 強健解析 rideId
export function parseRideId(rideId: string): { caseId?: string; serviceDate?: string; legSeq?: number } {
  if (!rideId) return {}

  // 1. 若已在 mockRideOverrides 中存在且有完整資訊
  const existing = mockRideOverrides[rideId]
  if (existing?.caseId && existing?.serviceDate && existing?.legSeq) {
    return {
      caseId: existing.caseId,
      serviceDate: existing.serviceDate,
      legSeq: existing.legSeq
    }
  }

  // 2. 特殊 Demo ID 容錯
  if (rideId === 'ride_conflict_1' || rideId === 'ride_case_2_2026-07-20_1') {
    return { caseId: 'case_2', serviceDate: '2026-07-20', legSeq: 1 }
  }
  if (rideId === 'ride_unrep_1') {
    return { caseId: 'case_1', serviceDate: '2026-07-15', legSeq: 2 }
  }

  // 3. 標準格式：(ride_)?{caseId}_{YYYY-MM-DD}_{legSeq}
  const fullDateRegex = /^(?:ride_)?([a-zA-Z0-9_\-]+)_(\d{4}-\d{2}-\d{2})_(\d+)$/
  const m1 = rideId.match(fullDateRegex)
  if (m1) {
    return {
      caseId: m1[1].replace(/^ride_/, ''),
      serviceDate: m1[2],
      legSeq: parseInt(m1[3], 10)
    }
  }

  // 4. 舊版短格式：(ride_)?{caseId}_{day}_{legSeq}（預設 2026-07）
  const dayRegex = /^(?:ride_)?([a-zA-Z0-9_\-]+)_(\d{1,2})_(\d+)$/
  const m2 = rideId.match(dayRegex)
  if (m2) {
    const day = parseInt(m2[2], 10)
    if (day >= 1 && day <= 31) {
      const dayStr = String(day).padStart(2, '0')
      return {
        caseId: m2[1].replace(/^ride_/, ''),
        serviceDate: `2026-07-${dayStr}`,
        legSeq: parseInt(m2[3], 10)
      }
    }
  }

  return {}
}

// 同步將 override 寫入所有可能被查詢的完整日期別名 key（嚴格隔離月份）
export function saveRideOverride(
  override: RideOverride,
  info?: {
    rideId?: string
    caseId?: string
    serviceDate?: string
    legSeq?: number | string
  }
) {
  const rideId = override.id || info?.rideId || ''
  const parsed = parseRideId(rideId)

  const caseId = info?.caseId || override.caseId || parsed.caseId
  const serviceDate = info?.serviceDate || override.serviceDate || parsed.serviceDate
  const legSeq = info?.legSeq ? Number(info.legSeq) : (override.legSeq || parsed.legSeq)

  if (caseId) override.caseId = caseId
  if (serviceDate) override.serviceDate = serviceDate
  if (legSeq) override.legSeq = legSeq

  if (rideId) {
    mockRideOverrides[rideId] = override
  }

  // 僅使用包含完整 YYYY-MM-DD 的 key，嚴禁使用單日 dayNum，確保各月份獨立不互串
  if (caseId && serviceDate && legSeq) {
    mockRideOverrides[`${caseId}_${serviceDate}_${legSeq}`] = override
    mockRideOverrides[`ride_${caseId}_${serviceDate}_${legSeq}`] = override
  }
}

export function findRideOverride(
  caseId: string,
  dateKey: string,
  legSeq: number,
  primaryRideId?: string
): RideOverride | undefined {
  if (primaryRideId && mockRideOverrides[primaryRideId]) {
    return mockRideOverrides[primaryRideId]
  }

  const fullRideKey = `ride_${caseId}_${dateKey}_${legSeq}`
  if (mockRideOverrides[fullRideKey]) {
    return mockRideOverrides[fullRideKey]
  }

  const fullDateKey = `${caseId}_${dateKey}_${legSeq}`
  if (mockRideOverrides[fullDateKey]) {
    return mockRideOverrides[fullDateKey]
  }

  return undefined
}

export const ridesHandlers = [
  // 搭乘月曆矩陣
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
      const days: Record<string, any> = {}
      for (let day = 1; day <= totalDays; day++) {
        const dayStr = String(day).padStart(2, '0')
        const monthStr = String(targetMonth).padStart(2, '0')
        const dateKey = `${targetYear}-${monthStr}-${dayStr}`
        const dateObj = new Date(targetYear, targetMonth - 1, day)
        const dayOfWeek = dateObj.getDay() === 0 ? 7 : dateObj.getDay()
        const isExpected = (c.activeSchedule?.weekdays || []).includes(dayOfWeek)

        if (isExpected) {
          const isConflict = targetYear === 2026 && targetMonth === 7 && day === 20 && c.id === 'case_2'
          const isCorrected = targetYear === 2026 && targetMonth === 7 && ((day === 10 && c.id === 'case_1') || (day === 13 && c.id === 'case_5'))
          const isAbsent = targetYear === 2026 && targetMonth === 7 && ((day === 15 && c.id === 'case_1') || (day === 18 && c.id === 'case_7'))
          const isUnreported = targetYear === 2026 && targetMonth === 7 && (day === 28 && c.id === 'case_1')
          const isNotClaimed = targetYear === 2026 && targetMonth === 7 && (day === 9 && c.id === 'case_3')

          days[dateKey] = {
            date: dateKey,
            dayOfWeek,
            isExpected: true,
            records: (c.activeSchedule?.legs || []).map((leg) => {
              const legUnreported = isUnreported || (day === 24 && c.id === 'case_2' && leg.legSeq === 4)
              const baseEffectiveStatus = legUnreported ? 'unreported' : (isAbsent ? 'absent' : 'boarded')
              const legConflict = isConflict && leg.legSeq === 1

              const rideId = `ride_${c.id}_${dateKey}_${leg.legSeq}`
              const override = findRideOverride(c.id, dateKey, leg.legSeq, rideId)

              const defaultDriverName = leg.vehicleName?.includes('竹北一')
                ? '郭澤威'
                : (leg.vehicleName?.includes('竹北二')
                  ? '林志豪'
                  : (leg.vehicleName?.includes('竹南1')
                    ? '曾建宏'
                    : (leg.vehicleName?.includes('苗栗')
                      ? '吳秀珠'
                      : '陳國華')))
              const defaultDriver = mockDrivers.find((d) => d.name === defaultDriverName)
              const defaultDriverId = defaultDriver ? defaultDriver.id : undefined

              const effectiveStatus = override?.effectiveStatus ?? baseEffectiveStatus
              const hasConflict = override?.hasConflict ?? legConflict

              let vehicleId: string | undefined
              let vehicleName: string | undefined
              let driverId: string | undefined
              let driverName: string | undefined

              if (effectiveStatus === 'absent') {
                vehicleId = undefined
                vehicleName = undefined
                driverId = undefined
                driverName = undefined
              } else {
                vehicleId = override?.vehicleId !== undefined ? override.vehicleId : leg.vehicleId
                vehicleName = override?.vehicleName !== undefined ? override.vehicleName : leg.vehicleName
                driverId = override?.driverId !== undefined ? override.driverId : defaultDriverId
                driverName = override?.driverName !== undefined ? override.driverName : defaultDriverName
              }

              const departTimeOverride = override?.departTimeOverride !== undefined
                ? override.departTimeOverride
                : (isCorrected ? (c.id === 'case_1' ? '10:05' : '09:15') : null)
              const durationMinOverride = override?.durationMinOverride !== undefined
                ? override.durationMinOverride
                : null
              const notClaimedAa09 = override?.notClaimedAa09 !== undefined
                ? override.notClaimedAa09
                : isNotClaimed
              const correctedAt = override?.correctedAt ?? (isCorrected ? (c.id === 'case_1' ? '2026-07-11 09:30' : '2026-07-14 14:00') : undefined)
              const correctedByName = override?.correctedByName ?? (isCorrected ? '行政承辦' : undefined)
              const correctionReason = override?.correctionReason ?? (isCorrected ? (c.id === 'case_1' ? '司機填錯時間' : '事後補報') : undefined)

              return {
                id: rideId,
                caseId: c.id,
                caseName: c.name,
                serviceDate: dateKey,
                legSeq: leg.legSeq,
                direction: leg.direction,
                mergedStatus: effectiveStatus,
                effectiveStatus: effectiveStatus,
                hasConflict: hasConflict,
                vehicleId: vehicleId,
                vehicleName: vehicleName,
                driverId: driverId,
                driverName: driverName,
                scheduledDepartTime: leg.departTime,
                scheduledDurationMin: c.activeSchedule?.serviceDurationMin || 10,
                departTimeOverride: departTimeOverride,
                durationMinOverride: durationMinOverride,
                notClaimedAa09: notClaimedAa09,
                correctedAt: correctedAt,
                correctedByName: correctedByName,
                correctionReason: correctionReason,
                sources: legUnreported && !override ? [] : (override?.sources || [
                  {
                    id: `src_${c.id}_${day}_1`,
                    submissionId: `sub_${c.id}_${day}`,
                    vehicleName: vehicleName || leg.vehicleName,
                    driverName: driverName || defaultDriverName,
                    reported: effectiveStatus === 'absent' ? 'absent' as const : 'boarded' as const,
                    submittedAt: `${dateKey} 17:30`
                  },
                  ...(hasConflict
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
                ])
              }
            })
          }
        } else {
          const nonScheduledRecords: any[] = []
          for (let legSeq = 1; legSeq <= 4; legSeq++) {
            const rideId = `ride_${c.id}_${dateKey}_${legSeq}`
            const override = findRideOverride(c.id, dateKey, legSeq, rideId)
            if (override) {
              const effectiveStatus = override.effectiveStatus || 'boarded'
              const hasConflict = !!override.hasConflict

              let vehicleId = override.vehicleId
              let vehicleName = override.vehicleName
              let driverId = override.driverId
              let driverName = override.driverName

              if (effectiveStatus === 'absent') {
                vehicleId = undefined
                vehicleName = undefined
                driverId = undefined
                driverName = undefined
              } else if (!vehicleId && !driverId) {
                const leg = c.activeSchedule?.legs?.find((l) => l.legSeq === legSeq)
                vehicleId = leg?.vehicleId || 'veh_1'
                vehicleName = leg?.vehicleName || '竹北一車'
                driverId = 'drv_1'
                driverName = '郭澤威'
              }

              nonScheduledRecords.push({
                id: override.id || rideId,
                caseId: c.id,
                caseName: c.name,
                serviceDate: dateKey,
                legSeq: legSeq,
                direction: legSeq % 2 === 1 ? 'outbound' : 'inbound',
                mergedStatus: effectiveStatus,
                effectiveStatus: effectiveStatus,
                hasConflict: hasConflict,
                vehicleId: vehicleId,
                vehicleName: vehicleName,
                driverId: driverId,
                driverName: driverName,
                scheduledDepartTime: legSeq === 1 ? '09:00' : '16:00',
                scheduledDurationMin: 10,
                departTimeOverride: override.departTimeOverride || null,
                durationMinOverride: override.durationMinOverride || null,
                notClaimedAa09: override.notClaimedAa09 || false,
                correctedAt: override.correctedAt,
                correctedByName: override.correctedByName,
                correctionReason: override.correctionReason,
                sources: override.sources || []
              })
            }
          }

          days[dateKey] = {
            date: dateKey,
            dayOfWeek,
            isExpected: false,
            records: nonScheduledRecords
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
      month: monthParam,
      totalCases: rows.length,
      daysInMonth: totalDays,
      cases: rows
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
    ].filter((item) => {
      const override = findRideOverride(item.caseId, item.serviceDate, item.legSeq, item.id)
      return !override || override.hasConflict !== false
    })

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
    ].filter((item) => {
      const override = findRideOverride(item.caseId, item.serviceDate, item.legSeq, item.id)
      return !override || override.effectiveStatus === 'unreported'
    })

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
