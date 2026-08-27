import { apiClient } from './client'
import type {
  FormDTO,
  FormColumnDTO,
  CreateFormAssociationRequest,
  SyncFormOptions,
  UpdateColumnMappingRequest,
  BatchMappingRequest,
  GoogleDriveSheetDTO,
  InspectSheetResultDTO,
  InspectSheetRequest
} from '@/types/api'

export async function listGoogleDriveSheets(): Promise<GoogleDriveSheetDTO[]> {
  return apiClient.get('/forms/google-drive-files')
}

export async function inspectGoogleSheet(params: InspectSheetRequest): Promise<InspectSheetResultDTO> {
  return apiClient.post('/forms/inspect-sheet', params)
}

export async function listForms(params?: { q?: string }): Promise<FormDTO[]> {
  return apiClient.get('/forms', { params })
}

export async function createFormAssociation(data: CreateFormAssociationRequest): Promise<FormDTO> {
  return apiClient.post('/forms', data)
}

export async function syncForm(
  formId: string,
  options?: SyncFormOptions
): Promise<{ syncedRows: number; newColumns: number; month?: string; sheetTab?: string }> {
  return apiClient.post(`/forms/${formId}/sync`, options)
}

export async function deleteFormAssociation(formId: string): Promise<{ success: boolean }> {
  return apiClient.delete(`/forms/${formId}`)
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
