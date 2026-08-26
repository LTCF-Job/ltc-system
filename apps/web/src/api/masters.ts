import { apiClient } from './client'
import type {
  Paged,
  RegionDTO,
  CreateRegionRequest,
  UpdateRegionRequest,
  SiteDTO,
  CreateSiteRequest,
  UpdateSiteRequest,
  VehicleDTO,
  CreateVehicleRequest,
  UpdateVehicleRequest,
  DriverDTO,
  CreateDriverRequest,
  UpdateDriverRequest,
  DryRunImportResultDTO
} from '@/types/api'

// 區域 Regions
export async function listRegions(params?: {
  page?: number
  pageSize?: number
  q?: string
  status?: string
  all?: boolean
}): Promise<Paged<RegionDTO>> {
  return apiClient.get('/regions', { params })
}

export async function listAllRegions(): Promise<{ data: RegionDTO[] }> {
  return apiClient.get('/regions', { params: { all: true } })
}

export async function createRegion(data: CreateRegionRequest): Promise<RegionDTO> {
  return apiClient.post('/regions', data)
}

export async function updateRegion(id: string, data: UpdateRegionRequest): Promise<RegionDTO> {
  return apiClient.patch(`/regions/${id}`, data)
}

export async function deleteRegion(id: string): Promise<void> {
  return apiClient.delete(`/regions/${id}`)
}


// 據點 Sites
export async function listSites(params?: {
  page?: number
  pageSize?: number
  region?: string
  q?: string
}): Promise<Paged<SiteDTO>> {
  return apiClient.get('/sites', { params })
}

export async function createSite(data: CreateSiteRequest): Promise<SiteDTO> {
  return apiClient.post('/sites', data)
}

export async function updateSite(id: string, data: UpdateSiteRequest): Promise<SiteDTO> {
  return apiClient.patch(`/sites/${id}`, data)
}

export async function deleteSite(id: string): Promise<void> {
  return apiClient.delete(`/sites/${id}`)
}

// 車輛 Vehicles
export async function listVehicles(params?: {
  page?: number
  pageSize?: number
  region?: string
  active?: boolean
  q?: string
}): Promise<Paged<VehicleDTO>> {
  return apiClient.get('/vehicles', { params })
}

export async function createVehicle(data: CreateVehicleRequest): Promise<VehicleDTO> {
  return apiClient.post('/vehicles', data)
}

export async function updateVehicle(id: string, data: UpdateVehicleRequest): Promise<VehicleDTO> {
  return apiClient.patch(`/vehicles/${id}`, data)
}

export async function deleteVehicle(id: string): Promise<void> {
  return apiClient.delete(`/vehicles/${id}`)
}

// 司機 Drivers
export async function listDrivers(params?: {
  page?: number
  pageSize?: number
  active?: boolean
  q?: string
}): Promise<Paged<DriverDTO>> {
  return apiClient.get('/drivers', { params })
}

export async function createDriver(data: CreateDriverRequest): Promise<DriverDTO> {
  return apiClient.post('/drivers', data)
}

export async function updateDriver(id: string, data: UpdateDriverRequest): Promise<DriverDTO> {
  return apiClient.patch(`/drivers/${id}`, data)
}

export async function deleteDriver(id: string): Promise<void> {
  return apiClient.delete(`/drivers/${id}`)
}

export async function revealDriverId(id: string): Promise<{ nationalId: string }> {
  return apiClient.post(`/drivers/${id}/reveal`)
}

export async function assignDriverVehicle(driverId: string, data: {
  vehicleId: string
  startDate: string
  endDate?: string
  isPrimary: boolean
}): Promise<void> {
  return apiClient.post(`/drivers/${driverId}/assignments`, data)
}

// 班表/主檔批次匯入
export async function dryRunImportMasters(file: File): Promise<DryRunImportResultDTO> {
  const formData = new FormData()
  formData.append('file', file)
  return apiClient.post('/masters/import?dryRun=true', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

export async function commitImportMasters(file: File): Promise<{ count: number }> {
  const formData = new FormData()
  formData.append('file', file)
  return apiClient.post('/masters/import?dryRun=false', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}
