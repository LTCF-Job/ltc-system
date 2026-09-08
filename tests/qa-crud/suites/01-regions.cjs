exports.name = '地區管理 CRUD'

exports.run = async ({ page, net, record, L }) => {
  const NAME = `${L.TAG}地區`
  await L.goto(page, '/masters/regions')

  await L.openCreate(page, '新增地區')
  record('空白送出應被前端擋下', await L.submit(page, net))

  await L.openCreate(page, '新增地區')
  await L.fill(page, '地區名稱', '   ')
  record('純空白名稱', await L.submit(page, net))

  await L.openCreate(page, '新增地區')
  await L.fill(page, '地區名稱', 'X'.repeat(300))
  record('超長名稱 300 字', await L.submit(page, net))

  await L.openCreate(page, '新增地區')
  await L.fill(page, '地區名稱', '<img src=x onerror=alert(1)>')
  record('HTML 注入字串', await L.submit(page, net))

  await L.openCreate(page, '新增地區')
  await L.fill(page, '地區名稱', NAME)
  await L.fill(page, '排序權重', '101')
  await L.radio(page, '狀態', '啟用')
  record('完整新增', await L.submit(page, net))

  await L.openCreate(page, '新增地區')
  await L.fill(page, '地區名稱', NAME)
  record('重複名稱', await L.submit(page, net))

  await L.goto(page, '/masters/regions')
  await L.rowAction(page, NAME, '編輯')
  record('編輯載入值', await L.dumpForm(page))
  await L.fill(page, '地區名稱', NAME + '改')
  await L.fill(page, '排序權重', '102')
  await L.radio(page, '狀態', '停用')
  record('編輯送出', await L.submit(page, net))

  await L.goto(page, '/masters/regions')
  const before = net.calls.length
  await (await L.resolveRow(page, NAME + '改')).locator('.status-toggle-pill').first().click()
  await page.waitForTimeout(1500)
  record('列表狀態切換', { calls: net.calls.slice(before), toasts: await L.toasts(page) })
  await L.clearToasts(page)

  await L.goto(page, '/masters/regions')
  await L.rowAction(page, NAME + '改', '刪除')
  record('刪除', await L.confirmBox(page, net))
}
