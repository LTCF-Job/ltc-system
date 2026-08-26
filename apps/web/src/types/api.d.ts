import type {
  UserRole,
  Region,
  CaseStatus,
  Direction,
  TripPattern,
  EffectiveRideStatus,
  RideReportedStatus,
  MappingStatus,
  ColumnKind,
  ExportJobType,
  ExportJobStatus,
  ServiceCategory,
  ServiceUsageType,
  NotificationTopic,
  AuditAction,
  AuditEntityType,
  SystemPermissions
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

// 個案與排班
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
}

export interface CaseDTO {
  id: string
  code: string
  name: string
  nameNormalized?: string
  nationalId: string
  phone?: string
  homeAddress: string
  region: Region
  ltcLevel?: string
  serviceCategory: ServiceCategory
  serviceUsageType: ServiceUsageType
  claimStartDate: string
  claimEndDate?: string
  status: CaseStatus
  createdAt: string
  updatedAt: string
  activeSchedule?: CaseScheduleDTO
}

export interface CreateCaseRequest {
  name: string
  nationalId: string
  phone?: string
  homeAddress: string
  region: Region
  ltcLevel?: string
  serviceCategory: ServiceCategory
  serviceUsageType: ServiceUsageType
  claimStartDate: string
  claimEndDate?: string
  status?: CaseStatus
}

export interface UpdateCaseRequest extends Partial<CreateCaseRequest> {}

export interface CreateScheduleRequest {
  siteId: string
  effectiveFrom: string
  effectiveTo?: string
  weekdays: number[]
  tripPattern: TripPattern
  unitPrice: number
  distanceKm: number
  serviceDurationMin: number
  serviceCode?: string
  note?: string
  legs: Array<{
    legSeq: number
    direction: Direction
    departTime: string
    arriveTime?: string
    runNo: number
    vehicleId: string
  }>
}

// 主檔：區域、據點、車輛、司機
export interface RegionDTO {
  id: string
  code: string
  name: string
  description?: string
  status: 'active' | 'inactive'
  sortOrder: number
  createdAt: string
  updatedAt: string
}

export interface CreateRegionRequest {
  code: string
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
  createdAt: string
}


export interface CreateSiteRequest {
  name: string
  region: Region
  address: string
  openDays: number[]
}

export interface UpdateSiteRequest extends Partial<CreateSiteRequest> {}

export interface VehicleDTO {
  id: string
  plateNo: string
  displayName: string
  region: Region
  active: boolean
  createdAt: string
}

export interface CreateVehicleRequest {
  plateNo: string
  displayName: string
  region: Region
  active?: boolean
}

export interface UpdateVehicleRequest extends Partial<CreateVehicleRequest> {}

export interface DriverAssignmentDTO {
  id: string
  driverId: string
  vehicleId: string
  vehicleName?: string
  vehiclePlateNo?: string
  plateNo?: string
  startDate: string
  endDate?: string
  isPrimary: boolean
}

export interface DriverDTO {
  id: string
  code?: string
  name: string
  nameNormalized?: string
  nationalId: string
  phone?: string
  email?: string
  active: boolean
  createdAt: string
  assignments?: DriverAssignmentDTO[]
}

export interface CreateDriverRequest {
  name: string
  nationalId: string
  phone?: string
  email?: string
  active?: boolean
}

export interface UpdateDriverRequest extends Partial<CreateDriverRequest> {}

// Google 表單與欄位對應
export interface FormDTO {
  id: string
  formId: string
  title: string
  sheetUrl?: string
  vehicleId?: string
  vehicleName?: string
  region?: Region
  sheetTabs?: string[]
  activeTab?: string
  syncedMonths?: string[]
  lastSyncedAt?: string
  totalColumns: number
  pendingColumns: number
  hasSyncAlert?: boolean
}

export interface CreateFormAssociationRequest {
  title: string
  sheetUrl: string
  vehicleId?: string
  region?: Region
  sheetTabs?: string[]
  activeTab?: string
}

export interface SyncFormOptions {
  month?: string
  sheetTab?: string
  force?: boolean
}

export interface FormColumnDTO {
  id: string
  formId: string
  columnName: string
  columnSeq: number
  kind: ColumnKind
  mappingStatus: MappingStatus
  suggestedCaseId?: string
  suggestedCaseName?: string
  suggestedLegSeq?: number
  suggestionScore?: number
  mappedCaseId?: string
  mappedCaseName?: string
  mappedLegSeq?: number
  updatedAt: string
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
  isHoliday?: boolean
  holidayName?: string
  records: RideRecordDTO[]
}

export interface CaseRideCalendarRowDTO {
  caseId: string
  caseCode: string
  caseName: string
  region: Region
  tripPattern: TripPattern
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
  periodYm: string // RRR-MM 如 115-07
  region?: Region
  mode: 'single_multi_case' | 'case_per_file'
  caseIds?: string[]
  vehicleIds?: string[]
}

export interface ExportJobDTO {
  id: string
  jobType: ExportJobType
  periodYm: string
  region?: Region
  mode: 'single_multi_case' | 'case_per_file'
  status: ExportJobStatus
  totalCases?: number
  totalRows?: number
  fileName?: string
  fileSize?: number
  downloadUrl?: string
  precheck?: PrecheckResultDTO
  errorMessage?: string
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
  caseCode: string
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
  caseCode: string
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

export interface UpdateMaintenanceRequest extends Partial<CreateMaintenanceRequest> {}

// 14. 司機出勤與請假 (Attendance)
export interface DriverDayAttendanceDTO {
  date: string
  status: 'work' | 'leave' | 'sick' | 'off'
  note?: string
}

export interface DriverMonthAttendanceDTO {
  driverId: string
  driverCode: string
  driverName: string
  region: Region
  days: Record<string, DriverDayAttendanceDTO>
  workDays: number
  leaveDays: number
  sickDays: number
  offDays: number
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

export interface UpdateFuelLogRequest extends Partial<CreateFuelLogRequest> {}

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


