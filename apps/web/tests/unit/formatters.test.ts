import assert from 'node:assert/strict'
import test from 'node:test'

import { formatDate, formatDateTime, formatTime, formatYearMonth } from '../../src/utils/formatters.ts'

test('formatters keep date and time precision at seconds', () => {
  assert.equal(formatDateTime('2026-09-05 01:02:03.999'), '2026-09-05 01:02:03')
  assert.equal(formatDate('2026-09-05 01:02:03.999'), '2026-09-05')
  assert.equal(formatTime('08:30'), '08:30:00')
  assert.equal(formatYearMonth('115-07'), '115 年 07 月')
})

test('formatters use explicit fallback for empty values', () => {
  assert.equal(formatDateTime(null), '-')
  assert.equal(formatDate(undefined, '無日期'), '無日期')
  assert.equal(formatTime('', '無時間'), '無時間')
})
