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
  return apiClient.get('/cases', { params })
}

export async function getCase(id: string): Promise<CaseDTO> {
  return apiClient.get(`/cases/${id}`)
}

export async function createCase(data: CreateCaseRequest): Promise<CaseDTO> {
  return apiClient.post('/cases', data)
}

export async function updateCase(id: string, data: UpdateCaseRequest): Promise<CaseDTO> {
  return apiClient.patch(`/cases/${id}`, data)
}

export async function revealCaseId(id: string): Promise<{ nationalId: string }> {
  return apiClient.post(`/cases/${id}/reveal`)
}

export async function getCaseSchedule(caseId: string): Promise<CaseScheduleDTO | null> {
  return apiClient.get(`/cases/${caseId}/schedule`)
}

export async function saveCaseSchedule(caseId: string, data: CreateScheduleRequest): Promise<CaseScheduleDTO> {
  return apiClient.put(`/cases/${caseId}/schedule`, data)
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
