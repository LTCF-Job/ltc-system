import { apiClient, createPaginationMeta, unwrapData, unwrapPaged } from './client'
import type {
  MonthAttendanceReportDTO,
  AttendanceRecordDTO,
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
  return unwrapData<MonthAttendanceReportDTO>(res)
}

export async function upsertAttendance(data: UpsertAttendanceRequest): Promise<AttendanceRecordDTO> {
  const res = await apiClient.post('/attendance', data)
  return unwrapData<AttendanceRecordDTO>(res)
}

// listAttendanceConflicts 取回司機接送匯報匯入自動同步出勤時，與人工登記不一致的待維護衝突。
export async function listAttendanceConflicts(): Promise<AttendanceConflictDTO[]> {
  const res = await apiClient.get('/attendance/conflicts')
  return unwrapData<AttendanceConflictDTO[]>(res) ?? []
}

// resolveAttendanceConflict 依使用者選擇解決一筆出勤待維護衝突。
export async function resolveAttendanceConflict(
  id: string,
  data: ResolveAttendanceConflictRequest
): Promise<AttendanceConflictDTO> {
  const res = await apiClient.post(`/attendance/conflicts/${id}/resolve`, data)
  return unwrapData<AttendanceConflictDTO>(res)
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
  const res = await apiClient.get('/fuel-logs', { params })
  const fallback = createPaginationMeta(params?.page, params?.pageSize)
  return unwrapPaged<FuelLogDTO>(res, fallback)
}

export async function createFuelLog(data: CreateFuelLogRequest): Promise<FuelLogDTO> {
  const res = await apiClient.post('/fuel-logs', data)
  return unwrapData<FuelLogDTO>(res)
}

export async function updateFuelLog(id: string, data: UpdateFuelLogRequest): Promise<FuelLogDTO> {
  const res = await apiClient.patch(`/fuel-logs/${id}`, data)
  return unwrapData<FuelLogDTO>(res)
}

export async function deleteFuelLog(id: string): Promise<{ success: boolean }> {
  const res = await apiClient.delete(`/fuel-logs/${id}`)
  return unwrapData<{ success: boolean }>(res)
}
