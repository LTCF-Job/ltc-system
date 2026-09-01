import { mockDrivers } from '../data/mockData'
import { findRideOverride } from './rideOverrides'
import type { CaseDTO } from '@/types/api'

// 展示模式沒有 ride_records 資料表，搭乘紀錄一律由個案排班逐日展開後套上示範例外（缺席、
// 未回報、衝突、更正、不申報 AA09）與使用者的人工更正產生。搭乘月曆與政府申報匯出都讀這一份，
// 兩邊才不會對同一天給出不同答案。

export interface DemoRideRecord {
  id: string
  caseId: string
  caseName: string
  serviceDate: string
  legSeq: number
  direction: 'outbound' | 'inbound'
  mergedStatus: 'boarded' | 'absent' | 'unreported'
  effectiveStatus: 'boarded' | 'absent' | 'unreported'
  hasConflict: boolean
  vehicleId?: string
  vehicleName?: string
  driverId?: string
  driverName?: string
  scheduledDepartTime: string
  scheduledDurationMin: number
  departTimeOverride: string | null
  durationMinOverride: number | null
  notClaimedAa09: boolean
  correctedAt?: string
  correctedByName?: string
  correctionReason?: string
  sources: unknown[]
}

export interface DemoCaseDay {
  date: string
  dayOfWeek: number
  isExpected: boolean
  expectedTripCount: number
  records: DemoRideRecord[]
}

// 車輛目前的司機由 driver_assignments 反查，一台車可能有多位
function driversOfVehicle(vehicleId: string) {
  return mockDrivers.filter((d) => (d.assignments || []).some((a) => a.vehicleId === vehicleId))
}

