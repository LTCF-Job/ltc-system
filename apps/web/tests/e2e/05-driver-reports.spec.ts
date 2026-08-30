import { test, expect, type Page } from '@playwright/test'
import * as XLSX from 'xlsx'
import { loginAs } from './helpers/auth'
import { waitForTableLoaded, expectElMessage } from './helpers/ui'

// 依實際匯報檔格式組出上傳檔：民國日期、駕駛人、各個案趟次欄、備註
function buildReportWorkbook(rows: string[][]): Buffer {
  const sheet = XLSX.utils.aoa_to_sheet(rows)
  const workbook = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(workbook, sheet, '司機接送匯報')
  return XLSX.write(workbook, { bookType: 'xlsx', type: 'buffer' }) as Buffer
}

async function uploadReport(page: Page, rows: string[][]) {
  await page.locator('.el-dialog input[type="file"]').setInputFiles({
    name: 'driver-report.xlsx',
    mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    buffer: buildReportWorkbook(rows)
  })
  await page.getByRole('button', { name: '開始解析與預覽' }).click()
}

const HEADER = ['民國日期', '駕駛人', '1.張詹竹妹 [去程]', '1.吳𣵛桂(去程竹3) [去程]', '備註']

test.describe('05. 司機接送匯報匯入與欄位對應 (Driver Report Import & Field Mapping)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin')
  })

  test('匯報表清單以車輛為主體，顯示欄位對應進度與最後匯入時間', async ({ page }) => {
    await page.goto('/driver-reports')
    await waitForTableLoaded(page)

    await expect(page.locator('.el-table').first()).toBeVisible()
    await expect(page.getByText('司機接送匯報（共 4 台車）')).toBeVisible()
    await expect(page.locator('.el-table').first().getByText('竹南2車').first()).toBeVisible()
    await expect(page.getByRole('button', { name: '新增車輛匯報表' })).toBeVisible()
  })

  test('上傳匯報檔後就地確認欄位對應，寫入搭乘紀錄', async ({ page }) => {
    await page.goto('/driver-reports')
    await waitForTableLoaded(page)

    const firstRow = page.locator('.el-table__row').first()
    await firstRow.getByRole('button', { name: '匯入' }).click()

    const dialog = page.locator('.el-dialog').filter({ hasText: '匯入司機接送匯報' })
    await expect(dialog).toBeVisible()
    await expect(dialog.getByRole('button', { name: /下載該車匯報範本/ })).toBeVisible()

    await uploadReport(page, [
      HEADER,
      ['1150302', '郭澤威', '有坐', '沒坐', '無'],
      ['115/3/3', '郭澤威', '沒坐', '有坐', ''],
      ['壞掉的日期', '郭澤威', '有坐', '有坐', '日期打錯']
    ])

    // 預覽統計：兩天可匯入、一天日期錯誤
    await expect(dialog.getByText('匯報天數：3')).toBeVisible()
    await expect(dialog.getByText('可匯入：2')).toBeVisible()
    await expect(dialog.getByText('日期錯誤：1')).toBeVisible()

    // 欄位對應段：張詹竹妹該欄尚未對應，系統已帶出推薦個案
    await expect(dialog.getByText('1.張詹竹妹 [去程]')).toBeVisible()
    await expect(dialog.getByText(/待對應欄位：1 \/ 2/)).toBeVisible()

    // 每日匯報段可檢視解析後的服務日期與檢核訊息
    await dialog.getByRole('tab', { name: /每日匯報/ }).click()
    await expect(dialog.getByText('2026-03-02')).toBeVisible()
    await expect(dialog.getByText(/日期格式無法解析/)).toBeVisible()

    await dialog.getByRole('button', { name: /^匯入 \(2 天\)$/ }).click()
    await expectElMessage(page, /已匯入 2 天的接送匯報/, 'success')
    await expect(dialog.getByText(/產生 \d+ 筆搭乘紀錄/)).toBeVisible()
  })

  test('表頭不符時明確擋下並說明應有的欄位順序', async ({ page }) => {
    await page.goto('/driver-reports')
    await waitForTableLoaded(page)

    await page.locator('.el-table__row').first().getByRole('button', { name: '匯入' }).click()
    const dialog = page.locator('.el-dialog').filter({ hasText: '匯入司機接送匯報' })
    await expect(dialog).toBeVisible()

    await uploadReport(page, [
      ['時間戳記', '今天日期', '今日駕駛人', '問題回報'],
      ['46084', '1150302', '郭澤威', '']
    ])

    // 失敗原因以通知條列呈現，操作人員看得到該修哪一欄
    const notification = page.locator('.el-notification')
    await expect(notification).toBeVisible({ timeout: 10000 })
    await expect(notification).toContainText('匯入司機接送匯報失敗')
    await expect(notification).toContainText('找不到匯報表表頭')

    // 解析失敗時停在上傳步驟，不會誤進預覽
    await expect(dialog.getByRole('button', { name: '開始解析與預覽' })).toBeVisible()
  })

  test('欄位對應設定：檢視推薦信心度、單筆綁定與略過欄位', async ({ page }) => {
    await page.goto('/driver-reports/mappings')
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table').first()).toBeVisible()

    await expect(page.getByText('原始表單欄位名稱 (左側)')).toBeVisible()
    await expect(page.getByText('目標個案與排班時段 (右側)')).toBeVisible()

    const ignoreBtn = page.locator('.el-table__row').locator('button').filter({ hasText: '略過此欄' }).first()
    if (await ignoreBtn.isVisible()) {
      await ignoreBtn.click()
      await expectElMessage(page, /略過/, 'info')
    }
  })
})
