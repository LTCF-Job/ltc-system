import { apiClient, createPaginationMeta, unwrapData, unwrapPaged } from './client'
import type {
  Paged,
  CaseDTO,
  CreateCaseRequest,
  UpdateCaseRequest,
  UpdateCaseTransportPreferenceRequest,
  CaseScheduleDTO,
  SaveScheduleRequest,
  DryRunImportResultDTO,
  CaseImportCommitResult,
  CaseDuplicateCandidateDTO,
  ResolveDuplicateCandidateRequest
} from '@/types/api'

export async function listCases(params?: {
  page?: number
  pageSize?: number
  status?: string
  q?: string
  // 依已關聯據點的區域篩選（COALESCE(sites.region, '') ILIKE 比對）。
  region?: string
  // 待維護個案預設不會回傳；unresolvedLink 只取待維護，includePending 取全部。
  unresolvedLink?: boolean
  includePending?: boolean
}): Promise<Paged<CaseDTO>> {
  const res = await apiClient.get('/cases', { params })
  const fallback = createPaginationMeta(params?.page, params?.pageSize)
  return unwrapPaged<CaseDTO>(res, fallback)
}

export async function listAllCases(params?: Omit<NonNullable<Parameters<typeof listCases>[0]>, 'page' | 'pageSize'>): Promise<CaseDTO[]> {
  const items: CaseDTO[] = []
  let page = 1
  const pageSize = 100
  while (true) {
    const result = await listCases({ ...params, page, pageSize })
    const pageItems = result.data || []
    items.push(...pageItems)
    if (page >= (result.meta?.totalPages || Math.ceil((result.meta?.total || items.length) / pageSize)) || pageItems.length === 0) break
    page += 1
  }
  return items
}

export async function getCase(id: string): Promise<CaseDTO> {
  const res = await apiClient.get(`/cases/${id}`)
  return unwrapData<CaseDTO>(res)
}

export async function createCase(data: CreateCaseRequest): Promise<CaseDTO> {
  const res = await apiClient.post('/cases', data)
  return unwrapData<CaseDTO>(res)
}

export async function updateCase(id: string, data: UpdateCaseRequest): Promise<CaseDTO> {
  const res = await apiClient.patch(`/cases/${id}`, data)
  return unwrapData<CaseDTO>(res)
}

export async function deleteCase(id: string): Promise<void> {
	const res = await apiClient.delete(`/cases/${id}`)
	unwrapData<unknown>(res)
}

export async function updateCaseTransportPreference(
  id: string,
  data: UpdateCaseTransportPreferenceRequest
): Promise<CaseDTO> {
  const res = await apiClient.put(`/cases/${id}/transport-preference`, data)
  return unwrapData<CaseDTO>(res)
}

export async function downloadCaseImportTemplate(): Promise<Blob> {
  return apiClient.get('/cases/template', { responseType: 'blob' })
}

export async function exportCaseProfileWorkbook(caseIds?: string[]): Promise<Blob> {
  const params = caseIds && caseIds.length > 0 ? { caseIds: caseIds.join(',') } : undefined
  return apiClient.get('/cases/export', { params, responseType: 'blob' })
}

export async function getCaseSchedule(caseId: string): Promise<CaseScheduleDTO | null> {
  const res = await apiClient.get(`/cases/${caseId}/schedule`)
  return unwrapData<CaseScheduleDTO | null>(res)
}

export async function saveCaseSchedule(caseId: string, data: SaveScheduleRequest): Promise<CaseScheduleDTO> {
  const res = await apiClient.put(`/cases/${caseId}/schedule`, data)
  return unwrapData<CaseScheduleDTO>(res)
}

export async function dryRunImportCases(file: File): Promise<DryRunImportResultDTO> {
  const formData = new FormData()
  formData.append('file', file)
  const res = await apiClient.post('/cases/import?dryRun=true', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
  return unwrapData<DryRunImportResultDTO>(res)
}

export async function commitImportCases(file: File): Promise<CaseImportCommitResult> {
  const formData = new FormData()
  formData.append('file', file)
  const res = await apiClient.post('/cases/import?dryRun=false', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
  return unwrapData<CaseImportCommitResult>(res)
}

export async function listCaseDuplicateCandidates(): Promise<CaseDuplicateCandidateDTO[]> {
  const res = await apiClient.get('/cases/import/duplicates')
  return unwrapData<CaseDuplicateCandidateDTO[]>(res)
}

export async function resolveCaseDuplicateCandidate(id: string, data: ResolveDuplicateCandidateRequest): Promise<CaseDTO> {
  const res = await apiClient.post(`/cases/import/duplicates/${id}/resolve`, data)
  return unwrapData<CaseDTO>(res)
}

// discardCaseDuplicateCandidate 忽略一筆疑似重複個案，直接把暫存列從系統刪除，不建立個案。
export async function discardCaseDuplicateCandidate(id: string): Promise<void> {
  await apiClient.delete(`/cases/import/duplicates/${id}`)
}
