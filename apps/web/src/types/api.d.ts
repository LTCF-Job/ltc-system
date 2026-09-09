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
  DriverLicenseClass,
} from "./domain";

// 共通分頁與錯誤結構
export interface PaginationMeta {
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}

export interface Paged<T> {
  data: T[];
  meta: PaginationMeta;
}

export interface ErrorDetail {
  field?: string;
  reason: string;
}

export interface ApiError {
  code: string;
  message: string;
  details?: ErrorDetail[];
  // 伺服器端 log 的同一筆請求識別碼；使用者回報問題時可據以查詢。
  requestId?: string;
}

export interface ApiResponse<T> {
  data: T;
  meta?: PaginationMeta;
  error?: ApiError;
}

// 使用者與認證
export interface UserDTO {
  id: string;
  email: string;
  displayName: string;
  role: UserRole;
  phone?: string;
  status?: "active" | "inactive";
  customPermissions?: SystemPermissions | null;
  lastLoginAt?: string;
  createdAt?: string;
}

export interface CreateUserRequest {
  email: string;
  displayName: string;
  role: UserRole;
  phone?: string;
  password?: string;
  status?: "active" | "inactive";
  customPermissions?: SystemPermissions | null;
}

export interface UpdateUserRequest {
  email?: string;
  displayName?: string;
  role?: UserRole;
  phone?: string;
  status?: "active" | "inactive";
  customPermissions?: SystemPermissions | null;
}

export interface ChangePasswordRequest {
  oldPassword?: string;
  newPassword?: string;
}

export interface AuthSession {
  user: UserDTO;
  accessToken: string;
}

// 角色身分管理
export interface RoleDTO {
  id: string;
  key: string;
  name: string;
  description: string;
  tagType: "danger" | "primary" | "success" | "warning" | "info";
  isSystem: boolean;
  permissions: SystemPermissions;
  /** null 代表使用者來源不可用、人數未知，不等同於 0 人。 */
  userCount?: number | null;
  createdAt?: string;
  updatedAt?: string;
}

export interface CreateRoleRequest {
  key?: string;
  name: string;
  description?: string;
  tagType?: "danger" | "primary" | "success" | "warning" | "info";
  permissions: SystemPermissions;
}

export interface UpdateRoleRequest {
  name?: string;
  description?: string;
  tagType?: "danger" | "primary" | "success" | "warning" | "info";
  permissions?: SystemPermissions;
}

// 個案與排班
export type ScheduleMode = "monthly" | "by_weekday" | "unified";

export interface WeekdayScheduleConfig {
  weekday: number;
  label?: string;
  tripCount: number;
  departTime?: string;
  returnTime?: string;
  vehicleId?: string;
}

export interface DayScheduleConfig {
  date: string;
  tripCount: number;
  departTime?: string;
  returnTime?: string;
  vehicleId?: string;
  note?: string;
}

export interface ScheduleLegDTO {
  id: string;
  legSeq: number;
  direction: Direction;
  departTime: string;
  arriveTime?: string;
  runNo: number;
  vehicleId: string;
  vehicleName?: string;
}

export interface CaseScheduleDTO {
  id: string;
  caseId: string;
  effectiveFrom: string;
  effectiveTo?: string;
  weekdays: number[];
  tripPattern: TripPattern;
  unitPrice: number;
  distanceKm: number;
  serviceDurationMin: number;
  serviceCode: string;
  note?: string;
  legs: ScheduleLegDTO[];
  scheduleMode?: ScheduleMode;
  weeklyConfigs?: WeekdayScheduleConfig[];
  monthlyConfigs?: Record<string, DayScheduleConfig>;
}

export interface CaseDTO {
  id: string;
  name: string;
  nameNormalized?: string;
  nationalId?: string;
  nationalIdMasked?: string;
  nationalIdInvalid?: boolean;
  homeAddress?: string;
  region?: Region | null;
  ltcLevel?: string;
  serviceCategory?: ServiceCategory;
  serviceUsageType?: ServiceUsageType;
  claimEndDate?: string;
  status: CaseStatus;
  householdType?: string;
  gender?: string;
  birthDate?: string;
  birthDateRaw?: string;
  careContactRole?: string;
  careContactName?: string;
  registeredAddress?: string;
  remarks?: string;
  siteId?: string;
  siteName?: string;
  siteNameRaw?: string;
  outboundVehicleId?: string;
  outboundVehicle?: string;
  outboundVehicleNameRaw?: string;
  inboundVehicleId?: string;
  inboundVehicle?: string;
  inboundVehicleNameRaw?: string;
  createdAt: string;
  updatedAt: string;
  activeSchedule?: CaseScheduleDTO;
}

