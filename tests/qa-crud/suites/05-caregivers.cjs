exports.name = '照護人員管理 CRUD'

exports.run = async ({ page, net, record, L }) => {
  const NAME = `${L.TAG}照護`
  await L.goto(page, '/masters/caregivers')

  await L.openCreate(page, '新增照護人員')
  record('類型下拉選項', { options: await L.selectOptions(page, '類型') })

  await L.openCreate(page, '新增照護人員')
  record('空白送出', await L.submit(page, net, '確認送出'))

  await L.openCreate(page, '新增照護人員')
  await L.pick(page, '類型', '個管')
  await L.pickOrCreate(page, '單位', '苗栗縣站')
  await L.fill(page, '姓名', NAME + 'A')
  await L.fill(page, '聯絡方式', '0912-345-678')
  await L.fill(page, '備註', 'QA 建立的完整資料')
  await L.radio(page, '狀態', '停用')
  record('全欄位新增', await L.submit(page, net, '確認送出'))

  await L.openCreate(page, '新增照護人員')
  await L.pick(page, '類型', '照專')
  await L.fill(page, '姓名', NAME + 'B')
  record('只填必填（不選單位）', await L.submit(page, net, '確認送出'))

  await L.openCreate(page, '新增照護人員')
  await L.pick(page, '類型', '個管')
  await L.fill(page, '姓名', NAME + 'A')
  record('同名同類型重複', await L.submit(page, net, '確認送出'))

  await L.openCreate(page, '新增照護人員')
  await L.pick(page, '類型', '個管')
  await L.fill(page, '姓名', '   ')
  record('姓名純空白', await L.submit(page, net, '確認送出'))

  await L.goto(page, '/masters/caregivers')
  await L.rowAction(page, NAME + 'A', '編輯')
  record('編輯載入值（此筆有單位）', await L.dumpForm(page))
  await L.fill(page, '聯絡方式', '0900-000-000')
  record('只改聯絡方式，不動單位', await L.submit(page, net, '確認送出'))

  await L.goto(page, '/masters/caregivers')
  record('編輯後列表（確認單位是否保留）', { rows: await L.tableRows(page) })

  await L.rowAction(page, NAME + 'A', '編輯')
  await L.fill(page, '姓名', NAME + 'A改')
  await L.fill(page, '聯絡方式', '')
  await L.fill(page, '備註', '')
  record('編輯並清空選填欄位', await L.submit(page, net, '確認送出'))

  await L.goto(page, '/masters/caregivers')
  record('列表', { rows: await L.tableRows(page) })

  await L.rowAction(page, NAME + 'B', '刪除')
  record('刪除', await L.confirmBox(page, net))
}
