import dayjs from 'dayjs'

/**
 * 將日期時間格式化為「YYYY-MM-DD HH:mm:ss」（精確至秒，去除毫秒與時區字尾）
 */
export function formatDateTime(value?: string | number | Date | null, fallback = '-'): string {
  if (!value) return fallback
  const d = dayjs(value)
  if (!d.isValid()) return String(value)
  return d.format('YYYY-MM-DD HH:mm:ss')
}

/**
 * 將日期格式化為「YYYY-MM-DD」
 */
export function formatDate(value?: string | number | Date | null, fallback = '-'): string {
  if (!value) return fallback
  const d = dayjs(value)
  if (!d.isValid()) return String(value)
  return d.format('YYYY-MM-DD')
}

/**
 * 將時間格式化為「HH:mm:ss」（精確至秒）
 */
export function formatTime(value?: string | number | Date | null, fallback = '-'): string {
  if (!value) return fallback
  // 處理純時分格式（如 08:30 轉為 08:30:00）
  if (typeof value === 'string' && /^\d{2}:\d{2}$/.test(value.trim())) {
    return `${value.trim()}:00`
  }
  const d = dayjs(value)
  if (!d.isValid()) return String(value)
  return d.format('HH:mm:ss')
}


/**
 * 將日期格式化為民國格式「114/12/23」
 */
export function formatRocDate(value?: string | number | Date | null, fallback = '-'): string {
  if (!value) return fallback
  const d = dayjs(value)
  if (!d.isValid()) return String(value)
  return `${d.year() - 1911}/${d.format('MM/DD')}`
}

/**
 * 將「YYYY-MM」年月格式化為「2013 年 03 月」
 */
export function formatYearMonth(value?: string | null, fallback = '-'): string {
  if (!value) return fallback
  const [year, month] = value.split('-')
  if (!year || !month) return value
  return `${year} 年 ${month} 月`
}
