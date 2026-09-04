import type {
  UserRole,
  Region,
  CaseStatus,
  Direction,
  TripPattern,
  CalendarTripPattern,
  EffectiveRideStatus,
  RideReportedStatus,
  MappingStatus,
  ColumnKind,
  ExportJobType,
  ExportJobStatus,
  ExportMode,
  ServiceCategory,
  ServiceUsageType,
  NotificationTopic,
  AuditAction,
  AuditEntityType,
  SystemPermissions,
  CaregiverType,
  DriverLicenseClass
} from './domain'

// 共通分頁與錯誤結構
export interface PaginationMeta {
  page: number
  pageSize: number
  total: number
  totalPages: number
}

export interface Paged<T> {
  data: T[]
  meta: PaginationMeta
}

export interface ErrorDetail {
  field?: string
  reason: string
}

export interface ApiError {
  code: string
  message: string
  details?: ErrorDetail[]
}

export interface ApiResponse<T> {
  data: T
  meta?: PaginationMeta
  error?: ApiError
}

// 使用者與認證
export interface UserDTO {
  id: string
  email: string
  displayName: string
  role: UserRole
  phone?: string
  status?: 'active' | 'inactive'
  customPermissions?: SystemPermissions | null
  lastLoginAt?: string
  createdAt?: string
}

export interface CreateUserRequest {
  email: string
  displayName: string
  role: UserRole
  phone?: string
  password?: string
  status?: 'active' | 'inactive'
  customPermissions?: SystemPermissions | null
}

export interface UpdateUserRequest {
  email?: string
  displayName?: string
  role?: UserRole
  phone?: string
  status?: 'active' | 'inactive'
  customPermissions?: SystemPermissions | null
}

export interface ChangePasswordRequest {
  oldPassword?: string
  newPassword?: string
}

export interface AuthSession {
  user: UserDTO
  accessToken: string
}

// 角色身分管理
export interface RoleDTO {
  id: string
  key: string
  name: string
  description: string
  tagType: 'danger' | 'primary' | 'success' | 'warning' | 'info'
  isSystem: boolean
  permissions: SystemPermissions
  userCount?: number
  createdAt?: string
  updatedAt?: string
}

export interface CreateRoleRequest {
  key?: string
  name: string
  description?: string
  tagType?: 'danger' | 'primary' | 'success' | 'warning' | 'info'
  permissions: SystemPermissions
}

export interface UpdateRoleRequest {
  name?: string
  description?: string
  tagType?: 'danger' | 'primary' | 'success' | 'warning' | 'info'
  permissions?: SystemPermissions
}

// 個案與排班
export type ScheduleMode = 'monthly' | 'by_weekday' | 'unified'

export interface WeekdayScheduleConfig {
  weekday: number
  label?: string
  tripCount: number
  departTime?: string
  returnTime?: string
  vehicleId?: string
}

export interface DayScheduleConfig {
  date: string
  tripCount: number
  departTime?: string
  returnTime?: string
  vehicleId?: string
  note?: string
}

export interface ScheduleLegDTO {
  id: string
  legSeq: number
  direction: Direction
  departTime: string
  arriveTime?: string
  runNo: number
  vehicleId: string
  vehicleName?: string
}

export interface CaseScheduleDTO {
  id: string
  caseId: string
  siteId: string
  siteName?: string
  effectiveFrom: string
  effectiveTo?: string
  weekdays: number[]
  tripPattern: TripPattern
  unitPrice: number
  distanceKm: number
  serviceDurationMin: number
  serviceCode: string
  note?: string
  legs: ScheduleLegDTO[]
  scheduleMode?: ScheduleMode
  weeklyConfigs?: WeekdayScheduleConfig[]
  monthlyConfigs?: Record<string, DayScheduleConfig>
}

