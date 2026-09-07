import assert from 'node:assert/strict'
import test from 'node:test'

import { resolveErrorMessage } from '../../src/api/errorCodes.ts'

test('error mapping never exposes unknown backend messages', () => {
  assert.equal(resolveErrorMessage('VALIDATION_FAILED'), '輸入資料不符合規則，請確認後再試')
  assert.equal(resolveErrorMessage('UNKNOWN', '情境提示'), '情境提示')
  assert.equal(resolveErrorMessage('UNKNOWN'), '系統發生錯誤，請稍後再試')
})
