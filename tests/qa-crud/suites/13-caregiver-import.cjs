const path = require('node:path')

exports.name = '照護人員批次匯入'

async function importFile(page, net, L, file) {
  const before = net.calls.length
  await page.getByRole('button', { name: '批次匯入照護人員', exact: false }).first().click()
  await page.waitForTimeout(1200)
  await page.setInputFiles('.el-dialog input[type=file]', file)
  await page.waitForTimeout(1200)
  const parse = L.dialog(page).locator('button', { hasText: /解析|預覽|上傳/ }).last()
  if (await parse.count() > 0) {
    await parse.click()
    await page.waitForTimeout(5000)
  }
  const confirm = L.dialog(page).locator('button', { hasText: /^匯入|確認匯入|確認新增/ }).last()
  let confirmed = false
  if (await confirm.count() > 0 && await confirm.isEnabled()) {
    await confirm.click()
    await page.waitForTimeout(5000)
    confirmed = true
  }
  return {
    confirmed,
    calls: net.calls.slice(before),
    dialogText: (await page.locator('.el-dialog:visible').count()) > 0
      ? (await L.dialog(page).innerText()).replace(/\s+/g, ' ').trim().slice(0, 2000) : '(已關閉)',
    toasts: await L.toasts(page)
  }
}

exports.run = async ({ page, net, step, L }) => {
  await L.goto(page, '/masters/caregivers')

  await step('直接匯入未修改的官方範本', () =>
    importFile(page, net, L, path.join(L.DOWNLOAD_DIR, '照護人員批次匯入範本.xlsx')))
  await L.closeDialog(page)

  await L.goto(page, '/masters/caregivers')
  await step('匯入欄位組合測試檔', () =>
    importFile(page, net, L, path.join(__dirname, '..', 'fixtures', 'caregiver-import-filled.xlsx')))
  await L.closeDialog(page)

  await L.goto(page, '/masters/caregivers')
  await step('匯入後清單', async () => ({ rows: await L.tableRows(page) }))

  await step('待維護頁籤', async () => {
    const tab = page.locator('.el-tabs__item', { hasText: '待維護' }).first()
    if (await tab.count() === 0) return { note: '找不到待維護頁籤' }
    await tab.click()
    await page.waitForTimeout(2000)
    return { text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 1200) }
  })
}
