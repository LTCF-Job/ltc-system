import { apiClient, createPaginationMeta, unwrapPaged } from './client'
import type { AuditLogDTO, ListAuditLogsParams, Paged } from '@/types/api'

export async function listAuditLogs(params?: ListAuditLogsParams): Promise<Paged<AuditLogDTO>> {
  const res = await apiClient.get('/audit', { params })
  return unwrapPaged<AuditLogDTO>(res, createPaginationMeta(params?.page, params?.pageSize))
}
