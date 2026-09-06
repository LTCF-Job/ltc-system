import { apiClient, createPaginationMeta, unwrapData, unwrapPaged } from './client'
import type {
  NotificationRecipientDTO,
  CreateNotificationRecipientRequest,
  UpdateNotificationRecipientRequest,
  BatchCreateNotificationRecipientsRequest,
  NotificationLogDTO,
  Paged
} from '@/types/api'

// 通知收件人管理
export async function listNotificationRecipients(params?: {
  topic?: string
  recipientType?: string
  targetRole?: string
  active?: boolean
  q?: string
}): Promise<NotificationRecipientDTO[]> {
  const res = await apiClient.get('/settings/notification-recipients', { params })
  return unwrapData<NotificationRecipientDTO[]>(res) ?? []
}

export async function createNotificationRecipient(
  data: CreateNotificationRecipientRequest
): Promise<NotificationRecipientDTO> {
  const res = await apiClient.post('/settings/notification-recipients', data)
  return unwrapData<NotificationRecipientDTO>(res)
}

export async function batchCreateNotificationRecipients(
  data: BatchCreateNotificationRecipientsRequest
): Promise<NotificationRecipientDTO[]> {
  const res = await apiClient.post('/settings/notification-recipients/batch', data)
  return unwrapData<NotificationRecipientDTO[]>(res) ?? []
}

export async function updateNotificationRecipient(
  id: string,
  data: UpdateNotificationRecipientRequest
): Promise<NotificationRecipientDTO> {
  const res = await apiClient.patch(`/settings/notification-recipients/${id}`, data)
  return unwrapData<NotificationRecipientDTO>(res)
}

export async function deleteNotificationRecipient(id: string): Promise<void> {
	const res = await apiClient.delete(`/settings/notification-recipients/${id}`)
	unwrapData<unknown>(res)
}

export async function batchDeleteNotificationRecipients(ids: string[]): Promise<{ count: number }> {
  const res = await apiClient.post('/settings/notification-recipients/batch-delete', { ids })
  return unwrapData<{ count: number }>(res)
}

// 通知歷史紀錄
export async function listNotificationLogs(params?: {
  page?: number
  pageSize?: number
  topic?: string
  status?: string
  q?: string
}): Promise<Paged<NotificationLogDTO>> {
  const res = await apiClient.get('/notifications/logs', { params })
  return unwrapPaged<NotificationLogDTO>(res, createPaginationMeta(params?.page, params?.pageSize))
}

// 手動觸發未回報檢核與通知排程
export async function triggerMissingReportsCheck(): Promise<{
  triggeredCount: number
  message: string
}> {
  const res = await apiClient.post('/tasks/check-missing-reports')
  return unwrapData<{ triggeredCount: number; message: string }>(res)
}