export interface CaseDTO {
  id: string
  name: string
  nameNormalized?: string
  nationalId?: string
  homeAddress?: string
  region?: Region
  ltcLevel?: string
  serviceCategory?: ServiceCategory
  serviceUsageType?: ServiceUsageType
  claimEndDate?: string
  status: CaseStatus
  householdType?: string
  gender?: string
  birthDate?: string
  careContactRole?: string
  careContactName?: string
  registeredAddress?: string
  remarks?: string
  siteId?: string
  siteName?: string
  siteNameRaw?: string
  outboundVehicleId?: string
  outboundVehicle?: string
  outboundVehicleNameRaw?: string
  inboundVehicleId?: string
  inboundVehicle?: string
  inboundVehicleNameRaw?: string
  createdAt: string
  updatedAt: string
  activeSchedule?: CaseScheduleDTO
}

export interface CreateCaseRequest {
  name: string
  nationalId?: string
  homeAddress?: string
  region?: Region
  ltcLevel?: string
  serviceCategory?: ServiceCategory
  serviceUsageType?: ServiceUsageType
  claimEndDate?: string
  status?: CaseStatus
  householdType?: string
  gender?: string
  birthDate?: string
  careContactRole?: string
  careContactName?: string
  registeredAddress?: string
  remarks?: string
}

// 三欄位皆選填：未帶入的欄位維持既有關聯不變，僅更新有帶值的那一項
export interface UpdateCaseTransportPreferenceRequest {
  siteId?: string
  outboundVehicleId?: string
  inboundVehicleId?: string
}

export interface UpdateCaseRequest extends Partial<CreateCaseRequest> { }

export interface CreateScheduleRequest {
  siteId: string
  effectiveFrom: string
  effectiveTo?: string
  weekdays: number[]
  tripPattern: TripPattern
  unitPrice: number
  distanceKm: number
  serviceDurationMin: number
  serviceCode: string
  note?: string
  scheduleMode?: ScheduleMode
  weeklyConfigs?: WeekdayScheduleConfig[]
  monthlyConfigs?: Record<string, DayScheduleConfig>
  legs: Array<{
    legSeq: number
    direction: Direction
    departTime: string
    arriveTime?: string
    runNo: number
    vehicleId: string
  }>
}

// 主檔：區域、單位、車輛、司機
export interface RegionDTO {
  id: string
  name: string
  description?: string
  status: 'active' | 'inactive'
  sortOrder: number
  createdAt: string
  updatedAt: string
}

export interface CreateRegionRequest {
  name: string
  description?: string
  status?: 'active' | 'inactive'
  sortOrder?: number
}

export interface UpdateRegionRequest {
  name?: string
  description?: string
  status?: 'active' | 'inactive'
  sortOrder?: number
}

export interface SiteDTO {
  id: string
  name: string
  region: Region
  address: string
  openDays: number[]
  status: 'active' | 'inactive'
  createdAt: string
}


export interface CreateSiteRequest {
  name: string
  region: Region
  address: string
  openDays: number[]
  status?: 'active' | 'inactive'
}

export interface UpdateSiteRequest extends Partial<CreateSiteRequest> { }

export interface VehicleDTO {
  id: string
  plateNo: string
  displayName: string
  siteId: string | null
  siteName: string
  // 由所屬單位帶出的唯讀區域，車輛本身不再自存
  region: Region | ''
  brand: string
  model: string
  // 出廠年月，格式為 YYYY-MM
  manufactureYm: string
  compulsoryInsuranceExpiry: string | null
  passengerInsuranceExpiry: string | null
  thirdPartyInsuranceExpiry: string | null
  lastInspectionDate: string | null
  wheelchairAccessible: boolean | null
  status: 'active' | 'inactive'
  createdAt: string
  drivers?: VehicleDriverDTO[]
}

// 掛在車輛上的司機摘要。一台車可以有多位司機，一位司機同期只會有一台車。
export interface VehicleDriverDTO {
  id: string
  code?: string
  name: string
}

