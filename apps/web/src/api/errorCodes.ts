import type { ApiError } from '@/types/api'

// 後端 API 統一錯誤碼對應之非技術性提示文字，需與 apps/api/internal/platform/httpx/response.go 的
// codeMessages 保持一致（由 tests/unit/error-codes.test.ts 直接比對 Go 原始碼把關）。
// 這份表是「後端沒給訊息」時的後備文字，不是唯一顯示來源。
export const API_ERROR_MESSAGES: Record<string, string> = {
  VALIDATION_FAILED: '輸入資料不符合規則，請確認後再試',
  UNAUTHENTICATED: '請重新登入',
  FORBIDDEN: '權限不足，無法執行此操作',
  NOT_FOUND: '查無資料',
  ASSIGNMENT_OVERLAP: '該時段已有其他排班，請調整後再試',
  EXPORT_IN_PROGRESS: '匯出作業進行中，請稍後再試',
  PRECHECK_FAILED: '資料檢核未通過，請確認後再試',
  NO_EXPORT_DATA: '指定條件下沒有可申報的資料',
  MAPPING_REQUIRED: '尚未完成欄位對應設定',
  DRIVER_REPORT_IMPORT_FAILED: '匯入司機接送匯報失敗，請確認檔案格式後再試',
  FORM_MAPPING_FAILED: '更新欄位對應設定失敗，請稍後再試',
  INTERNAL_ERROR: '系統發生錯誤，請稍後再試',
  SERVICE_UNAVAILABLE: '服務暫時無法使用，請稍後再試',
  RESOURCE_IN_USE: '此資料已被其他紀錄使用，無法刪除',
  UNSUPPORTED_FILE_TYPE: '檔案格式不支援，請改用 .xlsx 檔案',
  FILE_TOO_LARGE: '檔案超過大小上限，請分批匯入',
  FILE_UNREADABLE: '檔案無法讀取，可能已損毀或非有效的 Excel 檔',
  IMPORT_TEMPLATE_MISMATCH: '檔案欄位與匯入範本不符，請下載標準範本重新填寫',
  ROUTE_NOT_FOUND: '找不到此功能的服務位址，請重新整理頁面或聯繫系統管理員',
  STALE_WRITE: '資料已被其他人更新，請重新整理後再試',
  CONFLICT_ALREADY_RESOLVED: '此衝突已由其他人處理，請重新整理'
}

// 連線層錯誤沒有後端回應可查，前端自行分辨並給出對應處置，避免與伺服器錯誤混為一談。
export const NETWORK_ERROR_MESSAGE = '無法連線到伺服器，請確認網路狀態後再試'
export const TIMEOUT_ERROR_MESSAGE = '伺服器回應逾時，請稍後再試；若為大量資料請縮小查詢範圍'

const FALLBACK_MESSAGE = '系統發生錯誤，請稍後再試'

// 後端訊息一律是 handler 寫死的非技術性字串（由 apps/api/internal/arch 的
// TestErrorMessagesAreStaticText 保證不含 err.Error() 之類的執行期內容），因此可以直接顯示；
// 顯示它才拿得到「哪個欄位、哪個月份格式」這種具體原因，而不是每次都同一句通用訊息。
export function resolveApiErrorMessage(apiError: ApiError | undefined, contextFallback?: string): string {
  const serverMessage = apiError?.message?.trim()
  if (serverMessage) return serverMessage
  return resolveErrorMessage(apiError?.code, contextFallback)
}

// resolveErrorMessage 依錯誤碼查表取得顯示文字；未知或缺少錯誤碼時回退為呼叫端提供的情境訊息
// （找不到時用通用非技術性訊息）。
export function resolveErrorMessage(code: string | undefined, contextFallback?: string): string {
  if (code && API_ERROR_MESSAGES[code]) return API_ERROR_MESSAGES[code]
  return contextFallback || FALLBACK_MESSAGE
}
