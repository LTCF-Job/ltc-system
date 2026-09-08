exports.name = '通知收件人 CRUD'

exports.run = async ({ page, net, record, step, L }) => {
  await L.goto(page, '/settings/notifications')

  await L.openCreate(page, '新增外部信箱')
  await step('主題與信箱都空白', () => L.submit(page, net, '確認新增'))

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
    return { before, result: await L.submit(page, net, /確認|儲存/.source) }
  })

  await L.goto(page, '/settings/notifications')
  await step('刪除 qa-one', async () => {
    await L.rowAction(page, 'qa-one@example.com', '刪除')
    return L.confirmBox(page, net)
  })
}
