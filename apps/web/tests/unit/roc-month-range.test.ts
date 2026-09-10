import { test } from 'node:test'
import assert from 'node:assert/strict'

import { useRocMonth } from '../../src/composables/useRocMonth.ts'

const { toRocPeriodYmList } = useRocMonth()

// 多月匯出把使用者勾選的西元月份換算成後端要的民國 5 碼。順序與重複都必須先收斂：
// 這份清單決定要產幾份申報檔，也決定批次壓縮檔名的起訖月份。
test('toRocPeriodYmList converts each selected month to the ROC reporting period', () => {
  assert.deepEqual(toRocPeriodYmList(['2026-05', '2026-07']), ['11505', '11507'])
})

// 使用者點選的順序不固定，但申報月份的順序必須穩定，否則批次檔名的起訖月份會顛倒。
test('toRocPeriodYmList sorts ascending regardless of selection order', () => {
  assert.deepEqual(toRocPeriodYmList(['2026-07', '2026-05', '2026-06']), ['11505', '11506', '11507'])
})

test('toRocPeriodYmList drops duplicate months', () => {
  assert.deepEqual(toRocPeriodYmList(['2026-05', '2026-05']), ['11505'])
})

// 跨年是最容易出錯的邊界：民國 114 年 12 月的下一個月是民國 115 年 1 月。
test('toRocPeriodYmList crosses the year boundary correctly', () => {
  assert.deepEqual(toRocPeriodYmList(['2026-01', '2025-12']), ['11412', '11501'])
})

test('toRocPeriodYmList returns an empty list when nothing is selected', () => {
  assert.deepEqual(toRocPeriodYmList([]), [])
})

// 字串排序在五碼定長且零補齊的前提下才等同於時間排序，明確釘住這個前提。
test('toRocPeriodYmList keeps single-digit months zero-padded so string sort matches time order', () => {
  const periods = toRocPeriodYmList(['2026-10', '2026-09', '2026-01'])

  assert.deepEqual(periods, ['11501', '11509', '11510'])
  for (const period of periods) {
    assert.equal(period.length, 5)
  }
})

// Date 物件與純日期字串必須換算出相同結果，否則不同來源的選取值會分岔。
test('toRocPeriodYmList accepts Date objects alongside date strings', () => {
  assert.deepEqual(toRocPeriodYmList([new Date(2026, 4, 15), '2026-05']), ['11505'])
})
