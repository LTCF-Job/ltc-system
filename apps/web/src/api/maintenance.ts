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
  q?: string
}): Promise<Paged<MaintenanceLogDTO>> {
  const res = await apiClient.get<MaintenanceLogDTO[]>('/vehicles/maintenance', { params })
  const data = (res as any).data || (res as any)
  // 後端未回傳分頁 meta 時，以實際筆數推算，避免顯示與清單內容矛盾的假總數
  return {
    data,
    meta: (res as any).meta || { page: params?.page || 1, pageSize: params?.pageSize || 20, total: data.length, totalPages: 1 }
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
