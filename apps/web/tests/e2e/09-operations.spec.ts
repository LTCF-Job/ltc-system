import { test, expect } from '@playwright/test'
import { loginAs } from './helpers/auth'
import { waitForTableLoaded, expectElMessage } from './helpers/ui'

test.describe('09. 車輛保養與出勤油資 (Operations: Maintenance & Attendance)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin')
  })

  test('車輛保養：清單載入、新增保養紀錄與範本下載', async ({ page }) => {
    await page.goto('/vehicles/maintenance')
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table').first()).toBeVisible()

    // 點選新增保養紀錄
    await page.getByRole('button', { name: '新增保養紀錄' }).click()
    const dialog = page.locator('.el-dialog').filter({ hasText: '新增保養紀錄' })
    await expect(dialog).toBeVisible()

    // 填寫表單
    const vehicleSelect = dialog.locator('.el-select').first()
    await vehicleSelect.click()
    await page.locator('.el-select-dropdown:visible .el-select-dropdown__item').first().click()

    await dialog.getByPlaceholder('例如：更換機油、煞車皮、檢查五油三水').fill('定期更換機油與機油濾芯')
    await dialog.getByPlaceholder('例如：順益汽車、原廠保修站').fill('竹北原廠保修站')
    const costInput = dialog.locator('.el-input-number input').last()
    await costInput.fill('2500')

    await dialog.getByRole('button', { name: '確定儲存' }).click()
    await expectElMessage(page, /成功/, 'success')
  })

  test('出勤與油資登錄：出勤狀態與油資列表檢視', async ({ page }) => {
    await page.goto('/attendance')
    await waitForTableLoaded(page)
    await expect(page.locator('.filter-card, .table-card, .attendance-view, .el-table').first()).toBeVisible()

    // 驗證出勤彙總指標標籤包含 O / 假別文字與國定假日
    await expect(page.locator('.attendance-summary-pills')).toContainText('出勤 (O)')
    await expect(page.locator('.attendance-summary-pills')).toContainText('事假 (事)')
    await expect(page.locator('.attendance-summary-pills')).toContainText('國定假日')

    // 驗證出勤矩陣表格包含 O 與 假別文字
    const matrix = page.locator('.attendance-matrix')
    await expect(matrix).toBeVisible()
    await expect(matrix.locator('.symbol-work').first()).toHaveText('O')
    await expect(matrix.locator('.symbol-leave').first()).toHaveText('事')

    // 點選儲存格開啟登記 Dialog 驗證狀態選項
    const firstCell = matrix.locator('.day-cell').first()
    await firstCell.click()
    const dialog = page.locator('.el-dialog').filter({ hasText: '登記司機出勤狀態' })
    await expect(dialog).toBeVisible()
    await expect(dialog.getByRole('radio', { name: /出勤 \(O\)/ })).toBeVisible()
    await expect(dialog.getByRole('radio', { name: /事假 \(事\)/ })).toBeVisible()
    await dialog.getByRole('button', { name: '取消' }).click()
  })
})

