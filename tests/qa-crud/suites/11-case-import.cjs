const path = require('node:path')

exports.name = '個案批次匯入'

exports.run = async ({ page, net, record, step, expect, L }) => {
  const tpl = path.join(L.DOWNLOAD_DIR, '個案批次匯入範本.xlsx')

  await L.goto(page, '/cases')
  await step('開啟批次匯入對話框', async () => {
    await page.getByRole('button', { name: '批次匯入個案', exact: false }).first().click()
    await page.waitForTimeout(1200)
    return {
      dialogText: (await L.dialog(page).innerText()).replace(/\s+/g, ' ').trim().slice(0, 600),
      fileInputs: await page.locator('.el-dialog:visible input[type=file]').count(),
      buttons: (await L.dialog(page).locator('button').allInnerTexts()).map(b => b.trim()).filter(Boolean)
    }
  })

  await step('上傳未修改的官方範本並解析預覽', async () => {
    const before = net.calls.length
    await page.setInputFiles('.el-dialog input[type=file]', tpl)
    await page.waitForTimeout(1200)
    await L.dialog(page).locator('button', { hasText: '開始解析與預覽' }).last().click()
    await page.waitForTimeout(5000)
    return {
      calls: net.calls.slice(before),
      dialogText: (await L.dialog(page).innerText()).replace(/\s+/g, ' ').trim().slice(0, 2000),
      buttons: (await L.dialog(page).locator('button').allInnerTexts()).map(b => b.trim()).filter(Boolean),
      toasts: await L.toasts(page)
    }
  })

  await step('範本示範列不得被當成可匯入資料', async () => {
    const dialogText = (await L.dialog(page).innerText()).replace(/\s+/g, ' ')
    const total = /總筆數：(\d+)/.exec(dialogText)?.[1]
    expect('官方範本原封不動上傳 → 0 筆可匯入', total === '0', { total, dialogText: dialogText.slice(0, 200) })
    return { total }
  })

  await step('確認寫入', async () => {
    const before = net.calls.length
    const confirm = L.dialog(page).locator('button', { hasText: /^匯入|確認匯入|確認寫入|開始匯入|確認新增/ }).last()
    if (await confirm.count() === 0) return { note: '沒有確認按鈕' }
    await confirm.click()
    await page.waitForTimeout(5000)
    return {
      calls: net.calls.slice(before),
      dialogText: (await L.dialog(page).count()) > 0
        ? (await L.dialog(page).innerText()).replace(/\s+/g, ' ').trim().slice(0, 1500) : '(已關閉)',
      toasts: await L.toasts(page)
    }
  })
  await L.closeDialog(page)

  await L.goto(page, '/cases')
  await step('匯入後個案清單', async () => ({ rows: await L.tableRows(page) }))

  await step('待維護（疑似重複）頁籤', async () => {
    const tab = page.locator('.el-tabs__item', { hasText: '待維護' }).first()
    if (await tab.count() === 0) return { note: '找不到待維護頁籤' }
    await tab.click()
    await page.waitForTimeout(2000)
    return { text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 900) }
  })
}
