exports.name = '頁面控制項探索（寫測試前用）'

const PAGES = ['/attendance', '/settings/holidays', '/settings/notifications', '/rides', '/exports', '/rides/issues', '/driver-reports/import']

exports.run = async ({ page, step, L }) => {
  for (const route of PAGES) {
    await L.goto(page, route, 2500)
    await step(route, async () => ({
      tabs: (await page.locator('.el-tabs__item').allInnerTexts()).map(s => s.trim()),
      buttons: (await page.locator('main button').allInnerTexts()).map(s => s.replace(/\s+/g, ' ').trim()).filter(Boolean),
      fileInputs: await page.locator('main input[type=file]').count(),
      text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 600)
    }))
  }
}
