exports.name = '單位管理 CRUD'

exports.run = async ({ page, net, record, expect, L }) => {
  const NAME = `${L.TAG}單位`
  await L.goto(page, '/masters/sites')

  await L.openCreate(page, '新增單位')
  const regionOptions = await L.selectOptions(page, '所屬區域')
  record('所屬區域下拉選項', { options: regionOptions })
  // 下拉必須來自地區主檔；寫死清單不會包含使用者自建的地區，也會列出 DB 存不進去的值。
  const masterNames = await page.evaluate(async () => {
    const res = await fetch('/api/v1/regions?all=true', {
      headers: { Authorization: `Bearer ${localStorage.getItem('ltc_auth_token')}` }
    })
    const body = await res.json()
    return (body.data || []).filter((r) => r.status === 'active').map((r) => r.name)
  })
  expect('區域下拉＝地區主檔 active 清單', regionOptions.length === masterNames.length,
    { dropdown: regionOptions.length, master: masterNames.length })

  await L.openCreate(page, '新增單位')
  record('空白送出', await L.submit(page, net))

  await L.openCreate(page, '新增單位')
  await L.fill(page, '單位名稱', NAME)
  record('只填名稱', await L.submit(page, net))

  await L.openCreate(page, '新增單位')
  await L.fill(page, '單位名稱', NAME)
  await L.pick(page, '所屬區域', '苗栗縣')
  record('名稱+區域未填地址', await L.submit(page, net))

  await L.openCreate(page, '新增單位')
  await L.fill(page, '單位名稱', NAME)
  await L.pick(page, '所屬區域', '苗栗縣')
  await L.fill(page, '單位地址', '苗栗縣測試路1號')
  record('完整新增（苗栗縣）', await L.submit(page, net))

  await L.openCreate(page, '新增單位')
  await L.fill(page, '單位名稱', NAME + '台北')
  await L.pick(page, '所屬區域', '臺北市')
  await L.fill(page, '單位地址', '臺北市測試路2號')
  record('選 DB 不允許的區域（臺北市）', await L.submit(page, net))

  await L.openCreate(page, '新增單位')
  await L.fill(page, '單位名稱', NAME)
  await L.pick(page, '所屬區域', '苗栗縣')
  await L.fill(page, '單位地址', '重複測試')
  record('同名同區域重複', await L.submit(page, net))

  await L.openCreate(page, '新增單位')
  await L.fill(page, '單位名稱', NAME + '無開放日')
  await L.pick(page, '所屬區域', '苗栗縣')
  await L.fill(page, '單位地址', '無開放日測試')
  const checked = await L.item(page, '開放星期').locator('.el-checkbox.is-checked').count()
  for (let i = 0; i < checked; i++) {
    await L.item(page, '開放星期').locator('.el-checkbox.is-checked').first().click()
  }
  record('取消全部開放星期', await L.submit(page, net))

  await L.goto(page, '/masters/sites')
  await L.rowAction(page, NAME, '編輯')
  record('編輯載入值', await L.dumpForm(page))
  await L.fill(page, '單位名稱', NAME + '改')
  await L.fill(page, '單位地址', '改後地址999號')
  await L.radio(page, '狀態', '停用')
  record('編輯送出', await L.submit(page, net))

  await L.goto(page, '/masters/sites')
  record('列表', { rows: await L.tableRows(page) })

  await L.rowAction(page, NAME + '無開放日', '刪除')
  record('刪除未被引用的單位', await L.confirmBox(page, net))

  // 刪除既有 demo 單位會連帶影響車輛與個案，預設不跑。
  if (process.env.QA_DESTRUCTIVE === '1') {
    await L.goto(page, '/masters/sites')
    await L.rowAction(page, '測試單位', '刪除')
    record('刪除已被車輛引用的單位', await L.confirmBox(page, net))
  }
}
