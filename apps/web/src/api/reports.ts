import { apiClient } from './client'
import type { TripSummaryReportDTO, HsinchuScheduleReportDTO } from '@/types/api'

export async function getTripSummaryReport(params: {
  periodYm: string
  region?: string
  vehicleId?: string
  q?: string
}): Promise<TripSummaryReportDTO> {
  const res = await apiClient.get('/reports/trip-summary', { params })
  return (res as any).data ?? (res as any)
}

export async function exportTripSummaryExcel(params: {
  periodYm: string
  region?: string
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
  return (res as any).data ?? (res as any)
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
