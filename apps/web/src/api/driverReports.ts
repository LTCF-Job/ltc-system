import { apiClient } from './client'
import type {
  DriverReportFormDTO,
  DriverReportColumnDTO,
  CreateDriverReportFormRequest,
  DriverReportPreviewDTO,
  DriverReportCommitResultDTO,
  DriverReportColumnDecision,
  UpdateColumnMappingRequest,
  BatchMappingRequest
} from '@/types/api'

export async function listDriverReportForms(params?: { q?: string }): Promise<DriverReportFormDTO[]> {
  return apiClient.get('/driver-reports', { params })
}

export async function createDriverReportForm(
  data: CreateDriverReportFormRequest
): Promise<DriverReportFormDTO> {
  return apiClient.post('/driver-reports', data)
}

export async function deleteDriverReportForm(formId: string): Promise<{ success: boolean }> {
  return apiClient.delete(`/driver-reports/${formId}`)
}

// downloadDriverReportTemplate 取回範本原始位元組；呼叫端負責觸發瀏覽器下載。
export async function downloadDriverReportTemplate(formId: string): Promise<Blob> {
  return apiClient.get(`/driver-reports/${formId}/template`, { responseType: 'blob' })
}

export async function dryRunImportDriverReport(
  formId: string,
  file: File
): Promise<DriverReportPreviewDTO> {
  const formData = new FormData()
  formData.append('file', file)
  return apiClient.post(`/driver-reports/${formId}/import?dryRun=true`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

export async function commitImportDriverReport(
  formId: string,
  file: File,
  columnDecisions: DriverReportColumnDecision[]
): Promise<DriverReportCommitResultDTO> {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('columnDecisions', JSON.stringify(columnDecisions))
  return apiClient.post(`/driver-reports/${formId}/import?dryRun=false`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

export async function listDriverReportColumns(params?: {
  formId?: string
  mappingStatus?: string
}): Promise<DriverReportColumnDTO[]> {
  return apiClient.get('/driver-reports/columns', { params })
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
