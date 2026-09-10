// 2026-09-09 commit d59060f「前端隱藏批次匯入與下載範本按鈕，保留後端完整 API 功能」：
// apps/web/src/views/masters/CaregiverListView.vue 的 #actions 只剩「新增照護人員」一顆按鈕，
// openImportDialog() 仍存在但模板已無任何按鈕呼叫它，「批次匯入照護人員」觸發入口確實已從畫面移除。
// 這支 suite 找不到按鈕必定逾時，故整支跳過；若產品之後重新開放前端匯入入口，再復原本檔案。
exports.name = '照護人員批次匯入（前端入口已隱藏，跳過）'

exports.run = async ({ record }) => {
  record('skip', { reason: '批次匯入入口已於 commit d59060f 從 CaregiverListView.vue 隱藏，僅後端 API 保留' })
}
