exports.name = '假日設定 CRUD 與政府行事曆匯入'

exports.run = async ({ page, net, record, step, L }) => {
  await L.goto(page, '/settings/holidays')

  await L.openCreate(page, '新增休假日／上班日')
  await step('全部欄位留白直接送出', () => L.submit(page, net))

  await L.openCreate(page, '新增休假日／上班日')
  await step('新增休假日', async () => {
    await L.pickDate(page, '日期', '2026-10-10')
    await L.radio(page, '類型', '休假日')
    await L.fill(page, '名稱', 'QA國慶日')
    return L.submit(page, net)
  })

  await L.openCreate(page, '新增休假日／上班日')
  await step('同一天重複新增', async () => {
    await L.pickDate(page, '日期', '2026-10-10')
    await L.radio(page, '類型', '休假日')
    await L.fill(page, '名稱', 'QA重複日')
    return L.submit(page, net)
  })

  await L.openCreate(page, '新增休假日／上班日')
  await step('新增上班日（補班）', async () => {
    await L.pickDate(page, '日期', '2026-10-11')
    await L.radio(page, '類型', '上班日')
    await L.fill(page, '名稱', 'QA補班日')
    return L.submit(page, net)
  })

  await L.openCreate(page, '新增休假日／上班日')
  await step('只選日期不填名稱', async () => {
    await L.pickDate(page, '日期', '2026-10-12')
    return L.submit(page, net)
  })

  await L.goto(page, '/settings/holidays')
  record('清單', { rows: await L.tableRows(page) })

  await step('從政府行事曆匯入', async () => {
    const before = net.calls.length
    await page.getByRole('button', { name: '從政府行事曆匯入', exact: false }).first().click()
    await page.waitForTimeout(1500)
    const dialogText = (await page.locator('.el-dialog:visible').count()) > 0
      ? (await L.dialog(page).innerText()).replace(/\s+/g, ' ').trim().slice(0, 800) : '(無對話框)'
    const confirm = L.dialog(page).locator('button', { hasText: /匯入|確認|開始/ }).last()
    if (await confirm.count() > 0) {
      await confirm.click()
      await page.waitForTimeout(6000)
    }
    return { dialogText, calls: net.calls.slice(before), toasts: await L.toasts(page) }
  })
  await L.closeDialog(page)

  await L.goto(page, '/settings/holidays')
  record('匯入後清單', { rows: (await L.tableRows(page)).slice(0, 15) })

  await step('刪除 QA國慶日', async () => {
    await L.rowAction(page, 'QA國慶日', '刪除')
    return L.confirmBox(page, net)
  })
}