// 展開單一個案在指定西元年月的每日搭乘紀錄。
// scheduledTripCounts 供月曆列判斷該案這個月是固定趟數還是自訂趟數。
export function buildDemoCaseMonth(
  c: CaseDTO,
  targetYear: number,
  targetMonth: number
): { days: Record<string, DemoCaseDay>; scheduledTripCounts: Set<number> } {
  const totalDays = new Date(targetYear, targetMonth, 0).getDate()

  const days: Record<string, any> = {}
  const scheduledTripCounts = new Set<number>()

  for (let day = 1; day <= totalDays; day++) {
    const dayStr = String(day).padStart(2, '0')
    const monthStr = String(targetMonth).padStart(2, '0')
    const dateKey = `${targetYear}-${monthStr}-${dayStr}`
    const dateObj = new Date(targetYear, targetMonth - 1, day)
    const dayOfWeek = dateObj.getDay() === 0 ? 7 : dateObj.getDay()

    let isExpected = false
    let dayTripCount = 0
    let dayLegs: Array<{ legSeq: number; direction: 'outbound' | 'inbound'; departTime: string; vehicleId?: string; vehicleName?: string }> = []

    const monthlyCfg = c.activeSchedule?.monthlyConfigs?.[dateKey]
    if (monthlyCfg) {
      dayTripCount = monthlyCfg.tripCount || 0
      isExpected = dayTripCount > 0
      if (isExpected) {
        if (dayTripCount === 1) {
          dayLegs = [
            { legSeq: 1, direction: 'outbound', departTime: monthlyCfg.departTime || '09:00', vehicleId: monthlyCfg.vehicleId || c.activeSchedule?.legs?.[0]?.vehicleId, vehicleName: c.activeSchedule?.legs?.[0]?.vehicleName || '竹南2車' }
          ]
        } else if (dayTripCount === 2) {
          dayLegs = [
            { legSeq: 1, direction: 'outbound', departTime: monthlyCfg.departTime || '09:00', vehicleId: monthlyCfg.vehicleId || c.activeSchedule?.legs?.[0]?.vehicleId, vehicleName: c.activeSchedule?.legs?.[0]?.vehicleName || '竹南2車' },
            { legSeq: 2, direction: 'inbound', departTime: monthlyCfg.returnTime || '16:00', vehicleId: monthlyCfg.vehicleId || c.activeSchedule?.legs?.[1]?.vehicleId, vehicleName: c.activeSchedule?.legs?.[1]?.vehicleName || '竹南2車' }
          ]
        } else if (dayTripCount === 4) {
          dayLegs = [
            { legSeq: 1, direction: 'outbound', departTime: monthlyCfg.departTime || '08:30', vehicleId: monthlyCfg.vehicleId || 'veh_4', vehicleName: '竹南2車' },
            { legSeq: 2, direction: 'inbound', departTime: '11:30', vehicleId: monthlyCfg.vehicleId || 'veh_4', vehicleName: '竹南2車' },
            { legSeq: 3, direction: 'outbound', departTime: '13:30', vehicleId: monthlyCfg.vehicleId || 'veh_4', vehicleName: '竹南2車' },
            { legSeq: 4, direction: 'inbound', departTime: monthlyCfg.returnTime || '16:30', vehicleId: monthlyCfg.vehicleId || 'veh_4', vehicleName: '竹南2車' }
          ]
        }
      }
    } else if (c.activeSchedule?.scheduleMode === 'by_weekday') {
      const weeklyCfg = c.activeSchedule?.weeklyConfigs?.find((w) => w.weekday === dayOfWeek)
      dayTripCount = weeklyCfg?.tripCount || 0
      isExpected = dayTripCount > 0
      if (isExpected) {
        if (dayTripCount === 1) {
          dayLegs = [
            { legSeq: 1, direction: 'outbound', departTime: weeklyCfg?.departTime || '08:50', vehicleId: weeklyCfg?.vehicleId || 'veh_4', vehicleName: '竹南2車' }
          ]
        } else if (dayTripCount === 2) {
          dayLegs = [
            { legSeq: 1, direction: 'outbound', departTime: weeklyCfg?.departTime || '08:50', vehicleId: weeklyCfg?.vehicleId || 'veh_4', vehicleName: '竹南2車' },
            { legSeq: 2, direction: 'inbound', departTime: weeklyCfg?.returnTime || '15:50', vehicleId: weeklyCfg?.vehicleId || 'veh_4', vehicleName: '竹南2車' }
          ]
        } else if (dayTripCount === 4) {
          dayLegs = [
            { legSeq: 1, direction: 'outbound', departTime: '08:30', vehicleId: 'veh_4', vehicleName: '竹南2車' },
            { legSeq: 2, direction: 'inbound', departTime: '11:30', vehicleId: 'veh_4', vehicleName: '竹南2車' },
            { legSeq: 3, direction: 'outbound', departTime: '13:30', vehicleId: 'veh_4', vehicleName: '竹南2車' },
            { legSeq: 4, direction: 'inbound', departTime: '16:30', vehicleId: 'veh_4', vehicleName: '竹南2車' }
          ]
        }
      }
    } else {
      isExpected = (c.activeSchedule?.weekdays || []).includes(dayOfWeek)
      if (isExpected) {
        dayTripCount = (c.activeSchedule?.legs || []).length || (c.activeSchedule?.tripPattern || 2)
        dayLegs = (c.activeSchedule?.legs || []).map((leg) => ({
          legSeq: leg.legSeq,
          direction: leg.direction,
          departTime: leg.departTime,
          vehicleId: leg.vehicleId,
          vehicleName: leg.vehicleName
        }))
      }
    }

    if (isExpected && dayTripCount > 0) {
      scheduledTripCounts.add(dayTripCount)
    }

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
        expectedTripCount: dayTripCount,
        records: dayLegs.map((leg) => {
          const legUnreported = isUnreported || (day === 24 && c.id === 'case_2' && leg.legSeq === 4)
          const baseEffectiveStatus = legUnreported ? 'unreported' : (isAbsent ? 'absent' : 'boarded')
          const legConflict = isConflict && leg.legSeq === 1

          const rideId = `ride_${c.id}_${dateKey}_${leg.legSeq}`
          const override = findRideOverride(c.id, dateKey, leg.legSeq, rideId)

          // 一台車可能有多位司機，展示資料以日期輪替呈現輪班，實際承載司機仍以回報或人工更正為準
          const vehicleDrivers = leg.vehicleId ? driversOfVehicle(leg.vehicleId) : []
          const defaultDriver = vehicleDrivers.length > 0
            ? vehicleDrivers[day % vehicleDrivers.length]
            : undefined
          const defaultDriverName = defaultDriver?.name
          const defaultDriverId = defaultDriver?.id

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
        expectedTripCount: 0,
        records: nonScheduledRecords
      }
    }
  }

  return { days: days as Record<string, DemoCaseDay>, scheduledTripCounts }
}

// 取出該個案該月實際成行的趟次，依日期與趟序排好。
// 政府申報只申報實際搭乘，缺席與未回報都不列入，與後端 effective_status = 'boarded' 的判準一致。
export function listDemoBoardedRides(c: CaseDTO, targetYear: number, targetMonth: number): DemoRideRecord[] {
  const { days } = buildDemoCaseMonth(c, targetYear, targetMonth)
  return Object.keys(days)
    .sort()
    .flatMap((dateKey) => days[dateKey].records)
    .filter((record) => record.effectiveStatus === 'boarded')
    .sort((a, b) => a.legSeq - b.legSeq || a.serviceDate.localeCompare(b.serviceDate))
}
