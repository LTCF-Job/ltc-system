exports.name = '據點管理 CRUD'

exports.run = async ({ page, net, record, L }) => {
  const NAME = `${L.TAG}據點`
  await L.goto(page, '/masters/sites')

  await L.openCreate(page, '新增據點')
  record('空白送出', await L.submit(page, net))

  await L.openCreate(page, '新增據點')
  await L.fill(page, '據點名稱', NAME)
  record('只填名稱', await L.submit(page, net))

  await L.openCreate(page, '新增據點')
  await L.fill(page, '據點名稱', NAME)
  await L.pickOrCreate(page, '區域', '苗栗縣')
  record('名稱+區域未填地址', await L.submit(page, net))

  await L.openCreate(page, '新增據點')
  await L.fill(page, '據點名稱', NAME)
  await L.pickOrCreate(page, '區域', '苗栗縣')
  await L.fill(page, '據點地址', '苗栗縣測試路1號')
  record('完整新增（苗栗縣）', await L.submit(page, net))

  await L.openCreate(page, '新增據點')
  await L.fill(page, '據點名稱', NAME + '自訂區域')
  await L.pickOrCreate(page, '區域', '臺北市')
  await L.fill(page, '據點地址', '臺北市測試路2號')
  // 000048 migration 移除地區主檔後，區域不再受 DB 值域限制，這裡改成驗證自由輸入的新區域值仍能正常新增。
  record('自訂新區域（無主檔限制）', await L.submit(page, net))

  await L.openCreate(page, '新增據點')
  await L.fill(page, '據點名稱', NAME)
  await L.pickOrCreate(page, '區域', '苗栗縣')
  await L.fill(page, '據點地址', '重複測試')
  record('同名同區域重複', await L.submit(page, net))

  await L.goto(page, '/masters/sites')
  await L.rowAction(page, NAME, '編輯')
  record('編輯載入值', await L.dumpForm(page))
  await L.fill(page, '據點名稱', NAME + '改')
  await L.fill(page, '據點地址', '改後地址999號')
  await L.radio(page, '狀態', '停用')
  record('編輯送出', await L.submit(page, net))

  await L.goto(page, '/masters/sites')
  record('列表', { rows: await L.tableRows(page) })

  await L.rowAction(page, NAME + '自訂區域', '刪除')
  record('刪除未被引用的據點', await L.confirmBox(page, net))

  // 車輛（migration 000046）與照護人員（migration 000047）已改為自由輸入的據點名稱文字欄位，
  // 不再 FK 關聯 sites，刪除不會被擋。目前僅個案（cases.site_id, ON DELETE RESTRICT）仍會擋刪除，
  // 用 demo seed 既有的「苗栗縣站」測試；此分支仍會異動既有 demo 資料，預設不跑。
  if (process.env.QA_DESTRUCTIVE === '1') {
    await L.goto(page, '/masters/sites')
    await L.rowAction(page, '苗栗縣站', '刪除')
    record('刪除已被個案引用的據點', await L.confirmBox(page, net))
  }
}
