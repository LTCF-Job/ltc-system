import { apiClient } from './client'
import type { TripSummaryReportDTO } from '@/types/api'

export async function getTripSummaryReport(params: {
  periodYm: string
  region?: string
  vehicleId?: string
}): Promise<TripSummaryReportDTO> {
  return apiClient.get('/reports/trip-summary', { params })
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