export interface CreateCaseRequest {
  name: string;
  siteId: string;
  nationalId?: string;
  homeAddress?: string;
  region?: Region | null;
  ltcLevel?: string;
  serviceCategory?: ServiceCategory;
  serviceUsageType?: ServiceUsageType;
  claimEndDate?: string;
  status?: CaseStatus;
  householdType?: string;
  gender?: string;
  birthDate?: string;
  careContactRole?: string;
  careContactName?: string;
  registeredAddress?: string;
  remarks?: string;
}

// 兩欄位皆選填：未帶入的欄位維持既有關聯不變，僅更新有帶值的那一項
// PUT 為完整替換語意：未帶上的 *NameRaw 會被後端清成 NULL。呼叫端必須把該欄
// 尚未完成關聯的匯入原始名稱一併回送，只有真的關聯成功的那一欄才送空字串。
// 據點已改由個案本身持有，請改用 UpdateCaseRequest 的 siteId。
export interface UpdateCaseTransportPreferenceRequest {
  outboundVehicleId: string | null;
  inboundVehicleId: string | null;
  outboundVehicleNameRaw?: string;
  inboundVehicleNameRaw?: string;
}

export interface UpdateCaseRequest extends Partial<CreateCaseRequest> {}

export interface SaveScheduleRequest {
  effectiveFrom: string;
  effectiveTo?: string;
  weekdays: number[];
  tripPattern: TripPattern;
  unitPrice: number;
  distanceKm: number;
  serviceDurationMin: number;
  serviceCode: string;
  note?: string;
  scheduleMode?: ScheduleMode;
  weeklyConfigs?: WeekdayScheduleConfig[];
  monthlyConfigs?: Record<string, DayScheduleConfig>;
  legs: Array<{
    legSeq: number;
    direction: Direction;
    departTime: string;
    arriveTime?: string;
    runNo: number;
    vehicleId: string;
  }>;
}

// 主檔：據點、車輛、司機
export interface SiteDTO {
  id: string;
  name: string;
  /** 使用者自由填寫的區域文字，不再參照地區主檔。 */
  region?: string;
  address?: string;
  status: "active" | "inactive";
  createdAt: string;
}

export interface CreateSiteRequest {
  name: string;
  region?: string;
  address?: string;
  status?: "active" | "inactive";
}

export interface UpdateSiteRequest extends Partial<CreateSiteRequest> {}

export interface VehicleDTO {
  id: string;
  plateNo: string;
  displayName: string;
  // 據點是車輛自己的自由輸入文字，非必填，不關聯據點主檔
  siteName: string;
  brand: string;
  model: string;
  // 出廠年月，格式為 YYYY-MM
  manufactureYm: string;
  compulsoryInsuranceExpiry: string | null;
  passengerInsuranceExpiry: string | null;
  thirdPartyInsuranceExpiry: string | null;
  lastInspectionDate: string | null;
  wheelchairAccessible: boolean | null;
  // 四項證件持有註記
  hasVehicleLicense: boolean;
  hasPurchaseContract: boolean;
  hasPlateRegistration: boolean;
  hasTransferRegistration: boolean;
  status: "active" | "inactive";
  createdAt: string;
  drivers?: VehicleDriverDTO[];
}

// 掛在車輛上的司機摘要。一台車可以有多位司機，一位司機同期只會有一台車。
export interface VehicleDriverDTO {
  id: string;
  code?: string;
  name: string;
}

