import axios, { type AxiosError } from 'axios'
import { ElMessage, ElNotification } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { supabase } from '@/lib/supabase'
import router from '@/router'
import type { ApiError } from '@/types/api'
import { resolveApiErrorMessage, NETWORK_ERROR_MESSAGE, TIMEOUT_ERROR_MESSAGE } from './errorCodes'
export { createPaginationMeta, unwrapData, unwrapDataWithMeta, unwrapPaged, type PendingRelinkedMeta } from './envelope'

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

    // 當 responseType 為 'blob' 時，後端回傳的 JSON 錯誤會被包在 Blob 內，需讀取轉回物件
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
      ElMessage.warning(resolveApiErrorMessage(apiError, '權限不足，無法執行此操作'))
      return Promise.reject(error)
    }

    // Cloud Run（及其前面的 Google Front End）逾時是由平台直接回 504，不會經過我們的
    // 錯誤 envelope，也不是 axios 端的 ECONNABORTED；跟下面「完全沒有回應」的斷線情境
    // 分開判斷，否則使用者會被誤導成「連不上網路」，但實際上是伺服器處理時間過長。
    if (status === 504) {
      ElMessage.error(TIMEOUT_ERROR_MESSAGE)
      return Promise.reject(error)
    }

    // 完全沒有回應代表請求沒走到後端，這與「後端回報錯誤」是兩種不同的處置，
    // 混成同一句通用訊息會讓使用者以為是系統壞掉而不是自己斷線。
    if (!error.response) {
      const offline = typeof navigator !== 'undefined' && navigator.onLine === false
      const timedOut = error.code === 'ECONNABORTED' || error.code === 'ETIMEDOUT'
      ElMessage.error(timedOut && !offline ? TIMEOUT_ERROR_MESSAGE : NETWORK_ERROR_MESSAGE)
      return Promise.reject(error)
    }

    // 後端 message 由 handler 寫死且經 arch test 把關不含技術細節，優先顯示它才能說明
    // 具體原因；缺漏時才退回錯誤碼字典。
    const message = resolveApiErrorMessage(apiError)

    // 常用欄位代碼轉繁體中文標籤，讓錯誤清單明確告知使用者有問題的欄位
    const FIELD_LABELS: Record<string, string> = {
      plateNo: '車號',
      siteId: '所屬據點',
      displayName: '車別',
      brand: '廠牌',
      model: '車型',
      manufactureYm: '出廠年月',
      compulsoryInsuranceExpiry: '強制責任險',
      passengerInsuranceExpiry: '乘客責任險',
      thirdPartyInsuranceExpiry: '第三人責任險',
      lastInspectionDate: '驗車日期',
      wheelchairAccessible: '符合輪椅載運規定',
      status: '狀態',
      name: '姓名',
      nationalId: '身分證字號',
      email: '電子信箱',
      region: '區域',
      address: '地址',
      code: '代碼'
    }

    // 伺服器端錯誤才附上請求識別碼；使用者回報時後端能直接用它查 log。
    const traceSuffix = status && status >= 500 && apiError?.requestId ? `（錯誤編號 ${apiError.requestId}）` : ''

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
          .concat(traceSuffix ? [traceSuffix] : [])
          .join('\n'),
        duration: 6000
      })
    } else {
      ElMessage.error(`${message}${traceSuffix}`)
    }

    return Promise.reject(error)
  }
)
