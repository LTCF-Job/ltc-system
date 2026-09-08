exports.name = '司機管理 CRUD'

// 這些身分證字號與 apps/web/tests/unit/national-id.test.ts 的合法向量相同。
const VALID_IDS = ['A202559750', 'G121806465', 'K120098177']

exports.run = async ({ page, net, record, L }) => {
  const NAME = `${L.TAG}司機`
  await L.goto(page, '/masters/drivers')

  await L.openCreate(page, '新增司機')
  record('所屬區域下拉選項', { options: await L.selectOptions(page, '所屬區域') })

  await L.openCreate(page, '新增司機')
  record('空白送出', await L.submit(page, net))

  await L.openCreate(page, '新增司機')
  await L.fill(page, '司機姓名', NAME + 'A')
  await L.fill(page, '身分證字號', 'A202559751')
  await L.pick(page, '所屬區域', '苗栗縣')
  record('身分證檢查碼錯誤', await L.submit(page, net))

  await L.openCreate(page, '新增司機')
  await L.fill(page, '司機姓名', NAME + 'A')
  await L.fill(page, '身分證字號', VALID_IDS[0])
  await L.pick(page, '所屬區域', '苗栗縣')
  await L.fill(page, '電子信箱', 'qa-driver@example.com')
  await L.pick(page, '駕照類別', '大客車')
  await L.pickDate(page, '駕照有效日期', '2028-12-31')
  record('全欄位新增', await L.submit(page, net))

  await L.openCreate(page, '新增司機')
  await L.fill(page, '司機姓名', NAME + 'B')
  await L.fill(page, '身分證字號', VALID_IDS[1])
  await L.pick(page, '所屬區域', '新竹縣')
  record('只填必填欄位', await L.submit(page, net))

  await L.openCreate(page, '新增司機')
  await L.fill(page, '司機姓名', NAME + 'C')
  await L.fill(page, '身分證字號', VALID_IDS[0])
  await L.pick(page, '所屬區域', '苗栗縣')
  record('身分證字號重複', await L.submit(page, net))

  await L.openCreate(page, '新增司機')
  await L.fill(page, '司機姓名', NAME + 'D')
  await L.fill(page, '身分證字號', VALID_IDS[2])
  await L.pick(page, '所屬區域', '臺北市')
  record('選 DB 不允許的區域（臺北市）', await L.submit(page, net))

  await L.openCreate(page, '新增司機')
  await L.fill(page, '司機姓名', NAME + 'E')
  await L.fill(page, '身分證字號', VALID_IDS[2])
  await L.pick(page, '所屬區域', '苗栗縣')
  await L.fill(page, '電子信箱', 'not-an-email')
  record('電子信箱格式錯誤', await L.submit(page, net))

  await L.goto(page, '/masters/drivers')
  await L.rowAction(page, NAME + 'A', '編輯')
  record('編輯載入值', await L.dumpForm(page))
  await L.fill(page, '司機姓名', NAME + 'A改')
  await L.fill(page, '駕照有效日期', '')
  record('編輯並清空駕照有效日期', await L.submit(page, net))

  await L.goto(page, '/masters/drivers')
  await L.rowAction(page, NAME + 'B', '指派車輛')
  record('指派車輛對話框', await L.dumpForm(page))
  const opts = await L.selectOptions(page, '指派車輛')
  record('可指派車輛選項', { options: opts })
  await L.closeDialog(page)

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
