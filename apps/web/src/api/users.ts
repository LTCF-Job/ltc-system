import { apiClient, createPaginationMeta, unwrapData, unwrapPaged } from './client'
import type {
  Paged,
  UserDTO,
  CreateUserRequest,
  UpdateUserRequest,
  ChangePasswordRequest
} from '@/types/api'
import type { SystemPermissions } from '@/types/domain'

export async function listUsers(params?: {
  q?: string
  role?: string
  page?: number
  pageSize?: number
}): Promise<Paged<UserDTO>> {
  const res = await apiClient.get('/users', { params })
  return unwrapPaged<UserDTO>(res, createPaginationMeta(params?.page, params?.pageSize))
}

export async function getUser(id: string): Promise<UserDTO> {
  const res = await apiClient.get(`/users/${id}`)
  return unwrapData<UserDTO>(res)
}

export async function createUser(data: CreateUserRequest): Promise<UserDTO> {
  const res = await apiClient.post('/users', data)
  return unwrapData<UserDTO>(res)
}

export async function updateUser(id: string, data: UpdateUserRequest): Promise<UserDTO> {
  const res = await apiClient.patch(`/users/${id}`, data)
  return unwrapData<UserDTO>(res)
}

export async function updateUserPermissions(
  id: string,
  permissions: SystemPermissions | null
): Promise<UserDTO> {
  const res = await apiClient.put(`/users/${id}/permissions`, { customPermissions: permissions })
  return unwrapData<UserDTO>(res)
}

export async function deleteUser(id: string): Promise<{ success: boolean }> {
  const res = await apiClient.delete(`/users/${id}`)
  return unwrapData<{ success: boolean }>(res)
}

export async function changeSelfPassword(data: ChangePasswordRequest): Promise<{ success: boolean }> {
  const res = await apiClient.post('/auth/change-password', data)
  return unwrapData<{ success: boolean }>(res)
}
