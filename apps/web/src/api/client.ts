import axios, { type AxiosError } from 'axios'
import { ElMessage, ElNotification } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import router from '@/router'
import type { ApiError } from '@/types/api'
import { isMockRuntimeEnabled, isDemoModeActive } from '@/lib/demoMode'
import { resolveErrorMessage } from './errorCodes'

export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 請求攔截器：附加 JWT Token 與 Mock 身分標頭
apiClient.interceptors.request.use(
  (config) => {
    const authStore = useAuthStore()
    // Demo 帳號的請求一律轉送到獨立的 Demo API，避免與正式環境共用同一個後端
    if (authStore.user?.dataPlane === 'demo' && import.meta.env.VITE_DEMO_API_BASE_URL) {
      config.baseURL = import.meta.env.VITE_DEMO_API_BASE_URL
    }
    if (authStore.token) {
      config.headers.Authorization = `Bearer ${authStore.token}`
    }
    if ((isMockRuntimeEnabled() || isDemoModeActive()) && authStore.user) {
      config.headers['X-Mock-Role'] = authStore.user.role
      config.headers['X-Mock-User-ID'] = authStore.user.id || '00000000-0000-0000-0000-000000000001'
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 回應攔截器：處理 401、403 與通用錯誤提示
apiClient.interceptors.response.use(
  (response) => response.data,
  async (error: AxiosError<{ error?: ApiError }>) => {
    const authStore = useAuthStore()
    const status = error.response?.status
    let apiError = error.response?.data?.error

    // 當 responseType 為 'blob' 時，後端返回的 JSON 錯誤會被包在 Blob 內，需讀取轉回物件
    if (!apiError && error.response?.data instanceof Blob) {
      try {
        const text = await error.response.data.text()
        const parsed = JSON.parse(text)
        apiError = parsed?.error
      } catch {
        // 忽略解析失敗
      }
    }

    if (status === 401) {
      const wasAuthenticated = authStore.isAuthenticated
      await authStore.logout()
      if (router.currentRoute.value.path !== '/login') {
        router.push('/login')
        if (wasAuthenticated) {
          ElMessage.error('登入憑證已過期，請重新登入')
        }
      }
      return Promise.reject(error)
    }

    if (status === 403) {
      ElMessage.warning('權限不足，無法執行此操作')
      return Promise.reject(error)
    }

    // 一律依錯誤碼查表顯示非技術性訊息，不直接信任後端或 axios 回傳的原始 message 字串
    const message = resolveErrorMessage(apiError?.code)

    // 若有詳細欄位錯誤清單，以通知元件條列呈現
    if (apiError?.details && apiError.details.length > 0) {
      ElNotification({
        title: message,
        type: 'error',
        message: apiError.details.map((d) => `${d.field ? `[${d.field}] ` : ''}${d.reason}`).join('\n'),
        duration: 6000
      })
    } else {
      ElMessage.error(message)
    }

    return Promise.reject(error)
  }
)
