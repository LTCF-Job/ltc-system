import { apiClient } from './client'
import type {
  ExportJobDTO,
  PrecheckResultDTO,
  CreateExportJobRequest,
  DashboardStatsDTO,
  Paged
} from '@/types/api'

export async function precheckExport(params: {
  periodYm: string
  region?: string
}): Promise<PrecheckResultDTO> {
  return apiClient.post('/exports/precheck', params)
}

export async function createExportJob(data: CreateExportJobRequest): Promise<ExportJobDTO> {
  return apiClient.post('/exports', data)
}

export async function getExportJob(jobId: string): Promise<ExportJobDTO> {
  return apiClient.get(`/exports/${jobId}`)
}

export async function listExportJobs(params?: {
  page?: number
  pageSize?: number
}): Promise<Paged<ExportJobDTO>> {
  return apiClient.get('/exports', { params })
}

export async function getDashboardStats(): Promise<DashboardStatsDTO> {
  return apiClient.get('/dashboard/stats')
}