export interface CreateVehicleRequest {
  plateNo: string;
  displayName: string;
  siteName?: string;
  brand?: string | null;
  model?: string | null;
  manufactureYm?: string | null;
  compulsoryInsuranceExpiry?: string | null;
  passengerInsuranceExpiry?: string | null;
  thirdPartyInsuranceExpiry?: string | null;
  lastInspectionDate?: string | null;
  wheelchairAccessible: boolean;
  hasVehicleLicense?: boolean;
  hasPurchaseContract?: boolean;
  hasPlateRegistration?: boolean;
  hasTransferRegistration?: boolean;
  status?: "active" | "inactive";
}

export interface UpdateVehicleRequest extends Partial<CreateVehicleRequest> {}

export interface DriverAssignmentDTO {
  id: string;
  driverId: string;
  vehicleId: string;
  vehicleName?: string;
  vehiclePlateNo?: string;
  plateNo?: string;
}

export interface DriverDTO {
  id: string;
  name: string;
  nameNormalized?: string;
  nationalIdMasked: string;
  email?: string;
  gender?: string;
  birthDate?: string | null;
  hasProfessionalLicense?: boolean;
  employmentDate?: string | null;
  hasTransferCert?: boolean;
  remarks?: string;
  status: "active" | "inactive";
  // 駕照類別與有效日期為選填，未補登時為 null
  licenseClass?: DriverLicenseClass | null;
  licenseExpiryDate?: string | null;
  createdAt: string;
  assignments?: DriverAssignmentDTO[];
}

export interface CreateDriverRequest {
  name: string;
  nationalId: string;
  email?: string;
  gender?: string;
  birthDate?: string | null;
  hasProfessionalLicense?: boolean;
  employmentDate?: string | null;
  hasTransferCert?: boolean;
  remarks?: string;
  licenseClass?: DriverLicenseClass | null;
  licenseExpiryDate?: string | null;
}

export interface UpdateDriverRequest {
  name?: string;
  /** 留空代表不變更；提供時後端會重新驗證檢查碼並重算密文與遮罩值。 */
  nationalId?: string;
  email?: string;
  gender?: string;
  birthDate?: string | null;
  hasProfessionalLicense?: boolean;
  employmentDate?: string | null;
  hasTransferCert?: boolean;
  remarks?: string;
  status?: "active" | "inactive";
  licenseClass?: DriverLicenseClass | null;
  licenseExpiryDate?: string | null;
}

// 照護人員：據點是自由輸入的文字註記，非必填，不關聯據點主檔
export interface CaregiverDTO {
  id: string;
  siteName?: string;
  name: string;
  type: CaregiverType;
  contact?: string;
  notes?: string;
  status: "active" | "inactive";
  createdAt: string;
  updatedAt: string;
}

export interface CreateCaregiverRequest {
  siteName?: string;
  name: string;
  type: CaregiverType;
  contact?: string;
  notes?: string;
  status?: "active" | "inactive";
}

export interface UpdateCaregiverRequest extends Partial<CreateCaregiverRequest> {}

// 司機接送匯報表與欄位對應
export interface DriverReportFormDTO {
  id: string;
  vehicleId: string;
  vehicleName: string;
  title: string;
  region?: Region | null;
  lastImportedAt?: string | null;
  totalColumns: number;
  mappedColumns: number;
  pendingColumns: number;
  submissionCount: number;
  status: string;
}

export interface CreateDriverReportFormRequest {
  vehicleId: string;
  title: string;
}

// 某份匯報表在某個月已匯入的統計；月份由 service_date 推得，不是資料庫欄位
export interface DriverReportImportedMonthDTO {
  formId: string;
  yearMonth: string;
  submissionCount: number;
  lastImportedAt: string;
}

export interface DriverReportColumnDTO {
  id: string;
  formId: string;
  formTitle: string;
  vehicleName: string;
  columnIndex: number;
  columnHeader: string;
  cleanedName: string;
  kind: ColumnKind;
  mappingStatus: MappingStatus;
  caseId?: string | null;
  caseName?: string | null;
  legSeq?: number | null;
  suggestedCaseId?: string | null;
  suggestedCaseName?: string | null;
  suggestedLegSeq?: number | null;
  suggestionScore: number;
}