export interface CreateVehicleRequest {
  plateNo: string
  displayName: string
  siteId: string
  brand?: string | null
  model?: string | null
  manufactureYm?: string | null
  compulsoryInsuranceExpiry?: string | null
  passengerInsuranceExpiry?: string | null
  thirdPartyInsuranceExpiry?: string | null
  lastInspectionDate?: string | null
  wheelchairAccessible: boolean
  status?: 'active' | 'inactive'
}

export interface UpdateVehicleRequest extends Partial<CreateVehicleRequest> { }

export interface DriverAssignmentDTO {
  id: string
  driverId: string
  vehicleId: string
  vehicleName?: string
  vehiclePlateNo?: string
  plateNo?: string
  startDate: string
  endDate?: string
}

export interface DriverDTO {
  id: string
  name: string
  nameNormalized?: string
  nationalId: string
  region: string
  phone?: string
  email?: string
  status: 'active' | 'inactive'
  // 駕照類別與有效日期為選填，未補登時為 null
  licenseClass?: DriverLicenseClass | null
  licenseExpiryDate?: string | null
  createdAt: string
  assignments?: DriverAssignmentDTO[]
}

export interface CreateDriverRequest {
  name: string
  nationalId: string
  region: string
  phone?: string
  email?: string
  status?: 'active' | 'inactive'
  licenseClass?: DriverLicenseClass | null
  licenseExpiryDate?: string | null
}

export interface UpdateDriverRequest extends Partial<CreateDriverRequest> { }

// 照護人員：siteId 為空但 siteNameRaw 有值時，代表匯入時的單位名稱尚未關聯既有單位
export interface CaregiverDTO {
  id: string
  siteId?: string
  siteName?: string
  siteNameRaw?: string
  name: string
  type: CaregiverType
  contact?: string
  notes?: string
  status: 'active' | 'inactive'
  createdAt: string
  updatedAt: string
}

export interface CreateCaregiverRequest {
  siteId?: string
  name: string
  type: CaregiverType
  contact?: string
  notes?: string
  status?: 'active' | 'inactive'
}

export interface UpdateCaregiverRequest extends Partial<CreateCaregiverRequest> { }

// 司機接送匯報表與欄位對應
export interface DriverReportFormDTO {
  id: string
  vehicleId: string
  vehicleName: string
  title: string
  region?: Region
  lastImportedAt?: string | null
  totalColumns: number
  mappedColumns: number
  pendingColumns: number
  submissionCount: number
  status: string
}

export interface CreateDriverReportFormRequest {
  vehicleId: string
  title: string
}

// 某份匯報表在某個月已匯入的統計；月份由 service_date 推得，不是資料庫欄位
export interface DriverReportImportedMonthDTO {
  formId: string
  yearMonth: string
  submissionCount: number
  lastImportedAt: string
}

export interface DriverReportColumnDTO {
  id: string
  formId: string
  formTitle: string
  vehicleName: string
  columnIndex: number
  columnHeader: string
  cleanedName: string
  kind: ColumnKind
  mappingStatus: MappingStatus
  caseId?: string | null
  caseName?: string | null
  legSeq?: number | null
  suggestedCaseId?: string | null
  suggestedCaseName?: string | null
  suggestedLegSeq?: number | null
  suggestionScore: number
}

// 匯入預覽的欄位段：每個未對應欄位在此就地確認個案與趟次
export interface DriverReportPreviewColumn {
  columnId?: string
  columnIndex: number
  columnHeader: string
  cleanedName: string
  direction?: Direction
  mappingStatus: MappingStatus
  caseId?: string
  caseName?: string
  legSeq?: number
  suggestedCaseId?: string
  suggestedCaseName?: string
  suggestedLegSeq?: number
  suggestionScore: number
  boardedCount: number
  absentCount: number
}

// 匯入預覽的資料段：一列一天
export interface DriverReportPreviewRow {
  rowIndex: number
  reportDate: string
  serviceDate: string
  driverRaw: string
  driverId?: string
  driverName?: string
  remark?: string
  boardedCount: number
  absentCount: number
  errorMessage?: string
  warningMessage?: string
}

