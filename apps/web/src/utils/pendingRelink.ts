import { ElMessage } from 'element-plus'
import type { PendingRelinkedMeta } from '@/api/client'

export function notifyPendingRelinked(meta?: PendingRelinkedMeta): void {
  const n = meta?.pendingRelinked
  if (n && n > 0) {
    ElMessage.info(`已自動關聯 ${n} 筆待維護資料`)
  }
}
