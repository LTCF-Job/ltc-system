exports.name = '檢視人員（viewer）權限：唯讀是否真的唯讀'

// 這支 suite 必須以 viewer 身分登入才有意義：QA_LOGIN_EMAIL=viewer@example.com node tests/qa-crud/run.cjs 24-viewer
const PAGES = ['/cases', '/masters/sites', '/masters/vehicles', '/masters/drivers',
  '/masters/caregivers', '/vehicles/maintenance', '/settings/holidays', '/settings/notifications',
  '/settings/roles', '/settings/users', '/audit', '/exports']

exports.run = async ({ page, record, L }) => {
  for (const route of PAGES) {
    await L.goto(page, route, 2500)
    record(route, {
      landedOn: new URL(page.url()).pathname,
      buttons: (await page.locator('main button').allInnerTexts()).map(s => s.replace(/\s+/g, ' ').trim()).filter(Boolean),
      rowActions: (await page.locator('.el-table__body tr').first().locator('button').allInnerTexts()).map(s => s.trim())
    })
  }
}
