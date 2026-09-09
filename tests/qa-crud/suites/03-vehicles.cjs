exports.name = '車輛管理 CRUD'

exports.run = async ({ page, net, record, L }) => {
  const P = (n) => `${L.TAG.slice(-4)}-000${n}`
  const D = (n) => `${L.TAG}車${n}`
  await L.goto(page, '/masters/vehicles')

  await L.openCreate(page, '新增車輛')
  record('空白送出', await L.submit(page, net))

  await L.openCreate(page, '新增車輛')
  await L.fill(page, '車號', P(1))
  await L.fill(page, '車別', D(1))
  await L.pick(page, '所屬單位', '測試單位')
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
  await L.pick(page, '所屬單位', '測試單位')
  record('只填必填欄位', await L.submit(page, net))

  await L.openCreate(page, '新增車輛')
  await L.fill(page, '車號', P(1))
  await L.fill(page, '車別', D(3))
  await L.pick(page, '所屬單位', '測試單位')
  record('車號重複', await L.submit(page, net))

  await L.openCreate(page, '新增車輛')
  await L.fill(page, '車號', P(3))
  await L.fill(page, '車別', D(1))
  await L.pick(page, '所屬單位', '測試單位')
  record('車別重複', await L.submit(page, net))

  await L.openCreate(page, '新增車輛')
  await L.fill(page, '車號', '   ')
  await L.fill(page, '車別', '   ')
  await L.pick(page, '所屬單位', '測試單位')
  record('車號車別純空白', await L.submit(page, net))

  await L.openCreate(page, '新增車輛')
  await L.fill(page, '車號', P(4))
  await L.fill(page, '車別', D(4))
  await L.pick(page, '所屬單位', '測試單位')
  await L.pickDate(page, '出廠年月', '1800-01')
  record('出廠年月填 1800-01', await L.submit(page, net))

  await L.goto(page, '/masters/vehicles')
  await L.rowAction(page, D(1), '編輯')
  record('編輯載入值', await L.dumpForm(page))
  await L.fill(page, '廠牌', '')
  await L.fill(page, '車型', '')
  record('清空選填欄位後儲存', await L.submit(page, net))

  await L.goto(page, '/masters/vehicles')
  await L.rowAction(page, D(2), '司機')
  record('維護司機對話框', await L.dumpForm(page))
  const driverOptions = await L.selectOptions(page, '本車司機')
  record('可指派司機選項', { options: driverOptions })
  if (driverOptions.length > 0) {
    await L.pick(page, '本車司機', driverOptions[0])
    record('指派司機', await L.submit(page, net, '儲存'))
  }

  await L.goto(page, '/masters/vehicles')
  await L.rowAction(page, D(4), '刪除')
  record('刪除未被引用的車輛', await L.confirmBox(page, net))

  await L.goto(page, '/masters/vehicles')
  await L.rowAction(page, D(2), '刪除')
  record('刪除已有生效中司機指派的車輛', await L.confirmBox(page, net))

  // 刪除「已有搭乘紀錄」的車輛會真的刪掉既有 demo 資料，預設不跑。
  if (process.env.QA_DESTRUCTIVE === '1') {
    await L.goto(page, '/masters/vehicles')
    await L.rowAction(page, '竹北一車', '刪除')
    record('刪除已有搭乘紀錄的車輛', await L.confirmBox(page, net))
  }
}
