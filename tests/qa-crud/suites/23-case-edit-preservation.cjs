exports.name = '個案編輯頁：未觸碰欄位是否被保留'

exports.run = async ({ page, net, step, L }) => {
  await L.goto(page, '/cases')

  await step('開啟有完整背景資料的個案並直接儲存', async () => {
    await L.rowAction(page, 'QA匯入F主檔存在', '編輯')
    await page.waitForTimeout(2500)
    const loaded = (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 900)
    const before = net.calls.length
    await page.locator('main button', { hasText: '儲存基本資料' }).first().click()
    await page.waitForTimeout(2500)
    return { loaded, calls: net.calls.slice(before), toasts: await L.toasts(page) }
  })
}
