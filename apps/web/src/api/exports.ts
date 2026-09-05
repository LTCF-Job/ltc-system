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
  caseIds?: string[]
}): Promise<PrecheckResultDTO> {
  const res = await apiClient.post('/exports/precheck', params)
  return (res as any).data ?? (res as any)
}

export async function createExportJob(data: CreateExportJobRequest): Promise<ExportJobDTO> {
  const res = await apiClient.post('/exports', data)
  return (res as any).data ?? (res as any)
}

export async function getExportJob(jobId: string): Promise<ExportJobDTO> {
  const res = await apiClient.get(`/exports/${jobId}`)
  return (res as any).data ?? (res as any)
}

export async function listExportJobs(params?: {
  page?: number
  pageSize?: number
}): Promise<Paged<ExportJobDTO>> {
  return apiClient.get('/exports', { params })
}

export async function getDashboardStats(): Promise<DashboardStatsDTO> {
  const res = await apiClient.get('/dashboard/stats')
  return (res as any).data ?? (res as any)
}

// 逐案下載：一個個案一個月一份工作簿
export async function downloadExportCaseFile(jobId: string, caseId: string): Promise<Blob> {
  return apiClient.get(`/exports/${jobId}/files/${caseId}/download`, { responseType: 'blob' })
}

// 整包下載：僅壓縮檔模式的工作可用
export async function downloadExportZip(jobId: string): Promise<Blob> {
  return apiClient.get(`/exports/${jobId}/download`, { responseType: 'blob' })
}
