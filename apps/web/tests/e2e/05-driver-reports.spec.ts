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

const HEADER = ['民國日期', '駕駛人', '1.張詹竹妹 [去程]', '1.吳𣵛桂(去程竹3) [去程]', '備註']
// 「王小明」不存在於任何個案主檔，用來製造完全比對不到個案、必須進待維護的欄位；
// 另外保留一欄比對得到的「張詹竹妹」，確保至少有一欄可對應，檔案本身仍能成功匯入
const HEADER_UNMATCHED = ['民國日期', '駕駛人', '1.張詹竹妹 [去程]', '1.王小明 [去程]', '備註']

test.describe('05. 司機接送匯報總覽 (Driver Report Status Overview)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin')
  })

  test('接送匯報總覽：只顯示各車已有資料的月份，不提供上傳功能', async ({ page }) => {
    await page.goto('/driver-reports/status')
    await waitForTableLoaded(page)

    await expect(page.getByRole('heading', { name: '接送匯報總覽' })).toBeVisible()
    await expect(page.locator('.el-table').first()).toBeVisible()
    await expect(page.getByRole('button', { name: '批次上傳' })).toHaveCount(0)
    await expect(page.getByRole('columnheader', { name: '車輛' })).toBeVisible()
    await expect(page.getByRole('columnheader', { name: '匯報表名稱' })).toHaveCount(0)
  })

  test('/driver-reports 重定向至 /driver-reports/status；批次上傳對應舊路徑重定向至 /driver-reports/import', async ({ page }) => {
    await page.goto('/driver-reports')
    await waitForTableLoaded(page)
    await expect(page).toHaveURL(/.*\/driver-reports\/status/)

    await page.goto('/driver-reports/batch-import')
    await waitForTableLoaded(page)
    await expect(page).toHaveURL(/.*\/driver-reports\/import/)

    await page.goto('/driver-reports/mappings')
    await waitForTableLoaded(page)
    await expect(page).toHaveURL(/.*\/driver-reports\/import/)
  })
})