export interface DriverReportPreviewDTO {
  formId: string
  vehicleId: string
  vehicleName: string
  totalRows: number
  validRows: number
  errorRows: number
  warningRows: number
  unmappedColumns: number
  columns: DriverReportPreviewColumn[]
  previewRows: DriverReportPreviewRow[]
  errors: Array<{ rowIndex: number; field?: string; message: string }>
  warnings: Array<{ rowIndex: number; field?: string; message: string }>
}

export interface DriverReportColumnDecision {
  columnHeader: string
  mappingStatus: MappingStatus
  caseId?: string | null
  legSeq?: number | null
}

export interface DriverReportCommitResultDTO {
  importedRows: number
  rideRecordRows: number
  mappedColumns: number
  skippedRows: Array<{ rowIndex: number; reportDate: string; reasons: string[] }>
  warnings?: Array<{ rowIndex: number; field?: string; message: string }>
}

export interface UpdateColumnMappingRequest {
  caseId?: string
  legSeq?: number
  mappingStatus: MappingStatus
}

export interface BatchMappingRequest {
  mappings: Array<{
    columnId: string
    caseId?: string
    legSeq?: number
    mappingStatus: MappingStatus
  }>
}

// 待維護資料頁籤：以匯報表列（一天一列）為單位，一列可能同時有個案欄位與駕駛人兩種問題
export interface SubmissionReviewDTO {
  submissionId: string
  formTitle: string
  vehicleName: string
  serviceDate: string
  caseIssues: DriverReportColumnDTO[]
  driverIssue?: { driverNameRaw: string }
}

export interface BindDriverRequest {
  driverNameRaw: string
  driverId: string
}

// 總覽頁鑽取單一匯報表、單一月份時的完整內容：逐日回報明細與展開後的個案搭乘紀錄
export interface DriverReportMonthSubmissionDTO {
  serviceDate: string
  driverNameRaw: string
  remark: string
  answers: Record<string, string>
}

export interface DriverReportMonthRideEntryDTO {
  caseId: string
  caseName: string
  serviceDate: string
  legSeq: number
  reported: string
  driverId?: string | null
  driverName: string
  vehicleId: string
}

export interface DriverReportMonthDetailDTO {
  submissions: DriverReportMonthSubmissionDTO[]
  rideEntries: DriverReportMonthRideEntryDTO[]
}

// 搭乘紀錄與月曆矩陣
export interface RideSourceDTO {
  id: string
  submissionId: string
  vehicleId: string
  vehicleName?: string
  driverId?: string
  driverName?: string
  reported: RideReportedStatus
  submittedAt: string
}

export interface RideRecordDTO {
  id: string
  caseId: string
  caseName?: string
  serviceDate: string
  legSeq: number
  direction?: Direction
  mergedStatus: EffectiveRideStatus
  effectiveStatus: EffectiveRideStatus
  hasConflict: boolean
  vehicleId?: string
  vehicleName?: string
  driverId?: string
  driverName?: string
  departTimeOverride?: string
  durationMinOverride?: number
  scheduledDepartTime?: string
  scheduledDurationMin?: number
  notClaimedAa09: boolean
  correctedBy?: string
  correctedByName?: string
  correctedAt?: string
  correctionReason?: string
  sourceChanged?: boolean
  sources: RideSourceDTO[]
}

export interface RideCalendarCellDTO {
  date: string
  dayOfWeek: number
  isExpected: boolean
  expectedTripCount?: number
  isHoliday?: boolean
  holidayName?: string
  records: RideRecordDTO[]
}

export interface CaseRideCalendarRowDTO {
  caseId: string
  caseName: string
  region: Region
  tripPattern: CalendarTripPattern
  tripPatternText?: string
  days: Record<string, RideCalendarCellDTO>
}

export interface RideCalendarMatrixDTO {
  month: string
  totalCases: number
  daysInMonth: number
  cases: CaseRideCalendarRowDTO[]
}

