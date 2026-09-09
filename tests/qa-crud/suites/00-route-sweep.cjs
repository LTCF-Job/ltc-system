const ROUTES = [
  '/', '/cases', '/masters/regions', '/masters/sites', '/masters/vehicles',
  '/masters/drivers', '/masters/caregivers', '/driver-reports/status',
  '/driver-reports/import', '/rides', '/rides/issues', '/rides/missing',
  '/reports/trip-summary', '/reports/hsinchu-schedule', '/vehicles/maintenance',
  '/attendance', '/audit', '/settings/users', '/settings/roles',
  '/settings/notifications', '/settings/holidays', '/exports'
]

exports.name = '全路由煙霧測試'

exports.run = async ({ page, net, record, L }) => {
  for (const route of ROUTES) {
    const failedBefore = net.failed.length
    const consoleBefore = net.console.length
    await L.goto(page, route, 2500)
    let text = ''
    try { text = (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 240) } catch (_) { text = '(no main)' }
    record(route, {
      landedOn: new URL(page.url()).pathname,
      text,
      httpErrors: net.failed.slice(failedBefore),
      consoleErrors: net.console.slice(consoleBefore)
    })
  }
}