// 批次上傳直接常駐在「批次上傳」頁籤（左側上傳與送出、右側逐檔卡片），待維護資料則是同頁另一個頁籤：
// 解析後立即匯入，完全比對不到個案的欄位進入待維護頁籤，稍後連結既有個案或建立新個案。
test.describe('05b. 司機接送匯報批次上傳 (Driver Report Batch Import)', () => {
  function rowOf(page: Page, fileName: string) {
    return page.locator('.file-card').filter({ hasText: fileName }).first()
  }

  async function uploadFiles(page: Page, files: Array<{ name: string; rows: string[][] }>) {
    await page.locator('.drop-zone input[type="file"]').setInputFiles(
      files.map((f) => ({
        name: f.name,
        mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        buffer: buildReportWorkbook(f.rows)
      }))
    )
  }

  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin')
    await page.goto('/driver-reports/import')
    await waitForTableLoaded(page)
  })

  test('B1 檔名含車輛名稱時自動判斷車輛並自動解析出涵蓋月份，確認後直接匯入', async ({ page }) => {
    await uploadFiles(page, [
      {
        name: '竹北一車.xlsx',
        rows: [
          HEADER,
          ['1150302', '郭澤威', '有坐', '沒坐', '無'],
          ['115/3/3', '郭澤威', '沒坐', '有坐', '']
        ]
      }
    ])
    const row = rowOf(page, '竹北一車.xlsx')
    await expect(row).toContainText('竹北一車')
    await expect(row).toContainText('2026-03', { timeout: 10000 })

    await page.getByRole('button', { name: /開始解析與匯入/ }).click()
    await expect(row).toContainText('已匯入', { timeout: 10000 })
    await expect(row).toContainText('可匯入 2 天')
    await expect(page.locator('.result-banner')).toContainText('成功 1 個檔案、共 2 天')
  })

  test('B2 對沒有匯報表的車上傳，自動建表後成功匯入', async ({ page }) => {
    await uploadFiles(page, [
      { name: '苗栗市1車 (回覆).xlsx', rows: [HEADER, ['1150302', '郭澤威', '有坐', '沒坐', '無']] }
    ])
    await page.getByRole('button', { name: /開始解析與匯入/ }).click()
    await expect(rowOf(page, '苗栗市1車')).toContainText('已匯入', { timeout: 10000 })
    await expect(rowOf(page, '苗栗市1車')).toContainText('可匯入 1 天')
  })

  test('B3 單一檔案橫跨兩個月份時，各月份各自整月匯入', async ({ page }) => {
    await uploadFiles(page, [
      {
        name: '竹北一車.xlsx',
        rows: [
          HEADER,
          ['1150302', '郭澤威', '有坐', '沒坐', '無'],
          ['1150402', '郭澤威', '有坐', '沒坐', '無']
        ]
      }
    ])
    const row = rowOf(page, '竹北一車.xlsx')
    await expect(row).toContainText('2026-03', { timeout: 10000 })
    await expect(row).toContainText('2026-04')

    await page.getByRole('button', { name: /開始解析與匯入/ }).click()
    await expect(row).toContainText('可匯入 2 天', { timeout: 10000 })
  })

  test('B4 檔名比對不到車輛時需手動選擇，選定後自動解析並可開始匯入', async ({ page }) => {
    await uploadFiles(page, [
      { name: 'driver-report.xlsx', rows: [HEADER, ['1150302', '郭澤威', '有坐', '沒坐', '無']] }
    ])
    const row = rowOf(page, 'driver-report.xlsx')
    await expect(row).toContainText('待選車輛')
    await expect(page.getByRole('button', { name: /開始解析與匯入/ })).toBeDisabled()

    await row.locator('.el-select').click()
    await page.locator('.el-select-dropdown__item').filter({ hasText: '竹南1車' }).click()
    await expect(row).toContainText('2026-03', { timeout: 10000 })
    await expect(page.getByRole('button', { name: /開始解析與匯入/ })).toBeEnabled()

    await page.getByRole('button', { name: /開始解析與匯入/ }).click()
    await expect(row).toContainText('已匯入', { timeout: 10000 })
  })

  test('B5 對已匯入過的月份重傳時，就地顯示覆蓋警示並需勾選確認才能送出', async ({ page }) => {
    const file = [
      HEADER,
      ['1150302', '郭澤威', '有坐', '沒坐', '無'],
      ['115/3/3', '郭澤威', '沒坐', '有坐', '']
    ]

    await uploadFiles(page, [{ name: '竹北一車.xlsx', rows: file }])
    await page.getByRole('button', { name: /開始解析與匯入/ }).click()
    await expect(rowOf(page, '竹北一車.xlsx')).toContainText('已匯入', { timeout: 10000 })

    // 重傳同一份檔案（換個檔名避免被去重擋下）：涵蓋月份標籤變成 warning，就地跳出覆蓋警示
    await uploadFiles(page, [{ name: '竹北一車再傳.xlsx', rows: file }])
    const row = rowOf(page, '竹北一車再傳.xlsx')
    await expect(row).toContainText('2026-03', { timeout: 10000 })

    const overlapAlert = page.locator('.overlap-alert')
    await expect(overlapAlert).toBeVisible()
    await expect(overlapAlert).toContainText('竹北一車')
    await expect(page.getByRole('button', { name: /開始解析與匯入/ })).toBeDisabled()

    await overlapAlert.locator('.el-checkbox').click()
    await expect(page.getByRole('button', { name: /開始解析與匯入/ })).toBeEnabled()
    await page.getByRole('button', { name: /開始解析與匯入/ }).click()

    await expect(row).toContainText('已匯入', { timeout: 10000 })

    await page.goto('/driver-reports/status')
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table__row').filter({ hasText: '竹北一車' })).toContainText('2026-03（2天）')
  })

  test('B6 完全比對不到個案的欄位進入待維護，匯入完成後提示是否前往處理', async ({ page }) => {
    await uploadFiles(page, [
      { name: '竹北二車.xlsx', rows: [HEADER_UNMATCHED, ['1150302', '郭澤威', '有坐', '沒坐', '無']] }
    ])
    await page.getByRole('button', { name: /開始解析與匯入/ }).click()
    await expect(rowOf(page, '竹北二車.xlsx')).toContainText('已匯入', { timeout: 10000 })
    await expect(rowOf(page, '竹北二車.xlsx')).toContainText('1 欄待維護')
    await expect(page.locator('.result-banner')).toContainText('個欄位找不到對應個案，已進入待維護資料')

    const confirmBox = page.locator('.el-message-box')
    await expect(confirmBox).toBeVisible()
    await expect(confirmBox).toContainText('是否立即前往處理')
    await confirmBox.getByRole('button', { name: '前往待維護' }).click()

    await expect(page.getByRole('tab', { name: /待維護資料/ })).toHaveAttribute('aria-selected', 'true')
    await expect(page.locator('.el-table__row').filter({ hasText: '王小明' })).toBeVisible()
  })

  test('B7 一個檔案找不到匯報表表頭時單獨失敗，不影響其他檔案', async ({ page }) => {
    const validFile = [HEADER, ['1150302', '郭澤威', '有坐', '沒坐', '無']]
    await uploadFiles(page, [
      { name: '竹北一車.xlsx', rows: validFile },
      { name: '竹南1車.xlsx', rows: [['車牌', '備註']] }
    ])
    await page.getByRole('button', { name: /開始解析與匯入/ }).click()

    await expect(rowOf(page, '竹北一車.xlsx')).toContainText('已匯入', { timeout: 10000 })
    await expect(rowOf(page, '竹南1車.xlsx')).toContainText('失敗')
    await expect(rowOf(page, '竹南1車.xlsx')).toContainText('找不到匯報表表頭')
  })

  test('B8 移除已加入的檔案列', async ({ page }) => {
    await uploadFiles(page, [
      { name: '竹北一車.xlsx', rows: [HEADER, ['1150302', '郭澤威', '有坐', '沒坐', '無']] }
    ])
    const row = rowOf(page, '竹北一車.xlsx')
    await expect(row).toBeVisible()
    await row.getByRole('button', { name: /移除/ }).click()
    await expect(row).toHaveCount(0)
  })
})

