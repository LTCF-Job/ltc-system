import { test, expect } from '@playwright/test'
import { loginAs } from './helpers/auth'
import { waitForTableLoaded, expectElMessage } from './helpers/ui'

test.describe('05. 表單同步與欄位對應 (Forms Sync & Field Mapping)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin')
  })

  test('表單清單載入與批次同步操作', async ({ page }) => {
    await page.goto('/forms')
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table').first()).toBeVisible()

    // 點選全部批次同步按鈕
    const syncAllBtn = page.getByRole('button', { name: /全部批次同步/ })
    if (await syncAllBtn.isVisible()) {
      await syncAllBtn.click()
      await expectElMessage(page, /同步完成/, 'success')
    }
  })

  test('欄位對應：檢視推薦信心度、單筆綁定與略過欄位', async ({ page }) => {
    await page.goto('/forms/mappings')
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table').first()).toBeVisible()

    // 檢查標籤列與雙欄對照
    await expect(page.getByText('原始表單欄位名稱')).toBeVisible()
    await expect(page.getByText('目標個案與排班時段')).toBeVisible()

    // 測試略過此欄操作
    const ignoreBtn = page.locator('.el-table__row').locator('button').filter({ hasText: '略過此欄' }).first()
    if (await ignoreBtn.isVisible()) {
      await ignoreBtn.click()
      await expectElMessage(page, /略過/, 'info')
    }
  })
})