// 匯入預覽的欄位段：每個未對應欄位在此就地確認個案與趟次
export interface DriverReportPreviewColumn {
  columnId?: string;
  columnIndex: number;
  columnHeader: string;
  cleanedName: string;
  direction?: Direction;
  mappingStatus: MappingStatus;
  caseId?: string;
  caseName?: string;
  legSeq?: number;
  suggestedCaseId?: string;
  suggestedCaseName?: string;
  suggestedLegSeq?: number;
  suggestionScore: number;
  boardedCount: number;
  absentCount: number;
}

// 匯入預覽的資料段：一列一天
export interface DriverReportPreviewRow {
  rowId: string;
  rowIndex: number;
  reportDate: string;
  serviceDate: string;
  driverRaw: string;
  driverId?: string;
  driverName?: string;
  remark?: string;
  boardedCount: number;
  absentCount: number;
  errorMessage?: string;
  warningMessage?: string;
}

export interface DriverReportPreviewDTO {
  formId: string;
  vehicleId: string;
  vehicleName: string;
  canCommit: boolean;
  totalRows: number;
  validRows: number;
  errorRows: number;
  warningRows: number;
  unmappedColumns: number;
  columns: DriverReportPreviewColumn[];
  previewRows: DriverReportPreviewRow[];
  errors: Array<{
    rowId?: string;
    rowIndex: number;
    field?: string;
    message: string;
  }>;
  warnings: Array<{
    rowId?: string;
    rowIndex: number;
    field?: string;
    message: string;
  }>;
}

export interface DriverReportColumnDecision {
  columnHeader: string;
  mappingStatus: MappingStatus;
  caseId?: string | null;
  legSeq?: number | null;
}

export interface DriverReportCommitResultDTO {
  status: "pending" | "succeeded";
  importedRows: number;
  rideRecordRows: number;
  // reaffirmedRows 是值與既有資料相同的重複回報；pendingConflictRows 是與既有資料不同、
  // 已進入待維護等待使用者選擇的筆數；backfilledRows 是本次有欄位剛完成對應而補寫的
  // 先前月份筆數。這幾個數字與 rideRecordRows 分開計算。
  reaffirmedRows: number;
  pendingConflictRows: number;
  backfilledRows: number;
  mappedColumns: number;
  skippedRows: Array<{
    rowIndex: number;
    reportDate: string;
    reasons: string[];
  }>;
  warnings?: Array<{
    rowId?: string;
    rowIndex: number;
    field?: string;
    message: string;
  }>;
}

export interface UpdateColumnMappingRequest {
  caseId?: string;
  legSeq?: number;
  mappingStatus: MappingStatus;
}

export interface BatchMappingRequest {
  mappings: Array<{
    columnId: string;
    caseId?: string;
    legSeq?: number;
    mappingStatus: MappingStatus;
  }>;
}

// 待維護資料頁籤：以匯報表列（一天一列）為單位，一列可能同時有個案欄位、駕駛人、
// 或同車同個案資料衝突三種問題
export interface SubmissionReviewDTO {
  submissionId: string;
  formTitle: string;
  vehicleName: string;
  serviceDate: string;
  caseIssues: DriverReportColumnDTO[];
  driverIssue?: { driverNameRaw: string };
  rowConflicts?: RowConflictDTO[];
}

// 一筆「同車同個案」的資料與既有資料衝突，需要使用者選擇要保留哪一筆
export interface RowConflictDTO {
  id: string;
  caseId: string;
  caseName: string;
  legSeq: number;
  previousReported: "boarded" | "absent";
  previousDriverName: string;
  newReported: "boarded" | "absent";
  newDriverName: string;
  detectedAt: string;
}

export interface BindDriverRequest {
  driverNameRaw: string;
  driverId: string;
}

export interface ResolveRowConflictRequest {
  useNew: boolean;
}

// 總覽頁鑽取單一匯報表、單一月份時的完整內容：逐日回報明細與展開後的個案搭乘紀錄
export interface DriverReportMonthSubmissionDTO {
  serviceDate: string;
  driverNameRaw: string;
  remark: string;
  answers: Record<string, string>;
}

export interface DriverReportMonthRideEntryDTO {
  caseId: string;
  caseName: string;
  serviceDate: string;
  legSeq: number;
  reported: string;
  driverId?: string | null;
  driverName: string;
  vehicleId: string;
}

