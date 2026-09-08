const path = require('node:path')

exports.name = '個案匯入欄位組合驗證'

exports.run = async ({ page, net, record, step, expect, L }) => {
  const file = path.join(__dirname, '..', 'fixtures', 'case-import-filled.xlsx')

  // net.calls 的 body 有長度上限，截斷後 JSON.parse 會失敗，改直接取計數欄位。
  const commitCounts = (calls) => {
    const body = calls.find((c) => c.phase === 'res' && c.url.includes('dryRun=false'))?.body || ''
    const num = (key) => {
      const m = new RegExp(`"${key}":(\\d+)`).exec(body)
      return m ? Number(m[1]) : null
    }
    return {
      importedCount: num('importedCount'),
      alreadyImportedCount: num('alreadyImportedCount'),
      stagedDuplicateCount: num('stagedDuplicateCount')
    }
  }

  await L.goto(page, '/cases')
  await step('上傳組合測試檔並預覽', async () => {
    const before = net.calls.length
    await page.getByRole('button', { name: '批次匯入個案', exact: false }).first().click()
    await page.waitForTimeout(1000)
    await page.setInputFiles('.el-dialog input[type=file]', file)
    await page.waitForTimeout(1000)
    await L.dialog(page).locator('button', { hasText: '開始解析與預覽' }).last().click()
    await page.waitForTimeout(5000)
    const dialogText = (await L.dialog(page).innerText()).replace(/\s+/g, ' ').trim()
    // fixture 共 10 列資料，其中第 8 列姓名空白：必須列成錯誤列而不是靜默丟棄。
    const total = /總筆數：(\d+)/.exec(dialogText)?.[1]
    expect('空白姓名列不得被靜默丟棄（總筆數應為 10）', total === '10', { total })
    return { calls: net.calls.slice(before), dialogText: dialogText.slice(0, 3000) }
  })

  await step('確認寫入', async () => {
    const before = net.calls.length
    const confirm = L.dialog(page).locator('button', { hasText: /^匯入|確認匯入/ }).last()
    if (await confirm.count() === 0) return { note: '沒有可按的匯入鈕（全部列都不合法）' }
    await confirm.click()
    await page.waitForTimeout(6000)
    return {
      calls: net.calls.slice(before),
      dialogText: (await L.dialog(page).count()) > 0
        ? (await L.dialog(page).innerText()).replace(/\s+/g, ' ').trim().slice(0, 2500) : '(已關閉)'
    }
  })
  await L.closeDialog(page)

  await step('重複匯入同一份檔案（冪等性）', async () => {
    const before = net.calls.length
    await L.goto(page, '/cases')
    await page.getByRole('button', { name: '批次匯入個案', exact: false }).first().click()
    await page.waitForTimeout(1000)
    await page.setInputFiles('.el-dialog input[type=file]', file)
    await page.waitForTimeout(1000)
    await L.dialog(page).locator('button', { hasText: '開始解析與預覽' }).last().click()
    await page.waitForTimeout(5000)
    const confirm = L.dialog(page).locator('button', { hasText: /^匯入|確認匯入/ }).last()
    if (await confirm.count() > 0) {
      await confirm.click()
      await page.waitForTimeout(6000)
    }
    const calls = net.calls.slice(before)
    const counts = commitCounts(calls)
    // 第二次上傳同一份檔案，先前建立的列必須被認出「已匯入」而不是變成假的待裁決。
    expect('重複匯入不產生新的待裁決列', counts.stagedDuplicateCount === 0, counts)
    expect('重複匯入不重複建立個案', counts.importedCount === 0, counts)
    expect('重複匯入的列被認出已匯入', (counts.alreadyImportedCount ?? 0) > 0, counts)
    return {
      calls,
      dialogText: (await L.dialog(page).count()) > 0
        ? (await L.dialog(page).innerText()).replace(/\s+/g, ' ').trim().slice(0, 2500) : '(已關閉)'
    }
  })
  await L.closeDialog(page)

  await L.goto(page, '/cases')
  await step('匯入後全部個案（含待維護）', async () => {
    const includePending = page.locator('.el-tabs__item', { hasText: '待維護' }).first()
    if (await includePending.count() > 0) {
      await includePending.click()
      await page.waitForTimeout(2000)
    }
    return { text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 1500) }
  })
}
