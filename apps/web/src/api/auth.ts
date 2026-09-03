import { apiClient } from './client'
import type { UserDTO } from '@/types/api'
import type { SystemPermissions } from '@/types/domain'

// 後端合併個人 customPermissions 後的最終有效權限矩陣，前端一律以此為準，不再自行推導
export interface AuthMeResponse extends UserDTO {
  permissions: SystemPermissions
}

export async function getAuthMe(): Promise<AuthMeResponse> {
  const res = await apiClient.get('/auth/me')
  return (res as any).data ?? res
}
