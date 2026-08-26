import { apiClient } from './client'
import type {
  FormDTO,
  FormColumnDTO,
  UpdateColumnMappingRequest,
  BatchMappingRequest
} from '@/types/api'

export async function listForms(params?: { q?: string }): Promise<FormDTO[]> {
  return apiClient.get('/forms', { params })
}

export async function syncForm(formId: string): Promise<{ syncedRows: number; newColumns: number }> {
  return apiClient.post(`/forms/${formId}/sync`)
}

export async function listFormColumns(params?: {
  formId?: string
  mappingStatus?: string
  kind?: string
}): Promise<FormColumnDTO[]> {
  return apiClient.get('/forms/columns', { params })
}

export async function updateColumnMapping(
  columnId: string,
  data: UpdateColumnMappingRequest
): Promise<FormColumnDTO> {
  return apiClient.patch(`/forms/columns/${columnId}/mapping`, data)
}

export async function batchUpdateColumnMappings(data: BatchMappingRequest): Promise<{ updatedCount: number }> {
  return apiClient.post('/forms/columns/batch-mapping', data)
}
