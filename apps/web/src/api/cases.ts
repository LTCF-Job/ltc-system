import { apiClient } from './client'
import type {
  Paged,
  CaseDTO,
  CreateCaseRequest,
  UpdateCaseRequest,
  CaseScheduleDTO,
  CreateScheduleRequest,
  DryRunImportResultDTO
} from '@/types/api'

export async function listCases(params?: {
  page?: number
  pageSize?: number
  region?: string
  status?: string
  q?: string
}): Promise<Paged<CaseDTO>> {
  const res: any = await apiClient.get('/cases', { params })
  const rawData = res?.data ?? res
  const meta = res?.meta ?? {
    page: params?.page || 1,
    pageSize: params?.pageSize || 20,
    total: Array.isArray(rawData) ? rawData.length : (rawData?.total || 0),
    totalPages: 1
  }
  const list: any[] = Array.isArray(rawData) ? rawData : (rawData?.data || [])
  list.forEach((item) => {
    item.nationalId = item.nationalId || item.nationalIdMasked || ''
  })
  return {
    data: list,
    meta
  }
}

export async function getCase(id: string): Promise<CaseDTO> {
  const res: any = await apiClient.get(`/cases/${id}`)
  const data = res?.data ?? res
  if (data) {
    data.nationalId = data.nationalId || data.nationalIdMasked || ''
  }
  return data
}

export async function createCase(data: CreateCaseRequest): Promise<CaseDTO> {
  const res: any = await apiClient.post('/cases', data)
  return res?.data ?? res
}

export async function updateCase(id: string, data: UpdateCaseRequest): Promise<CaseDTO> {
  const res: any = await apiClient.patch(`/cases/${id}`, data)
  return res?.data ?? res
}

export async function deleteCase(id: string): Promise<void> {
  return apiClient.delete(`/cases/${id}`)
}

export async function downloadCaseImportTemplate(): Promise<Blob> {
  return apiClient.get('/cases/template', {
    responseType: 'blob'
  })
}

export async function revealCaseId(id: string): Promise<{ nationalId: string }> {
  const res: any = await apiClient.post(`/cases/${id}/reveal`)
  return res?.data ?? res
}

export async function getCaseSchedule(caseId: string): Promise<CaseScheduleDTO | null> {
  const res: any = await apiClient.get(`/cases/${caseId}/schedule`)
  return res?.data ?? res ?? null
}

export async function saveCaseSchedule(caseId: string, data: CreateScheduleRequest): Promise<CaseScheduleDTO> {
  const res: any = await apiClient.put(`/cases/${caseId}/schedule`, data)
  return res?.data ?? res
}

export async function dryRunImportCases(file: File): Promise<DryRunImportResultDTO> {
  const formData = new FormData()
  formData.append('file', file)
  return apiClient.post('/cases/import?dryRun=true', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

export async function commitImportCases(file: File): Promise<{ count: number }> {
  const formData = new FormData()
  formData.append('file', file)
  return apiClient.post('/cases/import?dryRun=false', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}
