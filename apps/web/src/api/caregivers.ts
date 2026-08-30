import { apiClient } from './client'
import type {
  Paged,
  CaregiverDTO,
  CreateCaregiverRequest,
  UpdateCaregiverRequest,
  CaregiverImportCommitResult,
  DryRunImportResultDTO
} from '@/types/api'

// 照護人員 Caregivers
export async function listCaregivers(params?: {
  page?: number
  pageSize?: number
  q?: string
  unresolvedLink?: boolean
  incomplete?: boolean
}): Promise<Paged<CaregiverDTO>> {
  return apiClient.get('/caregivers', { params })
}

export async function createCaregiver(data: CreateCaregiverRequest): Promise<CaregiverDTO> {
  return apiClient.post('/caregivers', data)
}

export async function updateCaregiver(id: string, data: UpdateCaregiverRequest): Promise<CaregiverDTO> {
  return apiClient.patch(`/caregivers/${id}`, data)
}

export async function deleteCaregiver(id: string): Promise<void> {
  return apiClient.delete(`/caregivers/${id}`)
}

export async function linkCaregiverSite(id: string, siteId: string): Promise<CaregiverDTO> {
  return apiClient.put(`/caregivers/${id}/site`, { siteId })
}

// 照護人員批次匯入
export async function dryRunImportCaregivers(file: File): Promise<DryRunImportResultDTO> {
  const formData = new FormData()
  formData.append('file', file)
  return apiClient.post('/caregivers/import?dryRun=true', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

export async function commitImportCaregivers(file: File): Promise<CaregiverImportCommitResult> {
  const formData = new FormData()
  formData.append('file', file)
  return apiClient.post('/caregivers/import?dryRun=false', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

export async function downloadCaregiverTemplate(): Promise<Blob> {
  return apiClient.get('/caregivers/template', { responseType: 'blob' })
}
