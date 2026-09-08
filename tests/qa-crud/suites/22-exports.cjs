exports.name = '政府申報匯出'

exports.run = async ({ page, net, step, L }) => {
  await L.goto(page, '/exports')

  await step('未選個案直接前置檢核', async () => {
    const before = net.calls.length
    await page.getByRole('button', { name: '執行前置檢核', exact: false }).first().click()
    await page.waitForTimeout(4000)
    return { calls: net.calls.slice(before), toasts: await L.toasts(page) }
  })

  await step('未選個案直接產生申報檔', async () => {
    const before = net.calls.length
    await page.getByRole('button', { name: '開始產生申報檔', exact: false }).first().click()
    await page.waitForTimeout(4000)
    return { calls: net.calls.slice(before), toasts: await L.toasts(page) }
  })

  await step('選擇個案', async () => {
    await page.getByRole('button', { name: '選擇個案', exact: false }).first().click()
    await page.waitForTimeout(1500)
    const header = L.dialog(page).locator('.el-table__header .el-checkbox').first()
    if (await header.count() > 0) await header.click()
    await page.waitForTimeout(600)
    const confirm = L.dialog(page).locator('button', { hasText: /確認|確定|套用/ }).last()
    if (await confirm.count() > 0) await confirm.click()
    await page.waitForTimeout(1500)
    return { text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 500) }
  })

  await step('選好個案後執行前置檢核', async () => {
    const before = net.calls.length
    await page.getByRole('button', { name: '執行前置檢核', exact: false }).first().click()
    await page.waitForTimeout(6000)
    return {
      calls: net.calls.slice(before),
      text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 1500),
      toasts: await L.toasts(page)
    }
  })

  await step('產生申報檔（直接下載模式）', async () => {
    const before = net.calls.length
    await page.getByRole('button', { name: '開始產生申報檔', exact: false }).first().click()
    await page.waitForTimeout(8000)
    await L.confirmBox(page, net).catch(() => {})
    await page.waitForTimeout(4000)
    return {
      calls: net.calls.slice(before),
      text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 1200),
      toasts: await L.toasts(page)
    }
  })

  await L.goto(page, '/exports')
  await step('歷史匯出紀錄', async () => ({ rows: (await L.tableRows(page)).slice(0, 5) }))
}
