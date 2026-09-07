import { apiClient, createPaginationMeta, unwrapData, unwrapPaged } from './client'
import type {
  MaintenanceLogDTO,
  CreateMaintenanceRequest,
  UpdateMaintenanceRequest,
  Paged
} from '@/types/api'

export async function listMaintenance(params?: {
  page?: number
  pageSize?: number
  vehicleId?: string
  startDate?: string
  endDate?: string
  q?: string
}): Promise<Paged<MaintenanceLogDTO>> {
  const res = await apiClient.get('/vehicles/maintenance', { params })
  return unwrapPaged<MaintenanceLogDTO>(res, createPaginationMeta(params?.page, params?.pageSize))
}

export async function createMaintenance(data: CreateMaintenanceRequest): Promise<MaintenanceLogDTO> {
  const res = await apiClient.post('/vehicles/maintenance', data)
  return unwrapData<MaintenanceLogDTO>(res)
}

export async function updateMaintenance(id: string, data: UpdateMaintenanceRequest): Promise<MaintenanceLogDTO> {
  const res = await apiClient.patch(`/vehicles/maintenance/${id}`, data)
  return unwrapData<MaintenanceLogDTO>(res)
}

export async function deleteMaintenance(id: string): Promise<{ success: boolean }> {
  const res = await apiClient.delete(`/vehicles/maintenance/${id}`)
  return unwrapData<{ success: boolean }>(res)
}

export async function downloadBlankMaintenanceExcel(): Promise<Blob> {
  return apiClient.get('/vehicles/maintenance/blank-template', {
    responseType: 'blob'
  })
}
