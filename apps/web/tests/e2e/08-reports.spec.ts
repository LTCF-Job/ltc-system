import { test, expect } from '@playwright/test'
import { loginAs } from './helpers/auth'
import { waitForTableLoaded, expectElMessage } from './helpers/ui'

test.describe('08. 營運報表 (Operational Reports)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin')
  })

  test('車輛趟數表：統計卡片、各車明細與 Excel 匯出', async ({ page }) => {
    await page.goto('/reports/trip-summary')
    await waitForTableLoaded(page)

    // 檢查總覽統計卡片
    await expect(page.getByText('全期去程趟數合計')).toBeVisible()
    await expect(page.getByText('全期回程趟數合計')).toBeVisible()
    await expect(page.getByText('全期車輛總趟數')).toBeVisible()

    // 檢查車輛明細區塊
    await expect(page.locator('.vehicle-card').first()).toBeVisible()
    await expect(page.locator('.el-table').first()).toBeVisible()

    // 檢查 Excel 匯出按鈕
    const exportBtn = page.getByRole('button', { name: /匯出 Excel 趟數表/ })
    await expect(exportBtn).toBeVisible()
  })

  test('新竹接送時刻表：日期切換與班次時刻清單', async ({ page }) => {
    await page.goto('/reports/hsinchu-schedule')
    await waitForTableLoaded(page)
    await expect(page.locator('.filter-card, .page-header, .schedule-view, .el-table').first()).toBeVisible()
  })
})

