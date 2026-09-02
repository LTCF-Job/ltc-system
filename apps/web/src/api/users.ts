import { apiClient } from './client'
import type {
  UserDTO,
  CreateUserRequest,
  UpdateUserRequest,
  ChangePasswordRequest
} from '@/types/api'
import type { SystemPermissions } from '@/types/domain'

export async function listUsers(params?: { q?: string; role?: string }): Promise<UserDTO[]> {
  const res = await apiClient.get('/users', { params })
  return (res as any).data ?? res
}

export async function getUser(id: string): Promise<UserDTO> {
  const res = await apiClient.get(`/users/${id}`)
  return (res as any).data ?? res
}

export async function createUser(data: CreateUserRequest): Promise<UserDTO> {
  const res = await apiClient.post('/users', data)
  return (res as any).data ?? res
}

export async function updateUser(id: string, data: UpdateUserRequest): Promise<UserDTO> {
  const res = await apiClient.patch(`/users/${id}`, data)
  return (res as any).data ?? res
}

export async function updateUserPermissions(
  id: string,
  permissions: SystemPermissions | null
): Promise<UserDTO> {
  const res = await apiClient.put(`/users/${id}/permissions`, { customPermissions: permissions })
  return (res as any).data ?? res
}

export async function deleteUser(id: string): Promise<{ success: boolean }> {
  const res = await apiClient.delete(`/users/${id}`)
  return (res as any).data ?? res
}

export async function changeSelfPassword(data: ChangePasswordRequest): Promise<{ success: boolean }> {
  const res = await apiClient.post('/auth/change-password', data)
  return (res as any).data ?? res
}
