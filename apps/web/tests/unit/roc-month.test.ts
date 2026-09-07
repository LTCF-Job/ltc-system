import { test } from 'node:test'
import assert from 'node:assert/strict'

import { useRocMonth } from '../../src/composables/useRocMonth.ts'
import { formatDate } from '../../src/utils/formatters.ts'

const { toRocMonth, toRocPeriodYm, formatRocMonthLabel, rocToGregorianMonth } = useRocMonth()

// 民國年 = 西元年 - 1911。政府申報期別直接使用這個換算，差一年整批資料就會報到錯的期別。
test('toRocMonth converts the Gregorian year to the ROC year', () => {
  assert.equal(toRocMonth(new Date(2026, 2, 15)), '115-03')
  assert.equal(toRocMonth(new Date(2025, 0, 1)), '114-01')
  assert.equal(toRocMonth(new Date(1912, 11, 31)), '1-12')
})

test('toRocMonth pads single-digit months to two digits', () => {
  for (let month = 0; month < 12; month += 1) {
    const rocMonth = toRocMonth(new Date(2026, month, 10))
    assert.equal(rocMonth, `115-${String(month + 1).padStart(2, '0')}`)
  }
})

// 申報期別是不帶連字號的六碼；少了補零會變成五碼，政府端會直接退件。
test('toRocPeriodYm produces the zero-padded reporting period', () => {
  assert.equal(toRocPeriodYm(new Date(2026, 2, 15)), '11503')
  assert.equal(toRocPeriodYm(new Date(2026, 11, 1)), '11512')
  assert.equal(toRocPeriodYm(new Date(2026, 0, 31)).length, 5)
})

test('formatRocMonthLabel renders a readable label and drops the leading zero of the month', () => {
  assert.equal(formatRocMonthLabel('115-03'), '民國 115 年 3 月')
  assert.equal(formatRocMonthLabel('115-12'), '民國 115 年 12 月')
})

test('formatRocMonthLabel passes through values it cannot parse', () => {
  assert.equal(formatRocMonthLabel(''), '')
  assert.equal(formatRocMonthLabel('11503'), '11503')
})

// rocToGregorianMonth 是送查詢參數前的反向換算，必須與 toRocMonth 完全互逆。
test('rocToGregorianMonth is the exact inverse of toRocMonth', () => {
  assert.equal(rocToGregorianMonth('115-03'), '2026-03')
  assert.equal(rocToGregorianMonth('114-01'), '2025-01')
  assert.equal(rocToGregorianMonth(''), '')

  for (let month = 0; month < 12; month += 1) {
    const date = new Date(2026, month, 10)
    const roundTripped = rocToGregorianMonth(toRocMonth(date))
    assert.equal(roundTripped, `2026-${String(month + 1).padStart(2, '0')}`)
  }
})

// 民國月份只有個位數時也要補回兩位數的西元月份，否則會送出 "2026-3" 這種後端解析不了的值。
test('rocToGregorianMonth pads single-digit ROC months', () => {
  assert.equal(rocToGregorianMonth('115-3'), '2026-03')
})

// 原生 new Date('2026-03-01') 依規格解析為 UTC，在負時區會退回 2 月；formatters 用 dayjs
// 解析為本地時間。兩者若不一致，同一個日期會在民國月份與其他日期欄位顯示成不同的月。
test('roc month parses date-only strings the same way as the shared formatters', () => {
  const dateOnly = ['2026-01-01', '2026-03-01', '2026-12-31', '2025-06-15']

  for (const value of dateOnly) {
    const [year, month] = formatDate(value).split('-')
    assert.equal(toRocMonth(value), `${Number(year) - 1911}-${month}`, value)
    assert.equal(toRocPeriodYm(value), `${Number(year) - 1911}${month}`, value)
  }
})

// 月初與月底最容易因為時區偏移跨到相鄰月份，明確釘住這兩個邊界。
test('roc month keeps month boundaries stable for date-only strings', () => {
  assert.equal(toRocMonth('2026-03-01'), '115-03')
  assert.equal(toRocMonth('2026-03-31'), '115-03')
  assert.equal(toRocPeriodYm('2026-01-01'), '11501')
  assert.equal(toRocPeriodYm('2026-12-31'), '11512')
})

// Date 物件的行為不得因為改用 dayjs 而改變。
test('roc month still accepts Date objects', () => {
  assert.equal(toRocMonth(new Date(2026, 2, 15)), '115-03')
  assert.equal(toRocPeriodYm(new Date(2026, 2, 15)), '11503')
})
