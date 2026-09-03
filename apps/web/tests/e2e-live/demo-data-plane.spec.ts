import { test, expect, request as playwrightRequest } from '@playwright/test'

// 對真正部署的 Supabase Auth 與 Demo Cloud Run API 做 API 層級驗證。
// 必要環境變數缺一則整份跳過，本機或還沒設好 CI secrets 時不會讓 pipeline 失敗。
const SUPABASE_URL = process.env.LIVE_SUPABASE_URL
const SUPABASE_ANON_KEY = process.env.LIVE_SUPABASE_ANON_KEY
const DEMO_API_BASE_URL = process.env.LIVE_DEMO_API_BASE_URL
const DEMO_TEST_EMAIL = process.env.LIVE_DEMO_TEST_EMAIL
const DEMO_TEST_PASSWORD = process.env.LIVE_DEMO_TEST_PASSWORD
const PROD_API_BASE_URL = process.env.LIVE_PROD_API_BASE_URL
const PROD_TEST_EMAIL = process.env.LIVE_PROD_TEST_EMAIL
const PROD_TEST_PASSWORD = process.env.LIVE_PROD_TEST_PASSWORD

const canRunDemoSuite = Boolean(SUPABASE_URL && SUPABASE_ANON_KEY && DEMO_API_BASE_URL && DEMO_TEST_EMAIL && DEMO_TEST_PASSWORD)
const canRunCrossPlaneMatrix = Boolean(canRunDemoSuite && PROD_API_BASE_URL && PROD_TEST_EMAIL && PROD_TEST_PASSWORD)

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

test.describe('Demo data-plane（真實 Supabase + 真實部署 API）', () => {
  test.skip(!canRunDemoSuite, '缺少 LIVE_SUPABASE_URL / LIVE_SUPABASE_ANON_KEY / LIVE_DEMO_API_BASE_URL / LIVE_DEMO_TEST_EMAIL / LIVE_DEMO_TEST_PASSWORD')

  test('Demo 測試帳號可用真實 Supabase 登入，且 Demo API 接受其 JWT', async () => {
    const token = await signIn(DEMO_TEST_EMAIL!, DEMO_TEST_PASSWORD!)
    const api = await playwrightRequest.newContext()
    const res = await api.get(`${DEMO_API_BASE_URL}/regions`, {
      headers: { Authorization: `Bearer ${token}` }
    })
    expect(res.status()).toBe(200)
    await api.dispose()
  })

  test('重置端點回傳 datasetVersion 與 resetAt，重置後基準資料可讀', async () => {
    const token = await signIn(DEMO_TEST_EMAIL!, DEMO_TEST_PASSWORD!)
    const api = await playwrightRequest.newContext()

    const resetRes = await api.post(`${DEMO_API_BASE_URL}/demo/reset`, {
      headers: { Authorization: `Bearer ${token}` }
    })
    expect(resetRes.status()).toBe(200)
    const resetBody = await resetRes.json()
    expect(resetBody.data.datasetVersion).toBeTruthy()
    expect(resetBody.data.resetAt).toBeTruthy()

    const casesRes = await api.get(`${DEMO_API_BASE_URL}/cases`, {
      headers: { Authorization: `Bearer ${token}` }
    })
    expect(casesRes.status()).toBe(200)
    const casesBody = await casesRes.json()
    expect(Array.isArray(casesBody.data)).toBeTruthy()
    expect(casesBody.data.length).toBeGreaterThan(0)

    await api.dispose()
  })

  test.describe('JWT data-plane 拒絕矩陣', () => {
    test.skip(!canRunCrossPlaneMatrix, '缺少 LIVE_PROD_API_BASE_URL / LIVE_PROD_TEST_EMAIL / LIVE_PROD_TEST_PASSWORD')

    test('正式帳號的 JWT 會被 Demo API 拒絕', async () => {
      const prodToken = await signIn(PROD_TEST_EMAIL!, PROD_TEST_PASSWORD!)
      const api = await playwrightRequest.newContext()
      const res = await api.get(`${DEMO_API_BASE_URL}/regions`, {
        headers: { Authorization: `Bearer ${prodToken}` }
      })
      expect(res.status()).toBe(401)
      await api.dispose()
    })

    test('Demo 帳號的 JWT 會被正式 API 拒絕', async () => {
      const demoToken = await signIn(DEMO_TEST_EMAIL!, DEMO_TEST_PASSWORD!)
      const api = await playwrightRequest.newContext()
      const res = await api.get(`${PROD_API_BASE_URL}/regions`, {
        headers: { Authorization: `Bearer ${demoToken}` }
      })
      expect(res.status()).toBe(401)
      await api.dispose()
    })
  })
})
