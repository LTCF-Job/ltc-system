import { isValidNationalID } from '@/utils/nationalId'

type Rule = Record<string, unknown>

/**
 * 身分證字號的表單規則。新增司機時必填；編輯司機時留空代表不變更，
 * 因此只在有輸入時檢查格式。兩處共用同一份規則，避免檢查碼邏輯各寫一份而漂移。
 */
export function nationalIdRules(required: boolean): Rule[] {
  const rules: Rule[] = []
  if (required) {
    rules.push({ required: true, message: '請輸入身分證字號', trigger: 'blur' })
  }
  rules.push({
    validator: (_rule: unknown, value: string, callback: (error?: Error) => void) => {
      if (value && !isValidNationalID(value)) {
        callback(new Error('身分證字號格式錯誤，請確認後再試'))
        return
      }
      callback()
    },
    trigger: 'blur'
  })
  return rules
}
