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
import { sanitizeVehiclePayload } from '@/utils/vehicleForm'

const MAX_PAGE_SIZE = 100

async function collectAllPages<T>(fetchPage: (page: number, pageSize: number) => Promise<Paged<T>>): Promise<T[]> {
  const items: T[] = []
  let page = 1

  while (true) {
    const result = await fetchPage(page, MAX_PAGE_SIZE)
    const pageItems = result.data || []
    items.push(...pageItems)
    const totalPages = result.meta?.totalPages || Math.ceil((result.meta?.total || items.length) / MAX_PAGE_SIZE)
    if (page >= totalPages || pageItems.length === 0) break
    page += 1
  }

  return items
}

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


// 單位 Sites
export async function listSites(params?: {
  page?: number
  pageSize?: number
  region?: string
  status?: string
  q?: string
}): Promise<Paged<SiteDTO>> {
  return apiClient.get('/sites', { params })
}

export async function listAllSites(params?: Omit<NonNullable<Parameters<typeof listSites>[0]>, 'page' | 'pageSize'>): Promise<SiteDTO[]> {
  return collectAllPages((page, pageSize) => listSites({ ...params, page, pageSize }))
}

export async function createSite(data: CreateSiteRequest): Promise<SiteDTO> {
  const res = await apiClient.post('/sites', data)
  return (res as any).data ?? (res as any)
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
  siteId?: string
  region?: string
  status?: string
  q?: string
}): Promise<Paged<VehicleDTO>> {
  return apiClient.get('/vehicles', { params })
}

export async function listAllVehicles(params?: Omit<NonNullable<Parameters<typeof listVehicles>[0]>, 'page' | 'pageSize'>): Promise<VehicleDTO[]> {
  return collectAllPages((page, pageSize) => listVehicles({ ...params, page, pageSize }))
}

export async function createVehicle(data: CreateVehicleRequest): Promise<VehicleDTO> {
  const payload = sanitizeVehiclePayload(data)
  const res = await apiClient.post('/vehicles', payload)
  return (res as any).data ?? (res as any)
}

export async function updateVehicle(id: string, data: UpdateVehicleRequest): Promise<VehicleDTO> {
  const payload = sanitizeVehiclePayload(data as CreateVehicleRequest)
  return apiClient.patch(`/vehicles/${id}`, payload)
}

export async function deleteVehicle(id: string): Promise<void> {
  return apiClient.delete(`/vehicles/${id}`)
}

// 司機 Drivers
export async function listDrivers(params?: {
  page?: number
  pageSize?: number
  status?: string
  q?: string
}): Promise<Paged<DriverDTO>> {
  return apiClient.get('/drivers', { params })
}

export async function listAllDrivers(params?: Omit<NonNullable<Parameters<typeof listDrivers>[0]>, 'page' | 'pageSize'>): Promise<DriverDTO[]> {
  return collectAllPages((page, pageSize) => listDrivers({ ...params, page, pageSize }))
}

export async function createDriver(data: CreateDriverRequest): Promise<DriverDTO> {
  const res: any = await apiClient.post('/drivers', data)
  return res?.data ?? res
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
}): Promise<void> {
  return apiClient.post(`/drivers/${driverId}/assignments`, data)
}

// 整批設定車輛司機；被指派到本車的司機，其他車上尚未結束的指派會一併收掉
export async function setVehicleDrivers(vehicleId: string, data: {
  driverIds: string[]
  effectiveFrom?: string
}): Promise<void> {
  return apiClient.put(`/vehicles/${vehicleId}/drivers`, data)
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
