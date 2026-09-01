import { test, expect } from '@playwright/test'
import { loginAs } from './helpers/auth'
import { waitForTableLoaded, expectElMessage } from './helpers/ui'

test.describe('04. 基礎主檔管理 (Master Data Management)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin')
  })

  test('單位管理：清單載入、新增與編輯單位', async ({ page }) => {
    await page.goto('/masters/sites')
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table').first()).toBeVisible()

    // 點選新增單位
    await page.getByRole('button', { name: '新增單位' }).click()
    const dialog = page.locator('.el-dialog').filter({ hasText: '新增單位' })
    await expect(dialog).toBeVisible()

    await dialog.getByPlaceholder('如：竹北日照中心').fill('測試日照中心')
    await dialog.getByPlaceholder('請輸入完整地址').fill('新竹縣竹北市光明六路 1 號')
    await dialog.getByRole('button', { name: '儲存' }).click()
    await expectElMessage(page, /成功/, 'success')
  })

  test('車輛管理：清單顯示車籍與保險欄位，並可新增車輛', async ({ page }) => {
    await page.goto('/masters/vehicles')
    await waitForTableLoaded(page)
    const table = page.locator('.el-table').first()
    await expect(table).toBeVisible()

    // 服務車輛清冊欄位需直接看得到，保險與檢驗日期以民國年呈現
    await expect(table).toContainText('竹北日照中心')
    await expect(table).toContainText('DE241LB8')
    await expect(table).toContainText('2013 年 03 月')
    await expect(table).toContainText('114/12/23')

    await page.getByRole('button', { name: '新增車輛' }).click()
    const dialog = page.locator('.el-dialog').filter({ hasText: '新增車輛' })
    await expect(dialog).toBeVisible()

    await fillVehicleForm(page, dialog, { plateNo: 'E2E-8888', displayName: '測試測試車' })
    await dialog.getByRole('button', { name: '儲存' }).click()
    await expectElMessage(page, /成功/, 'success')
  })

  test('車輛管理：新增車輛缺必填欄位時擋下並提示', async ({ page }) => {
    await page.goto('/masters/vehicles')
    await waitForTableLoaded(page)

    await page.getByRole('button', { name: '新增車輛' }).click()
    const dialog = page.locator('.el-dialog').filter({ hasText: '新增車輛' })
    await dialog.getByPlaceholder(/BZG-7915/).fill('E2E-7777')
    await dialog.getByRole('button', { name: '儲存' }).click()

    await expect(dialog.locator('.el-form-item__error').first()).toBeVisible()
    await expect(dialog).toBeVisible()
  })

  test('車輛管理：可依所屬單位篩選清單', async ({ page }) => {
    await page.goto('/masters/vehicles')
    await waitForTableLoaded(page)

    await page.locator('.el-select').filter({ hasText: '全部單位' }).first().click()
    await page.locator('.el-select-dropdown:visible .el-select-dropdown__item')
      .filter({ hasText: '竹北日照中心' })
      .first()
      .click()
    await waitForTableLoaded(page)

    const table = page.locator('.el-table').first()
    await expect(table).toContainText('竹北一車')
    await expect(table).not.toContainText('竹南1車')
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

    // 駕照類別與有效日期為政府「服務駕駛清冊」必填，清單需直接看得到
    await expect(page.locator('.el-table').first()).toContainText('職業小型車')
    await expect(page.locator('.el-table').first()).toContainText('2031-04-22')

    // 點選新增司機
    await page.getByRole('button', { name: '新增司機' }).click()
    const dialog = page.locator('.el-dialog').filter({ hasText: '新增司機' })
    await expect(dialog).toBeVisible()

    await dialog.getByPlaceholder(/請輸入姓名/).fill('測試司機')
    await dialog.getByPlaceholder(/1 碼英文/).fill('B123456789')
    await dialog.getByPlaceholder(/0912345678/).fill('0988777666')
    await dialog.locator('.el-select').first().click()
    await page.locator('.el-select-dropdown:visible .el-select-dropdown__item')
      .filter({ hasText: '職業大客車' })
      .first()
      .click()
    const expiryInput = dialog.locator('.el-date-editor input')
    await expiryInput.fill('2030-12-31')
    await expiryInput.press('Enter')
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

// 車輛表單所有欄位皆為必填，測試只在意其中一兩個值，其餘欄位統一由這裡補齊
async function fillVehicleForm(
  page: import('@playwright/test').Page,
  dialog: import('@playwright/test').Locator,
  values: { plateNo: string; displayName: string }
) {
  await dialog.getByPlaceholder(/BZG-7915/).fill(values.plateNo)
  await dialog.getByPlaceholder(/竹北一車/).fill(values.displayName)
  await dialog.locator('.el-select').first().click()
  await page.locator('.el-select-dropdown:visible .el-select-dropdown__item').first().click()
  await dialog.getByPlaceholder('如：中華').fill('中華')
  await dialog.getByPlaceholder('如：DE241L8').fill('DE241L8')

  const monthInput = dialog.getByPlaceholder('請選擇年月')
  await monthInput.fill('2013-03')
  await monthInput.press('Enter')

  const dateInputs = dialog.getByPlaceholder('請選擇日期')
  for (let i = 0; i < (await dateInputs.count()); i++) {
    const input = dateInputs.nth(i)
    await input.fill('2026-12-23')
    await input.press('Enter')
  }
}
