import { apiClient, unwrapData } from './client'
import type { TripSummaryReportDTO, HsinchuScheduleReportDTO } from '@/types/api'

export async function getTripSummaryReport(params: {
  periodYm: string
  vehicleId?: string
  q?: string
}): Promise<TripSummaryReportDTO> {
  const res = await apiClient.get('/reports/trip-summary', { params })
  return unwrapData<TripSummaryReportDTO>(res)
}

export async function exportTripSummaryExcel(params: {
  periodYm: string
  vehicleId?: string
}): Promise<Blob> {
  return apiClient.get('/reports/trip-summary/export', {
    params,
    responseType: 'blob'
  })
}

export async function getHsinchuSchedule(params?: {
  siteId?: string
  vehicleId?: string
  q?: string
}): Promise<HsinchuScheduleReportDTO> {
  const res = await apiClient.get('/reports/hsinchu-schedule', { params })
  return unwrapData<HsinchuScheduleReportDTO>(res)
}

export async function exportHsinchuScheduleExcel(params?: {
  siteId?: string
  vehicleId?: string
}): Promise<Blob> {
  return apiClient.get('/reports/hsinchu-schedule/export', {
    params,
    responseType: 'blob'
  })
}
