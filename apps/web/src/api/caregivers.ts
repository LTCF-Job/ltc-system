import { apiClient, createPaginationMeta, unwrapData, unwrapPaged } from './client'
import type {
  Paged,
  CaregiverDTO,
  CreateCaregiverRequest,
  UpdateCaregiverRequest,
  CaregiverImportCommitResult,
  DryRunImportResultDTO
} from '@/types/api'

// 依查詢條件取得照護人員分頁清單
export async function listCaregivers(params?: {
  page?: number
  pageSize?: number
  q?: string
  status?: string
  // 待維護資料預設不會回傳；pending 只取待維護，includePending 取全部。
  pending?: boolean
  includePending?: boolean
}): Promise<Paged<CaregiverDTO>> {
  const res = await apiClient.get('/caregivers', { params })
  return unwrapPaged<CaregiverDTO>(res, createPaginationMeta(params?.page, params?.pageSize))
}

export async function listAllCaregivers(params?: Omit<NonNullable<Parameters<typeof listCaregivers>[0]>, 'page' | 'pageSize'>): Promise<CaregiverDTO[]> {
  const items: CaregiverDTO[] = []
  let page = 1
  const pageSize = 100
  while (true) {
    const result = await listCaregivers({ ...params, page, pageSize })
    const pageItems = result.data || []
    items.push(...pageItems)
    if (page >= (result.meta?.totalPages || Math.ceil((result.meta?.total || items.length) / pageSize)) || pageItems.length === 0) break
    page += 1
  }
  return items
}

export async function createCaregiver(data: CreateCaregiverRequest): Promise<CaregiverDTO> {
  const res = await apiClient.post('/caregivers', data)
  return unwrapData<CaregiverDTO>(res)
}

export async function updateCaregiver(id: string, data: UpdateCaregiverRequest): Promise<CaregiverDTO> {
  const res = await apiClient.patch(`/caregivers/${id}`, data)
  return unwrapData<CaregiverDTO>(res)
}

export async function deleteCaregiver(id: string): Promise<void> {
	const res = await apiClient.delete(`/caregivers/${id}`)
	unwrapData<unknown>(res)
}

export async function linkCaregiverSite(id: string, siteId: string): Promise<CaregiverDTO> {
  const res = await apiClient.put(`/caregivers/${id}/site`, { siteId })
  return unwrapData<CaregiverDTO>(res)
}

// 上傳照護人員名單試算匯入結果，不寫入資料庫
export async function dryRunImportCaregivers(file: File): Promise<DryRunImportResultDTO> {
  const formData = new FormData()
  formData.append('file', file)
  const res = await apiClient.post('/caregivers/import?dryRun=true', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
  return unwrapData<DryRunImportResultDTO>(res)
}

// 確認試算結果後正式寫入照護人員匯入資料
export async function commitImportCaregivers(
	file: File,
	includeDuplicateRows: string[] = []
): Promise<CaregiverImportCommitResult> {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('includeDuplicateRows', JSON.stringify(includeDuplicateRows))
  const res = await apiClient.post('/caregivers/import?dryRun=false', formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  })
  return unwrapData<CaregiverImportCommitResult>(res)
}

export async function downloadCaregiverTemplate(): Promise<Blob> {
  return apiClient.get('/caregivers/template', { responseType: 'blob' })
}
