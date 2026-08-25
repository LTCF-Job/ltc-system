import { apiClient } from './client'
import type {
  MonthAttendanceReportDTO,
  UpsertAttendanceRequest,
  FuelLogDTO,
  CreateFuelLogRequest,
  UpdateFuelLogRequest,
  Paged
} from '@/types/api'

export async function getMonthAttendance(month?: string, driverId?: string): Promise<MonthAttendanceReportDTO> {
  return apiClient.get('/attendance', { params: { month, driverId } })
}

export async function upsertAttendance(data: UpsertAttendanceRequest): Promise<any> {
  return apiClient.post('/attendance', data)
}

export async function listFuelLogs(params?: {
  page?: number
  pageSize?: number
  vehicleId?: string
  driverId?: string
  startDate?: string
  endDate?: string
}): Promise<Paged<FuelLogDTO>> {
  const res = await apiClient.get<FuelLogDTO[]>('/fuel-logs', { params })
  return {
    data: (res as any).data || (res as any),
    meta: (res as any).meta || { page: 1, pageSize: 20, total: 0, totalPages: 1 }
  }
}

export async function createFuelLog(data: CreateFuelLogRequest): Promise<FuelLogDTO> {
  return apiClient.post('/fuel-logs', data)
}

export async function updateFuelLog(id: string, data: UpdateFuelLogRequest): Promise<FuelLogDTO> {
  return apiClient.patch(`/fuel-logs/${id}`, data)
}

export async function deleteFuelLog(id: string): Promise<{ success: boolean }> {
  return apiClient.delete(`/fuel-logs/${id}`)
}
