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

  // 多月匯出用：把一組西元月份換算成民國 5 碼，去重並升冪排序。
  // 使用者勾選的順序不固定，但送出去的月份清單決定要產幾份申報檔、批次檔名的起訖月份
  // 也由頭尾決定，順序與重複都必須先收斂。
  function toRocPeriodYmList(months: (Date | string)[]): string[] {
    return Array.from(new Set(months.map((month) => toRocPeriodYm(month)))).sort()
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
    toRocPeriodYmList,
    formatRocMonthLabel,
    rocToGregorianMonth
  }
}