// 待維護資料頁籤：連結既有個案或建立新個案，建立時預帶匯報表原始欄位解析出的姓名
test.describe('05c. 司機接送匯報待維護資料 (Driver Report Pending Mappings)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin')
  })

  test('C1 新增個案並綁定：預帶原始欄位姓名，完成後從待維護清單移除', async ({ page }) => {
    // 先透過批次上傳製造一筆完全比對不到個案的待維護欄位
    await page.goto('/driver-reports/import')
    await waitForTableLoaded(page)
    await page.locator('.drop-zone input[type="file"]').setInputFiles({
      name: '竹北二車.xlsx',
      mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      buffer: buildReportWorkbook([HEADER_UNMATCHED, ['1150302', '郭澤威', '有坐', '沒坐', '無']])
    })
    await page.getByRole('button', { name: /開始解析與匯入/ }).click()
    await expect(page.locator('.result-banner')).toContainText('1 個欄位找不到對應個案', { timeout: 10000 })

    const confirmBox = page.locator('.el-message-box')
    await confirmBox.getByRole('button', { name: '前往待維護' }).click()

    const pendingRow = page.locator('.el-table__row').filter({ hasText: '王小明' })
    await expect(pendingRow).toBeVisible()

    await pendingRow.getByRole('button', { name: '新增個案' }).click()
    const dialog = page.locator('.el-dialog').filter({ hasText: '新增個案並綁定' })
    await expect(dialog.locator('input').first()).toHaveValue('王小明')
    await dialog.getByRole('button', { name: '建立並綁定' }).click()

    await expectElMessage(page, /已建立個案「王小明」並完成綁定/, 'success')
    await expect(page.locator('.el-table__row').filter({ hasText: '王小明' })).toHaveCount(0)
  })
})
