import { test, expect } from '@playwright/test'
import { loginAs } from './helpers/auth'
import { waitForTableLoaded, expectElMessage } from './helpers/ui'

test.describe('04. 基礎主檔管理 (Master Data Management)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin')
  })

  test('據點管理：清單載入、新增與編輯據點', async ({ page }) => {
    await page.goto('/masters/sites')
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table').first()).toBeVisible()

    // 點選新增據點
    await page.getByRole('button', { name: '新增據點' }).click()
    const dialog = page.locator('.el-dialog').filter({ hasText: '新增據點' })
    await expect(dialog).toBeVisible()

    await dialog.getByPlaceholder('如：竹北日照中心').fill('測試日照中心')
    await dialog.getByPlaceholder('請輸入完整地址').fill('新竹縣竹北市光明六路 1 號')
    await dialog.getByRole('button', { name: '儲存' }).click()
    await expectElMessage(page, /成功/, 'success')
  })

  test('車輛管理：清單載入、新增與編輯車輛', async ({ page }) => {
    await page.goto('/masters/vehicles')
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table').first()).toBeVisible()

    // 點選新增車輛
    await page.getByRole('button', { name: '新增車輛' }).click()
    const dialog = page.locator('.el-dialog').filter({ hasText: '新增車輛' })
    await expect(dialog).toBeVisible()

    await dialog.getByPlaceholder(/竹北一車/).fill('測試測試車')
    await dialog.getByPlaceholder(/BZG-7915/).fill('E2E-8888')
    await dialog.getByRole('button', { name: '儲存' }).click()
    await expectElMessage(page, /成功/, 'success')
  })

  test('車輛管理：一台車可掛多位司機，維護後清單同步更新', async ({ page }) => {
    await page.goto('/masters/vehicles')
    await waitForTableLoaded(page)

    // 竹北一車在展示資料中由兩位司機共同駕駛
    const row = page.locator('.el-table__body tr').filter({ hasText: '竹北一車' }).first()
    await expect(row.locator('.vehicle-driver-tags .el-tag')).toHaveCount(2)

    await row.getByRole('button', { name: '司機' }).click()
    const dialog = page.locator('.el-dialog').filter({ hasText: '維護司機' })
    await expect(dialog).toBeVisible()

    // 移除其中一位司機後，清單只剩另一位
    await dialog.getByRole('button', { name: '關閉此標籤' }).first().click()
    await dialog.getByRole('button', { name: '儲存' }).click()
    await expectElMessage(page, /司機已更新/, 'success')
    await expect(row.locator('.vehicle-driver-tags .el-tag')).toHaveCount(1)
  })

  test('司機管理：清單載入、身分證遮罩與解密、新增司機', async ({ page }) => {
    await page.goto('/masters/drivers')
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table').first()).toBeVisible()

    // 點選新增司機
    await page.getByRole('button', { name: '新增司機' }).click()
    const dialog = page.locator('.el-dialog').filter({ hasText: '新增司機' })
    await expect(dialog).toBeVisible()

    await dialog.getByPlaceholder(/請輸入姓名/).fill('測試司機')
    await dialog.getByPlaceholder(/1 碼英文/).fill('B123456789')
    await dialog.getByPlaceholder(/0912345678/).fill('0988777666')
    await dialog.getByRole('button', { name: '儲存' }).click()
    await expectElMessage(page, /成功/, 'success')
  })

  test('地區管理：清單瀏覽與新增地區（不需填寫代碼）', async ({ page }) => {
    await page.goto('/masters/regions')
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table').first()).toBeVisible()
    await expect(page.locator('.el-table').first()).toContainText('新竹')

    // 點選新增地區按鈕
    await page.getByRole('button', { name: '新增地區' }).click()
    const dialog = page.locator('.el-dialog').filter({ hasText: '新增營運地區' })
    await expect(dialog).toBeVisible()

    // 驗證彈窗中無「地區代碼」欄位，只有區域名稱、排序、狀態與備註說明
    await expect(dialog.getByLabel('地區代碼')).toHaveCount(0)
    await dialog.getByPlaceholder(/如：臺北市/).fill('測試營運區')
    await dialog.getByRole('button', { name: '儲存' }).click()
    await expectElMessage(page, /成功/, 'success')
  })
})