export interface PatchRideRequest {
  effectiveStatus?: EffectiveRideStatus
  vehicleId?: string
  driverId?: string
  departTimeOverride?: string | null
  durationMinOverride?: number | null
  legSeq?: number
  notClaimedAa09?: boolean
  reason?: string
}

export interface ResolveConflictRequest {
  vehicleId: string
  driverId: string
  reason?: string
}

export interface ManualReportRideRequest {
  id?: string
  caseId: string
  serviceDate: string
  legSeq: number
  effectiveStatus: EffectiveRideStatus
  vehicleId?: string
  driverId?: string
  departTimeOverride?: string | null
  durationMinOverride?: number | null
  notClaimedAa09?: boolean
  reason?: string
}

export interface IssueRideDTO {
  id: string
  caseId: string
  caseName: string
  serviceDate: string
  legSeq: number
  issueType: 'conflict' | 'unreported' | 'import_error'
  hasConflict: boolean
  description: string
  vehicles?: string[]
  sources?: RideSourceDTO[]
  rawPayload?: string
}

// 申報匯出
export interface PrecheckItemDTO {
  level: 'error' | 'warning' | 'info'
  code: string
  message: string
  details?: Array<{
    caseId?: string
    caseName?: string
    field?: string
    serviceDate?: string
    rideId?: string
    description?: string
  }>
}

export interface PrecheckResultDTO {
  passed: boolean
  hasErrors: boolean
  hasWarnings: boolean
  summary: {
    totalErrors: number
    totalWarnings: number
    totalInfos: number
  }
  items: PrecheckItemDTO[]
}

export interface CreateExportJobRequest {
  jobType: ExportJobType
  periodYm: string // 民國 5 碼，如 11507
  region?: Region
  mode: ExportMode
  caseIds: string[]
}

// 匯出結果中的單一個案工作簿（一個個案一個月一份）
export interface ExportJobFileDTO {
  caseId: string
  caseName: string
  region?: Region
  rowCount: number
  fileName: string
  downloadUrl: string
}

// 因資料缺漏未納入申報的趟次統計
export interface ExportJobSkipDTO {
  caseId: string
  caseName: string
  reason: string
  count: number
}

export interface ExportJobDTO {
  id: string
  jobType: ExportJobType
  periodYm: string
  region?: Region
  mode: ExportMode
  status: ExportJobStatus
  totalCases?: number
  totalRows?: number
  files?: ExportJobFileDTO[]
  // skipped 只在建立當下回傳；跳過統計不落地，歷史查詢不會重現
  skipped?: ExportJobSkipDTO[]
  zipFileName?: string
  // 僅壓縮檔模式有值；逐案下載的連結掛在 files 上
  downloadUrl?: string
  precheck?: PrecheckResultDTO
  errorMessage?: string
  createdByName?: string
  createdAt: string
  completedAt?: string
}

// 儀表板
export interface DashboardStatsDTO {
  currentMonth: string
  totalCasesCount: number
  reportedTripsCount: number
  unreportedVehiclesToday: number
  pendingConflictsCount: number
  pendingFormColumnsCount: number
  recentExports: ExportJobDTO[]
}

// 批次匯入預覽
export interface ImportRowErrorDTO {
  rowIndex: number
  caseName?: string
  field?: string
  message: string
}

export interface ImportRowWarningDTO {
  rowIndex: number
  caseName?: string
  field?: string
  message: string
  useDefault?: boolean
}

export interface DryRunImportResultDTO {
  totalRows: number
  validRows: number
  errorRows: number
  warningRows: number
  previewRows: Array<Record<string, any>>
  errors: ImportRowErrorDTO[]
  warnings: ImportRowWarningDTO[]
}

// 個案匯入預覽列疑似重複所指向的既有個案（欄位型別未確認前對呼叫端一律視為選填）
export interface CaseImportDuplicateOfDTO {
  code?: string
  name?: string
}

