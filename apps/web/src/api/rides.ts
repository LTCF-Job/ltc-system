import { apiClient, createPaginationMeta, unwrapData, unwrapPaged } from './client'
import type {
  RideCalendarMatrixDTO,
  RideRecordDTO,
  PatchRideRequest,
  ResolveConflictRequest,
  ManualReportRideRequest,
  IssueRideDTO,
  MissingRideDTO,
  Paged
} from '@/types/api'

export async function getRideCalendarMatrix(params: {
  month: string // RRR-MM 或 YYYY-MM
  q?: string
}): Promise<RideCalendarMatrixDTO> {
  const res = await apiClient.get('/rides/calendar', { params })
  return unwrapData<RideCalendarMatrixDTO>(res)
}

export async function getRideRecord(id: string): Promise<RideRecordDTO> {
  const res = await apiClient.get(`/rides/${id}`)
  return unwrapData<RideRecordDTO>(res)
}

export async function patchRideRecord(id: string, data: PatchRideRequest): Promise<RideRecordDTO> {
  const res = await apiClient.patch(`/rides/${id}`, data)
  return unwrapData<RideRecordDTO>(res)
}

export async function submitManualRideReport(data: ManualReportRideRequest): Promise<RideRecordDTO> {
  const res = await apiClient.post('/rides/manual-report', data)
  return unwrapData<RideRecordDTO>(res)
}

export async function resolveConflict(rideId: string, data: ResolveConflictRequest): Promise<RideRecordDTO> {
  const res = await apiClient.post(`/rides/${rideId}/resolve-conflict`, data)
  return unwrapData<RideRecordDTO>(res)
}

export async function listIssueRides(params?: {
  page?: number
  pageSize?: number
  month?: string
  issueType?: 'conflict' | 'unreported' | 'import_error'
  keyword?: string
}): Promise<Paged<IssueRideDTO>> {
  const res = await apiClient.get('/rides/issues', { params })
  return unwrapPaged<IssueRideDTO>(res, createPaginationMeta(params?.page, params?.pageSize))
}

export async function listMissingRides(params?: {
  page?: number
  pageSize?: number
  startDate?: string
  endDate?: string
  vehicleId?: string
  caseId?: string
  q?: string
}): Promise<Paged<MissingRideDTO>> {
  const res = await apiClient.get('/rides/missing', { params })
  return unwrapPaged<MissingRideDTO>(res, createPaginationMeta(params?.page, params?.pageSize))
}


