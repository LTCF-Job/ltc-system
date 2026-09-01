import { test, expect } from '@playwright/test'
import fs from 'node:fs'
import * as XLSX from 'xlsx'
import { loginAs } from './helpers/auth'
import { waitForTableLoaded } from './helpers/ui'

/** 在申報個案對話框中勾選前 N 筆個案並確認 */
async function selectCases(page: import('@playwright/test').Page, count: number) {
  await page.getByRole('button', { name: '選擇個案' }).click()

  const dialog = page.locator('.el-dialog:visible').filter({ hasText: '選擇申報個案' })
  await expect(dialog).toBeVisible()
  await expect(dialog.locator('.el-table__body tr')).not.toHaveCount(0)

  for (let i = 0; i < count; i++) {
    await dialog.locator('.el-table__body tr').nth(i).locator('.el-checkbox').click()
  }
  await dialog.getByRole('button', { name: /確認選擇/ }).click()
  await expect(dialog).toBeHidden()
}

async function runExport(page: import('@playwright/test').Page) {
  await page.getByRole('button', { name: '開始產生申報檔' }).click()
  // 展示資料的前置檢核帶有警告，匯出前會先要求確認
  const warningBox = page.locator('.el-message-box')
  await expect(warningBox).toBeVisible({ timeout: 10000 })
  await warningBox.getByRole('button', { name: '繼續匯出' }).click()
  await expect(page.getByText('本次匯出結果')).toBeVisible({ timeout: 15000 })
}

