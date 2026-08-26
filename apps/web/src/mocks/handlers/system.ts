import { http, HttpResponse } from 'msw'
import {
  mockDashboardStats,
  mockDashboardMetrics,
  mockAuditLogs,
  mockNotificationRecipients,
  mockNotificationLogs,
  mockMissingRides
} from '../data/mockData'

export const systemHandlers = [
  // 儀表板指標與統計
  http.get('/api/v1/dashboard/stats', () => {
    return HttpResponse.json(mockDashboardStats)
  }),

  http.get('/api/v1/dashboard/metrics', () => {
    return HttpResponse.json(mockDashboardMetrics)
  }),

  // 系統稽核紀錄
  http.get('/api/v1/audit', ({ request }) => {
    const url = new URL(request.url)
    const action = url.searchParams.get('action')
    const entityType = url.searchParams.get('entityType')
    const q = url.searchParams.get('q')?.trim().toLowerCase()

    let list = [...mockAuditLogs]
    if (action) {
      list = list.filter((a) => a.action === action)
    }
    if (entityType) {
      list = list.filter((a) => a.entityType === entityType)
    }
    if (q) {
      list = list.filter(
        (a) =>
          a.action.toLowerCase().includes(q) ||
          a.entityType.toLowerCase().includes(q) ||
          (a.entityId && a.entityId.toLowerCase().includes(q)) ||
          ((a as any).actorName && (a as any).actorName.toLowerCase().includes(q)) ||
          ((a as any).beforeData && JSON.stringify((a as any).beforeData).toLowerCase().includes(q)) ||
          ((a as any).afterData && JSON.stringify((a as any).afterData).toLowerCase().includes(q))
      )
    }
    return HttpResponse.json({
      data: list,
      meta: {
        page: 1,
        pageSize: 20,
        total: list.length,
        totalPages: 1
      }
    })
  }),

  // 通知收件人管理
  http.get('/api/v1/settings/notification-recipients', ({ request }) => {
    const url = new URL(request.url)
    const topic = url.searchParams.get('topic')
    const q = url.searchParams.get('q')?.trim().toLowerCase()

    let list = [...mockNotificationRecipients]
    if (topic) {
      list = list.filter((r) => r.topic === topic)
    }
    if (q) {
      list = list.filter(
        (r) =>
          r.email.toLowerCase().includes(q) ||
          (r.displayName && r.displayName.toLowerCase().includes(q)) ||
          (r.createdByName && r.createdByName.toLowerCase().includes(q))
      )
    }
    return HttpResponse.json(list)
  }),

  http.post('/api/v1/settings/notification-recipients', async ({ request }) => {
    const body = (await request.json()) as any
    const newRecipient = {
      id: `rec_${Date.now()}`,
      topic: body.topic,
      email: body.email,
      displayName: body.displayName || '',
      active: body.active !== undefined ? body.active : true,
      createdByName: '系統管理員',
      createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
    }
    mockNotificationRecipients.push(newRecipient)
    return HttpResponse.json(newRecipient)
  }),

  http.patch('/api/v1/settings/notification-recipients/:id', async ({ params, request }) => {
    const target = mockNotificationRecipients.find((r) => r.id === params.id)
    if (!target) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    Object.assign(target, body)
    return HttpResponse.json(target)
  }),

  http.delete('/api/v1/settings/notification-recipients/:id', ({ params }) => {
    const idx = mockNotificationRecipients.findIndex((r) => r.id === params.id)
    if (idx !== -1) mockNotificationRecipients.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // 通知歷史紀錄
  http.get('/api/v1/notifications/logs', ({ request }) => {
    const url = new URL(request.url)
    const topic = url.searchParams.get('topic')
    const q = url.searchParams.get('q')?.trim().toLowerCase()

    let list = [...mockNotificationLogs]
    if (topic) {
      list = list.filter((l) => l.topic === topic)
    }
    if (q) {
      list = list.filter(
        (l) =>
          l.subject.toLowerCase().includes(q) ||
          (l.contentSummary && l.contentSummary.toLowerCase().includes(q)) ||
          (l.recipientEmails && l.recipientEmails.some((e) => e.toLowerCase().includes(q))) ||
          (l.triggeredByName && l.triggeredByName.toLowerCase().includes(q))
      )
    }
    return HttpResponse.json({
      data: list,
      meta: {
        page: 1,
        pageSize: 20,
        total: list.length,
        totalPages: 1
      }
    })
  }),

  // 任務手動觸發
  http.post('/api/v1/tasks/check-missing-reports', () => {
    const newLog = {
      id: `nlog_${Date.now()}`,
      topic: 'missing_report' as const,
      channel: 'email' as const,
      recipientEmails: ['admin@ltc.example.com'],
      subject: `【長照交通系統】手動催報執行通知 (${new Date().toLocaleDateString()})`,
      contentSummary: `已發送未回報提醒，共計 ${mockMissingRides.length} 筆待回報項目。`,
      status: 'sent' as const,
      triggeredByName: '當前操作人員 (手動觸發)',
      sentAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
    }
    mockNotificationLogs.unshift(newLog)
    return HttpResponse.json({
      triggeredCount: mockMissingRides.length,
      message: `已成功執行未回報檢核，並發送催報通知至收件人信箱。`
    })
  })
]
