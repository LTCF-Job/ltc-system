import axios, { type AxiosError } from 'axios'
import { ElMessage, ElNotification } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import router from '@/router'
import type { ApiError } from '@/types/api'

export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 請求攔截器：附加 JWT Token 與 Mock 角色標頭
apiClient.interceptors.request.use(
  (config) => {
    const authStore = useAuthStore()
    if (authStore.token) {
      config.headers.Authorization = `Bearer ${authStore.token}`
    }
    if (authStore.user) {
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
  (error: AxiosError<{ error?: ApiError }>) => {
    const authStore = useAuthStore()
    const status = error.response?.status
    const apiError = error.response?.data?.error

    if (status === 401) {
      const wasAuthenticated = authStore.isAuthenticated
      authStore.logout()
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

    const message = apiError?.message || error.message || '系統發生錯誤，請稍後再試'

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
