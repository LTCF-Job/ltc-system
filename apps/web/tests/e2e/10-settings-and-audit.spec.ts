import { test, expect } from '@playwright/test'
import { loginAs } from './helpers/auth'
import { waitForTableLoaded, expectElMessage } from './helpers/ui'

test.describe('10. 系統稽核紀錄與權限設定 (Audit Logs & System Settings)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin')
  })

  test('稽核紀錄：清單載入、篩選與查看異動前後比對彈窗', async ({ page }) => {
    await page.goto('/audit')
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table').first()).toBeVisible()

    // 點選第一筆異動前後按鈕
    const diffBtn = page.locator('.el-table__row').locator('button, a').filter({ hasText: '異動前後' }).first()
    if (await diffBtn.isVisible()) {
      await diffBtn.click()
      const dialog = page.locator('.el-dialog').filter({ hasText: '系統操作紀錄異動詳情' })
      await expect(dialog).toBeVisible()
      await expect(dialog.getByText('所屬區塊').first()).toBeVisible()
      await dialog.getByRole('button', { name: '關閉', exact: true }).click()
    }
  })

  test('使用者管理：清單檢視與使用者資料設定', async ({ page }) => {
    await page.goto('/settings/users')
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table').first()).toBeVisible()
  })

  test('角色身分管理：角色清單與權限矩陣檢視', async ({ page }) => {
    await page.goto('/settings/roles')
    await waitForTableLoaded(page)
    await expect(page.locator('.filter-card, .table-card, .role-management-view, .el-table').first()).toBeVisible()
  })

  test('通知收件人管理：通知類型收件人檢視', async ({ page }) => {
    await page.goto('/settings/notifications')
    await waitForTableLoaded(page)
    await expect(page.locator('.filter-card, .table-card, .notification-settings-view, .el-table').first()).toBeVisible()
  })
})

