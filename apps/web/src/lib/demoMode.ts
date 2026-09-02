const DEMO_MODE_KEY = 'ltc_demo_mode'

let workerStarted = false

export function isMockRuntimeEnabled(): boolean {
  return import.meta.env.VITE_ENABLE_MSW === 'true' && import.meta.env.VITE_E2E === 'true'
}

// 檢查是否為展示模式登入憑證
export function isDemoCredentials(email: string, password: string): boolean {
  const normalizedEmail = (email || '').trim().toLowerCase()
  const isDemoUser = normalizedEmail === 'demo' || normalizedEmail.startsWith('demo@')
  const isDemoPass = password === 'demo' || password === 'demo123'
  return isDemoUser && isDemoPass
}

// 取得當前是否處於展示模式
export function isDemoModeActive(): boolean {
  return localStorage.getItem(DEMO_MODE_KEY) === 'true'
}

async function ensureWorkerStarted() {
  if (isMockRuntimeEnabled()) return
  if (workerStarted) return
  const { worker } = await import('@/mocks/browser')
  const { onUnhandledRequest } = await import('@/mocks/onUnhandledRequest')
  await worker.start({ onUnhandledRequest })
  workerStarted = true
}

async function ensureWorkerStopped() {
  if (isMockRuntimeEnabled()) return
  if (!workerStarted) return
  const { worker } = await import('@/mocks/browser')
  worker.stop()
  workerStarted = false
}

// 登入真實帳號時清理展示模式殘留狀態
export async function exitDemoModeIfActive() {
  await ensureWorkerStopped()
  localStorage.removeItem(DEMO_MODE_KEY)
}

// 應用程式啟動時依本機儲存狀態還原 Mock 攔截
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

// 重設記憶體中所有展示資料至初始狀態
export async function resetDemoData() {
  const { resetDemoData: reset } = await import('@/mocks/data/demoStore')
  reset()
}
