exports.name = '出勤與油資 CRUD'

exports.run = async ({ page, net, record, step, L }) => {
  await L.goto(page, '/attendance')

  await step('出勤月曆統計卡片', async () => ({
    text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 400)
  }))

  await step('點擊出勤格子開啟登記對話框', async () => {
    await page.locator('.el-table__body tr').first().locator('td').nth(3).click()
    await page.waitForTimeout(1500)
    return L.dumpForm(page)
  })

  await step('登記出勤狀態', async () => {
    if ((await page.locator('.el-dialog:visible').count()) === 0) return { note: '沒有開啟對話框' }
    // 原本傳入 /儲存|確認/.source 會把正則物件轉成字面字串 "儲存|確認"，
    // hasText 傳字串只會做子字串比對，永遠找不到按鈕；DialogFooter 預設文字就是「儲存」。
    return L.submit(page, net)
  })
  await L.closeDialog(page)

  await step('切換到車輛油資登錄頁籤', async () => {
    await page.locator('.el-tabs__item', { hasText: '車輛油資登錄' }).first().click()
    await page.waitForTimeout(2000)
    return { text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 400) }
  })

  await L.openCreate(page, '新增加油紀錄')
  await step('油資表單欄位', () => L.dumpForm(page))
  await step('油資空白送出', () => L.submit(page, net))

  await step('油資全欄位新增', async () => {
    const form = await L.dumpForm(page)
    for (const it of form.items) {
      if (it.label.includes('車輛')) await L.pick(page, it.label, 'AAA-123')
      else if (it.label.includes('司機')) await L.pick(page, it.label, '林彥衡')
      else if (it.label.includes('日期')) await L.pickDate(page, it.label, '2026-09-05')
      else if (it.label.includes('金額')) await L.fill(page, it.label, '1200')
      else if (it.label.includes('公升') || it.label.includes('油量')) await L.fill(page, it.label, '45.5')
      else if (it.label.includes('里程')) await L.fill(page, it.label, '55000')
      else if (it.label.includes('加油站') || it.label.includes('地點')) await L.fill(page, it.label, 'QA加油站')
      else if (it.label.includes('備註')) await L.fill(page, it.label, `${L.TAG}油資`)
    }
    return L.submit(page, net)
  })

  await L.goto(page, '/attendance')
  await page.locator('.el-tabs__item', { hasText: '車輛油資登錄' }).first().click()
  await page.waitForTimeout(2000)
  record('油資清單', { rows: await L.tableRows(page) })
}
