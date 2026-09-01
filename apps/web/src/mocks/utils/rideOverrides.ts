// 搭乘紀錄的更正／補登狀態：展示模式下沒有資料庫，人工更正與補登都存在這裡，
// 由 demoStore 一併持久化。月曆與政府申報匯出都要讀到同一份，因此放在 utils 而非單一 handler。

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
