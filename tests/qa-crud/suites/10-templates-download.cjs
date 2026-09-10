exports.name = '各功能範本與匯出檔下載'

exports.run = async ({ page, record, step, L }) => {
  await L.goto(page, '/cases')
  await step('個案匯入範本', () => L.download(page, '下載匯入範本'))

  await step('個案資料匯出（需先勾選個案）', async () => {
    await page.getByRole('button', { name: '匯出個案資料', exact: false }).first().click()
    await page.waitForTimeout(1200)
    const header = L.dialog(page).locator('.el-table__header .el-checkbox').first()
    if (await header.count() > 0) await header.click()
    await page.waitForTimeout(500)
    return await L.download(page, () => L.dialog(page).locator('button', { hasText: '確認匯出' }).last().click())
  })
  await L.closeDialog(page)

  // commit d59060f「前端隱藏批次匯入與下載範本按鈕，保留後端完整 API 功能」：
  // 「下載匯入範本」按鈕跟批次匯入入口在同一個對話框內，該對話框已無任何按鈕能開啟（見 13-caregiver-import.cjs），
  // 故此步驟必定逾時，改為記錄跳過原因。
  await L.goto(page, '/masters/caregivers')
  await step('照護人員匯入範本（前端入口已隱藏，跳過）', async () =>
    ({ skipped: true, reason: '批次匯入與下載範本按鈕已於 commit d59060f 從畫面隱藏，僅後端 API 保留' }))

  await L.goto(page, '/vehicles/maintenance')
  await step('車輛保養空白表', () => L.download(page, '下載空白保養表'))

  await L.goto(page, '/reports/trip-summary')
  await step('車輛趟數表匯出', () => L.download(page, '匯出 Excel'))

  await L.goto(page, '/reports/hsinchu-schedule')
  await step('新竹接送時刻表匯出', () => L.download(page, '匯出 Excel'))

  await L.goto(page, '/driver-reports/import')
  await step('批次上傳頁可見按鈕', async () => ({
    buttons: (await page.locator('button').allInnerTexts()).map(b => b.replace(/\s+/g, ' ').trim()).filter(Boolean)
  }))

  await L.goto(page, '/driver-reports/status')
  await step('接送匯報總覽頁可見按鈕', async () => ({
    buttons: (await page.locator('button, a').allInnerTexts()).map(b => b.replace(/\s+/g, ' ').trim()).filter(Boolean)
  }))
}
