import dayjs from 'dayjs'

/**
 * 將時間字串格式化為「YYYY-MM-DD HH:mm:ss」（精確至秒，無時區字尾）
 */
export function formatDateTime(value?: string | number | Date | null, fallback = '-'): string {
  if (!value) return fallback
  const d = dayjs(value)
  if (!d.isValid()) return String(value)
  return d.format('YYYY-MM-DD HH:mm:ss')
}

/**
 * 將日期字串格式化為「YYYY-MM-DD」
 */
export function formatDate(value?: string | number | Date | null, fallback = '-'): string {
  if (!value) return fallback
  const d = dayjs(value)
  if (!d.isValid()) return String(value)
  return d.format('YYYY-MM-DD')
}
