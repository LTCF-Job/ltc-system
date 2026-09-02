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

    // 混車衝突分頁預設載入，展示資料應至少有一列，且「涉及車輛」欄要能看到
    // 兩個以上的車名（用頓號分隔），這是混車衝突本身的定義：同一趟次有多台車回報
    const conflictRows = page.locator('#pane-conflict .el-table__row')
    await expect(conflictRows.first()).toBeVisible()
    await expect(page.locator('.vehicle-separator').first()).toBeVisible()

    // 切換至未回報分頁
    await page.getByRole('tab', { name: /未回報/ }).click()
    await waitForTableLoaded(page)
    const unreportedTable = page.locator('#pane-unreported .el-table')
    await expect(unreportedTable).toBeVisible()
    await expect(page.locator('#pane-unreported .el-table__row').first()).toBeVisible()
    await expect(page.locator('#pane-unreported .el-table__row').first()).toContainText('未')

    // 切換至匯入異常分頁
    await page.getByRole('tab', { name: /匯入異常/ }).click()
    await waitForTableLoaded(page)
    await expect(page.locator('#pane-import_error .el-table')).toBeVisible()
    await expect(page.locator('#pane-import_error .el-table__row').first()).toBeVisible()
  })

  test('混車衝突人工裁決流程：送出後該列從清單消失', async ({ page }) => {
    await page.goto('/rides/issues')
    await waitForTableLoaded(page)

    const rows = page.locator('#pane-conflict .el-table__row')
    const initialCount = await rows.count()
    expect(initialCount).toBeGreaterThan(0)

    const resolveBtn = rows.first().locator('button').filter({ hasText: '人工裁決' })
    await expect(resolveBtn).toBeVisible()
    await resolveBtn.click()

    const dialog = page.locator('.el-dialog').filter({ hasText: '混車衝突裁決' })
    await expect(dialog).toBeVisible()

    // 裁決車輛與司機下拉已預設帶入第一筆選項，直接送出即可驗證流程
    await dialog.getByRole('button', { name: '確認送出' }).click()
    await confirmMessageBox(page)
    await expectElMessage(page, /已裁決/, 'success')

    await waitForTableLoaded(page)
    await expect(rows).toHaveCount(initialCount - 1)
  })

  test('表單匯入異常：查看詳情彈窗顯示非空的原始 Payload', async ({ page }) => {
    await page.goto('/rides/issues')
    await waitForTableLoaded(page)
    await page.getByRole('tab', { name: /匯入異常/ }).click()
    await waitForTableLoaded(page)

    const viewBtn = page.locator('#pane-import_error .el-table__row').first().getByRole('button', { name: '查看' })
    await expect(viewBtn).toBeVisible()
    await viewBtn.click()

    const dialog = page.locator('.el-dialog').filter({ hasText: '表單匯入異常詳情' })
    await expect(dialog).toBeVisible()
    const payload = dialog.locator('.raw-payload')
    await expect(payload).toBeVisible()
    await expect(payload).not.toHaveText('（無原始 Payload 紀錄）')

    await dialog.getByRole('button', { name: '關閉', exact: true }).click()
  })

  test('異常集中處理搜尋：依個案姓名篩選混車衝突清單', async ({ page }) => {
    await page.goto('/rides/issues')
    await waitForTableLoaded(page)

    const rows = page.locator('#pane-conflict .el-table__row')
    const firstCaseName = await rows.first().locator('.case-name-col').innerText()

    await page.getByPlaceholder('搜尋個案姓名／涉及車輛／說明').fill(firstCaseName)
    await page.getByRole('button', { name: '查詢' }).click()
    await waitForTableLoaded(page)

    const filteredCount = await rows.count()
    expect(filteredCount).toBeGreaterThan(0)
    for (let i = 0; i < filteredCount; i++) {
      await expect(rows.nth(i)).toContainText(firstCaseName)
    }
  })

  test('未回報搭乘清單與催報功能', async ({ page }) => {
    await page.goto('/rides/missing')
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table').first()).toBeVisible()
  })
})
