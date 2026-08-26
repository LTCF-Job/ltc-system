import { apiClient } from './client'
import type {
  UserDTO,
  CreateUserRequest,
  UpdateUserRequest,
  ChangePasswordRequest
} from '@/types/api'
import type { SystemPermissions } from '@/types/domain'

export async function listUsers(params?: { q?: string; role?: string }): Promise<UserDTO[]> {
  return apiClient.get('/users', { params })
}

export async function getUser(id: string): Promise<UserDTO> {
  return apiClient.get(`/users/${id}`)
}

export async function createUser(data: CreateUserRequest): Promise<UserDTO> {
  return apiClient.post('/users', data)
}

export async function updateUser(id: string, data: UpdateUserRequest): Promise<UserDTO> {
  return apiClient.patch(`/users/${id}`, data)
}

export async function updateUserPermissions(
  id: string,
  permissions: SystemPermissions | null
): Promise<UserDTO> {
  return apiClient.put(`/users/${id}/permissions`, { permissions })
}

export async function deleteUser(id: string): Promise<{ success: boolean }> {
  return apiClient.delete(`/users/${id}`)
}

export async function changeSelfPassword(data: ChangePasswordRequest): Promise<{ success: boolean }> {
  return apiClient.post('/auth/change-password', data)
}
