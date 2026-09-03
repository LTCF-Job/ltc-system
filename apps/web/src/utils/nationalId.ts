// 與 apps/api/internal/domain/crypto/crypto.go 的 ValidateNationalID 演算法保持一致，
// 讓前端能在送出前就攔截檢查碼錯誤的身分證字號，避免依賴後端回傳的通用驗證錯誤訊息。
const LETTER_CODES: Record<string, number> = {
  A: 10, B: 11, C: 12, D: 13, E: 14, F: 15, G: 16, H: 17, I: 34,
  J: 18, K: 19, L: 20, M: 21, N: 22, O: 35, P: 23, Q: 24, R: 25,
  S: 26, T: 27, U: 28, V: 29, W: 32, X: 30, Y: 31, Z: 33
}

export function isValidNationalID(input: string): boolean {
  const nid = input.trim().toUpperCase()
  if (nid.length !== 10) return false

  const code = LETTER_CODES[nid[0]]
  if (code === undefined) return false

  // 第 2 碼（性別碼）：本國人 1、2；外來人口居留證 8、9
  if (!['1', '2', '8', '9'].includes(nid[1])) return false

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
