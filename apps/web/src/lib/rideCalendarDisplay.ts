import type { CalendarTripPattern } from '../types/domain'

type CalendarDisplayCell = {
  isExpected?: boolean
  expectedTripCount?: number
  records?: unknown[]
}

export type CalendarDisplayRow = {
  tripPattern?: CalendarTripPattern | 0
  tripPatternText?: string
  days?: Record<string, CalendarDisplayCell>
}

export function getTripPatternDisplay(row: CalendarDisplayRow): string {
  if (row.tripPattern === 'custom' || row.tripPatternText === '自訂') {
    return '自訂'
  }

  if (row.days) {
    const scheduledTripCounts = new Set<number>()
    for (const dateKey in row.days) {
      const cell = row.days[dateKey]
      if (cell?.isExpected) {
        const count = cell.expectedTripCount ?? cell.records?.length ?? 0
        if (count > 0) {
          scheduledTripCounts.add(count)
        }
      }
    }

    if (scheduledTripCounts.size > 1) {
      return '自訂'
    }
    if (scheduledTripCounts.size === 1) {
      return `${Array.from(scheduledTripCounts)[0]} 趟`
    }
  }

  // API 以 0 表示本月份只有搭乘紀錄，沒有可供顯示的排班資料。
  if (row.tripPattern === 0) {
    return '未設定排班'
  }
  if (typeof row.tripPattern === 'number') {
    return `${row.tripPattern} 趟`
  }
  return '未提供排班趟次'
}
