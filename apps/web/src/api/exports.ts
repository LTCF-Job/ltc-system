import { apiClient, createPaginationMeta, unwrapData, unwrapPaged } from './client'
import type {
  ExportJobDTO,
  PrecheckResultDTO,
  CreateExportJobRequest,
  RegionExportRequest,
  RegionExportResultDTO,
  SiteTripSummaryRequest,
  DashboardStatsDTO,
  Paged
} from '@/types/api'

// 前置檢核同時服務三種選取模式：periodYms 未給時退回單一 periodYm，
// regions 未給時沿用 caseIds，逐案勾選的既有請求形狀完全不變。
export async function precheckExport(params: {
  periodYm?: string
  periodYms?: string[]
  caseIds?: string[]
  regions?: string[]
}): Promise<PrecheckResultDTO> {
  const res = await apiClient.post('/exports/precheck', params)
  return unwrapData<PrecheckResultDTO>(res)
}

// 以區域批次匯出：逐月各建立一個匯出工作，回傳每月結果與跨月合併下載連結
export async function createRegionExportJobs(
  data: RegionExportRequest
): Promise<RegionExportResultDTO> {
  // 後端會逐月產檔並上傳 object storage，時間可能超過全域 30 秒 timeout；
  // 在非同步工作架構完成前，這個同步端點必須由 client 等待完整回應。
  const res = await apiClient.post('/exports/by-region', data, { timeout: 0 })
  return unwrapData<RegionExportResultDTO>(res)
}

export async function createExportJob(data: CreateExportJobRequest): Promise<ExportJobDTO> {
  const res = await apiClient.post('/exports', data)
  return unwrapData<ExportJobDTO>(res)
}

export async function getExportJob(jobId: string): Promise<ExportJobDTO> {
  const res = await apiClient.get(`/exports/${jobId}`)
  return unwrapData<ExportJobDTO>(res)
}

export async function listExportJobs(params?: {
  page?: number
  pageSize?: number
}): Promise<Paged<ExportJobDTO>> {
	const res = await apiClient.get('/exports', { params })
	return unwrapPaged<ExportJobDTO>(res, createPaginationMeta(params?.page, params?.pageSize))
}

export async function getDashboardStats(): Promise<DashboardStatsDTO> {
  const res = await apiClient.get('/dashboard/stats')
  return unwrapData<DashboardStatsDTO>(res)
}

// 逐案下載：一個個案一個月一份工作簿
export async function downloadExportCaseFile(jobId: string, caseId: string): Promise<Blob> {
  return apiClient.get(`/exports/${jobId}/files/${caseId}/download`, { responseType: 'blob' })
}

// 整包下載：僅壓縮檔模式的工作可用
export async function downloadExportZip(jobId: string): Promise<Blob> {
  return apiClient.get(`/exports/${jobId}/download`, { responseType: 'blob' })
}

// 跨月批次下載：把多筆匯出工作的檔案合併成一包，zip 內以民國年月分資料夾。
// jobIds 以重複參數送出（jobIds=a&jobIds=b），與後端的 QueryArray 解析一致。
export async function downloadExportBatchZip(jobIds: string[]): Promise<Blob> {
  return apiClient.get('/exports/batch-download', {
    params: { jobIds },
    paramsSerializer: {
      indexes: null
    },
    responseType: 'blob'
  })
}

// 據點趟數彙總表：即時產檔直接回傳位元組，不建立匯出工作、不進歷史紀錄
export async function downloadSiteTripSummary(data: SiteTripSummaryRequest): Promise<Blob> {
  return apiClient.post('/exports/site-trip-summary', data, { responseType: 'blob' })
}
