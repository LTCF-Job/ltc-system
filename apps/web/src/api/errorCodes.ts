// 後端 API 統一錯誤碼對應之非技術性提示文字，需與 apps/api/internal/platform/httpx/response.go 的錯誤碼常數保持一致。
// 前端一律依錯誤碼決定顯示文字，不直接信任後端回傳的 message（避免技術性錯誤外洩或未預期字串）。
export const API_ERROR_MESSAGES: Record<string, string> = {
  VALIDATION_FAILED: '輸入資料不符合規則，請確認後再試',
  UNAUTHENTICATED: '請重新登入',
  FORBIDDEN: '權限不足，無法執行此操作',
  NOT_FOUND: '查無資料',
  ASSIGNMENT_OVERLAP: '該時段已有其他排班，請調整後再試',
  EXPORT_IN_PROGRESS: '匯出作業進行中，請稍後再試',
  PRECHECK_FAILED: '資料檢核未通過，請確認後再試',
  MAPPING_REQUIRED: '尚未完成欄位對應設定',
  DRIVER_REPORT_IMPORT_FAILED: '匯入司機接送匯報失敗，請確認檔案格式後再試',
  FORM_MAPPING_FAILED: '更新欄位對應設定失敗，請稍後再試',
  INTERNAL_ERROR: '系統發生錯誤，請稍後再試'
}

const FALLBACK_MESSAGE = '系統發生錯誤，請稍後再試'

// resolveErrorMessage 依錯誤碼查表取得顯示文字；未知或缺少錯誤碼時回退為呼叫端提供的情境訊息（找不到時用通用非技術性訊息），
// 一律不使用後端或 axios 回傳的原始 message，避免技術性錯誤外洩。
export function resolveErrorMessage(code: string | undefined, contextFallback?: string): string {
  if (code && API_ERROR_MESSAGES[code]) return API_ERROR_MESSAGES[code]
  return contextFallback || FALLBACK_MESSAGE
}
