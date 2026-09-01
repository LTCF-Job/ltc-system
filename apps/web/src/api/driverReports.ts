import { apiClient } from './client'
import type {
  DriverReportFormDTO,
  DriverReportColumnDTO,
  DriverReportImportedMonthDTO,
  CreateDriverReportFormRequest,
  DriverReportPreviewDTO,
  DriverReportCommitResultDTO,
  DriverReportColumnDecision,
  UpdateColumnMappingRequest,
  BatchMappingRequest
} from '@/types/api'

export async function listDriverReportForms(params?: { q?: string }): Promise<DriverReportFormDTO[]> {
  const res = await apiClient.get('/driver-reports', { params })
  return (res as any).data ?? (res as any)
}

// listDriverReportImportedMonths 取回每份匯報表各月份已匯入的筆數，供批次上傳判斷重傳。
export async function listDriverReportImportedMonths(): Promise<DriverReportImportedMonthDTO[]> {
  const res = await apiClient.get('/driver-reports/imported-months')
  return (res as any).data ?? (res as any)
}

export async function createDriverReportForm(
  data: CreateDriverReportFormRequest
): Promise<DriverReportFormDTO> {
  const res = await apiClient.post('/driver-reports', data)
  return (res as any).data ?? (res as any)
}

export async function deleteDriverReportForm(formId: string): Promise<{ success: boolean }> {
  return apiClient.delete(`/driver-reports/${formId}`)
}

// downloadDriverReportTemplate 取回範本原始位元組；呼叫端負責觸發瀏覽器下載。
export async function downloadDriverReportTemplate(formId: string): Promise<Blob> {
  return apiClient.get(`/driver-reports/${formId}/template`, { responseType: 'blob' })
}

// buildImportUrl 組出匯入端點；yearMonth 為選填的宣告匯入月份（YYYY-MM）。
// 帶月份時後端以該月為單位覆蓋，並拒收含有該月以外日期的檔案。
function buildImportUrl(formId: string, dryRun: boolean, yearMonth?: string): string {
  const query = new URLSearchParams({ dryRun: String(dryRun) })
  if (yearMonth) query.set('yearMonth', yearMonth)
  return `/driver-reports/${formId}/import?${query.toString()}`
}

export async function dryRunImportDriverReport(
  formId: string,
  file: File,
  yearMonth?: string
): Promise<DriverReportPreviewDTO> {
  const formData = new FormData()
  formData.append('file', file)
  return apiClient.post(buildImportUrl(formId, true, yearMonth), formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

export async function commitImportDriverReport(
  formId: string,
  file: File,
  columnDecisions: DriverReportColumnDecision[],
  yearMonth?: string
): Promise<DriverReportCommitResultDTO> {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('columnDecisions', JSON.stringify(columnDecisions))
  return apiClient.post(buildImportUrl(formId, false, yearMonth), formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

export async function listDriverReportColumns(params?: {
  formId?: string
  mappingStatus?: string
}): Promise<DriverReportColumnDTO[]> {
  const res = await apiClient.get('/driver-reports/columns', { params })
  return (res as any).data ?? (res as any)
}

export async function updateColumnMapping(
  columnId: string,
  data: UpdateColumnMappingRequest
): Promise<DriverReportColumnDTO> {
  return apiClient.patch(`/driver-reports/columns/${columnId}/mapping`, data)
}

export async function batchUpdateColumnMappings(
  data: BatchMappingRequest
): Promise<{ updatedCount: number }> {
  return apiClient.post('/driver-reports/columns/batch-mapping', data)
}
