exports.name = '車輛管理 CRUD'

exports.run = async ({ page, net, record, L }) => {
  const P = (n) => `${L.TAG.slice(-4)}-000${n}`
  const D = (n) => `${L.TAG}車${n}`
  const SITE = `${L.TAG}據點`
  await L.goto(page, '/masters/vehicles')

  await L.openCreate(page, '新增車輛')
  record('空白送出', await L.submit(page, net))

  await L.openCreate(page, '新增車輛')
  await L.fill(page, '車號', P(1))
  await L.fill(page, '車別', D(1))
  await L.fill(page, '據點', SITE)
  await L.fill(page, '廠牌', '中華')
  await L.fill(page, '車型', 'DE241L8')
  await L.pickDate(page, '出廠年月', '2020-05')
  await L.pickDate(page, '強制責任險', '2027-01-31')
  await L.pickDate(page, '乘客責任險', '2027-02-28')
  await L.pickDate(page, '第三人責任險', '2027-03-31')
  await L.pickDate(page, '驗車日期', '2026-06-15')
  await L.radio(page, '符合輪椅載運規定', '否')
  await L.check(page, '證件資料', '行照')
  await L.check(page, '證件資料', '領牌登記書')
  await L.radio(page, '狀態', '停用')
  record('全欄位新增（含證件資料）', await L.submit(page, net))

  await L.openCreate(page, '新增車輛')
  await L.fill(page, '車號', P(2))
  await L.fill(page, '車別', D(2))
  record('只填必填欄位（不填據點）', await L.submit(page, net))

  await L.openCreate(page, '新增車輛')
  await L.fill(page, '車號', P(1))
  await L.fill(page, '車別', D(3))
  record('車號重複', await L.submit(page, net))

  await L.openCreate(page, '新增車輛')
  await L.fill(page, '車號', P(3))
  await L.fill(page, '車別', D(1))
  record('車別重複', await L.submit(page, net))

  await L.openCreate(page, '新增車輛')
  await L.fill(page, '車號', '   ')
  await L.fill(page, '車別', '   ')
  record('車號車別純空白', await L.submit(page, net))

  await L.openCreate(page, '新增車輛')
  await L.fill(page, '車號', P(4))
  await L.fill(page, '車別', D(4))
  await L.pickDate(page, '出廠年月', '1800-01')
  record('出廠年月填 1800-01', await L.submit(page, net))

  await L.goto(page, '/masters/vehicles')
  await L.rowAction(page, D(1), '編輯')
  record('編輯載入值', await L.dumpForm(page))
  await L.fill(page, '廠牌', '')
  await L.fill(page, '車型', '')
  record('清空選填欄位後儲存', await L.submit(page, net))

  // 指派司機已從「司機」對話框按鈕（現已移除）改為列表「駕駛司機」欄位的行內多選 InlineOptionPicker。
  await L.goto(page, '/masters/vehicles')
  const driverRow = await L.resolveRow(page, D(2))
  const driverTrigger = driverRow.locator('.inline-picker-trigger').first()
  await driverTrigger.click()
  await page.waitForTimeout(400)
  const driverOptions = await page.locator('.inline-picker-popper:visible .inline-picker-item').allInnerTexts()
  record('可指派司機選項', { options: driverOptions.map(s => s.trim()) })
  if (driverOptions.length > 0) {
    const before = net.calls.length
    await page.locator('.inline-picker-popper:visible .inline-picker-item').first().click()
    await page.waitForTimeout(200)
    await driverTrigger.click() // 多選面板只在關閉當下送出，再點一次 trigger 收起面板
    await page.waitForTimeout(1200)
    record('指派司機', { calls: net.calls.slice(before), toasts: await L.toasts(page) })
    await L.clearToasts(page)
  }

  await L.goto(page, '/masters/vehicles')
  await L.rowAction(page, D(4), '刪除')
  record('刪除未被引用的車輛', await L.confirmBox(page, net))

  await L.goto(page, '/masters/vehicles')
  await L.rowAction(page, D(2), '刪除')
  record('刪除已有生效中司機指派的車輛', await L.confirmBox(page, net))

  // 車輛（migration 000046）已改為自由輸入的據點文字欄位，刪除車輛不再受「所屬據點」牽制。
  // 這裡改測「已有搭乘紀錄」是否仍會擋刪除；demo seed 目前未內建搭乘紀錄，
  // 需先跑過 21-rides.cjs（或已存在搭乘紀錄）此分支才有意義，預設仍不跑以避免異動既有 demo 資料。
  if (process.env.QA_DESTRUCTIVE === '1') {
    await L.goto(page, '/masters/vehicles')
    await L.rowAction(page, '新竹一號車', '刪除')
    record('刪除已有搭乘紀錄的車輛', await L.confirmBox(page, net))
  }
}
