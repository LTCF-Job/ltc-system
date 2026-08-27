import { test, expect } from '@playwright/test'
import { loginAs } from './helpers/auth'
import { waitForTableLoaded } from './helpers/ui'

test.describe('02. 總覽儀表板 (Dashboard)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin', '/')
  })

  test('指標卡片與數值正常顯示', async ({ page }) => {
    await waitForTableLoaded(page)
    await expect(page.getByText('在案個案總數')).toBeVisible()
    await expect(page.getByText('本月已回報趟數')).toBeVisible()
    await expect(page.getByText('司機平均請假率')).toBeVisible()
    await expect(page.getByText('待處理混車衝突')).toBeVisible()
    await expect(page.getByText('待對應表單欄位')).toBeVisible()
  })

  test('視覺化圖表容器載入完成', async ({ page }) => {
    await waitForTableLoaded(page)
    await expect(page.getByText('各車當月接送趟數分佈')).toBeVisible()
    await expect(page.getByText('車隊出勤與請假狀態')).toBeVisible()
    const canvases = page.locator('canvas')
    await expect(canvases.first()).toBeVisible({ timeout: 10000 })
  })

  test('快捷功能按鈕可順暢跳轉至各功能頁面', async ({ page }) => {
    await waitForTableLoaded(page)
    // 點選搭乘月曆表
    await page.getByRole('button', { name: '搭乘月曆表' }).click()
    await page.waitForURL('**/rides', { timeout: 10000 })
    await expect(page).toHaveURL(/.*\/rides$/)

    // 返回首頁測試異常集中處理
    await page.goto('/')
    await page.waitForLoadState('networkidle')
    await page.getByRole('button', { name: '異常集中處理' }).click()
    await page.waitForURL('**/rides/issues', { timeout: 10000 })
    await expect(page).toHaveURL(/.*\/rides\/issues/)
  })

  test('最近申報匯出紀錄表格顯示正常', async ({ page }) => {
    await waitForTableLoaded(page)
    await expect(page.getByText('最近申報匯出紀錄')).toBeVisible()
    await expect(page.locator('.recent-exports-card .el-table').first()).toBeVisible()
  })
})