export interface DriverReportMonthDetailDTO {
  submissions: DriverReportMonthSubmissionDTO[];
  rideEntries: DriverReportMonthRideEntryDTO[];
}

// 搭乘紀錄與月曆矩陣
export interface RideSourceDTO {
  id: string;
  submissionId: string;
  vehicleId: string;
  vehicleName?: string;
  driverId?: string;
  driverName?: string;
  reported: RideReportedStatus;
  submittedAt: string;
}

export interface RideRecordDTO {
  id: string;
  caseId: string;
  caseName?: string;
  serviceDate: string;
  legSeq: number;
  direction?: Direction;
  mergedStatus: EffectiveRideStatus;
  effectiveStatus: EffectiveRideStatus;
  hasConflict: boolean;
  vehicleId?: string;
  vehicleName?: string;
  driverId?: string;
  driverName?: string;
  departTimeOverride?: string;
  durationMinOverride?: number;
  scheduledDepartTime?: string;
  scheduledDurationMin?: number;
  notClaimedAa09: boolean;
  correctedBy?: string;
  correctedByName?: string;
  correctedAt?: string;
  correctionReason?: string;
  basedOnFingerprint: string;
  sourceChanged?: boolean;
  sources: RideSourceDTO[];
}

export interface RideCalendarCellDTO {
  date: string;
  dayOfWeek: number;
  isExpected: boolean;
  expectedTripCount?: number;
  isHoliday?: boolean;
  holidayName?: string;
  records: RideRecordDTO[];
}

export interface CaseRideCalendarRowDTO {
  caseId: string;
  caseName: string;
  region: Region;
  tripPattern: CalendarTripPattern;
  tripPatternText?: string;
  days: Record<string, RideCalendarCellDTO>;
}

export interface RideCalendarMatrixDTO {
  month: string;
  totalCases: number;
  daysInMonth: number;
  cases: CaseRideCalendarRowDTO[];
}

export interface PatchRideRequest {
  // PATCH 三態：省略=保留、null=清除、值=設定；vehicleId 因資料庫 NOT NULL 不可清除。
  effectiveStatus?: EffectiveRideStatus | null;
  vehicleId?: string | null;
  driverId?: string | null;
  departTimeOverride?: string | null;
  durationMinOverride?: number | null;
  legSeq?: number;
  notClaimedAa09?: boolean | null;
  reason?: string | null;
  basedOnFingerprint: string;
}

export interface ResolveConflictRequest {
  vehicleId: string;
  driverId: string;
  reason?: string;
}

export interface ManualReportRideRequest {
  id?: string;
  caseId: string;
  serviceDate: string;
  legSeq: number;
  effectiveStatus: EffectiveRideStatus;
  vehicleId?: string;
  driverId?: string;
  departTimeOverride?: string | null;
  durationMinOverride?: number | null;
  notClaimedAa09?: boolean;
  reason?: string;
}

export interface IssueRideDTO {
  id: string;
  caseId: string;
  caseName: string;
  serviceDate: string;
  legSeq: number;
  issueType: "conflict" | "unreported" | "import_error";
  hasConflict: boolean;
  description: string;
  vehicles?: string[];
  sources?: RideSourceDTO[];
  rawPayload?: string;
}

// 申報匯出
export interface PrecheckItemDTO {
  level: "error" | "warning" | "info";
  code: string;
  message: string;
  details?: Array<{
    caseId?: string;
    caseName?: string;
    field?: string;
    serviceDate?: string;
    rideId?: string;
    description?: string;
  }>;
}

export interface PrecheckResultDTO {
  passed: boolean;
  hasErrors: boolean;
  hasWarnings: boolean;
  summary: {
    totalErrors: number;
    totalWarnings: number;
    totalInfos: number;
  };
  items: PrecheckItemDTO[];
}

export interface CreateExportJobRequest {
  jobType: ExportJobType;
  periodYm: string; // 民國 5 碼，如 11507
  region?: Region | null;
  mode: ExportMode;
  caseIds: string[];
}

// 匯出結果中的單一個案工作簿（一個個案一個月一份）
export interface ExportJobFileDTO {
  caseId: string;
  caseName: string;
  region?: Region | null;
  rowCount: number;
  fileName: string;
  downloadUrl: string;
}

