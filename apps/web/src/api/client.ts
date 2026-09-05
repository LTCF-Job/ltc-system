import axios, { type AxiosError } from 'axios'
import { ElMessage, ElNotification } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { supabase } from '@/lib/supabase'
import router from '@/router'
import type { ApiError } from '@/types/api'
import { resolveErrorMessage } from './errorCodes'

export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 請求攔截器：附加 JWT Token
apiClient.interceptors.request.use(
  async (config) => {
    const authStore = useAuthStore()
    let activeToken = authStore.token
    if (supabase) {
      // 正式環境每次請求都向 Supabase 取得目前 session，避免沿用過期的舊 access token。
      const { data } = await supabase.auth.getSession()
      activeToken = data.session?.access_token || null
    }
    if (activeToken) {
      config.headers.Authorization = `Bearer ${activeToken}`
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

    // 常用欄位代碼轉繁體中文標籤，讓錯誤清單明確告知使用者有問題的欄位
    const FIELD_LABELS: Record<string, string> = {
      plateNo: '車號',
      siteId: '所屬單位',
      displayName: '代稱',
      brand: '廠牌',
      model: '車型',
      manufactureYm: '出廠年月',
      compulsoryInsuranceExpiry: '強制責任險',
      passengerInsuranceExpiry: '乘客責任險',
      thirdPartyInsuranceExpiry: '第三人責任險',
      lastInspectionDate: '前次檢驗日期',
      wheelchairAccessible: '符合輪椅載運規定',
      status: '狀態',
      name: '姓名',
      nationalId: '身分證字號',
      email: '電子信箱',
      region: '區域',
      address: '地址',
      code: '代碼'
    }

    // 若有詳細欄位錯誤清單，以通知元件條列呈現
    if (apiError?.details && apiError.details.length > 0) {
      ElNotification({
        title: message,
        type: 'error',
        message: apiError.details
          .map((d) => {
            const label = d.field ? FIELD_LABELS[d.field] || d.field : ''
            return `${label ? `【${label}】` : ''}${d.reason}`
          })
          .join('\n'),
        duration: 6000
      })
    } else {
      ElMessage.error(message)
    }

    return Promise.reject(error)
  }
)
