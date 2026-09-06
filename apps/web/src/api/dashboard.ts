import { apiClient, unwrapData } from './client'
import type { DashboardMetricsDTO } from '@/types/api'

export async function getDashboardMetrics(month?: string): Promise<DashboardMetricsDTO> {
  const res = await apiClient.get('/dashboard/metrics', { params: { month } })
  return unwrapData<DashboardMetricsDTO>(res)
}