// 因來源缺漏而在申報檔留白的欄位統計
export interface ExportJobDataGapDTO {
  caseId: string;
  caseName: string;
  reason: string;
  count: number;
}

export interface ExportJobDTO {
  id: string;
  jobType: ExportJobType;
  periodYm: string;
  region?: Region | null;
  mode: ExportMode;
  status: ExportJobStatus;
  totalCases?: number;
  totalRows?: number;
  files?: ExportJobFileDTO[];
  // dataGaps 只在建立當下回傳；缺漏統計不寫入儲存，歷史查詢不會重現
  dataGaps?: ExportJobDataGapDTO[];
  zipFileName?: string;
  // 僅壓縮檔模式有值；逐案下載的連結掛在 files 上
  downloadUrl?: string;
  precheck?: PrecheckResultDTO;
  errorMessage?: string;
  createdByName?: string;
  createdAt: string;
  completedAt?: string;
}

// 儀表板
export interface DashboardStatsDTO {
  currentMonth: string;
  totalCasesCount: number;
  reportedTripsCount: number;
  unreportedVehiclesToday: number;
  pendingConflictsCount: number;
  pendingFormColumnsCount: number;
  recentExports: ExportJobDTO[];
}

// 批次匯入預覽
export interface ImportRowErrorDTO {
  rowIndex: number;
  caseName?: string;
  field?: string;
  message: string;
}

export interface ImportRowWarningDTO {
  rowIndex: number;
  caseName?: string;
  field?: string;
  message: string;
  useDefault?: boolean;
}

export interface DryRunImportResultDTO {
  totalRows: number;
  validRows: number;
  errorRows: number;
  warningRows: number;
  previewRows: Array<Record<string, any>>;
  errors: ImportRowErrorDTO[];
  warnings: ImportRowWarningDTO[];
}

// 個案匯入預覽列疑似重複所指向的既有個案（欄位型別未確認前對呼叫端一律視為選填）
export interface CaseImportDuplicateOfDTO {
  code?: string;
  name?: string;
}

// 個案匯入預覽列：isDuplicate/duplicateOf 之外的欄位沿用 DryRunImportResultDTO.previewRows 的動態結構。
// isDuplicate 為 true 的列正式匯入時不會建立個案，改建立為待裁決暫存項目。
export interface CaseImportPreviewRowDTO extends Record<string, any> {
  rowId?: string;
  isDuplicate?: boolean;
  duplicateOf?: CaseImportDuplicateOfDTO;
  birthDateInvalid?: boolean;
  nationalIdInvalid?: boolean;
}

export interface CaseImportCommitResult {
  importedCount: number;
  alreadyImportedCount: number;
  failedCount: number;
  stagedDuplicateCount: number;
  skippedRows: Array<{
    rowId?: string;
    rowIndex: number;
    caseName: string;
    reasons: string[];
  }>;
  failedRows: Array<{
    rowId?: string;
    rowIndex: number;
    caseName: string;
    reasons: string[];
  }>;
  warnings?: Array<{
    rowIndex: number;
    caseName?: string;
    field?: string;
    message: string;
  }>;
}

// 待裁決的疑似重複個案暫存列；裁決前不對應任何 cases 資料列，故無 caseId。
// 身分證字號只提供遮罩值，明文需另外呼叫 reveal API 並留下稽核紀錄後才會回傳。
export interface CaseDuplicateCandidateDTO {
  id: string;
  rowIndex: number;
  sheetName?: string;
  name: string;
  nationalIdMasked?: string;
  nationalIdInvalid?: boolean;
  householdType?: string;
  gender?: string;
  birthDate?: string;
  birthDateRaw?: string;
  careContactRole?: string;
  careContactName?: string;
  registeredAddress?: string;
  homeAddress?: string;
  region?: Region | null;
  siteId?: string;
  siteName?: string;
  siteNameRaw?: string;
  outboundVehicleId?: string;
  outboundVehicle?: string;
  outboundVehicleNameRaw?: string;
  inboundVehicleId?: string;
  inboundVehicle?: string;
  inboundVehicleNameRaw?: string;
  remarks?: string;
  duplicateCaseId: string;
  duplicateCaseName: string;
  status: 'pending' | 'confirmed_new' | 'merged_existing';
  createdAt: string;
}

