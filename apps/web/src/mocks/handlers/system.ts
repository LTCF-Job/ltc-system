import { http, HttpResponse } from 'msw'
import {
  mockDashboardStats,
  mockDashboardMetrics,
  mockAuditLogs,
  mockNotificationRecipients,
  mockNotificationLogs,
  mockMissingRides,
  mockUsers,
  mockRoles
} from '../data/mockData'

export const systemHandlers = [
  // 儀表板指標與統計
  http.get('/api/v1/dashboard/stats', () => {
    // 儀表板只呈現最近匯出的摘要，不再提供下載入口
    const recentExports = mockDashboardStats.recentExports.map((job) => ({
      ...job,
      files: undefined,
      downloadUrl: undefined,
      zipFileName: undefined
    }))
    return HttpResponse.json({ ...mockDashboardStats, recentExports })
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
      id: `rec_${Date.now()}_${Math.random().toString(36).substring(2, 6)}`,
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

  // 批次新增外部信箱收件人
  http.post('/api/v1/settings/notification-recipients/batch', async ({ request }) => {
    const body = (await request.json()) as any
    const topic = body.topic
    const incomingRecipients = (body.recipients || []) as any[]

    const createdList: any[] = []
    for (const item of incomingRecipients) {
      // 避免同一主題下重複加入相同 email
      const duplicate = mockNotificationRecipients.some(
        (r) => r.topic === topic && r.email.toLowerCase() === item.email.toLowerCase()
      )

      if (!duplicate) {
        const newRecipient = {
          id: `rec_${Date.now()}_${Math.random().toString(36).substring(2, 6)}`,
          topic,
          email: item.email,
          displayName: item.displayName || '',
          active: true,
          createdByName: '系統管理員',
          createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
        }
        mockNotificationRecipients.push(newRecipient)
        createdList.push(newRecipient)
      }
    }

    if (createdList.length > 0) {
      mockAuditLogs.unshift({
        id: `audit_${Date.now()}`,
        actorId: 'usr_admin',
        actorName: '系統管理員',
        actorRole: 'admin',
        action: 'create',
        entityType: 'notification_recipients',
        entityId: topic,
        entityName: `批次新增通知信箱 (${createdList.length} 筆)`,
        beforeData: undefined,
        afterData: { topic, addedCount: createdList.length, recipients: createdList.map((c) => c.displayName || c.email) },
        ipAddress: '127.0.0.1',
        userAgent: 'Mock Agent',
        createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
      })
    }

    return HttpResponse.json(createdList)
  }),

  // 批次刪除通知收件人
  http.post('/api/v1/settings/notification-recipients/batch-delete', async ({ request }) => {
    const body = (await request.json()) as any
    const ids = (body.ids || []) as string[]
    let deletedCount = 0

    for (const id of ids) {
      const idx = mockNotificationRecipients.findIndex((r) => r.id === id)
      if (idx !== -1) {
        mockNotificationRecipients.splice(idx, 1)
        deletedCount++
      }
    }

    if (deletedCount > 0) {
      mockAuditLogs.unshift({
        id: `audit_${Date.now()}`,
        actorId: 'usr_admin',
        actorName: '系統管理員',
        actorRole: 'admin',
        action: 'delete',
        entityType: 'notification_recipients',
        entityId: ids.join(','),
        entityName: `批次刪除通知收件人 (${deletedCount} 位)`,
        beforeData: { deletedIds: ids },
        afterData: undefined,
        ipAddress: '127.0.0.1',
        userAgent: 'Mock Agent',
        createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
      })
    }

    return HttpResponse.json({ count: deletedCount })
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
  }),

  // 使用者帳號與權限管理
  http.get('/api/v1/users', ({ request }) => {
    const url = new URL(request.url)
    const q = url.searchParams.get('q')?.trim().toLowerCase()
    const role = url.searchParams.get('role')

    let list = [...mockUsers]
    if (role) {
      list = list.filter((u) => u.role === role)
    }
    if (q) {
      list = list.filter(
        (u) =>
          u.displayName.toLowerCase().includes(q) ||
          u.email.toLowerCase().includes(q) ||
          (u.phone && u.phone.includes(q))
      )
    }
    return HttpResponse.json(list)
  }),

  http.get('/api/v1/users/:id', ({ params }) => {
    const user = mockUsers.find((u) => u.id === params.id)
    if (!user) return new HttpResponse(null, { status: 404 })
    return HttpResponse.json(user)
  }),

  http.post('/api/v1/users', async ({ request }) => {
    const body = (await request.json()) as any
    const newUser = {
      id: `usr_${Date.now()}`,
      email: body.email,
      displayName: body.displayName,
      role: body.role || 'dispatcher',
      phone: body.phone || '',
      status: body.status || 'active',
      customPermissions: body.customPermissions || null,
      createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19),
      lastLoginAt: undefined
    }
    mockUsers.unshift(newUser)

    // 寫入系統操作日誌留痕
    mockAuditLogs.unshift({
      id: `audit_${Date.now()}`,
      actorId: 'usr_admin',
      actorName: '系統管理員',
      actorRole: 'admin',
      action: 'create',
      entityType: 'users',
      entityId: newUser.id,
      entityName: newUser.displayName,
      beforeData: undefined,
      afterData: {
        email: newUser.email,
        displayName: newUser.displayName,
        role: newUser.role,
        status: newUser.status
      },
      ipAddress: '127.0.0.1',
      userAgent: 'Mock Agent',
      createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
    })

    return HttpResponse.json(newUser)
  }),

  http.patch('/api/v1/users/:id', async ({ params, request }) => {
    const user = mockUsers.find((u) => u.id === params.id)
    if (!user) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    const before = { ...user }
    Object.assign(user, body)

    mockAuditLogs.unshift({
      id: `audit_${Date.now()}`,
      actorId: 'usr_admin',
      actorName: '系統管理員',
      actorRole: 'admin',
      action: 'update',
      entityType: 'users',
      entityId: user.id,
      entityName: user.displayName,
      beforeData: before,
      afterData: user,
      ipAddress: '127.0.0.1',
      userAgent: 'Mock Agent',
      createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
    })

    return HttpResponse.json(user)
  }),

  http.put('/api/v1/users/:id/permissions', async ({ params, request }) => {
    const user = mockUsers.find((u) => u.id === params.id)
    if (!user) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    const before = user.customPermissions ? { ...user.customPermissions } : null
    user.customPermissions = body.permissions

    mockAuditLogs.unshift({
      id: `audit_${Date.now()}`,
      actorId: 'usr_admin',
      actorName: '系統管理員',
      actorRole: 'admin',
      action: 'setting_change',
      entityType: 'users',
      entityId: user.id,
      entityName: `${user.displayName} (自訂權限變更)`,
      beforeData: before || { mode: '套用角色預設' },
      afterData: user.customPermissions || { mode: '重設為角色預設' },
      ipAddress: '127.0.0.1',
      userAgent: 'Mock Agent',
      createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
    })

    return HttpResponse.json(user)
  }),

  http.delete('/api/v1/users/:id', ({ params }) => {
    const idx = mockUsers.findIndex((u) => u.id === params.id)
    if (idx !== -1) {
      const removed = mockUsers.splice(idx, 1)[0]
      mockAuditLogs.unshift({
        id: `audit_${Date.now()}`,
        actorId: 'usr_admin',
        actorName: '系統管理員',
        actorRole: 'admin',
        action: 'delete',
        entityType: 'users',
        entityId: removed.id,
        entityName: removed.displayName,
        beforeData: removed,
        afterData: undefined,
        ipAddress: '127.0.0.1',
        userAgent: 'Mock Agent',
        createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
      })
    }
    return HttpResponse.json({ success: true })
  }),

  http.post('/api/v1/auth/change-password', async () => {
    mockAuditLogs.unshift({
      id: `audit_${Date.now()}`,
      actorId: 'usr_current',
      actorName: '當前使用者',
      actorRole: 'user',
      action: 'update',
      entityType: 'users',
      entityId: 'usr_current',
      entityName: '個人密碼修改',
      beforeData: { status: '舊密碼驗證通過' },
      afterData: { status: '新密碼已啟用' },
      ipAddress: '127.0.0.1',
      userAgent: 'Mock Agent',
      createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
    })
    return HttpResponse.json({ success: true })
  }),

  // 角色身分管理 (CRUD)
  http.get('/api/v1/roles', ({ request }) => {
    const url = new URL(request.url)
    const q = url.searchParams.get('q')?.trim().toLowerCase()

    let list = mockRoles.map((role) => {
      const userCount = mockUsers.filter((u) => u.role === role.key).length
      return { ...role, userCount }
    })

    if (q) {
      list = list.filter(
        (r) =>
          r.name.toLowerCase().includes(q) ||
          r.key.toLowerCase().includes(q) ||
          (r.description && r.description.toLowerCase().includes(q))
      )
    }

    return HttpResponse.json(list)
  }),

  http.get('/api/v1/roles/:id', ({ params }) => {
    const role = mockRoles.find((r) => r.id === params.id || r.key === params.id)
    if (!role) return new HttpResponse(null, { status: 404 })
    const userCount = mockUsers.filter((u) => u.role === role.key).length
    return HttpResponse.json({ ...role, userCount })
  }),

  http.post('/api/v1/roles', async ({ request }) => {
    const body = (await request.json()) as any
    const id = `role_${Date.now()}`
    const key = body.key ? String(body.key).trim().toLowerCase() : id

    const newRole = {
      id,
      key,
      name: body.name || '自訂角色',
      description: body.description || '',
      tagType: body.tagType || 'primary',
      isSystem: false,
      permissions: body.permissions || {},
      userCount: 0,
      createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19),
      updatedAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
    }
    mockRoles.push(newRole)

    mockAuditLogs.unshift({
      id: `audit_${Date.now()}`,
      actorId: 'usr_admin',
      actorName: '系統管理員',
      actorRole: 'admin',
      action: 'create',
      entityType: 'roles',
      entityId: newRole.id,
      entityName: newRole.name,
      beforeData: undefined,
      afterData: newRole,
      ipAddress: '127.0.0.1',
      userAgent: 'Mock Agent',
      createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
    })

    return HttpResponse.json(newRole)
  }),

  http.patch('/api/v1/roles/:id', async ({ params, request }) => {
    const role = mockRoles.find((r) => r.id === params.id || r.key === params.id)
    if (!role) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    const before = JSON.parse(JSON.stringify(role))

    if (body.name !== undefined) role.name = body.name
    if (body.description !== undefined) role.description = body.description
    if (body.tagType !== undefined) role.tagType = body.tagType
    if (body.permissions !== undefined) role.permissions = body.permissions
    role.updatedAt = new Date().toISOString().replace('T', ' ').substring(0, 19)

    const userCount = mockUsers.filter((u) => u.role === role.key).length
    const responseData = { ...role, userCount }

    mockAuditLogs.unshift({
      id: `audit_${Date.now()}`,
      actorId: 'usr_admin',
      actorName: '系統管理員',
      actorRole: 'admin',
      action: 'update',
      entityType: 'roles',
      entityId: role.id,
      entityName: role.name,
      beforeData: before,
      afterData: role,
      ipAddress: '127.0.0.1',
      userAgent: 'Mock Agent',
      createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
    })

    return HttpResponse.json(responseData)
  }),

  http.delete('/api/v1/roles/:id', ({ params }) => {
    const idx = mockRoles.findIndex((r) => r.id === params.id || r.key === params.id)
    if (idx === -1) return new HttpResponse(null, { status: 404 })

    const target = mockRoles[idx]
    if (target.isSystem) {
      return HttpResponse.json(
        { error: { message: `「${target.name}」為系統內建核心角色，受系統保護禁止刪除` } },
        { status: 400 }
      )
    }

    const boundUsers = mockUsers.filter((u) => u.role === target.key)
    if (boundUsers.length > 0) {
      return HttpResponse.json(
        { error: { message: `目前尚有 ${boundUsers.length} 位使用者被指派為此角色，請先將使用者轉移至其他身分後再執行刪除` } },
        { status: 400 }
      )
    }

    const removed = mockRoles.splice(idx, 1)[0]
    mockAuditLogs.unshift({
      id: `audit_${Date.now()}`,
      actorId: 'usr_admin',
      actorName: '系統管理員',
      actorRole: 'admin',
      action: 'delete',
      entityType: 'roles',
      entityId: removed.id,
      entityName: removed.name,
      beforeData: removed,
      afterData: undefined,
      ipAddress: '127.0.0.1',
      userAgent: 'Mock Agent',
      createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
    })

    return HttpResponse.json({ success: true })
  })
]
