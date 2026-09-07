import dayjs from 'dayjs'

// 民國年月顯示與格式轉換（僅呈現層輔助，無業務規則運算）
//
// 一律透過 dayjs 解析，與 @/utils/formatters 使用同一套規則：原生 new Date('2026-03-01')
// 依 ECMAScript 規格把純日期字串當成 UTC，在負時區會被算成前一個月，導致同一個日期在
// 民國月份與其他欄位顯示不一致。dayjs 把純日期字串解析為本地時間，兩邊才會一致。

export function useRocMonth() {
  function toRocMonth(date: Date | string): string {
    const d = dayjs(date)
    return `${d.year() - 1911}-${d.format('MM')}`
  }

  function toRocPeriodYm(date: Date | string): string {
    const d = dayjs(date)
    return `${d.year() - 1911}${d.format('MM')}`
  }

  function formatRocMonthLabel(rocMonth: string): string {
    if (!rocMonth) return ''
    const parts = rocMonth.split('-')
    if (parts.length === 2) {
      return `民國 ${parts[0]} 年 ${Number(parts[1])} 月`
    }
    return rocMonth
  }

  function rocToGregorianMonth(rocMonth: string): string {
    if (!rocMonth) return ''
    const [rocYear, month] = rocMonth.split('-').map(Number)
    const ceYear = rocYear + 1911
    return `${ceYear}-${String(month).padStart(2, '0')}`
  }

  return {
    toRocMonth,
    toRocPeriodYm,
    formatRocMonthLabel,
    rocToGregorianMonth
  }
}
