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
  BatchMappingRequest,
  SubmissionReviewDTO,
  BindDriverRequest,
  DriverReportMonthDetailDTO
} from '@/types/api'

export async function listDriverReportForms(params?: { q?: string }): Promise<DriverReportFormDTO[]> {
  const res = await apiClient.get('/driver-reports', { params })
  return (res as any)?.data ?? []
}

// listDriverReportImportedMonths 取回每份匯報表各月份已匯入的筆數，供批次上傳判斷重傳。
export async function listDriverReportImportedMonths(): Promise<DriverReportImportedMonthDTO[]> {
  const res = await apiClient.get('/driver-reports/imported-months')
  return (res as any)?.data ?? []
}

export async function createDriverReportForm(
  data: CreateDriverReportFormRequest
): Promise<DriverReportFormDTO> {
  const res = await apiClient.post('/driver-reports', data)
  return (res as any).data ?? (res as any)
}

export async function deleteDriverReportForm(formId: string): Promise<{ success: boolean }> {
  const res = await apiClient.delete(`/driver-reports/${formId}`)
  return (res as any)?.data ?? res
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
  const res = await apiClient.post(buildImportUrl(formId, true, yearMonth), formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
  return (res as any)?.data ?? res
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
  const res = await apiClient.post(buildImportUrl(formId, false, yearMonth), formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
  return (res as any)?.data ?? res
}

export async function listDriverReportColumns(params?: {
  formId?: string
  mappingStatus?: string
}): Promise<DriverReportColumnDTO[]> {
  const res = await apiClient.get('/driver-reports/columns', { params })
  return (res as any)?.data ?? []
}

export async function updateColumnMapping(
  columnId: string,
  data: UpdateColumnMappingRequest
): Promise<{ backfilledRows: number }> {
  const res = await apiClient.patch(`/driver-reports/columns/${columnId}/mapping`, data)
  return (res as any)?.data ?? res
}

export async function batchUpdateColumnMappings(
  data: BatchMappingRequest
): Promise<{ updatedCount: number }> {
  const res = await apiClient.post('/driver-reports/columns/batch-mapping', data)
  return (res as any)?.data ?? res
}

// matchPendingColumnsByName 找出目前待維護欄位中姓名與傳入姓名相符（含近似）的欄位，
// 供新建個案後主動詢問使用者這批欄位是否也是同一人。
export async function matchPendingColumnsByName(name: string): Promise<DriverReportColumnDTO[]> {
  const res = await apiClient.get('/driver-reports/columns/name-matches', { params: { name } })
  return (res as any)?.data ?? []
}

// listSubmissionReview 以匯報表列為單位列出待維護資料，一列可能同時有個案欄位與駕駛人兩種問題。
export async function listSubmissionReview(): Promise<SubmissionReviewDTO[]> {
  const res = await apiClient.get('/driver-reports/submissions/review')
  return (res as any)?.data ?? []
}

// bindPendingDriver 把某個比對不到司機主檔的原始姓名綁定到指定司機，立即回填既有回報
// 已寫入的搭乘紀錄，不需要重新上傳原始檔案。
export async function bindPendingDriver(data: BindDriverRequest): Promise<{ affectedCount: number }> {
  const res = await apiClient.post('/driver-reports/drivers/bind', data)
  return (res as any)?.data ?? res
}

// getDriverReportMonthDetail 取回某份匯報表指定月份（YYYY-MM）已匯入的完整內容，
// 供總覽頁鑽取單一月份時顯示逐日回報明細與展開後的個案搭乘紀錄。
export async function getDriverReportMonthDetail(
  formId: string,
  yearMonth: string
): Promise<DriverReportMonthDetailDTO> {
  const res = await apiClient.get(`/driver-reports/${formId}/months/${yearMonth}`)
  return (res as any)?.data ?? res
}
