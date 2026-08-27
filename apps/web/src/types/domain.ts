// 全系統列舉常數與正體中文標籤對照表（集中管理，禁止在視圖模板寫死）

export type UserRole =
  | 'admin'
  | 'dispatcher'
  | 'driver'
  | 'staff'
  | 'viewer'
  | (string & {})

export const ROLE_LABELS: Record<string, string> = {
  admin: '系統管理員',
  dispatcher: '調度員',
  driver: '司機',
  staff: '行政人員',
  viewer: '檢視者'
}

export type RoleTagType = 'danger' | 'primary' | 'success' | 'warning' | 'info'

export interface RoleItem {
  id: string
  key: string
  name: string
  description: string
  tagType: RoleTagType
  isSystem: boolean
  permissions: SystemPermissions
  userCount?: number
  createdAt?: string
  updatedAt?: string
}

export type Region =
  | 'hsinchu'
  | 'hsinchu_city'
  | 'miaoli'
  | 'taipei'
  | 'new_taipei'
  | 'keelung'
  | 'taoyuan'
  | 'taichung'
  | 'changhua'
  | 'nantou'
  | 'yunlin'
  | 'chiayi_city'
  | 'chiayi'
  | 'tainan'
  | 'kaohsiung'
  | 'pingtung'
  | 'yilan'
  | 'hualien'
  | 'taitung'
  | 'penghu'
  | 'kinmen'
  | 'lienchiang'
  | (string & {})

