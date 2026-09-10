/** 收據連結欄位共用檢查：允許空白或 http(s):// 開頭的網址，拒絕 javascript: 等其他協定，
 * 避免值被直接渲染成 <a href> 時觸發儲存型 XSS。與後端 isValidReceiptURL 的規則一致。 */
export function isSafeReceiptUrl(value?: string | null): boolean {
  if (!value) return true
  return /^https?:\/\//i.test(value)
}

/** el-form rules 用的驗證規則，供保養、油資等表單共用。 */
export const receiptUrlRule = {
  validator: (_rule: unknown, value: string, callback: (error?: Error) => void) => {
    if (!isSafeReceiptUrl(value)) {
      callback(new Error('收據連結必須為 http:// 或 https:// 開頭的網址'))
      return
    }
    callback()
  },
  trigger: 'blur'
}
