// 展示帳號固定為帳號密碼皆為 demo；正式環境要換展示帳密時，改這兩個常數並重新部署
const DEMO_ACCOUNT = 'demo'
const DEMO_PASSWORD = 'demo'
const DEMO_MODE_KEY = 'ltc_demo_mode'

let workerStarted = false

export function isMockRuntimeEnabled(): boolean {
  return import.meta.env.VITE_ENABLE_MSW === 'true' && import.meta.env.VITE_E2E === 'true'
}

export function isDemoCredentials(email: string, password: string): boolean {
  return email === DEMO_ACCOUNT && password === DEMO_PASSWORD
}

export function isDemoModeActive(): boolean {
  return localStorage.getItem(DEMO_MODE_KEY) === 'true'
}

async function ensureWorkerStarted() {
  if (workerStarted) return
  const { worker } = await import('@/mocks/browser')
  const { onUnhandledRequest } = await import('@/mocks/onUnhandledRequest')
  await worker.start({ onUnhandledRequest })
  workerStarted = true
}

async function ensureWorkerStopped() {
  if (!workerStarted) return
  const { worker } = await import('@/mocks/browser')
  worker.stop()
  workerStarted = false
}

// 帳號密碼皆為 demo 時呼叫：略過真實登入，啟用 mock 攔截並記住狀態
export async function enterDemoMode() {
  await ensureWorkerStarted()
  localStorage.setItem(DEMO_MODE_KEY, 'true')
}

// 非展示帳號完成真實登入後呼叫：確保沒有殘留前一次展示模式的攔截
export async function exitDemoModeIfActive() {
  await ensureWorkerStopped()
  localStorage.removeItem(DEMO_MODE_KEY)
}

// App 啟動時呼叫：重新整理頁面後，若上次是展示模式且尚未登出，還原 mock 攔截狀態
export async function restoreDemoModeOnBoot() {
  if (isDemoModeActive()) {
    await ensureWorkerStarted()
  }
}

// 登出時呼叫：停用 mock 攔截並清空展示資料，確保下一次 demo/demo 登入拿到乾淨的初始資料集
// 必須等待 worker 停止與資料重置完成才能返回，避免與緊接著的下一次登入互相競態
export async function clearDemoModeOnLogout() {
  const wasActive = isDemoModeActive()
  await ensureWorkerStopped()
  localStorage.removeItem(DEMO_MODE_KEY)
  if (wasActive) {
    const { resetDemoData } = await import('@/mocks/data/demoStore')
    resetDemoData()
  }
}