export const REGION_LABELS: Record<string, string> = {
  hsinchu: '新竹縣',
  hsinchu_city: '新竹市',
  miaoli: '苗栗縣',
  taipei: '臺北市',
  new_taipei: '新北市',
  keelung: '基隆市',
  taoyuan: '桃園市',
  taichung: '臺中市',
  changhua: '彰化縣',
  nantou: '南投縣',
  yunlin: '雲林縣',
  chiayi_city: '嘉義市',
  chiayi: '嘉義縣',
  tainan: '臺南市',
  kaohsiung: '高雄市',
  pingtung: '屏東縣',
  yilan: '宜蘭縣',
  hualien: '花蓮縣',
  taitung: '臺東縣',
  penghu: '澎湖縣',
  kinmen: '金門縣',
  lienchiang: '連江縣'
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
  | 'login'

export const AUDIT_ACTION_LABELS: Record<AuditAction, string> = {
  create: '主檔新增',
  update: '主檔修改',
  delete: '主檔停用/刪除',
  reveal_pii: '查看完整身分證',
  correct: '搭乘紀錄更正',
  resolve_conflict: '衝突裁決',
  export: '申報匯出',
  setting_change: '系統設定變更',
  import: '批次匯入',
  login: '使用者登入'
}

export type AuditEntityType =
  | 'cases'
  | 'sites'
  | 'vehicles'
  | 'drivers'
  | 'regions'
  | 'ride_records'
  | 'notification_recipients'
  | 'export_jobs'
  | 'app_settings'
  | 'users'
  | 'roles'
  | 'auth'
  | 'google_forms'

export const AUDIT_ENTITY_LABELS: Record<AuditEntityType, string> = {
  cases: '個案主檔',
  sites: '據點主檔',
  vehicles: '車輛主檔',
  drivers: '司機主檔',
  regions: '地區主檔',
  ride_records: '搭乘紀錄',
  notification_recipients: '通知收件人',
  export_jobs: '匯出工作',
  app_settings: '系統設定',
  users: '使用者帳號',
  roles: '角色身分',
  auth: '登入驗證',
  google_forms: 'Google 表單'
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

// 系統區塊模組定義
export interface SystemModuleInfo {
  id: string
  name: string
  category: 'overview' | 'masters' | 'operations' | 'reports' | 'system'
  categoryName: string
  description?: string
}

export const SYSTEM_MODULES: SystemModuleInfo[] = [
  { id: 'dashboard', name: '總覽儀表板', category: 'overview', categoryName: '首頁儀表' },
  { id: 'masters_regions', name: '地區管理', category: 'masters', categoryName: '主檔資料' },
  { id: 'masters_cases', name: '個案管理', category: 'masters', categoryName: '主檔資料' },
  { id: 'masters_sites', name: '據點管理', category: 'masters', categoryName: '主檔資料' },
  { id: 'masters_vehicles', name: '車輛管理', category: 'masters', categoryName: '主檔資料' },
  { id: 'masters_drivers', name: '司機管理', category: 'masters', categoryName: '主檔資料' },
  { id: 'forms_sync', name: '表單同步管理', category: 'operations', categoryName: '表單與搭乘' },
  { id: 'forms_mappings', name: '欄位對應設定', category: 'operations', categoryName: '表單與搭乘' },
  { id: 'rides_calendar', name: '搭乘月曆表', category: 'operations', categoryName: '表單與搭乘' },
  { id: 'rides_issues', name: '異常集中處理', category: 'operations', categoryName: '表單與搭乘' },
  { id: 'rides_missing', name: '未回報清單', category: 'operations', categoryName: '表單與搭乘' },
  { id: 'reports_trip_summary', name: '車輛趟數表', category: 'reports', categoryName: '營運報表' },
  { id: 'reports_hsinchu_schedule', name: '新竹接送時刻表', category: 'reports', categoryName: '營運報表' },
  { id: 'vehicles_maintenance', name: '車輛維修保養', category: 'operations', categoryName: '車輛維運' },
  { id: 'attendance_fuel', name: '出勤與油資管理', category: 'operations', categoryName: '車輛維運' },
  { id: 'exports', name: '政府申報匯出', category: 'reports', categoryName: '申報匯出' },
  { id: 'audit_logs', name: '系統操作紀錄', category: 'system', categoryName: '系統管理' },
  { id: 'settings_notifications', name: '通知收件人管理', category: 'system', categoryName: '系統管理' },
  { id: 'settings_users', name: '使用者與權限管理', category: 'system', categoryName: '系統管理' },
  { id: 'settings_roles', name: '角色身分管理', category: 'system', categoryName: '系統管理' }
]

export interface ModulePermission {
  view: boolean
  edit: boolean
}

export type SystemPermissions = Record<string, ModulePermission>

// 角色預設權限配置表
export const DEFAULT_ROLE_PERMISSIONS: Record<UserRole, SystemPermissions> = {
  admin: SYSTEM_MODULES.reduce((acc, m) => {
    acc[m.id] = { view: true, edit: true }
    return acc
  }, {} as SystemPermissions),

  dispatcher: {
    dashboard: { view: true, edit: false },
    masters_regions: { view: true, edit: true },
    masters_cases: { view: true, edit: true },
    masters_sites: { view: true, edit: true },
    masters_vehicles: { view: true, edit: true },
    masters_drivers: { view: true, edit: true },
    forms_sync: { view: true, edit: true },
    forms_mappings: { view: true, edit: true },
    rides_calendar: { view: true, edit: true },
    rides_issues: { view: true, edit: true },
    rides_missing: { view: true, edit: true },
    reports_trip_summary: { view: true, edit: false },
    reports_hsinchu_schedule: { view: true, edit: false },
    vehicles_maintenance: { view: true, edit: true },
    attendance_fuel: { view: true, edit: true },
    exports: { view: true, edit: true },
    audit_logs: { view: false, edit: false },
    settings_notifications: { view: true, edit: false },
    settings_users: { view: false, edit: false },
    settings_roles: { view: false, edit: false }
  },

  driver: {
    dashboard: { view: true, edit: false },
    masters_regions: { view: false, edit: false },
    masters_cases: { view: false, edit: false },
    masters_sites: { view: false, edit: false },
    masters_vehicles: { view: true, edit: false },
    masters_drivers: { view: true, edit: false },
    forms_sync: { view: false, edit: false },
    forms_mappings: { view: false, edit: false },
    rides_calendar: { view: true, edit: false },
    rides_issues: { view: false, edit: false },
    rides_missing: { view: false, edit: false },
    reports_trip_summary: { view: false, edit: false },
    reports_hsinchu_schedule: { view: false, edit: false },
    vehicles_maintenance: { view: true, edit: true },
    attendance_fuel: { view: true, edit: true },
    exports: { view: false, edit: false },
    audit_logs: { view: false, edit: false },
    settings_notifications: { view: false, edit: false },
    settings_users: { view: false, edit: false },
    settings_roles: { view: false, edit: false }
  },

  staff: {
    dashboard: { view: true, edit: false },
    masters_regions: { view: true, edit: true },
    masters_cases: { view: true, edit: true },
    masters_sites: { view: true, edit: true },
    masters_vehicles: { view: true, edit: true },
    masters_drivers: { view: true, edit: true },
    forms_sync: { view: true, edit: true },
    forms_mappings: { view: true, edit: true },
    rides_calendar: { view: true, edit: true },
    rides_issues: { view: true, edit: true },
    rides_missing: { view: true, edit: true },
    reports_trip_summary: { view: true, edit: false },
    reports_hsinchu_schedule: { view: true, edit: false },
    vehicles_maintenance: { view: true, edit: true },
    attendance_fuel: { view: true, edit: true },
    exports: { view: true, edit: true },
    audit_logs: { view: false, edit: false },
    settings_notifications: { view: true, edit: false },
    settings_users: { view: false, edit: false },
    settings_roles: { view: false, edit: false }
  },

  viewer: SYSTEM_MODULES.reduce((acc, m) => {
    acc[m.id] = {
      view: !['audit_logs', 'settings_users', 'settings_roles'].includes(m.id),
      edit: false
    }
    return acc
  }, {} as SystemPermissions)
}

// 稽核日誌欄位名稱中文化字典
export const AUDIT_FIELD_LABELS: Record<string, string> = {
  id: '資料編號',
  name: '姓名／名稱',
  code: '代碼／識別碼',
  title: '標題／名稱',
  region: '所屬地區',
  status: '營運狀態',
  plate_no: '車牌號碼',
  plateNo: '車牌號碼',
  brandModel: '廠牌型號',
  brand_model: '廠牌型號',
  seatCapacity: '一般座位數',
  seat_capacity: '一般座位數',
  wheelchairCapacity: '輪椅座位數',
  wheelchair_capacity: '輪椅座位數',
  displayName: '顯示名稱',
  display_name: '顯示名稱',
  address: '地址',
  homeAddress: '住家地址',
  home_address: '住家地址',
  phone: '聯絡電話',
  email: '電子郵件',
  contactPerson: '聯絡人',
  contact_person: '聯絡人',
  contactPhone: '聯絡電話',
  contact_phone: '聯絡電話',
  serviceArea: '服務區域',
  service_area: '服務區域',
  nationalIdMasked: '身分證號（去識別）',
  national_id_masked: '身分證號（去識別）',
  ltcLevel: '長照等級',
  ltc_level: '長照等級',
  serviceCategory: '服務類別',
  service_category: '服務類別',
  serviceUsageType: '使用型態',
  service_usage_type: '使用型態',
  claimStartDate: '申報起始日',
  claim_start_date: '申報起始日',
  claimEndDate: '申報終止日',
  claim_end_date: '申報終止日',
  weekdays: '排班星期',
  tripPattern: '趟次模式',
  trip_pattern: '趟次模式',
  unitPrice: '每趟單價',
  unit_price: '每趟單價',
  distanceKm: '服務里程 (km)',
  distance_km: '服務里程 (km)',
  serviceDurationMin: '車程時間 (分)',
  service_duration_min: '車程時間 (分)',
  serviceCode: '服務代碼',
  service_code: '服務代碼',
  note: '備註說明',
  effectiveStatus: '有效搭乘狀態',
  effective_status: '有效搭乘狀態',
  mergedStatus: '合併回報狀態',
  merged_status: '合併回報狀態',
  serviceDate: '服務日期',
  service_date: '服務日期',
  legSeq: '趟次序號',
  leg_seq: '趟次序號',
  departTime: '預計發車時間',
  depart_time: '預計發車時間',
  arriveTime: '預計抵達時間',
  arrive_time: '預計抵達時間',
  departTimeOverride: '更正發車時間',
  depart_time_override: '更正發車時間',
  reason: '更正原因／異動說明',
  vehicleId: '指派車輛編號',
  vehicle_id: '指派車輛編號',
  vehicleName: '指派車輛名稱',
  vehicle_name: '指派車輛名稱',
  driverId: '指派司機編號',
  driver_id: '指派司機編號',
  driverName: '指派司機姓名',
  driver_name: '指派司機姓名',
  topic: '通知主題',
  channel: '發送管道',
  active: '啟用狀態',
  role: '使用者角色',
  password: '密碼設定',
  customPermissions: '自訂模組權限',
  custom_permissions: '自訂模組權限',
  description: '說明描述',
  sortOrder: '排序順序',
  sort_order: '排序順序',
  openDays: '營運開放日',
  open_days: '營運開放日',
  formId: '表單代碼',
  form_id: '表單代碼',
  sheetUrl: 'Google 試算表網址',
  sheet_url: 'Google 試算表網址',
  sheetTab: '工作表分頁',
  sheet_tab: '工作表分頁',
  syncedMonths: '已同步月份',
  synced_months: '已同步月份',
  lastSyncedAt: '最後同步時間',
  last_synced_at: '最後同步時間',
  correctionReason: '更正原因',
  correction_reason: '更正原因',
  hasConflict: '是否有衝突',
  has_conflict: '是否有衝突',
  anomalyFlags: '異常標記',
  anomaly_flags: '異常標記',
  periodYm: '申報年月',
  period_ym: '申報年月',
  totalCases: '總個案數',
  total_cases: '總個案數',
  totalRows: '總資料筆數',
  total_rows: '總資料筆數',
  fileChecksum: '檔案校驗碼',
  file_checksum: '檔案校驗碼',
  importedCount: '成功匯入筆數',
  imported_count: '成功匯入筆數',
  errorCount: '失敗筆數',
  error_count: '失敗筆數',
  warningCount: '警告筆數',
  warning_count: '警告筆數',
  loginTime: '登入時間',
  login_time: '登入時間',
  inspectionDueDate: '驗車到期日',
  inspection_due_date: '驗車到期日',
  insuranceDueDate: '保險到期日',
  insurance_due_date: '保險到期日',
  licenseType: '駕照種類',
  license_type: '駕照種類',
  licenseExpiryDate: '駕照到期日',
  license_expiry_date: '駕照到期日',
  updatedAt: '更新時間',
  updated_at: '更新時間',
  createdAt: '建立時間',
  created_at: '建立時間'
}

// 稽核日誌欄位所屬區塊字典
export const AUDIT_FIELD_SECTIONS: Record<string, string> = {
  id: '基本資料',
  name: '基本資料',
  code: '基本資料',
  title: '基本資料',
  displayName: '基本資料',
  display_name: '基本資料',
  plateNo: '基本規格',
  plate_no: '基本規格',
  brandModel: '基本規格',
  brand_model: '基本規格',
  seatCapacity: '基本規格',
  seat_capacity: '基本規格',
  wheelchairCapacity: '基本規格',
  wheelchair_capacity: '基本規格',
  address: '聯絡資訊',
  homeAddress: '聯絡資訊',
  home_address: '聯絡資訊',
  phone: '聯絡資訊',
  email: '聯絡資訊',
  contactPerson: '聯絡資訊',
  contact_person: '聯絡資訊',
  contactPhone: '聯絡資訊',
  contact_phone: '聯絡資訊',
  serviceArea: '營運所屬',
  service_area: '營運所屬',
  nationalIdMasked: '個資識別',
  national_id_masked: '個資識別',
  description: '基本資料',
  note: '備註說明',

  // 長照與補助
  ltcLevel: '長照與補助',
  ltc_level: '長照與補助',
  serviceCategory: '長照與補助',
  service_category: '長照與補助',
  serviceUsageType: '長照與補助',
  service_usage_type: '長照與補助',
  claimStartDate: '長照與補助',
  claim_start_date: '長照與補助',
  claimEndDate: '長照與補助',
  claim_end_date: '長照與補助',
  serviceCode: '長照與補助',
  service_code: '長照與補助',
  unitPrice: '費用與里程',
  unit_price: '費用與里程',
  distanceKm: '費用與里程',
  distance_km: '費用與里程',
  serviceDurationMin: '費用與里程',
  service_duration_min: '費用與里程',

  // 營運與指派
  region: '營運所屬',
  status: '營運狀態',
  active: '營運狀態',
  openDays: '營運排程',
  open_days: '營運排程',
  sortOrder: '顯示順序',
  sort_order: '顯示順序',
  vehicleId: '車輛與司機指派',
  vehicle_id: '車輛與司機指派',
  vehicleName: '車輛與司機指派',
  vehicle_name: '車輛與司機指派',
  driverId: '車輛與司機指派',
  driver_id: '車輛與司機指派',
  driverName: '車輛與司機指派',
  driver_name: '車輛與司機指派',
  inspectionDueDate: '證照與檢驗',
  inspection_due_date: '證照與檢驗',
  insuranceDueDate: '證照與檢驗',
  insurance_due_date: '證照與檢驗',
  licenseType: '證照與檢驗',
  license_type: '證照與檢驗',
  licenseExpiryDate: '證照與檢驗',
  license_expiry_date: '證照與檢驗',

  // 搭乘與回報
  serviceDate: '搭乘資訊',
  service_date: '搭乘資訊',
  legSeq: '搭乘資訊',
  leg_seq: '搭乘資訊',
  departTime: '班次時間',
  depart_time: '班次時間',
  arriveTime: '班次時間',
  arrive_time: '班次時間',
  departTimeOverride: '更正資訊',
  depart_time_override: '更正資訊',
  effectiveStatus: '搭乘狀態',
  effective_status: '搭乘狀態',
  mergedStatus: '回報狀態',
  merged_status: '回報狀態',
  reason: '更正紀錄',
  correctionReason: '更正紀錄',
  correction_reason: '更正紀錄',
  hasConflict: '衝突標記',
  has_conflict: '衝突標記',
  anomalyFlags: '異常標記',
  anomaly_flags: '異常標記',

  // 帳號安全與登入
  role: '帳號權限',
  password: '帳號安全',
  customPermissions: '模組權限',
  custom_permissions: '模組權限',
  loginTime: '登入活動',
  login_time: '登入活動',

  // 匯出與申報
  periodYm: '申報作業',
  period_ym: '申報作業',
  totalCases: '申報統計',
  total_cases: '申報統計',
  totalRows: '申報統計',
  total_rows: '申報統計',
  fileChecksum: '檔案安全',
  file_checksum: '檔案安全',

  // 表單與同步
  formId: '表單整合',
  form_id: '表單整合',
  sheetUrl: '試算表設定',
  sheet_url: '試算表設定',
  sheetTab: '工作表分頁',
  sheet_tab: '工作表分頁',
  syncedMonths: '同步記錄',
  synced_months: '同步記錄',
  lastSyncedAt: '同步記錄',
  last_synced_at: '同步記錄',
  importedCount: '匯入統計',
  imported_count: '匯入統計',
  errorCount: '匯入統計',
  error_count: '匯入統計',
  warningCount: '匯入統計',
  warning_count: '匯入統計',

  // 通知與系統
  topic: '通知設定',
  channel: '通知設定',
  updatedAt: '系統記錄',
  updated_at: '系統記錄',
  createdAt: '系統記錄',
  created_at: '系統記錄'
}

// 稽核日誌數值常數中文化字典
export const AUDIT_VALUE_LABELS: Record<string, string> = {
  boarded: '已搭乘',
  absent: '未搭乘／缺席',
  cancelled: '已取消',
  pending: '未確認',
  unreported: '未回報',
  normal: '正常搭乘',
  mixed: '混車搭乘',
  active: '啟用／正常營運',
  inactive: '已停用',
  suspended: '暫停服務',
  work: '正常出勤',
  leave: '事假／請假',
  sick: '病假',
  off: '排休',
  admin: '系統管理員',
  dispatcher: '調度員',
  driver: '司機',
  staff: '行政人員',
  viewer: '檢視者'
}


