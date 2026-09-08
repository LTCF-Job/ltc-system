import { apiClient, unwrapData } from './client'

export interface HolidayItem {
  holidayDate: string
  name: string
  /** 全區假日為 null。 */
  source: string
  isDayOff: boolean
  createdAt?: string
}

export async function listHolidays(params?: {
  startDate?: string
  endDate?: string
}): Promise<HolidayItem[]> {
  const res = await apiClient.get('/holidays', { params })
  return unwrapData<HolidayItem[]>(res) ?? []
}

export async function createHoliday(data: {
  holidayDate: string
  name: string
  source?: string
  isDayOff?: boolean
}): Promise<HolidayItem> {
  const res = await apiClient.post('/holidays', data)
  return unwrapData<HolidayItem>(res)
}

export async function importGovHolidays(year: number): Promise<{ importedCount: number; year: number }> {
  const res = await apiClient.post('/holidays/import', { year })
  return unwrapData<{ importedCount: number; year: number }>(res)
}

export async function deleteHoliday(dateStr: string): Promise<{ deleted: boolean }> {
  const res = await apiClient.delete(`/holidays/${dateStr}`)
  return unwrapData<{ deleted: boolean }>(res)
}
