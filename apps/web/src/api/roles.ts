import { apiClient } from './client'
import type {
  RoleDTO,
  CreateRoleRequest,
  UpdateRoleRequest
} from '@/types/api'

export async function listRoles(params?: { q?: string }): Promise<RoleDTO[]> {
  return apiClient.get('/roles', { params })
}

export async function getRole(id: string): Promise<RoleDTO> {
  return apiClient.get(`/roles/${id}`)
}

export async function createRole(data: CreateRoleRequest): Promise<RoleDTO> {
  return apiClient.post('/roles', data)
}

export async function updateRole(id: string, data: UpdateRoleRequest): Promise<RoleDTO> {
  return apiClient.patch(`/roles/${id}`, data)
}

export async function deleteRole(id: string): Promise<{ success: boolean }> {
  return apiClient.delete(`/roles/${id}`)
}
