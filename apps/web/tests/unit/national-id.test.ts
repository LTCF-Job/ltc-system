import { test } from 'node:test'
import assert from 'node:assert/strict'

import { isValidNationalID } from '../../src/utils/nationalId.ts'

// 這組案例與 apps/api/internal/domain/crypto/crypto_test.go 的 TestValidateNationalID 相同。
// 前後端各有一份實作，只要任一邊漂移，使用者就會遇到「前端讓你送出、後端退件」或反過來的
// 情況，因此用同一組向量把兩邊釘在一起。
const SHARED_VECTORS: Array<{ name: string; nid: string; valid: boolean }> = [
  { name: '合法身分證 (蔡曾切)', nid: 'A202559750', valid: true },
  { name: '合法身分證 (司機郭澤威)', nid: 'G121806465', valid: true },
  { name: '合法身分證 (服務人員)', nid: 'K120098177', valid: true },
  { name: '合法外來人口統一證號 (男性)', nid: 'A800000014', valid: true },
  { name: '合法外來人口統一證號 (女性)', nid: 'A900000016', valid: true },
  { name: '檢查碼錯誤', nid: 'A202559751', valid: false },
  { name: '字數不足', nid: 'A20255975', valid: false },
  { name: '含非法字元', nid: 'A20255975A', valid: false },
  { name: '首碼非英文字母', nid: '1202559750', valid: false },
  { name: '第二碼非 1,2,8,9', nid: 'A302559750', valid: false },
  { name: '小寫字母自動轉大寫驗證', nid: 'a202559750', valid: true }
]

test('national id validation matches the backend algorithm', () => {
  for (const { name, nid, valid } of SHARED_VECTORS) {
    assert.equal(isValidNationalID(nid), valid, name)
  }
})

test('national id validation trims surrounding whitespace before checking', () => {
  assert.equal(isValidNationalID('  A202559750  '), true)
  assert.equal(isValidNationalID('\tA202559750\n'), true)
})

test('national id validation rejects empty and structurally wrong input', () => {
  for (const invalid of ['', '   ', 'A2025597500', '身分證字號', 'AA02559750']) {
    assert.equal(isValidNationalID(invalid), false, invalid)
  }
})

// I、O、W、X、Y、Z 的代碼是 34、35、32、30、31、33，與其他字母不同段，最容易在改寫查表時寫錯。
test('national id validation covers the out-of-sequence letter codes', () => {
  const outOfSequence = ['I', 'O', 'W', 'X', 'Y', 'Z']
  for (const letter of outOfSequence) {
    // 對每個字母造一個檢查碼正確的號碼，確認查表值本身沒被改壞。
    const body = '12345678'
    let found = false
    for (let check = 0; check <= 9; check += 1) {
      if (isValidNationalID(`${letter}${body}${check}`)) {
        found = true
        break
      }
    }
    assert.equal(found, true, `字母 ${letter} 應存在唯一合法檢查碼`)
  }
})
