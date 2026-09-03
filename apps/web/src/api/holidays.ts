import { apiClient } from './client'
import type { ApiResponse } from '@/types/api'

export interface HolidayItem {
  holidayDate: string
  name: string
  region?: string
  source: string
  isDayOff: boolean
  createdAt?: string
}

export async function listHolidays(params?: {
  startDate?: string
  endDate?: string
  region?: string
}): Promise<ApiResponse<HolidayItem[]>> {
  return apiClient.get('/holidays', { params })
}

export async function createHoliday(data: {
  holidayDate: string
  name: string
  region?: string
  source?: string
  isDayOff?: boolean
}): Promise<ApiResponse<HolidayItem>> {
  return apiClient.post('/holidays', data)
}

export async function importGovHolidays(year: number): Promise<ApiResponse<{ importedCount: number; year: number }>> {
  return apiClient.post('/holidays/import', { year })
}

export async function deleteHoliday(dateStr: string): Promise<ApiResponse<{ deleted: boolean }>> {
  return apiClient.delete(`/holidays/${dateStr}`)
}
