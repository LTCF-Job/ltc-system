exports.name = '車輛維修保養 CRUD'

exports.run = async ({ page, net, record, step, L }) => {
  const NOTE = `${L.TAG}保養`
  await L.goto(page, '/vehicles/maintenance')

  await L.openCreate(page, '新增保養紀錄')
  await step('空白送出', () => L.submit(page, net))

  await L.openCreate(page, '新增保養紀錄')
  await step('全欄位新增', async () => {
    await L.pick(page, '保養車輛', 'AAA-123')
    await L.pickDate(page, '保養日期', '2026-09-01')
    await L.fill(page, '保養里程', '12345')
    await L.fill(page, '保養項目', '更換機油、煞車皮')
    await L.fill(page, '保養廠商', '順益汽車')
    await L.fill(page, '花費金額', '3500')
    await L.fill(page, '收據連結', 'https://example.com/receipt.pdf')
    await L.fill(page, '備註說明', NOTE)
    return L.submit(page, net)
  })

  await L.openCreate(page, '新增保養紀錄')
  await step('只填必填欄位', async () => {
    await L.pick(page, '保養車輛', 'AAA-123')
    await L.pickDate(page, '保養日期', '2026-09-02')
    await L.fill(page, '保養里程', '12400')
    await L.fill(page, '保養項目', '只填必填')
    await L.fill(page, '花費金額', '0')
    return L.submit(page, net)
  })

  await L.openCreate(page, '新增保養紀錄')
  await step('里程與金額填負數', async () => {
    await L.pick(page, '保養車輛', 'AAA-123')
    await L.pickDate(page, '保養日期', '2026-09-03')
    await L.fill(page, '保養里程', '-100')
    await L.fill(page, '保養項目', '負數測試')
    await L.fill(page, '花費金額', '-999')
    return L.submit(page, net)
  })

  await L.openCreate(page, '新增保養紀錄')
  await step('保養日期填未來一年', async () => {
    await L.pick(page, '保養車輛', 'AAA-123')
    await L.pickDate(page, '保養日期', '2027-12-31')
    await L.fill(page, '保養里程', '1')
    await L.fill(page, '保養項目', '未來日期測試')
    await L.fill(page, '花費金額', '1')
    return L.submit(page, net)
  })

  await L.openCreate(page, '新增保養紀錄')
  await step('收據連結填非網址', async () => {
    await L.pick(page, '保養車輛', 'AAA-123')
    await L.pickDate(page, '保養日期', '2026-09-04')
    await L.fill(page, '保養里程', '2')
    await L.fill(page, '保養項目', '收據連結測試')
    await L.fill(page, '花費金額', '2')
    await L.fill(page, '收據連結', 'javascript:alert(1)')
    return L.submit(page, net)
  })

  await L.goto(page, '/vehicles/maintenance')
  record('清單', { rows: await L.tableRows(page) })

  await step('編輯第一筆', async () => {
    await L.rowAction(page, '更換機油、煞車皮', '編輯')
    const before = await L.dumpForm(page)
    await L.fill(page, '花費金額', '4200')
    await L.fill(page, '備註說明', NOTE + '改')
    return { before, result: await L.submit(page, net) }
  })

  await L.goto(page, '/vehicles/maintenance')
  await step('刪除', async () => {
    await L.rowAction(page, '只填必填', '刪除')
    return L.confirmBox(page, net)
  })
}
