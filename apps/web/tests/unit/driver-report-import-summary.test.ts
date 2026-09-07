import assert from 'node:assert/strict'
import test from 'node:test'

import { describeImportResult, hasPendingWork } from '../../src/views/driverReports/importSummary.ts'

const empty = {
  importedDays: 0,
  rideRecords: 0,
  reaffirmed: 0,
  conflicts: 0,
  backfilled: 0,
  pendingColumns: 0
}

test('重傳同一份檔案時明講內容與既有相同，不能讓使用者以為沒匯入', () => {
  assert.equal(
    describeImportResult({ ...empty, importedDays: 22, reaffirmed: 41 }),
    '已處理 22 天，內容與既有資料完全相同，沒有需要更新的地方'
  )
})

test('有新增與待維護時分開列出各自筆數', () => {
  assert.equal(
    describeImportResult({ ...empty, importedDays: 22, rideRecords: 41, reaffirmed: 8, conflicts: 3 }),
    '已處理 22 天：新增 41 筆、與既有相同 8 筆、待維護 3 筆'
  )
})

test('欄位剛完成對應時一併說明補寫了先前月份', () => {
  assert.equal(
    describeImportResult({ ...empty, importedDays: 5, rideRecords: 10, backfilled: 12 }),
    '已處理 5 天：新增 10 筆、另補寫先前月份 12 筆'
  )
})

test('有處理天數但四個數字皆為零時說明沒有可判讀的回報值', () => {
  assert.equal(
    describeImportResult({ ...empty, importedDays: 22 }),
    '已處理 22 天，但這些欄位都沒有可判讀的回報值'
  )
})

test('沒有可寫入的日期但建立了待維護欄位時導向後續動作', () => {
  assert.equal(
    describeImportResult({ ...empty, pendingColumns: 4 }),
    '已建立 4 個待維護欄位，完成個案連結後會自動補寫搭乘紀錄'
  )
})

test('完全沒有任何結果時才說沒有可寫入的搭乘資料', () => {
  assert.equal(describeImportResult(empty), '沒有可寫入的搭乘資料')
})

test('hasPendingWork 只在有衝突或待維護欄位時為真', () => {
  assert.equal(hasPendingWork({ conflicts: 0, pendingColumns: 0 }), false)
  assert.equal(hasPendingWork({ conflicts: 1, pendingColumns: 0 }), true)
  assert.equal(hasPendingWork({ conflicts: 0, pendingColumns: 2 }), true)
})
