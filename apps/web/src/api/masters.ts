import { apiClient, createPaginationMeta, unwrapData, unwrapPaged } from './client'
import type {
  Paged,
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

// 單位 Sites
export async function listSites(params?: {
  page?: number
  pageSize?: number
  region?: string
  status?: string
  q?: string
}): Promise<Paged<SiteDTO>> {
  const res = await apiClient.get('/sites', { params })
  return unwrapPaged<SiteDTO>(res, createPaginationMeta(params?.page, params?.pageSize))
}

export async function listAllSites(params?: Omit<NonNullable<Parameters<typeof listSites>[0]>, 'page' | 'pageSize'>): Promise<SiteDTO[]> {
  return collectAllPages((page, pageSize) => listSites({ ...params, page, pageSize }))
}

export async function createSite(data: CreateSiteRequest): Promise<SiteDTO> {
  const res = await apiClient.post('/sites', data)
  return unwrapData<SiteDTO>(res)
}

export async function updateSite(id: string, data: UpdateSiteRequest): Promise<SiteDTO> {
  const res = await apiClient.patch(`/sites/${id}`, data)
  return unwrapData<SiteDTO>(res)
}

export async function deleteSite(id: string): Promise<void> {
	const res = await apiClient.delete(`/sites/${id}`)
	unwrapData<unknown>(res)
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
  const res = await apiClient.get('/vehicles', { params })
  return unwrapPaged<VehicleDTO>(res, createPaginationMeta(params?.page, params?.pageSize))
}

export async function listAllVehicles(params?: Omit<NonNullable<Parameters<typeof listVehicles>[0]>, 'page' | 'pageSize'>): Promise<VehicleDTO[]> {
  return collectAllPages((page, pageSize) => listVehicles({ ...params, page, pageSize }))
}

export async function createVehicle(data: CreateVehicleRequest): Promise<VehicleDTO> {
  const payload = sanitizeVehiclePayload(data)
  const res = await apiClient.post('/vehicles', payload)
  return unwrapData<VehicleDTO>(res)
}

export async function updateVehicle(id: string, data: UpdateVehicleRequest): Promise<VehicleDTO> {
  const payload = sanitizeVehiclePayload(data as CreateVehicleRequest)
  const res = await apiClient.patch(`/vehicles/${id}`, payload)
  return unwrapData<VehicleDTO>(res)
}

export async function deleteVehicle(id: string): Promise<void> {
	const res = await apiClient.delete(`/vehicles/${id}`)
	unwrapData<unknown>(res)
}

// 司機 Drivers
export async function listDrivers(params?: {
  page?: number
  pageSize?: number
  status?: string
  q?: string
}): Promise<Paged<DriverDTO>> {
  const res = await apiClient.get('/drivers', { params })
  return unwrapPaged<DriverDTO>(res, createPaginationMeta(params?.page, params?.pageSize))
}

export async function listAllDrivers(params?: Omit<NonNullable<Parameters<typeof listDrivers>[0]>, 'page' | 'pageSize'>): Promise<DriverDTO[]> {
  return collectAllPages((page, pageSize) => listDrivers({ ...params, page, pageSize }))
}

export async function createDriver(data: CreateDriverRequest): Promise<DriverDTO> {
  const res = await apiClient.post('/drivers', data)
  return unwrapData<DriverDTO>(res)
}

export async function updateDriver(id: string, data: UpdateDriverRequest): Promise<DriverDTO> {
  const res = await apiClient.patch(`/drivers/${id}`, data)
  return unwrapData<DriverDTO>(res)
}

export async function deleteDriver(id: string): Promise<void> {
	const res = await apiClient.delete(`/drivers/${id}`)
	unwrapData<unknown>(res)
}

export async function revealDriverId(id: string): Promise<{ nationalId: string }> {
  const res = await apiClient.post(`/drivers/${id}/reveal`)
  return unwrapData<{ nationalId: string }>(res)
}

export async function assignDriverVehicle(driverId: string, data: {
  vehicleId: string
}): Promise<void> {
	const res = await apiClient.post(`/drivers/${driverId}/assignments`, data)
	unwrapData<unknown>(res)
}

// 整批設定車輛司機；被指派到本車的司機，其他車上尚未結束的指派會一併收掉
export async function setVehicleDrivers(vehicleId: string, data: {
  driverIds: string[]
  effectiveFrom?: string
}): Promise<void> {
	const res = await apiClient.put(`/vehicles/${vehicleId}/drivers`, data)
	unwrapData<unknown>(res)
}

// 班表/主檔批次匯入
export async function dryRunImportMasters(file: File): Promise<DryRunImportResultDTO> {
  const formData = new FormData()
  formData.append('file', file)
  const res = await apiClient.post('/masters/import?dryRun=true', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
  return unwrapData<DryRunImportResultDTO>(res)
}

export async function commitImportMasters(file: File): Promise<{ count: number }> {
  const formData = new FormData()
  formData.append('file', file)
  const res = await apiClient.post('/masters/import?dryRun=false', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
  return unwrapData<{ count: number }>(res)
}
