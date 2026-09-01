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

// 逐案下載：一個個案一個月一份工作簿
export async function downloadExportCaseFile(jobId: string, caseId: string): Promise<Blob> {
  return apiClient.get(`/exports/${jobId}/files/${caseId}/download`, { responseType: 'blob' })
}

// 整包下載：僅壓縮檔模式的工作可用
export async function downloadExportZip(jobId: string): Promise<Blob> {
  return apiClient.get(`/exports/${jobId}/download`, { responseType: 'blob' })
}
