/**
 * 中華民國身分證與外來人口統一證號檢查碼驗證與遮罩，邏輯對齊後端
 * apps/api/internal/domain/crypto/crypto.go 的 ValidateNationalID／Mask，
 * 讓 mock 模式下的個案匯入驗證規則與正式環境一致。
 */

const LETTER_CODE_MAP: Record<string, number> = {
  A: 10, B: 11, C: 12, D: 13, E: 14, F: 15, G: 16, H: 17, I: 34,
  J: 18, K: 19, L: 20, M: 21, N: 22, O: 35, P: 23, Q: 24, R: 25,
  S: 26, T: 27, U: 28, V: 29, W: 32, X: 30, Y: 31, Z: 33
}

export function isValidTaiwanNationalId(raw: string): boolean {
  const nid = raw.trim().toUpperCase()
  if (nid.length !== 10) return false

  const code = LETTER_CODE_MAP[nid[0]]
  if (code === undefined) return false

  // 第 2 碼（性別碼）：本國人 1、2；外來人口居留證 8、9
  const secondChar = nid[1]
  if (!['1', '2', '8', '9'].includes(secondChar)) return false

  for (let i = 1; i < 10; i++) {
    if (!/[0-9]/.test(nid[i])) return false
  }

  const n1 = Math.floor(code / 10)
  const n2 = code % 10
  let sum = n1 + n2 * 9
  const weights = [8, 7, 6, 5, 4, 3, 2, 1]
  for (let i = 0; i < 8; i++) {
    sum += Number(nid[i + 1]) * weights[i]
  }
  sum += Number(nid[9])

  return sum % 10 === 0
}

export function maskNationalId(plain: string): string {
  const value = plain.trim()
  if (value.length < 10) return '*'.repeat(value.length)
  return value.slice(0, 3) + '***' + value.slice(-4)
}
