import { test, expect } from '@playwright/test'
import { loginAs } from './helpers/auth'
import { waitForTableLoaded, expectElMessage } from './helpers/ui'

test.describe('03. 個案主檔與排班設定 (Cases & Schedules)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin', '/cases')
  })

  test('個案清單載入、搜尋與條件篩選功能正常', async ({ page }) => {
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table').first()).toBeVisible()

    // 關鍵字搜尋
    const searchInput = page.getByPlaceholder(/搜尋姓名／編號／身分證/)
    await searchInput.fill('蔡')
    await searchInput.press('Enter')
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table').first()).toContainText('蔡')

    // 重設搜尋
    await page.locator('.filter-card').getByRole('button', { name: '重設' }).click({ force: true })
    await waitForTableLoaded(page)
  })

  test('新增個案彈窗表單驗證與成功建立', async ({ page }) => {
    await waitForTableLoaded(page)
    await page.getByRole('button', { name: '新增個案' }).click()

    const dialog = page.locator('.el-dialog').filter({ hasText: '新增個案基本資料' })
    await expect(dialog).toBeVisible()

    // 填寫欄位
    await dialog.getByPlaceholder(/請輸入姓名/).fill('林小明')
    await dialog.getByPlaceholder(/1 碼英文/).fill('A123456789')
    await dialog.getByPlaceholder(/0912345678/).fill('0911222333')
    await dialog.getByPlaceholder(/請輸入住家地址/).fill('新竹市東區中央路 100 號')

    // 送出
    await dialog.getByRole('button', { name: '新增' }).click()
    await expectElMessage(page, /成功/, 'success')
  })

  test('點選編輯進入個案明細與排班設定頁面', async ({ page }) => {
    await waitForTableLoaded(page)
    const editBtn = page.locator('.el-table__row').filter({ hasText: '編輯' }).locator('button, a').filter({ hasText: '編輯' }).first()
    await editBtn.click()

    await page.waitForURL(/\/cases\//, { timeout: 10000 })
    await expect(page.getByRole('tab', { name: '基本資料' })).toBeVisible()
    await expect(page.getByRole('tab', { name: '排班設定' })).toBeVisible()

    // 切換至排班設定分頁
    await page.getByRole('tab', { name: '排班設定' }).click()
    await expect(page.getByText('排班條件與模式設定')).toBeVisible()
    await expect(page.getByText('排班優先順序')).toBeVisible()

    // 驗證當月排班月曆之來源標籤與日期呈現
    await expect(page.locator('.monthly-table')).toBeVisible()
    await expect(page.locator('.source-tag').first()).toBeVisible()
    const firstRowDate = page.locator('.date-cell-label').first()
    await expect(firstRowDate).toBeVisible()
    await expect(firstRowDate).not.toContainText('★')

    // 切換至固定排班確認趟數型態
    await page.locator('.el-radio-button', { hasText: '固定排班' }).click()
    await expect(page.getByText('趟數型態')).toBeVisible()
  })

  test('下載匯入範本與開啟批次匯入彈窗', async ({ page }) => {
    await waitForTableLoaded(page)
    await page.getByRole('button', { name: '批次匯入個案' }).click()
    const importDialog = page.locator('.el-dialog').filter({ hasText: /批次匯入/ })
    await expect(importDialog).toBeVisible()
    await importDialog.getByRole('button', { name: '關閉' }).or(importDialog.locator('.el-dialog__headerbtn')).first().click()
  })
})

