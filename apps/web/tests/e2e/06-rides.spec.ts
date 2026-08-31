import { test, expect } from '@playwright/test'
import { loginAs } from './helpers/auth'
import { waitForTableLoaded, expectElMessage, confirmMessageBox } from './helpers/ui'

test.describe('06. 搭乘月曆表與紀錄更正 (Ride Calendar & Correction)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin', '/rides')
  })

  test('搭乘月曆表渲染、圖例與日期欄位顯示', async ({ page }) => {
    await waitForTableLoaded(page)
    await expect(page.getByText('搭乘圖例：')).toBeVisible()
    await expect(page.getByText('有坐 (Boarded)')).toBeVisible()
    await expect(page.getByText('沒坐 (Absent)')).toBeVisible()
    await expect(page.getByText('未回報 (Unreported)')).toBeVisible()
    await expect(page.getByText('混車衝突 (Conflict)')).toBeVisible()

    // 檢查表格與個案姓名欄
    await expect(page.locator('.calendar-table').first()).toBeVisible()
    await expect(page.locator('.calendar-table').first().getByText('個案姓名')).toBeVisible()
    await expect(page.locator('.calendar-table').first().getByText('趟數')).toBeVisible()

    // 檢查自訂與各趟次標籤渲染
    await expect(page.locator('.custom-trip-tag').first()).toBeVisible()
    await expect(page.locator('.custom-trip-tag').first()).toHaveText('自訂')
  })

  test('點選查詢月份左右方向按鈕可快速切換上下月份', async ({ page }) => {
    await waitForTableLoaded(page)
    const monthInput = page.locator('.month-picker-wrapper input')
    await expect(monthInput).toHaveValue('2026-07')

    // 點選上一月
    const prevBtn = page.getByRole('button', { name: '上一月' })
    await expect(prevBtn).toBeVisible()
    await prevBtn.click()
    await expect(monthInput).toHaveValue('2026-06')

    // 點選下一月
    const nextBtn = page.getByRole('button', { name: '下一月' })
    await expect(nextBtn).toBeVisible()
    await nextBtn.click()
    await expect(monthInput).toHaveValue('2026-07')
  })

  test('點選月曆搭乘格子開啟更正抽屜面板並執行更正', async ({ page }) => {
    await waitForTableLoaded(page)
    // 點選有紀錄之搭乘格子開啟更正抽屜
    const cell = page.locator('.calendar-cell:not(.status-non-scheduled)').first()
    await expect(cell).toBeVisible()
    await cell.click()

    // 檢查抽屜面板開啟
    const drawer = page.locator('.el-drawer').filter({ hasText: '搭乘紀錄更正' })
    await expect(drawer).toBeVisible()
    await expect(drawer.getByText('搭乘紀錄更正欄位')).toBeVisible()

    // 填寫更正原因
    const reasonInput = drawer.getByPlaceholder(/請輸入更正原因/).or(drawer.locator('input[type="text"]').last())
    if (await reasonInput.isVisible()) {
      await reasonInput.fill('測試 E2E 人工更正')
    }

    // 儲存
    const saveBtn = drawer.getByRole('button', { name: /儲存/ })
    if (await saveBtn.isVisible()) {
      await saveBtn.click()
      await confirmMessageBox(page)
      await expectElMessage(page, /成功|更新/, 'success')
    }
  })
})

