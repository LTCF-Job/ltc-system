const path = require('node:path')

exports.name = '司機接送匯報批次上傳'

const VEHICLE_NAME = 'QA匯報測試車'
const CASE_NAME = 'QA匯報測試個案'
const TEMPLATE_FILE = path.join(__dirname, '..', 'fixtures', `${VEHICLE_NAME}接送匯報範本.xlsx`)
const FILLED_FILE = path.join(__dirname, '..', 'fixtures', `${VEHICLE_NAME} (回覆).xlsx`)

// driver-report-rows.json 的 2、3 欄是「個案欄位比對真實個案姓名」的測試，2026-08-31 決定的匯報表流程
// 改成依「檔名比對車輛顯示名稱」自動選車（見 DriverReportImportView.vue 的 detectVehicle），也不再有下載
// 官方範本的按鈕（只保留後端 API）。這裡固定用一台跟一筆個案，缺了就先建，測試才不受這台長期活用的
// 本機 docker 資料庫裡其他 session 留下的資料影響。
async function ensureVehicle(page, L, displayName) {
  await L.goto(page, '/masters/vehicles')
  const exists = (await (await L.resolveRow(page, displayName)).count()) > 0
  if (exists) return
  await L.openCreate(page, '新增車輛')
  await L.fill(page, '車號', 'QA00-0001')
  await L.fill(page, '車別', displayName)
  await L.submit(page, { calls: [] })
}

async function ensureCase(page, L, name) {
  await L.goto(page, '/cases')
  const exists = (await (await L.resolveRow(page, name)).count()) > 0
  if (exists) return
  await L.openCreate(page, '新增個案')
  await L.fill(page, '個案姓名', name)
  const sites = await L.selectOptions(page, '所屬據點')
  if (sites.length > 0) await L.pick(page, '所屬據點', sites[0])
  const caregivers = await L.selectOptions(page, '照護人員')
  if (caregivers.length > 0) await L.pick(page, '照護人員', caregivers[0])
  await L.submit(page, { calls: [] })
}

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
  await step('確保匯報檔引用的車輛與個案存在', async () => {
    await ensureVehicle(page, L, VEHICLE_NAME)
    await ensureCase(page, L, CASE_NAME)
  })

  await L.goto(page, '/driver-reports/import')
  await step('上傳空白範本（只有表頭）', () =>
    uploadAndReport(page, net, L, TEMPLATE_FILE, '空白範本'))

  await L.goto(page, '/driver-reports/import')
  await step('上傳含各種資料列的匯報檔', async () => {
    const out = await uploadAndReport(page, net, L, FILLED_FILE, '組合測試檔')
    // 檔案含 2 列壞日期、5 列合法：單筆髒資料不得讓整月停擺（準則三 §3.2）。
    const commit = out.calls.find((c) => c.phase === 'res' && c.url.includes('dryRun=false'))
    expect('錯誤列不得讓整月匯入失敗', commit?.status === 200, { status: commit?.status, body: (commit?.body || '').slice(0, 200) })
    return out
  })

  await L.goto(page, '/driver-reports/import')
  await step('重複上傳同一份匯報檔', () =>
    uploadAndReport(page, net, L, FILLED_FILE, '重複上傳'))

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
