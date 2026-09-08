exports.name = '搭乘月曆、異常與未回報'

exports.run = async ({ page, net, record, step, L }) => {
  await L.goto(page, '/rides')

  await step('切換到有資料的月份 2026-07', async () => {
    const before = net.calls.length
    const monthInput = page.locator('main input').first()
    await monthInput.click()
    await page.waitForTimeout(400)
    await monthInput.fill('2026-07')
    await page.keyboard.press('Enter')
    await page.waitForTimeout(600)
    await page.getByRole('button', { name: '查詢', exact: false }).first().click()
    await page.waitForTimeout(3000)
    return {
      calls: net.calls.slice(before),
      text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 1200)
    }
  })

  await step('點擊月曆格子開啟人工補登', async () => {
    const before = net.calls.length
    const cells = page.locator('.el-table__body tr td')
    const count = await cells.count()
    if (count < 4) return { note: '月曆無資料列', cellCount: count }
    await cells.nth(3).click()
    await page.waitForTimeout(1500)
    const form = await L.dumpForm(page)
    return { calls: net.calls.slice(before), form }
  })

  await step('人工補登送出', async () => {
    if ((await page.locator('.el-dialog:visible').count()) === 0) return { note: '沒有開啟補登對話框' }
    return L.submit(page, net, /儲存|確認|送出/.source)
  })
  await L.closeDialog(page)

  await L.goto(page, '/rides/issues')
  for (const tab of ['混車衝突待裁決', '應搭未回報清單', '表單匯入異常']) {
    await step(`異常頁籤：${tab}`, async () => {
      await page.locator('.el-tabs__item', { hasText: tab }).first().click()
      await page.waitForTimeout(2500)
      return {
        rows: (await L.tableRows(page)).slice(0, 5),
        text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 500)
      }
    })
  }

  await L.goto(page, '/rides/missing')
  record('未回報清單', {
    rows: (await L.tableRows(page)).slice(0, 5),
    text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 500)
  })

  await step('立即執行未回報催報', async () => {
    const before = net.calls.length
    const btn = page.getByRole('button', { name: '立即執行未回報催報', exact: false }).first()
    if (await btn.count() === 0) return { note: '找不到催報按鈕' }
    await btn.click()
    await page.waitForTimeout(1500)
    const confirmed = await L.confirmBox(page, net)
    await page.waitForTimeout(3000)
    return { confirmed, calls: net.calls.slice(before), toasts: await L.toasts(page) }
  })
}