test.describe('11. 政府申報匯出與前置檢核 (Gov Claims Export & Precheck)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin', '/exports')
  })

  test('匯出設定介面與前置檢核執行', async ({ page }) => {
    await waitForTableLoaded(page)
    await expect(page.getByText('政府申報表匯出設定')).toBeVisible()

    // 設定只保留四個輸入：申報年月、申報地區、申報個案、匯出檔案模式
    await expect(page.getByText('申報年月 (民國)')).toBeVisible()
    await expect(page.getByText('申報地區')).toBeVisible()
    await expect(page.getByText('申報個案')).toBeVisible()
    await expect(page.getByText('匯出檔案模式')).toBeVisible()

    await page.getByRole('button', { name: '執行前置檢核' }).click()
    await expect(page.getByText('前置檢核報告')).toBeVisible({ timeout: 10000 })
  })

  test('本次匯出結果排在前置檢核報告之上', async ({ page }) => {
    await waitForTableLoaded(page)
    await selectCases(page, 2)
    await page.getByRole('button', { name: '執行前置檢核' }).click()
    await expect(page.getByText('前置檢核報告')).toBeVisible({ timeout: 10000 })
    await runExport(page)

    const jobTop = await page.locator('.job-card').boundingBox()
    const precheckTop = await page.locator('.precheck-card').boundingBox()
    expect(jobTop!.y).toBeLessThan(precheckTop!.y)
  })

  test('前置檢核的未回報項目可直接跳到未回報清單並篩出該個案', async ({ page }) => {
    await waitForTableLoaded(page)
    await page.getByRole('button', { name: '執行前置檢核' }).click()
    await expect(page.getByText('前置檢核報告')).toBeVisible({ timeout: 10000 })

    const unreportedItem = page
      .locator('.precheck-card .el-collapse-item')
      .filter({ hasText: '未回報' })
      .first()
    const targetRow = unreportedItem.locator('.el-table__body tr').filter({ hasText: '蔡曾切' }).first()
    await targetRow.getByRole('button', { name: '查看未回報' }).click()

    await expect(page).toHaveURL(/\/rides\/missing\?q=/)
    await expect(page.getByPlaceholder('搜尋個案姓名／車輛／司機')).toHaveValue('蔡曾切')

    const rows = page.locator('.el-tab-pane:visible .el-table__body tr')
    await expect(rows).not.toHaveCount(0)
    for (const row of await rows.all()) {
      await expect(row).toContainText('蔡曾切')
    }
  })

  test('未選個案時不執行匯出', async ({ page }) => {
    await waitForTableLoaded(page)
    await expect(page.getByText('尚未選擇個案')).toBeVisible()

    await page.getByRole('button', { name: '開始產生申報檔' }).click()
    await expect(page.locator('.el-message--warning')).toContainText('請先選擇要申報的個案')
    await expect(page.getByText('本次匯出結果')).toBeHidden()
  })

  test('個案多選視窗可篩選並回填已選筆數', async ({ page }) => {
    await waitForTableLoaded(page)
    await selectCases(page, 2)
    await expect(page.locator('.case-picker-summary')).toContainText('已選擇 2 筆')
  })

  test('直接下載模式逐案列出檔案並可單獨下載', async ({ page }) => {
    await waitForTableLoaded(page)
    await selectCases(page, 2)
    await runExport(page)

    // 一個個案一列，各自一顆下載鈕；不自動觸發多重下載
    const fileRows = page.locator('.file-table .el-table__body tr')
    await expect(fileRows).toHaveCount(2)
    await expect(page.locator('.file-table')).toContainText('11507.xlsx')

    const downloadPromise = page.waitForEvent('download')
    await fileRows.first().getByRole('button', { name: '下載' }).click()
    const download = await downloadPromise
    expect(download.suggestedFilename()).toMatch(/\.xlsx$/)
  })

  test('壓縮檔模式只提供單一壓縮檔下載', async ({ page }) => {
    await waitForTableLoaded(page)
    await selectCases(page, 2)
    await page.locator('.el-radio').filter({ hasText: '壓縮檔' }).click()
    await runExport(page)

    await expect(page.locator('.file-table')).toHaveCount(0)

    const zipButton = page.locator('.zip-download').getByRole('button')
    await expect(zipButton).toContainText('.zip')

    const downloadPromise = page.waitForEvent('download')
    await zipButton.click()
    const download = await downloadPromise
    expect(download.suggestedFilename()).toMatch(/\.zip$/)
  })

  // 展示模式的申報列必須跟搭乘月曆讀同一份資料，否則畫面說某天缺席、申報檔卻有那一天
  test('申報列只取月曆上實際成行的趟次', async ({ page }) => {
    await waitForTableLoaded(page)

    const expected = await page.evaluate(async () => {
      const res = await fetch('/api/v1/rides/calendar?month=115-07')
      const body = await res.json()
      const target = body.cases.find((row: any) => row.caseId === 'case_1')
      const records = Object.values(target.days).flatMap((day: any) => day.records)
      return {
        boardedCount: records.filter((r: any) => r.effectiveStatus === 'boarded').length,
        excludedDates: [
          ...new Set(
            records
              .filter((r: any) => r.effectiveStatus !== 'boarded')
              .map((r: any) => r.serviceDate)
          )
        ] as string[]
      }
    })
    expect(expected.boardedCount).toBeGreaterThan(0)
    expect(expected.excludedDates.length).toBeGreaterThan(0)

    const job = await page.evaluate(async () => {
      const res = await fetch('/api/v1/exports', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ jobType: 'gov_claim', periodYm: '11507', mode: 'direct', caseIds: ['case_1'] })
      })
      return res.json()
    })
    expect(job.files[0].rowCount).toBe(expected.boardedCount)

    const downloadPromise = page.waitForEvent('download')
    await page.evaluate(async (url) => {
      const res = await fetch(url)
      const anchor = document.createElement('a')
      anchor.href = URL.createObjectURL(await res.blob())
      anchor.download = 'claim.xlsx'
      document.body.appendChild(anchor)
      anchor.click()
    }, job.files[0].downloadUrl)
    const filePath = await (await downloadPromise).path()

    const workbook = XLSX.read(fs.readFileSync(filePath!), { type: 'buffer' })
    const sheet = XLSX.utils.sheet_to_json(workbook.Sheets['工作表1'], { header: 1, defval: '' }) as any[][]
    const claimDates = sheet.slice(1).map((row) => row[1])

    expect(claimDates).toHaveLength(expected.boardedCount)
    // 民國 7 碼：2026-07-28 -> 1150728
    const excludedRoc = expected.excludedDates.map((d) => Number(d.replaceAll('-', '')) - 19110000)
    for (const roc of excludedRoc) {
      expect(claimDates).not.toContain(roc)
    }
  })

  test('歷史匯出紀錄不提供下載，只能檢視個案清單', async ({ page }) => {
    await waitForTableLoaded(page)
    await expect(page.getByText('歷史匯出紀錄')).toBeVisible()

    const historyTable = page.locator('.history-card .el-table').first()
    await expect(historyTable).toBeVisible()
    await expect(page.locator('.history-card').getByRole('button', { name: '下載檔案' })).toHaveCount(0)

    await page.locator('.history-card').getByRole('button', { name: '檢視個案' }).first().click()

    const detailDialog = page.locator('.el-dialog:visible').filter({ hasText: '該次匯出的個案清單' })
    await expect(detailDialog).toBeVisible()
    await expect(detailDialog.locator('.el-table__body tr')).not.toHaveCount(0)
    await expect(detailDialog.getByRole('button', { name: '下載' })).toHaveCount(0)
  })
})
