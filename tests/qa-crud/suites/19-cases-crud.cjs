exports.name = '個案 CRUD、排班與交通偏好'

exports.run = async ({ page, net, record, step, L }) => {
  const NAME = `${L.TAG}個案`
  await L.goto(page, '/cases')

  await L.openCreate(page, '新增個案')
  await step('空白送出', () => L.submit(page, net))

  await L.openCreate(page, '新增個案')
  await step('只填姓名', async () => {
    await L.fill(page, '個案姓名', NAME + 'A')
    return L.submit(page, net)
  })

  await L.openCreate(page, '新增個案')
  await step('姓名+檢查碼錯誤身分證', async () => {
    await L.fill(page, '個案姓名', NAME + 'B')
    await L.fill(page, '身分證字號', 'A202559751')
    return L.submit(page, net)
  })

  await L.openCreate(page, '新增個案')
  await step('全欄位新增', async () => {
    await L.fill(page, '個案姓名', NAME + 'C')
    await L.fill(page, '身分證字號', 'A800000014')
    await L.pick(page, '申報區域', '苗栗縣')
    await L.fill(page, '住家地址', '苗栗縣QA路9號')
    await L.radio(page, '服務類別', '1. 補助')
    const usage = await L.selectOptions(page, '服務使用類型')
    if (usage.length > 0) await L.pick(page, '服務使用類型', usage[0])
    await L.fill(page, '備註', 'QA 全欄位個案')
    return { usageOptions: usage, result: await L.submit(page, net) }
  })

  await L.openCreate(page, '新增個案')
  await step('選 DB 不允許的申報區域（臺北市）', async () => {
    await L.fill(page, '個案姓名', NAME + 'D')
    await L.pick(page, '申報區域', '臺北市')
    return L.submit(page, net)
  })

  await L.openCreate(page, '新增個案')
  await step('身分證與既有個案重複', async () => {
    await L.fill(page, '個案姓名', NAME + 'E')
    await L.fill(page, '身分證字號', 'A800000014')
    return L.submit(page, net)
  })

  await L.goto(page, '/cases')
  record('清單', { rows: await L.tableRows(page) })

  await step('進入個案編輯頁', async () => {
    await L.rowAction(page, NAME + 'C', '編輯')
    await page.waitForTimeout(2500)
    return {
      url: new URL(page.url()).pathname,
      text: (await page.locator('main').innerText()).replace(/\s+/g, ' ').slice(0, 1200),
      buttons: (await page.locator('main button').allInnerTexts()).map(s => s.replace(/\s+/g, ' ').trim()).filter(Boolean)
    }
  })

  await step('編輯頁儲存基本資料', async () => {
    const before = net.calls.length
    const save = page.locator('main button', { hasText: /儲存|更新/ }).first()
    if (await save.count() === 0) return { note: '找不到儲存按鈕' }
    await save.click()
    await page.waitForTimeout(2500)
    return { calls: net.calls.slice(before), toasts: await L.toasts(page) }
  })

  await step('明文顯示身分證（reveal）', async () => {
    const before = net.calls.length
    const reveal = page.locator('main button, main span', { hasText: /顯示|明文|檢視身分證/ }).first()
    if (await reveal.count() === 0) return { note: '找不到 reveal 控制項' }
    await reveal.click()
    await page.waitForTimeout(2500)
    return { calls: net.calls.slice(before), toasts: await L.toasts(page) }
  })

  await L.goto(page, '/cases')
  await step('刪除個案', async () => {
    await L.rowAction(page, NAME + 'A', '刪除')
    return L.confirmBox(page, net)
  })
}
