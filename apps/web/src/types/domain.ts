// 全系統列舉常數與正體中文標籤對照表（集中管理，禁止在視圖模板寫死）

export type UserRole = 'admin' | 'staff' | 'viewer'

export const ROLE_LABELS: Record<UserRole, string> = {
  admin: '系統管理者',
  staff: '行政人員',
  viewer: '檢視者'
}

export type Region = 'miaoli' | 'hsinchu'

export const REGION_LABELS: Record<Region, string> = {
  miaoli: '苗栗',
  hsinchu: '新竹'
}

export type CaseStatus = 'active' | 'suspended' | 'closed'

export const CASE_STATUS_LABELS: Record<CaseStatus, string> = {
  active: '在案',
  suspended: '暫停',
  closed: '停案'
}

export type Direction = 'outbound' | 'inbound'

export const DIRECTION_LABELS: Record<Direction, string> = {
  outbound: '去程',
  inbound: '回程'
}

export type Period = 'am' | 'pm'

export const PERIOD_LABELS: Record<Period, string> = {
  am: '上午',
  pm: '下午'
}

export type TripPattern = 1 | 2 | 4

export const TRIP_PATTERN_LABELS: Record<TripPattern, string> = {
  1: '單向 1 趟',
  2: '一般 2 趟',
  4: '四趟'
}

export type RideReportedStatus = 'boarded' | 'absent'

export const RIDE_REPORTED_LABELS: Record<RideReportedStatus, string> = {
  boarded: '有坐',
  absent: '沒坐'
}

export type EffectiveRideStatus = 'boarded' | 'absent' | 'unreported'

export const RIDE_STATUS_LABELS: Record<EffectiveRideStatus, string> = {
  boarded: '有坐',
  absent: '沒坐',
  unreported: '未回報'
}

export const CALENDAR_STATUS_SYMBOLS = {
  boarded: '√',
  absent: '／',
  unreported: '?',
  conflict: '!'
} as const

export type MappingStatus = 'pending' | 'mapped' | 'ignored'

export const MAPPING_STATUS_LABELS: Record<MappingStatus, string> = {
  pending: '待對應',
  mapped: '已對應',
  ignored: '已略過'
}

export type ColumnKind = 'meta' | 'ride' | 'issue' | 'unknown'

export const COLUMN_KIND_LABELS: Record<ColumnKind, string> = {
  meta: '系統欄',
  ride: '搭乘欄',
  issue: '問題回報欄',
  unknown: '未判定'
}

export type IngestSource = 'webhook' | 'sheets_sync' | 'manual'

export const INGEST_SOURCE_LABELS: Record<IngestSource, string> = {
  webhook: '即時推送',
  sheets_sync: '每日對帳',
  manual: '人工補登'
}

export type ExportJobType = 'gov_claim' | 'trip_summary' | 'hsinchu_schedule' | 'maintenance_blank'

export const EXPORT_JOB_TYPE_LABELS: Record<ExportJobType, string> = {
  gov_claim: '政府申報表',
  trip_summary: '車輛趟數表',
  hsinchu_schedule: '新竹接送時刻表',
  maintenance_blank: '空白保養表'
}

export type ExportJobStatus = 'pending' | 'running' | 'succeeded' | 'failed'

export const EXPORT_STATUS_LABELS: Record<ExportJobStatus, string> = {
  pending: '等待中',
  running: '處理中',
  succeeded: '已完成',
  failed: '失敗'
}

export type ServiceCategory = 1 | 2

export const SERVICE_CATEGORY_LABELS: Record<ServiceCategory, string> = {
  1: '補助',
  2: '自費'
}

export type ServiceUsageType = 1 | 2 | 3 | 4

export const SERVICE_USAGE_TYPE_LABELS: Record<ServiceUsageType, string> = {
  1: '社區式長照機構',
  2: '社區服務據點(不含身障類)',
  3: '輔具中心',
  4: '身障日間照顧服務'
}

export const CORRECTION_REASONS = [
  '司機填錯',
  '司機漏填',
  '事後補報',
  '混車確認',
  '其他'
] as const

export type NotificationTopic = 'missing_report' | 'driver_leave' | 'month_end' | 'export_failed'

export const NOTIFICATION_TOPIC_LABELS: Record<NotificationTopic, string> = {
  missing_report: '未回報催報',
  driver_leave: '司機請假通知',
  month_end: '月底提醒',
  export_failed: '匯出失敗通知'
}

export type AuditAction =
  | 'create'
  | 'update'
  | 'delete'
  | 'reveal_pii'
  | 'correct'
  | 'resolve_conflict'
  | 'export'
  | 'setting_change'
  | 'import'

export const AUDIT_ACTION_LABELS: Record<AuditAction, string> = {
  create: '主檔新增',
  update: '主檔修改',
  delete: '主檔停用/刪除',
  reveal_pii: '查看完整身分證',
  correct: '搭乘紀錄更正',
  resolve_conflict: '衝突裁決',
  export: '申報匯出',
  setting_change: '系統設定變更',
  import: '批次匯入'
}

export type AuditEntityType =
  | 'cases'
  | 'sites'
  | 'vehicles'
  | 'drivers'
  | 'ride_records'
  | 'notification_recipients'
  | 'export_jobs'
  | 'app_settings'

export const AUDIT_ENTITY_LABELS: Record<AuditEntityType, string> = {
  cases: '個案主檔',
  sites: '據點主檔',
  vehicles: '車輛主檔',
  drivers: '司機主檔',
  ride_records: '搭乘紀錄',
  notification_recipients: '通知收件人',
  export_jobs: '匯出工作',
  app_settings: '系統設定'
}

export type AttendanceStatus = 'work' | 'leave' | 'sick' | 'off'

export const ATTENDANCE_STATUS_LABELS: Record<AttendanceStatus, string> = {
  work: '出勤',
  leave: '事假',
  sick: '病假',
  off: '休假'
}

export const ATTENDANCE_STATUS_TAGS: Record<AttendanceStatus, string> = {
  work: 'success',
  leave: 'warning',
  sick: 'danger',
  off: 'info'
}