// 個案匯入預覽列：isDuplicate/duplicateOf 之外的欄位沿用 DryRunImportResultDTO.previewRows 的動態結構
export interface CaseImportPreviewRowDTO extends Record<string, any> {
  isDuplicate?: boolean
  duplicateOf?: CaseImportDuplicateOfDTO
}

export interface CaseImportCommitResult {
  importedCount: number
  skippedRows: Array<{ rowIndex: number; caseName: string; reasons: string[] }>
  warnings?: Array<{ rowIndex: number; caseName?: string; field?: string; message: string }>
}

// 照護人員匯入結果：warnings 的 field 為 "site"／"contact"／"notes"，供「待維護」頁籤分類顯示
export interface CaregiverImportCommitResult {
  importedCount: number
  skippedRows: Array<{ rowIndex: number; name: string; reasons: string[] }>
  warnings?: Array<{ rowIndex: number; name?: string; field?: string; message: string }>
}

// 7. 系統稽核紀錄 (Audit Log)
export interface AuditLogDTO {
  id: string
  actorId?: string
  actorName?: string
  actorRole?: string
  action: AuditAction
  entityType: AuditEntityType
  entityId?: string
  entityName?: string
  beforeData?: Record<string, any>
  afterData?: Record<string, any>
  ipAddress?: string
  userAgent?: string
  createdAt: string
}

export interface ListAuditLogsParams {
  page?: number
  pageSize?: number
  actorId?: string
  action?: AuditAction
  entityType?: AuditEntityType
  fromDate?: string
  toDate?: string
  q?: string
}

// 8. 通知收件人管理 (Notification Recipients)
export type RecipientTargetType = 'role' | 'user' | 'custom'

export interface NotificationRecipientDTO {
  id: string
  topic: NotificationTopic
  recipientType?: RecipientTargetType
  targetRole?: UserRole
  userId?: string
  email: string
  displayName?: string
  active: boolean
  createdBy?: string
  createdByName?: string
  createdAt: string
}

export interface CreateNotificationRecipientRequest {
  topic: NotificationTopic
  recipientType?: RecipientTargetType
  targetRole?: UserRole
  userId?: string
  email: string
  displayName?: string
  active?: boolean
}

export interface UpdateNotificationRecipientRequest {
  topic?: NotificationTopic
  recipientType?: RecipientTargetType
  targetRole?: UserRole
  userId?: string
  email?: string
  displayName?: string
  active?: boolean
}

export interface BatchCreateNotificationRecipientsRequest {
  topic: NotificationTopic
  recipients: Array<{
    recipientType?: RecipientTargetType
    targetRole?: UserRole
    userId?: string
    email: string
    displayName?: string
  }>
}

// 9. 通知歷史紀錄 (Notification Log)
export interface NotificationLogDTO {
  id: string
  topic: NotificationTopic
  channel: 'email' | 'sms' | 'system'
  recipientEmails: string[]
  subject: string
  contentSummary?: string
  status: 'sent' | 'failed'
  errorMessage?: string
  triggeredBy?: string
  triggeredByName?: string
  sentAt: string
}

// 10. 未回報清單 (Missing Rides)
export interface MissingRideDTO {
  id: string
  caseId: string
  caseName: string
  serviceDate: string
  legSeq: number
  direction: Direction
  departTime: string
  vehicleId?: string
  vehicleName?: string
  driverName?: string
  daysOverdue: number
}

// 11. 車輛趟數表報表 (Trip Summary Report)
export interface TripSummaryCaseRowDTO {
  caseId: string
  caseName: string
  outboundCount: number
  inboundCount: number
  totalCount: number
}

export interface TripSummaryVehicleDTO {
  vehicleId: string
  vehicleName: string
  plateNo: string
  driverName?: string
  rows: TripSummaryCaseRowDTO[]
  subtotalOutbound: number
  subtotalInbound: number
  subtotalTotal: number
}

export interface TripSummaryReportDTO {
  periodYm: string
  region?: Region
  generatedAt: string
  vehicles: TripSummaryVehicleDTO[]
  grandTotalOutbound: number
  grandTotalInbound: number
  grandTotal: number
}

