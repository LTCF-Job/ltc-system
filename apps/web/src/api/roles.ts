import { apiClient } from './client'
import type {
  RoleDTO,
  CreateRoleRequest,
  UpdateRoleRequest
} from '@/types/api'

export async function listRoles(params?: { q?: string }): Promise<RoleDTO[]> {
  const res = await apiClient.get('/roles', { params })
  return (res as any).data ?? res
}

export async function getRole(id: string): Promise<RoleDTO> {
  const res = await apiClient.get(`/roles/${id}`)
  return (res as any).data ?? res
}

export async function createRole(data: CreateRoleRequest): Promise<RoleDTO> {
  const res = await apiClient.post('/roles', data)
  return (res as any).data ?? res
}

export async function updateRole(id: string, data: UpdateRoleRequest): Promise<RoleDTO> {
  const res = await apiClient.patch(`/roles/${id}`, data)
  return (res as any).data ?? res
}

export async function deleteRole(id: string): Promise<{ success: boolean }> {
  const res = await apiClient.delete(`/roles/${id}`)
  return (res as any).data ?? res
}
