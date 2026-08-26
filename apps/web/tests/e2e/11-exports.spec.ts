import { test, expect } from '@playwright/test'
import { loginAs } from './helpers/auth'
import { waitForTableLoaded, expectElMessage } from './helpers/ui'

test.describe('11. 政府申報匯出與前置檢核 (Gov Claims Export & Precheck)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin', '/exports')
  })

  test('匯出設定介面與前置檢核執行', async ({ page }) => {
    await waitForTableLoaded(page)
    await expect(page.getByText('政府申報表匯出設定')).toBeVisible()

    // 點選執行前置檢核
    const precheckBtn = page.getByRole('button', { name: '執行前置檢核' })
    await expect(precheckBtn).toBeVisible()
    await precheckBtn.click()

    // 檢查檢核結果區塊
    await expect(page.getByText('前置檢核報告')).toBeVisible({ timeout: 10000 })
  })

  test('歷史匯出紀錄表格正常顯示', async ({ page }) => {
    await waitForTableLoaded(page)
    await expect(page.getByText('歷史匯出紀錄')).toBeVisible()
    await expect(page.locator('.history-card .el-table').first()).toBeVisible()
  })
})