// 12. 新竹接送時刻表 (Hsinchu Schedule Report)
export interface HsinchuScheduleItemDTO {
  direction: Direction
  runNo: number
  caseName: string
  note?: string
  departTime: string
  origin: string
  arriveTime?: string
  destination: string
  vehicleName: string
  siteName: string
}

export interface HsinchuScheduleReportDTO {
  generatedAt: string
  siteName?: string
  vehicleName?: string
  outbound: HsinchuScheduleItemDTO[]
  inbound: HsinchuScheduleItemDTO[]
}

// 13. 車輛維修保養 (Vehicle Maintenance)
export interface MaintenanceLogDTO {
  id: string
  vehicleId: string
  vehicleName?: string
  plateNo?: string
  serviceDate: string
  mileage: number
  items: string
  vendor?: string
  cost: number
  receiptUrl?: string
  note?: string
  createdBy: string
  createdAt: string
}

export interface CreateMaintenanceRequest {
  vehicleId: string
  serviceDate: string
  mileage: number
  items: string
  vendor?: string
  cost: number
  receiptUrl?: string
  note?: string
}

export interface UpdateMaintenanceRequest extends Partial<CreateMaintenanceRequest> { }

// 14. 司機出勤與請假 (Attendance)
export interface DriverDayAttendanceDTO {
  date: string
  status: 'work' | 'leave' | 'sick' | 'off' | 'absent'
  note?: string
}

export interface DriverMonthAttendanceDTO {
  driverId: string
  driverName: string
  region: Region
  days: Record<string, DriverDayAttendanceDTO>
  workDays: number
  leaveDays: number
  sickDays: number
  offDays: number
  absentDays: number
}

export interface MonthAttendanceReportDTO {
  periodYm: string
  daysInMonth: number
  drivers: DriverMonthAttendanceDTO[]
}

export interface UpsertAttendanceRequest {
  driverId: string
  recordDate: string
  status: 'work' | 'leave' | 'sick' | 'off'
  note?: string
}

// 司機接送匯報匯入自動同步出勤時，與人工登記不一致的待維護衝突
export interface AttendanceConflictDTO {
  id: string
  driverId: string
  driverName: string
  recordDate: string
  existingStatus: 'work' | 'leave' | 'sick' | 'off'
  importedStatus: 'work' | 'leave' | 'sick' | 'off'
  status: 'pending' | 'resolved'
  resolvedChoice?: 'keep_manual' | 'use_import'
}

export interface ResolveAttendanceConflictRequest {
  choice: 'keep_manual' | 'use_import'
}

// 15. 車輛油資 (Fuel Logs)
export interface FuelLogDTO {
  id: string
  vehicleId: string
  vehicleName?: string
  plateNo?: string
  driverId?: string
  driverName?: string
  fuelDate: string
  liters: number
  cost: number
  receiptUrl?: string
  createdBy: string
  createdAt: string
}

export interface CreateFuelLogRequest {
  vehicleId: string
  driverId?: string
  fuelDate: string
  liters: number
  cost: number
  receiptUrl?: string
}

export interface UpdateFuelLogRequest extends Partial<CreateFuelLogRequest> { }

// 16. 完整版營運儀表板指標 (Dashboard Advanced Metrics)
export interface AttendanceDistributionDTO {
  workCount: number
  leaveCount: number
  sickCount: number
  offCount: number
  leavePercentage: number
}

export interface VehicleTripTrendItemDTO {
  vehicleName: string
  plateNo: string
  tripCount: number
}

export interface DashboardMetricsDTO {
  currentMonth: string
  totalCasesCount: number
  reportedTripsCount: number
  unreportedVehiclesToday: number
  pendingConflictsCount: number
  pendingFormColumnsCount: number
  attendanceDistribution: AttendanceDistributionDTO
  vehicleTripTrends: VehicleTripTrendItemDTO[]
  claimFulfillmentRate: number
}


