import { apiClient } from './client'
import type {
  RideCalendarMatrixDTO,
  RideRecordDTO,
  PatchRideRequest,
  ResolveConflictRequest,
  IssueRideDTO,
  Paged
} from '@/types/api'

export async function getRideCalendarMatrix(params: {
  month: string // RRR-MM 或 YYYY-MM
  region?: string
  q?: string
}): Promise<RideCalendarMatrixDTO> {
  return apiClient.get('/rides/calendar', { params })
}

export async function getRideRecord(id: string): Promise<RideRecordDTO> {
  return apiClient.get(`/rides/${id}`)
}

export async function patchRideRecord(id: string, data: PatchRideRequest): Promise<RideRecordDTO> {
  return apiClient.patch(`/rides/${id}`, data)
}

export async function resolveConflict(rideId: string, data: ResolveConflictRequest): Promise<RideRecordDTO> {
  return apiClient.post(`/rides/${rideId}/resolve-conflict`, data)
}

export async function listIssueRides(params?: {
  page?: number
  pageSize?: number
  month?: string
  issueType?: 'conflict' | 'unreported' | 'import_error'
}): Promise<Paged<IssueRideDTO>> {
  return apiClient.get('/rides/issues', { params })
}
