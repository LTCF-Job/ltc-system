// 民國年月顯示與格式轉換（僅呈現層輔助，無業務規則運算）

export function useRocMonth() {
  function toRocMonth(date: Date | string): string {
    const d = typeof date === 'string' ? new Date(date) : date
    const rocYear = d.getFullYear() - 1911
    const month = String(d.getMonth() + 1).padStart(2, '0')
    return `${rocYear}-${month}`
  }

  function toRocPeriodYm(date: Date | string): string {
    const d = typeof date === 'string' ? new Date(date) : date
    const rocYear = d.getFullYear() - 1911
    const month = String(d.getMonth() + 1).padStart(2, '0')
    return `${rocYear}${month}`
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
