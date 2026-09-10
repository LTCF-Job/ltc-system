exports.name = '司機管理 CRUD'

// 這些身分證字號與 apps/web/tests/unit/national-id.test.ts 的合法向量相同。
const VALID_IDS = ['A202559750', 'G121806465', 'K120098177']

exports.run = async ({ page, net, record, L }) => {
  const NAME = `${L.TAG}司機`
  await L.goto(page, '/masters/drivers')

  await L.openCreate(page, '新增司機')
  record('空白送出', await L.submit(page, net))

  await L.openCreate(page, '新增司機')
  await L.fill(page, '司機姓名', NAME + 'A')
  await L.fill(page, '身分證字號', 'A202559751')
  record('身分證檢查碼錯誤', await L.submit(page, net))

  await L.openCreate(page, '新增司機')
  await L.fill(page, '司機姓名', NAME + 'A')
  await L.fill(page, '身分證字號', VALID_IDS[0])
  await L.fill(page, '電子信箱', 'qa-driver@example.com')
  await L.pick(page, '駕照類別', '大客車')
  await L.pickDate(page, '駕照有效日期', '2028-12-31')
  record('全欄位新增', await L.submit(page, net))

  await L.openCreate(page, '新增司機')
  await L.fill(page, '司機姓名', NAME + 'B')
  await L.fill(page, '身分證字號', VALID_IDS[1])
  record('只填必填欄位', await L.submit(page, net))

  await L.openCreate(page, '新增司機')
  await L.fill(page, '司機姓名', NAME + 'C')
  await L.fill(page, '身分證字號', VALID_IDS[0])
  record('身分證字號重複', await L.submit(page, net))

  await L.openCreate(page, '新增司機')
  await L.fill(page, '司機姓名', NAME + 'E')
  await L.fill(page, '身分證字號', VALID_IDS[2])
  await L.fill(page, '電子信箱', 'not-an-email')
  record('電子信箱格式錯誤', await L.submit(page, net))

  await L.goto(page, '/masters/drivers')
  await L.rowAction(page, NAME + 'A', '編輯')
  record('編輯載入值', await L.dumpForm(page))
  await L.fill(page, '司機姓名', NAME + 'A改')
  // el-date-picker 直接 fill('') 會打開日曆面板且不會收起，面板可能疊住對話框的儲存鈕擋住點擊；改用 clearDate 走 clearable 的 x 圖示。
  await L.clearDate(page, '駕照有效日期')
  record('編輯並清空駕照有效日期', await L.submit(page, net))

  // 身分證字號改為可在編輯畫面變更：留空代表不變更，改成別人的號碼要有可讀的錯誤。
  await L.goto(page, '/masters/drivers')
  await L.rowAction(page, NAME + 'A改', '編輯')
  await L.fill(page, '身分證字號', 'A202559751')
  record('編輯時身分證檢查碼錯誤', await L.submit(page, net))

  await L.goto(page, '/masters/drivers')
  await L.rowAction(page, NAME + 'A改', '編輯')
  await L.fill(page, '身分證字號', VALID_IDS[1])
  record('編輯改成其他司機已用的身分證', await L.submit(page, net))

  await L.goto(page, '/masters/drivers')
  await L.rowAction(page, NAME + 'A改', '編輯')
  await L.fill(page, '身分證字號', VALID_IDS[2])
  record('編輯變更身分證字號', await L.submit(page, net))

  // 指派車輛已從「指派車輛」對話框按鈕（現已移除）改為列表「目前指派車輛」欄位的行內單選 InlineOptionPicker。
  await L.goto(page, '/masters/drivers')
  const assignRow = await L.resolveRow(page, NAME + 'B')
  const assignTrigger = assignRow.locator('.inline-picker-trigger').first()
  await assignTrigger.click()
  await page.waitForTimeout(400)
  // clearable 面板第一項固定是「未指定」清除選項，實際車輛選項要排除它。
  const vehicleItems = page.locator('.inline-picker-popper:visible .inline-picker-item--radio:not(.inline-picker-item--clear)')
  const opts = await vehicleItems.allInnerTexts()
  record('可指派車輛選項', { options: opts.map(s => s.trim()) })
  if (opts.length > 0) {
    const before = net.calls.length
    // 單選模式一點選項就立刻送出並自動收起面板，不需要再點 trigger 關閉。
    await vehicleItems.first().click()
    await page.waitForTimeout(1200)
    record('指派車輛送出', { calls: net.calls.slice(before), toasts: await L.toasts(page) })
    await L.clearToasts(page)
  }

  await L.goto(page, '/masters/drivers')
  await L.rowAction(page, NAME + 'A改', '刪除')
  record('刪除司機', await L.confirmBox(page, net))

  // 刪除既有 demo 司機會影響出勤與搭乘紀錄，預設不跑。
  if (process.env.QA_DESTRUCTIVE === '1') {
    await L.goto(page, '/masters/drivers')
    await L.rowAction(page, '林彥衡', '刪除')
    record('刪除已有出勤紀錄的司機', await L.confirmBox(page, net))
  }
}
