import { apiClient } from './client'
import type {
  NotificationRecipientDTO,
  CreateNotificationRecipientRequest,
  UpdateNotificationRecipientRequest,
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
  return apiClient.get('/settings/notification-recipients', { params })
}

export async function createNotificationRecipient(
  data: CreateNotificationRecipientRequest
): Promise<NotificationRecipientDTO> {
  return apiClient.post('/settings/notification-recipients', data)
}

export async function updateNotificationRecipient(
  id: string,
  data: UpdateNotificationRecipientRequest
): Promise<NotificationRecipientDTO> {
  return apiClient.patch(`/settings/notification-recipients/${id}`, data)
}

export async function deleteNotificationRecipient(id: string): Promise<void> {
  return apiClient.delete(`/settings/notification-recipients/${id}`)
}

// 通知歷史紀錄
export async function listNotificationLogs(params?: {
  page?: number
  pageSize?: number
  topic?: string
  status?: string
  q?: string
}): Promise<Paged<NotificationLogDTO>> {
  return apiClient.get('/notifications/logs', { params })
}

// 手動觸發未回報檢核與通知排程
export async function triggerMissingReportsCheck(): Promise<{
  triggeredCount: number
  message: string
}> {
  return apiClient.post('/tasks/check-missing-reports')
}