export interface ResolveDuplicateCandidateRequest {
  decision: 'confirmed_new' | 'merged_existing';
  targetCaseId?: string;
  mergeRemarks?: boolean;
}

// 照護人員匯入結果：warnings 的 field 為 "site"／"contact"／"notes"，供「待維護」頁籤分類顯示
export interface CaregiverImportCommitResult {
  importedCount: number;
  skippedRows: Array<{
    rowId?: string;
    rowIndex: number;
    name: string;
    reasons: string[];
  }>;
  warnings?: Array<{
    rowIndex: number;
    name?: string;
    field?: string;
    message: string;
  }>;
}

// 7. 系統稽核紀錄 (Audit Log)
export interface AuditLogDTO {
  id: string;
  actorId?: string;
  actorName?: string;
  actorRole?: string;
  action: AuditAction;
  entityType: AuditEntityType;
  entityId?: string;
  entityName?: string;
  beforeData?: Record<string, any>;
  afterData?: Record<string, any>;
  ipAddress?: string;
  userAgent?: string;
  createdAt: string;
}

export interface ListAuditLogsParams {
  page?: number;
  pageSize?: number;
  actorId?: string;
  action?: AuditAction;
  entityType?: AuditEntityType;
  fromDate?: string;
  toDate?: string;
  q?: string;
}

// 8. 通知收件人管理 (Notification Recipients)
export type RecipientTargetType = "role" | "user" | "custom";

export interface NotificationRecipientDTO {
  id: string;
  topic: NotificationTopic;
  recipientType?: RecipientTargetType;
  targetRole?: UserRole;
  userId?: string;
  email: string;
  displayName?: string;
  active: boolean;
  createdBy?: string;
  createdByName?: string;
  createdAt: string;
}

export interface CreateNotificationRecipientRequest {
  topic: NotificationTopic;
  email: string;
  displayName?: string;
}

export interface UpdateNotificationRecipientRequest {
  /** email 為必填：後端以完整列覆寫，不支援部分更新。topic 不可變更。 */
  email: string;
  displayName?: string;
  active?: boolean;
}

export interface BatchCreateNotificationRecipientsRequest {
  /** topic 屬於每一筆收件人；後端以 topic + email 為唯一鍵。 */
  recipients: Array<{
    topic: NotificationTopic;
    email: string;
    displayName?: string;
  }>;
}

// 9. 通知歷史紀錄 (Notification Log)
export interface NotificationLogDTO {
  id: string;
  topic: NotificationTopic;
  channel: "email" | "sms" | "system";
  recipientEmails: string[];
  subject: string;
  contentSummary?: string;
  status: "sent" | "failed";
  errorMessage?: string;
  triggeredBy?: string;
  triggeredByName?: string;
  sentAt: string;
}

// 10. 未回報清單 (Missing Rides)
export interface MissingRideDTO {
  id: string;
  caseId: string;
  caseName: string;
  serviceDate: string;
  legSeq: number;
  direction: Direction;
  departTime: string;
  vehicleId?: string;
  vehicleName?: string;
  driverName?: string;
  daysOverdue: number;
}

// 11. 車輛趟數表報表 (Trip Summary Report)
export interface TripSummaryCaseRowDTO {
  caseId: string;
  caseName: string;
  outboundCount: number;
  inboundCount: number;
  totalCount: number;
}

export interface TripSummaryVehicleDTO {
  vehicleId: string;
  vehicleName: string;
  plateNo: string;
  driverName?: string;
  rows: TripSummaryCaseRowDTO[];
  subtotalOutbound: number;
  subtotalInbound: number;
  subtotalTotal: number;
}

export interface TripSummaryReportDTO {
  periodYm: string;
  generatedAt: string;
  vehicles: TripSummaryVehicleDTO[];
  grandTotalOutbound: number;
  grandTotalInbound: number;
  grandTotal: number;
}

