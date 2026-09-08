const path = require('node:path')

exports.name = '司機接送匯報批次上傳'

async function uploadAndReport(page, net, L, file, label) {
  const before = net.calls.length
  await page.setInputFiles('main input[type=file]', file)
  await page.waitForTimeout(3000)
  const actions = (await page.locator('main button').allInnerTexts()).map(s => s.replace(/\s+/g, ' ').trim()).filter(Boolean)
  const go = page.locator('main button', { hasText: /開始匯入|確認匯入|開始解析|匯入/ }).first()
  if (await go.count() > 0 && await go.isEnabled()) {
    await go.click()
    await page.waitForTimeout(8000)
  }
  return {
    label,
    actionsBeforeImport: actions,
    calls: net.calls.slice(before),
    text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 1800),
    toasts: await L.toasts(page)
  }
}

exports.run = async ({ page, net, step, expect, L }) => {
  await L.goto(page, '/driver-reports/import')

  await step('上傳空白官方範本（只有表頭）', () =>
    uploadAndReport(page, net, L, path.join(L.DOWNLOAD_DIR, 'AAA-123接送匯報範本.xlsx'), '空白範本'))

  await L.goto(page, '/driver-reports/import')
  await step('上傳含各種資料列的匯報檔', async () => {
    const out = await uploadAndReport(page, net, L, path.join(__dirname, '..', 'fixtures', 'AAA-123 (回覆).xlsx'), '組合測試檔')
    // 檔案含 2 列壞日期、5 列合法：單筆髒資料不得讓整月停擺（準則三 §3.2）。
    const commit = out.calls.find((c) => c.phase === 'res' && c.url.includes('dryRun=false'))
    expect('錯誤列不得讓整月匯入失敗', commit?.status === 200, { status: commit?.status, body: (commit?.body || '').slice(0, 200) })
    return out
  })

  await L.goto(page, '/driver-reports/import')
  await step('重複上傳同一份匯報檔', () =>
    uploadAndReport(page, net, L, path.join(__dirname, '..', 'fixtures', 'AAA-123 (回覆).xlsx'), '重複上傳'))

  await L.goto(page, '/driver-reports/import')
  await step('待維護資料頁籤', async () => {
    await page.locator('.el-tabs__item', { hasText: '待維護資料' }).first().click()
    await page.waitForTimeout(3000)
    return { text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 2000) }
  })

  await L.goto(page, '/driver-reports/status')
  await step('接送匯報總覽', async () => ({
    text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 900)
  }))
}
