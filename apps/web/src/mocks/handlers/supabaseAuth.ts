import { http, HttpResponse } from 'msw'

interface MockSupabaseUser {
  role: string
  dataPlane: 'production' | 'demo'
  displayName: string
}

// 帳密對照表：只涵蓋登入頁與 E2E 測試會用到的帳號，其餘一律回傳登入失敗
const MOCK_USERS: Record<string, { password: string; user: MockSupabaseUser }> = {
  'demo@ltc.example.com': {
    password: 'demo',
    user: { role: 'admin', dataPlane: 'demo', displayName: '展示帳號' }
  },
  'ltcf-admin@ltc.example.com': {
    password: 'password123',
    user: { role: 'admin', dataPlane: 'production', displayName: '系統管理員' }
  }
}

export const supabaseAuthHandlers = [
  http.post('/api/v1/demo/reset', () => {
    return HttpResponse.json({
      data: { datasetVersion: '0001_baseline', resetAt: new Date().toISOString() }
    })
  }),

  http.post('http://mock.supabase.local/auth/v1/token', async ({ request }) => {
    const body = (await request.json()) as { email?: string; password?: string }
    const email = (body.email || '').trim().toLowerCase()
    // demo/demo123 皆視為展示帳號密碼，維持登入頁沿用已久的兩種輸入習慣
    const password = body.password === 'demo123' ? 'demo' : body.password
    const entry = MOCK_USERS[email]

    if (!entry || entry.password !== password) {
      return HttpResponse.json(
        { error: 'invalid_grant', error_description: 'Invalid login credentials' },
        { status: 400 }
      )
    }

    const userId = `mock-${email.split('@')[0]}`
    const nowIso = new Date().toISOString()
    return HttpResponse.json({
      access_token: `mock-access-token-${userId}`,
      token_type: 'bearer',
      expires_in: 3600,
      expires_at: Math.floor(Date.now() / 1000) + 3600,
      refresh_token: `mock-refresh-token-${userId}`,
      user: {
        id: userId,
        aud: 'authenticated',
        email,
        app_metadata: { provider: 'email', providers: ['email'], role: entry.user.role, data_plane: entry.user.dataPlane },
        user_metadata: { display_name: entry.user.displayName },
        created_at: nowIso
      }
    })
  })
]
