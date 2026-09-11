import { test, expect, request as playwrightRequest } from '@playwright/test'

// 對真正部署的 Supabase Auth 與 API 做 API 層級驗證。
// 必要環境變數缺一則整份跳過，本機或還沒設好 CI secrets 時不會讓 pipeline 失敗。
const SUPABASE_URL = process.env.LIVE_SUPABASE_URL
const SUPABASE_ANON_KEY = process.env.LIVE_SUPABASE_ANON_KEY
const API_BASE_URL = process.env.LIVE_API_BASE_URL?.replace(/\/+$/, '')
const TEST_EMAIL = process.env.LIVE_TEST_EMAIL
const TEST_PASSWORD = process.env.LIVE_TEST_PASSWORD

const canRunApiSuite = Boolean(SUPABASE_URL && SUPABASE_ANON_KEY && API_BASE_URL && TEST_EMAIL && TEST_PASSWORD)

async function signIn(email: string, password: string) {
  const api = await playwrightRequest.newContext()
  const res = await api.post(`${SUPABASE_URL}/auth/v1/token?grant_type=password`, {
    headers: { apikey: SUPABASE_ANON_KEY!, 'Content-Type': 'application/json' },
    data: { email, password }
  })
  expect(res.ok(), `Supabase 登入失敗：${await res.text()}`).toBeTruthy()
  const body = await res.json()
  await api.dispose()
  return body.access_token as string
}

test.describe('Live API data-plane（真實 Supabase + 真實部署 API）', () => {
  test.skip(!canRunApiSuite, '缺少 LIVE_SUPABASE_URL / LIVE_SUPABASE_ANON_KEY / LIVE_API_BASE_URL / LIVE_TEST_EMAIL / LIVE_TEST_PASSWORD')

  test('測試帳號可用真實 Supabase 登入，且 API /auth/me 接受其 JWT', async () => {
    const token = await signIn(TEST_EMAIL!, TEST_PASSWORD!)
    const api = await playwrightRequest.newContext()
    const res = await api.get(`${API_BASE_URL}/auth/me`, {
      headers: { Authorization: `Bearer ${token}` }
    })
    expect(res.status()).toBe(200)
    const body = await res.json()
    expect(body.data.id).toBeTruthy()
    expect(body.data.permissions).toBeTruthy()
    await api.dispose()
  })

  test('API health endpoint 僅回傳公開狀態欄位', async () => {
    const api = await playwrightRequest.newContext()
    const healthURL = API_BASE_URL!.replace(/\/api\/v1\/?$/, '/api/health')
    const res = await api.get(healthURL)
    expect([200, 503]).toContain(res.status())
    const body = await res.json()
    expect(Object.keys(body).sort()).toEqual(['status'])
    await api.dispose()
  })
})
