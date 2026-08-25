import { apiClient } from './client'
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
}): Promise<Paged<MaintenanceLogDTO>> {
  const res = await apiClient.get<MaintenanceLogDTO[]>('/vehicles/maintenance', { params })
  return {
    data: (res as any).data || (res as any),
    meta: (res as any).meta || { page: 1, pageSize: 20, total: 0, totalPages: 1 }
  }
}

export async function createMaintenance(data: CreateMaintenanceRequest): Promise<MaintenanceLogDTO> {
  return apiClient.post('/vehicles/maintenance', data)
}

export async function updateMaintenance(id: string, data: UpdateMaintenanceRequest): Promise<MaintenanceLogDTO> {
  return apiClient.patch(`/vehicles/maintenance/${id}`, data)
}

export async function deleteMaintenance(id: string): Promise<{ success: boolean }> {
  return apiClient.delete(`/vehicles/maintenance/${id}`)
}

export async function downloadBlankMaintenanceExcel(): Promise<Blob> {
  return apiClient.get('/vehicles/maintenance/blank-template', {
    responseType: 'blob'
  })
}
