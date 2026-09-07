import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import test from 'node:test'

import { API_ERROR_MESSAGES, resolveApiErrorMessage, resolveErrorMessage } from '../../src/api/errorCodes.ts'

const responseGoPath = fileURLToPath(
  new URL('../../../api/internal/platform/httpx/response.go', import.meta.url)
)

test('error mapping falls back to the code dictionary', () => {
  assert.equal(resolveErrorMessage('VALIDATION_FAILED'), '輸入資料不符合規則，請確認後再試')
  assert.equal(resolveErrorMessage('UNKNOWN', '情境提示'), '情境提示')
  assert.equal(resolveErrorMessage('UNKNOWN'), '系統發生錯誤，請稍後再試')
})

test('resolveApiErrorMessage prefers the backend message', () => {
  assert.equal(
    resolveApiErrorMessage({ code: 'VALIDATION_FAILED', message: '月份格式錯誤，請使用 RRR-MM' }),
    '月份格式錯誤，請使用 RRR-MM'
  )
})

test('resolveApiErrorMessage falls back when the backend message is missing or blank', () => {
  assert.equal(resolveApiErrorMessage({ code: 'NOT_FOUND', message: '' }), '查無資料')
  assert.equal(resolveApiErrorMessage({ code: 'NOT_FOUND', message: '   ' }), '查無資料')
  assert.equal(resolveApiErrorMessage(undefined, '匯入失敗'), '匯入失敗')
  assert.equal(resolveApiErrorMessage(undefined), '系統發生錯誤，請稍後再試')
})

// 後端新增錯誤碼卻忘了同步這份字典時，該錯誤只會顯示成通用的「系統發生錯誤」。
// 直接讀 Go 原始碼比對，讓漏改在測試就被擋下來，而不是等使用者回報看不懂的提示。
test('the frontend dictionary covers every backend error code', () => {
  const source = readFileSync(responseGoPath, 'utf8')
  const constBlock = source.slice(source.indexOf('const ('), source.indexOf(')', source.indexOf('const (')))
  const backendCodes = [...constBlock.matchAll(/Code[A-Za-z]+\s*=\s*"([^"]+)"/g)].map((m) => m[1])

  assert.ok(backendCodes.length > 0, '未能從 response.go 解析出錯誤碼')

  const missing = backendCodes.filter((code) => !(code in API_ERROR_MESSAGES))
  assert.deepEqual(missing, [], `前端字典缺少後端錯誤碼：${missing.join(', ')}`)

  const extra = Object.keys(API_ERROR_MESSAGES).filter((code) => !backendCodes.includes(code))
  assert.deepEqual(extra, [], `前端字典有後端未定義的錯誤碼：${extra.join(', ')}`)
})