// 12. 新竹接送時刻表 (Hsinchu Schedule Report)
export interface HsinchuScheduleItemDTO {
  direction: Direction;
  runNo: number;
  caseName: string;
  note?: string;
  departTime: string;
  origin: string;
  arriveTime?: string;
  destination: string;
  vehicleName: string;
  siteName: string;
}

export interface HsinchuScheduleReportDTO {
  generatedAt: string;
  siteName?: string;
  vehicleName?: string;
  outbound: HsinchuScheduleItemDTO[];
  inbound: HsinchuScheduleItemDTO[];
}

// 13. 車輛維修保養 (Vehicle Maintenance)
export interface MaintenanceLogDTO {
  id: string;
  vehicleId: string;
  vehicleName?: string;
  plateNo?: string;
  serviceDate: string;
  mileage: number;
  items: string;
  vendor?: string;
  cost: number;
  receiptUrl?: string;
  note?: string;
  createdBy: string;
  createdAt: string;
}

export interface CreateMaintenanceRequest {
  vehicleId: string;
  serviceDate: string;
  mileage: number;
  items: string;
  vendor?: string;
  cost: number;
  receiptUrl?: string;
  note?: string;
}

export interface UpdateMaintenanceRequest extends Partial<CreateMaintenanceRequest> {}

// 14. 司機出勤與請假 (Attendance)
export interface DriverDayAttendanceDTO {
  date: string;
  status: "work" | "leave" | "sick" | "off" | "absent";
  note?: string;
}

export interface AttendanceRecordDTO {
  id: string;
  driverId: string;
  recordDate: string;
  status: "work" | "leave" | "sick" | "off";
  note?: string;
  source?: string;
}

export interface DriverMonthAttendanceDTO {
  driverId: string;
  driverName: string;
  region: Region;
  days: Record<string, DriverDayAttendanceDTO>;
  workDays: number;
  leaveDays: number;
  sickDays: number;
  offDays: number;
  absentDays: number;
}

export interface MonthAttendanceReportDTO {
  periodYm: string;
  daysInMonth: number;
  drivers: DriverMonthAttendanceDTO[];
}

export interface UpsertAttendanceRequest {
  driverId: string;
  recordDate: string;
  status: "work" | "leave" | "sick" | "off";
  note?: string;
}

// 司機接送匯報匯入自動同步出勤時，與人工登記不一致的待維護衝突
export interface AttendanceConflictDTO {
  id: string;
  driverId: string;
  driverName: string;
  recordDate: string;
  existingStatus: "work" | "leave" | "sick" | "off";
  importedStatus: "work" | "leave" | "sick" | "off";
  status: "pending" | "resolved";
  resolvedChoice?: "keep_manual" | "use_import";
}

export interface ResolveAttendanceConflictRequest {
  choice: "keep_manual" | "use_import";
}

// 15. 車輛油資 (Fuel Logs)
export interface FuelLogDTO {
  id: string;
  vehicleId: string;
  vehicleName?: string;
  plateNo?: string;
  driverId?: string;
  driverName?: string;
  fuelDate: string;
  liters: number;
  cost: number;
  receiptUrl?: string;
  createdBy: string;
  createdAt: string;
}

export interface CreateFuelLogRequest {
  vehicleId: string;
  driverId?: string;
  fuelDate: string;
  liters: number;
  cost: number;
  receiptUrl?: string;
}

export interface UpdateFuelLogRequest extends Partial<CreateFuelLogRequest> {}

// 16. 完整版營運儀表板指標 (Dashboard Advanced Metrics)
export interface AttendanceDistributionDTO {
  workCount: number;
  leaveCount: number;
  sickCount: number;
  offCount: number;
  leavePercentage: number;
}

export interface VehicleTripTrendItemDTO {
  vehicleName: string;
  plateNo: string;
  tripCount: number;
}

export interface DashboardMetricsDTO {
  currentMonth: string;
  totalCasesCount: number;
  reportedTripsCount: number;
  unreportedVehiclesToday: number;
  pendingConflictsCount: number;
  pendingFormColumnsCount: number;
  attendanceDistribution: AttendanceDistributionDTO;
  vehicleTripTrends: VehicleTripTrendItemDTO[];
  claimFulfillmentRate: number;
}
