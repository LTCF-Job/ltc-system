import { apiClient } from './client'
import type { DashboardMetricsDTO } from '@/types/api'

export async function getDashboardMetrics(month?: string): Promise<DashboardMetricsDTO> {
  return apiClient.get('/dashboard/metrics', { params: { month } })
}
