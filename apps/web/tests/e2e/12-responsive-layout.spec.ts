import { test, expect } from '@playwright/test'
import { loginAs } from './helpers/auth'

const adminRoutes = [
  '/',
  '/cases',
  '/masters/regions',
  '/masters/sites',
  '/masters/vehicles',
  '/masters/drivers',
  '/driver-reports',
  '/driver-reports/status',
  '/driver-reports/import',
  '/rides',
  '/rides/issues',
  '/rides/missing',
  '/reports/trip-summary',
  '/reports/hsinchu-schedule',
  '/vehicles/maintenance',
  '/attendance',
  '/audit',
  '/settings/users',
  '/settings/roles',
  '/settings/notifications',
  '/settings/holidays',
  '/exports'
]

test.describe('12. 響應式版面', () => {
  test('手機版側邊導覽可開啟與關閉，且不遮擋主要內容', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await loginAs(page, 'admin', '/')

    const toggle = page.getByRole('button', { name: '展開側邊導覽' })
    await toggle.click()
    await expect(page.locator('.aside-menu')).toHaveClass(/is-mobile-open/)

    await page.locator('.navigation-backdrop').click({ position: { x: 380, y: 400 } })
    await expect(page.locator('.aside-menu')).not.toHaveClass(/is-mobile-open/)
    await expect(page.locator('#main-content')).toBeVisible()
  })

  test('所有管理頁在手機寬度不造成文件層級的水平捲動', async ({ page }) => {
    test.setTimeout(60000)
    await page.setViewportSize({ width: 390, height: 844 })
    await loginAs(page, 'admin', '/')

    for (const route of adminRoutes) {
      await page.goto(route)
      await expect(page.locator('#main-content')).toBeVisible()
      await expect.poll(() => page.evaluate(() => {
        return document.documentElement.scrollWidth <= window.innerWidth
      }), { message: `${route} 不應產生文件層級的水平捲動` }).toBe(true)
    }
  })
})
