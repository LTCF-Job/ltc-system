import { test } from 'node:test'
import assert from 'node:assert/strict'

import {
  VEHICLE_DATE_FIELDS,
  emptyVehicleForm,
  sanitizeVehiclePayload,
  vehicleFormRules
} from '../../src/utils/vehicleForm.ts'

// 空字串與 null 在後端是不同意思：null 代表「沒有這筆資料」，空字串會讓日期欄位解析失敗。
// 這是送出前最後一道轉換，寫錯不會有型別錯誤，只會在 API 回 400 時才發現。
test('sanitizeVehiclePayload turns optional blanks into null', () => {
  const payload = sanitizeVehiclePayload(emptyVehicleForm())

  assert.equal(payload.brand, null)
  assert.equal(payload.model, null)
  assert.equal(payload.manufactureYm, null)
  assert.equal(payload.compulsoryInsuranceExpiry, null)
  assert.equal(payload.passengerInsuranceExpiry, null)
  assert.equal(payload.thirdPartyInsuranceExpiry, null)
  assert.equal(payload.lastInspectionDate, null)
})

// 必填欄位維持空字串而非 null，讓後端的 binding:"required" 能一致地擋下來。
// 據點是自由輸入的選填文字，非必填，同樣不轉成 null（trim 後留空字串）。
test('sanitizeVehiclePayload keeps required fields as strings', () => {
  const payload = sanitizeVehiclePayload(emptyVehicleForm())

  assert.equal(payload.plateNo, '')
  assert.equal(payload.displayName, '')
  assert.equal(payload.siteName, '')
})

test('sanitizeVehiclePayload trims whitespace-only values down to null', () => {
  const payload = sanitizeVehiclePayload({
    ...emptyVehicleForm(),
    plateNo: '  BZG-7915  ',
    displayName: '  無障礙一號  ',
    brand: '   ',
    model: '  Hiace  ',
    lastInspectionDate: '   '
  })

  assert.equal(payload.plateNo, 'BZG-7915')
  assert.equal(payload.displayName, '無障礙一號')
  assert.equal(payload.brand, null, '只有空白的選填欄位要視為未填')
  assert.equal(payload.model, 'Hiace')
  assert.equal(payload.lastInspectionDate, null)
})

test('sanitizeVehiclePayload preserves non-string fields', () => {
  const payload = sanitizeVehiclePayload({
    ...emptyVehicleForm(),
    wheelchairAccessible: false,
    status: 'inactive'
  })

  assert.equal(payload.wheelchairAccessible, false)
  assert.equal(payload.status, 'inactive')
})

// 車號格式規則直接決定使用者能不能建立車輛，改壞會擋掉合法車號或放行錯誤格式。
test('vehicle plate rule accepts real plate shapes and rejects malformed ones', () => {
  const pattern = vehicleFormRules.plateNo[1].pattern as RegExp

  for (const valid of ['BZG-7915', 'AB-1234', 'ABC-123', '1234-AB', 'RAB-0001']) {
    assert.equal(pattern.test(valid), true, `${valid} 應為合法車號`)
  }

  for (const invalid of ['BZG7915', 'bzg-7915', 'B-1234', 'BZGA1-7915', 'BZG-79151', 'BZG--7915', '']) {
    assert.equal(pattern.test(invalid), false, `${invalid} 應為不合法車號`)
  }
})

// 四個日期欄位在表單、驗證與民國換算提示三處共用同一份宣告，缺一個就會出現沒有提示的欄位。
test('vehicle date fields cover every date column on the form', () => {
  const props = VEHICLE_DATE_FIELDS.map((field) => field.prop)
  const form = emptyVehicleForm()

  assert.deepEqual(props, [
    'compulsoryInsuranceExpiry',
    'passengerInsuranceExpiry',
    'thirdPartyInsuranceExpiry',
    'lastInspectionDate'
  ])
  for (const prop of props) {
    assert.ok(prop in form, `${prop} 必須存在於空白表單`)
  }
  for (const field of VEHICLE_DATE_FIELDS) {
    assert.notEqual(field.label, '', `${field.prop} 必須有標籤`)
  }
})

test('vehicle document flags default to false and survive sanitizing', () => {
  const form = emptyVehicleForm()
  assert.equal(form.hasVehicleLicense, false)
  assert.equal(form.hasPurchaseContract, false)
  assert.equal(form.hasPlateRegistration, false)
  assert.equal(form.hasTransferRegistration, false)

  // sanitize 的職責是把空字串轉成 null；布林的 false 必須原樣通過，
  // 寫成 `form.hasVehicleLicense || null` 會讓「未持有」被送成 null。
  const sanitized = sanitizeVehiclePayload({
    ...form,
    hasVehicleLicense: true,
    hasPurchaseContract: false
  })
  assert.equal(sanitized.hasVehicleLicense, true)
  assert.equal(sanitized.hasPurchaseContract, false)
  assert.equal(sanitized.hasPlateRegistration, false)
  assert.equal(sanitized.hasTransferRegistration, false)
})
