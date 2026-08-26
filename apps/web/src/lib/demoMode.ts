// 展示帳號固定為帳號密碼皆為 demo；正式環境要換展示帳密時，改這兩個常數並重新部署
const DEMO_ACCOUNT = 'demo'
const DEMO_PASSWORD = 'demo'
const DEMO_MODE_KEY = 'ltc_demo_mode'

let workerStarted = false

export function isDemoCredentials(email: string, password: string): boolean {
  return email === DEMO_ACCOUNT && password === DEMO_PASSWORD
}

function isDemoModeFlagged(): boolean {
  return localStorage.getItem(DEMO_MODE_KEY) === 'true'
}

async function ensureWorkerStarted() {
  if (workerStarted) return
  const { worker } = await import('@/mocks/browser')
  await worker.start({ onUnhandledRequest: 'bypass' })
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
  if (isDemoModeFlagged()) {
    await ensureWorkerStarted()
  }
}

// 登出時呼叫：停用 mock 攔截，避免下一次用真實帳號登入時仍攔到假資料
export function clearDemoModeOnLogout() {
  void ensureWorkerStopped()
  localStorage.removeItem(DEMO_MODE_KEY)
}
