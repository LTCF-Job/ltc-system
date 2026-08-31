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

// 批次上傳頁：每一列是「一輛車 × 一個月」。月份寫在 query 讓連結可分享，測試沿用同一個入口。
test.describe('05b. 司機接送匯報批次上傳 (Driver Report Batch Import)', () => {
  const MONTH = '2026-03'

  function rowOf(page: Page, vehicleName: string) {
    return page.locator('.el-table__row').filter({ hasText: vehicleName }).first()
  }

  async function uploadRowFile(page: Page, vehicleName: string, rows: string[][]) {
    await rowOf(page, vehicleName).locator('input[type="file"]').setInputFiles({
      name: 'driver-report.xlsx',
      mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      buffer: buildReportWorkbook(rows)
    })
  }

  async function analyzeRow(page: Page, vehicleName: string) {
    await rowOf(page, vehicleName).getByRole('button', { name: '試算' }).click()
  }

  // 試算後若該列仍有欄位待對應，就地把待對應的欄位綁到第一個個案，讓該列變成可匯入
  async function resolvePendingMappings(page: Page, vehicleName: string) {
    const statusTag = rowOf(page, vehicleName).locator('.el-tag')
    // 先等試算結束，狀態還停在「試算中」時判斷會誤以為沒有待對應欄位
    await expect(statusTag.filter({ hasText: /需處理|可匯入|失敗/ })).toBeVisible()
    if (!(await statusTag.filter({ hasText: '需處理' }).isVisible())) return

    const expanded = page.locator('.el-table__expanded-cell').filter({ hasText: vehicleName })
    await expect(expanded).toBeVisible()
    const mappingRows = expanded.locator('.el-table__row')
    await expect(mappingRows.first()).toBeVisible()
    for (let i = 0; i < (await mappingRows.count()); i++) {
      const mappingRow = mappingRows.nth(i)
      if ((await mappingRow.locator('.el-tag').filter({ hasText: '待對應' }).count()) === 0) continue
      await mappingRow.locator('.el-select').first().click()
      // 用鍵盤挑下一個選項：多列同時展開時會有多個 dropdown，直接點選容易命中別列的；
      // 而且重選同一個個案不會觸發 change，該欄會留在待對應
      await page.keyboard.press('ArrowDown')
      await page.keyboard.press('Enter')
      await expect(mappingRow.locator('.el-tag').filter({ hasText: '待對應' })).toHaveCount(0)
    }
    await expect(statusTag.filter({ hasText: '可匯入' })).toBeVisible()
  }

  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin')
  })

  test('B1 選兩個月時，表格列出全部啟用車輛 × 2 列，沒有匯報表的車也在其中', async ({ page }) => {
    await page.goto('/driver-reports/batch-import?months=2026-03,2026-04')
    await waitForTableLoaded(page)

    // 啟用車輛 5 台 × 2 個月
    await expect(page.locator('.el-table__row')).toHaveCount(10)
    await expect(rowOf(page, '苗栗市1車')).toContainText('尚未建表')
    await expect(page.locator('.el-table__row').filter({ hasText: '2026-04' }).first()).toBeVisible()
  })

  test('B2 對已有匯報表的車上傳並試算，只有該列出現結果', async ({ page }) => {
    await page.goto(`/driver-reports/batch-import?months=${MONTH}`)
    await waitForTableLoaded(page)

    await uploadRowFile(page, '竹北一車', [
      HEADER,
      ['1150302', '郭澤威', '有坐', '沒坐', '無'],
      ['115/3/3', '郭澤威', '沒坐', '有坐', '']
    ])
    await analyzeRow(page, '竹北一車')

    await expect(rowOf(page, '竹北一車')).toContainText('可匯入 2 天')
    await expect(rowOf(page, '竹北二車')).toContainText('待處理')
  })

  test('B3 對沒有匯報表的車上傳，自動建表後成功試算並出現在匯報表清單', async ({ page }) => {
    await page.goto(`/driver-reports/batch-import?months=${MONTH}`)
    await waitForTableLoaded(page)

    await uploadRowFile(page, '苗栗市1車', [
      HEADER,
      ['1150302', '郭澤威', '有坐', '沒坐', '無']
    ])
    await analyzeRow(page, '苗栗市1車')

    await expect(rowOf(page, '苗栗市1車')).toContainText('可匯入 1 天')
    await expect(rowOf(page, '苗栗市1車')).not.toContainText('尚未建表')

    await page.goto('/driver-reports')
    await waitForTableLoaded(page)
    await expect(page.getByText('司機接送匯報（共 5 台車）')).toBeVisible()
  })

  test('B4 檔案日期不屬於該列宣告的月份時，該列顯示錯誤且不得被匯入', async ({ page }) => {
    await page.goto(`/driver-reports/batch-import?months=${MONTH}`)
    await waitForTableLoaded(page)

    await uploadRowFile(page, '竹北一車', [
      HEADER,
      ['1150402', '郭澤威', '有坐', '沒坐', '無']
    ])
    await analyzeRow(page, '竹北一車')

    await expect(rowOf(page, '竹北一車')).toContainText('不屬於宣告匯入的 2026-03')
    await expect(rowOf(page, '竹北一車')).toContainText('失敗')
    await expect(page.getByRole('button', { name: /確認匯入 \(0\)/ })).toBeDisabled()
  })

  test('B5 有未對應欄位的列標記需處理，就地對應後才可確認匯入', async ({ page }) => {
    await page.goto(`/driver-reports/batch-import?months=${MONTH}`)
    await waitForTableLoaded(page)

    await uploadRowFile(page, '竹北一車', [
      HEADER,
      ['1150302', '郭澤威', '有坐', '沒坐', '無']
    ])
    await analyzeRow(page, '竹北一車')

    await expect(rowOf(page, '竹北一車')).toContainText('需處理')
    await expect(page.getByRole('button', { name: /確認匯入 \(0\)/ })).toBeDisabled()

    // 展開列的欄位對應表格：選定個案後該欄即視為已對應
    const expanded = page.locator('.el-table__expanded-cell')
    await expect(expanded).toBeVisible()
    await expanded.locator('.el-select').first().click()
    await page.locator('.el-select-dropdown__item').first().click()

    await expect(rowOf(page, '竹北一車')).toContainText('可匯入')
    await expect(page.getByRole('button', { name: /確認匯入 \(1\)/ })).toBeEnabled()
  })

  test('B6 對已匯入過的月份重傳時跳確認視窗，確認後筆數不翻倍', async ({ page }) => {
    const file: string[][] = [
      HEADER,
      ['1150302', '郭澤威', '有坐', '沒坐', '無'],
      ['115/3/3', '郭澤威', '沒坐', '有坐', '']
    ]

    await page.goto(`/driver-reports/batch-import?months=${MONTH}`)
    await waitForTableLoaded(page)

    await uploadRowFile(page, '竹北一車', file)
    await analyzeRow(page, '竹北一車')
    await resolvePendingMappings(page, '竹北一車')
    await page.getByRole('button', { name: /確認匯入/ }).click()
    await expectElMessage(page, /已匯入 1 列接送匯報/, 'success')
    await expect(rowOf(page, '竹北一車')).toContainText('已匯入')
    await expect(rowOf(page, '竹北一車')).toContainText('2 天')
    // 等第一則成功訊息消失，避免第二則與它同時存在而選到兩個元素
    await expect(page.locator('.el-message--success')).toHaveCount(0, { timeout: 10000 })

    // 重傳同一個月：先跳覆蓋確認，確認後仍是 2 天而非 4 天
    await uploadRowFile(page, '竹北一車', file)
    await analyzeRow(page, '竹北一車')
    await resolvePendingMappings(page, '竹北一車')
    await page.getByRole('button', { name: /確認匯入/ }).click()

    const confirmBox = page.locator('.el-message-box')
    await expect(confirmBox).toBeVisible()
    await expect(confirmBox).toContainText('既有 2 天')
    await confirmBox.getByRole('button', { name: '確認送出' }).click()

    await expectElMessage(page, /已匯入 1 列接送匯報/, 'success')
    await expect(rowOf(page, '竹北一車')).toContainText('2 天')
  })

  test('B10 重新選擇檔案後，試算與匯入用的是新檔而不是第一次選的那份', async ({ page }) => {
    await page.goto(`/driver-reports/batch-import?months=${MONTH}`)
    await waitForTableLoaded(page)

    await uploadRowFile(page, '竹北一車', [HEADER, ['1150302', '郭澤威', '有坐', '沒坐', '無']])
    await analyzeRow(page, '竹北一車')
    await expect(rowOf(page, '竹北一車')).toContainText('可匯入 1 天')

    // 換成三天的檔案：沿用舊檔的話這裡會停在 1 天
    await uploadRowFile(page, '竹北一車', [
      HEADER,
      ['1150302', '郭澤威', '有坐', '沒坐', '無'],
      ['1150303', '郭澤威', '沒坐', '有坐', ''],
      ['1150304', '郭澤威', '有坐', '有坐', '']
    ])
    await analyzeRow(page, '竹北一車')
    await expect(rowOf(page, '竹北一車')).toContainText('可匯入 3 天')

    await resolvePendingMappings(page, '竹北一車')
    await page.getByRole('button', { name: /確認匯入/ }).click()
    await expectElMessage(page, /已匯入 1 列接送匯報/, 'success')
    await expect(rowOf(page, '竹北一車')).toContainText('3 天')
  })

  test('B11 清空月份選擇後畫面仍可用，回到請先選擇月份的空狀態', async ({ page }) => {
    await page.goto(`/driver-reports/batch-import?months=${MONTH}`)
    await waitForTableLoaded(page)
    await expect(page.locator('.el-table__row').first()).toBeVisible()

    await page.locator('.el-date-editor').hover()
    await page.locator('.el-input__icon.clear-icon, .el-select__caret.is-clear').first().click()

    await expect(page.getByText('請先選擇要匯入的月份')).toBeVisible()
    await expect(page.locator('.el-table__row')).toHaveCount(0)
  })

  test('B12 列數超過並發上限時佇列仍會排乾，全部列都有最終結果', async ({ page }) => {
    await page.goto(`/driver-reports/batch-import?months=${MONTH}`)
    await waitForTableLoaded(page)

    const validFile: string[][] = [HEADER, ['1150302', '郭澤威', '有坐', '沒坐', '無']]
    const vehicles = ['竹北一車', '竹北二車', '竹南1車', '竹南2車', '苗栗市1車']
    for (const vehicle of vehicles) {
      await uploadRowFile(page, vehicle, validFile)
    }

    // 5 列 > MAX_CONCURRENT_REQUESTS(3)，後面兩列必須由 worker 從佇列取出處理
    await page.getByRole('button', { name: /全部試算/ }).click()
    for (const vehicle of vehicles) {
      await expect(rowOf(page, vehicle)).toContainText('可匯入 1 天')
    }
  })

  test('B7 三列同時處理，失敗列顯示原因且不影響其他列', async ({ page }) => {
    await page.goto(`/driver-reports/batch-import?months=${MONTH}`)
    await waitForTableLoaded(page)

    const validFile: string[][] = [HEADER, ['1150302', '郭澤威', '有坐', '沒坐', '無']]

    await uploadRowFile(page, '竹北一車', validFile)
    await uploadRowFile(page, '竹南1車', validFile)
    await uploadRowFile(page, '苗栗市1車', [['時間戳記', '今天日期', '今日駕駛人', '問題回報']])

    await page.getByRole('button', { name: /全部試算/ }).click()

    await expect(rowOf(page, '苗栗市1車')).toContainText('找不到匯報表表頭')
    await expect(rowOf(page, '苗栗市1車')).toContainText('失敗')
    await expect(rowOf(page, '竹北一車')).toContainText('可匯入 1 天')
    await expect(rowOf(page, '竹南1車')).toContainText('可匯入 1 天')

    // 失敗列被排除在確認匯入之外，其餘兩列照常寫入
    await resolvePendingMappings(page, '竹北一車')
    await resolvePendingMappings(page, '竹南1車')
    await page.getByRole('button', { name: /確認匯入 \(2\)/ }).click()

    await expectElMessage(page, /已匯入 2 列接送匯報/, 'success')
    await expect(rowOf(page, '竹北一車')).toContainText('已匯入')
    await expect(rowOf(page, '竹南1車')).toContainText('已匯入')
    await expect(rowOf(page, '苗栗市1車')).toContainText('找不到匯報表表頭')
  })
})
