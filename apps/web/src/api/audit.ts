import { apiClient } from './client'
import type { AuditLogDTO, ListAuditLogsParams, Paged } from '@/types/api'

export async function listAuditLogs(params?: ListAuditLogsParams): Promise<Paged<AuditLogDTO>> {
  return apiClient.get('/audit', { params })
}
