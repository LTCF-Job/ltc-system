import { apiClient } from './client'
import type {
  MonthAttendanceReportDTO,
  UpsertAttendanceRequest,
  AttendanceConflictDTO,
  ResolveAttendanceConflictRequest,
  FuelLogDTO,
  CreateFuelLogRequest,
  UpdateFuelLogRequest,
  Paged
} from '@/types/api'

export async function getMonthAttendance(month?: string, driverId?: string, q?: string): Promise<MonthAttendanceReportDTO> {
  const res = await apiClient.get('/attendance', { params: { month, driverId, q } })
  return (res as any).data ?? (res as any)
}

export async function upsertAttendance(data: UpsertAttendanceRequest): Promise<any> {
  return apiClient.post('/attendance', data)
}

// listAttendanceConflicts 取回司機接送匯報匯入自動同步出勤時，與人工登記不一致的待維護衝突。
export async function listAttendanceConflicts(): Promise<AttendanceConflictDTO[]> {
  const res = await apiClient.get('/attendance/conflicts')
  return (res as any)?.data ?? []
}

// resolveAttendanceConflict 依使用者選擇解決一筆出勤待維護衝突。
export async function resolveAttendanceConflict(
  id: string,
  data: ResolveAttendanceConflictRequest
): Promise<AttendanceConflictDTO> {
  const res = await apiClient.post(`/attendance/conflicts/${id}/resolve`, data)
  return (res as any)?.data ?? res
}

export async function listFuelLogs(params?: {
  page?: number
  pageSize?: number
  vehicleId?: string
  driverId?: string
  startDate?: string
  endDate?: string
  q?: string
}): Promise<Paged<FuelLogDTO>> {
  const res = await apiClient.get<FuelLogDTO[]>('/fuel-logs', { params })
  const data = (res as any).data || (res as any)
  // 後端未回傳分頁 meta 時，以實際筆數推算，避免顯示與清單內容矛盾的假總數
  return {
    data,
    meta: (res as any).meta || { page: params?.page || 1, pageSize: params?.pageSize || 20, total: data.length, totalPages: 1 }
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
