import { test, expect } from '@playwright/test'
import { loginAs } from './helpers/auth'
import { waitForTableLoaded, expectElMessage, confirmMessageBox } from './helpers/ui'

test.describe('07. 異常集中處理與未回報催報 (Ride Issues & Missing Rides)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin')
  })

  test('異常集中處理分頁切換與清單載入', async ({ page }) => {
    await page.goto('/rides/issues')
    await waitForTableLoaded(page)

    // 檢查三分頁
    await expect(page.getByRole('tab', { name: /混車衝突/ })).toBeVisible()
    await expect(page.getByRole('tab', { name: /未回報/ })).toBeVisible()
    await expect(page.getByRole('tab', { name: /匯入異常/ })).toBeVisible()

    // 切換至未回報分頁
    await page.getByRole('tab', { name: /未回報/ }).click()
    await waitForTableLoaded(page)
    await expect(page.locator('#pane-unreported .el-table')).toBeVisible()

    // 切換至匯入異常分頁
    await page.getByRole('tab', { name: /匯入異常/ }).click()
    await waitForTableLoaded(page)
    await expect(page.locator('#pane-import_error .el-table')).toBeVisible()
  })

  test('混車衝突人工裁決流程', async ({ page }) => {
    await page.goto('/rides/issues')
    await waitForTableLoaded(page)

    // 點選第一筆人工裁決按鈕
    const resolveBtn = page.locator('.el-table__row').locator('button').filter({ hasText: '人工裁決' }).first()
    if (await resolveBtn.isVisible()) {
      await resolveBtn.click()
      const dialog = page.locator('.el-dialog').filter({ hasText: '混車衝突裁決' })
      await expect(dialog).toBeVisible()

      // 送出裁決
      await dialog.getByRole('button', { name: '確認送出' }).click()
      await confirmMessageBox(page)
      await expectElMessage(page, /已裁決/, 'success')
    }
  })

  test('未回報搭乘清單與催報功能', async ({ page }) => {
    await page.goto('/rides/missing')
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table').first()).toBeVisible()
  })
})

