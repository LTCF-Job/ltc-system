exports.name = '角色、使用者與稽核紀錄'

exports.run = async ({ page, net, record, step, L }) => {
  await L.goto(page, '/settings/roles')
  record('角色頁載入結果', {
    text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 500),
    httpErrors: net.failed.slice(-3)
  })

  await step('新增自訂角色', async () => {
    await L.openCreate(page, '新增自訂角色')
    await L.fill(page, '角色名稱', `${L.TAG}稽核員`)
    await L.fill(page, '角色說明', 'QA 建立的自訂角色')
    return L.submit(page, net)
  })

  await L.goto(page, '/settings/users')
  record('使用者頁載入結果', {
    text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 400),
    httpErrors: net.failed.slice(-3)
  })

  await step('新增使用者', async () => {
    await L.openCreate(page, '新增使用者')
    const form = await L.dumpForm(page)
    return { form, result: await L.submit(page, net) }
  })
  await L.closeDialog(page)

  await L.goto(page, '/audit')
  record('稽核紀錄前 8 筆', { rows: (await L.tableRows(page)).slice(0, 8) })

  await step('稽核紀錄動作類型篩選', async () => {
    const before = net.calls.length
    const select = page.locator('main .el-select').first()
    await select.click()
    await page.waitForTimeout(600)
    const options = (await page.locator('.el-select-dropdown:visible .el-select-dropdown__item').allInnerTexts()).map(s => s.trim())
    await page.keyboard.press('Escape')
    await page.waitForTimeout(300)
    return { options, calls: net.calls.slice(before) }
  })
}
