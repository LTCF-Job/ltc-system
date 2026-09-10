exports.name = '通知收件人 CRUD'

exports.run = async ({ page, net, record, step, L }) => {
  await L.goto(page, '/settings/notifications')

  await L.openCreate(page, '新增外部信箱')
  await step('主題與信箱都空白（確認按鈕停用）', async () => {
    // 確認按鈕在沒有任何有效信箱時是 :confirm-disabled，Playwright 對停用按鈕的 click 只會一直等到逾時，
    // 改成直接檢查按鈕的 disabled 狀態，符合現在「送出前就擋住」的前端行為。
    const btn = L.dialog(page).locator('button', { hasText: '確認新增' }).last()
    return { disabled: await btn.isDisabled() }
  })

  await L.openCreate(page, '新增外部信箱')
  await step('單一合法信箱', async () => {
    const topics = await L.selectOptions(page, '通知主題')
    await L.pick(page, '通知主題', topics[0])
    await L.fill(page, '輸入外部信箱（支援直接貼上多行 Email，或「姓名 <信箱>」格式）：', 'qa-one@example.com')
    return { topics, result: await L.submit(page, net, '確認新增') }
  })

  await L.openCreate(page, '新增外部信箱')
  await step('多行混合格式貼上', async () => {
    const topics = await L.selectOptions(page, '通知主題')
    await L.pick(page, '通知主題', topics[0])
    await L.fill(page, '輸入外部信箱（支援直接貼上多行 Email，或「姓名 <信箱>」格式）：',
      'qa-two@example.com\nQA顧問 <qa-three@example.com>\nQA專員 qa-four@example.com\n不是信箱\nqa-two@example.com')
    return L.submit(page, net, '確認新增')
  })

  await L.openCreate(page, '新增外部信箱')
  await step('與既有收件人重複', async () => {
    const topics = await L.selectOptions(page, '通知主題')
    await L.pick(page, '通知主題', topics[0])
    await L.fill(page, '輸入外部信箱（支援直接貼上多行 Email，或「姓名 <信箱>」格式）：', 'qa-one@example.com')
    return L.submit(page, net, '確認新增')
  })

  await L.goto(page, '/settings/notifications')
  record('清單', { rows: await L.tableRows(page) })

  await step('編輯第一筆', async () => {
    await L.rowAction(page, 'qa-one@example.com', '編輯')
    const before = await L.dumpForm(page)
    // 編輯對話框的 DialogFooter 沒有覆寫 confirm-text，走預設「儲存」；
    // 原本傳 /確認|儲存/.source 會把正則物件轉成字面字串，hasText 傳字串只比對子字串，永遠找不到按鈕。
    return { before, result: await L.submit(page, net) }
  })

  await L.goto(page, '/settings/notifications')
  await step('刪除 qa-one', async () => {
    await L.rowAction(page, 'qa-one@example.com', '刪除')
    return L.confirmBox(page, net)
  })
}
