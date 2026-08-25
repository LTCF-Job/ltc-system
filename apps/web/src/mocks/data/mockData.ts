import type {
  CaseDTO,
  SiteDTO,
  VehicleDTO,
  DriverDTO,
  FormDTO,
  FormColumnDTO,
  RideCalendarMatrixDTO,
  ExportJobDTO,
  DashboardStatsDTO,
  PrecheckResultDTO,
  AuditLogDTO,
  NotificationRecipientDTO,
  NotificationLogDTO,
  MissingRideDTO,
  TripSummaryReportDTO,
  MaintenanceLogDTO,
  MonthAttendanceReportDTO,
  FuelLogDTO,
  HsinchuScheduleReportDTO,
  DashboardMetricsDTO
} from '@/types/api'

export const mockSites: SiteDTO[] = [
  { id: 'site_1', name: '竹北日照中心', region: 'hsinchu', address: '新竹縣竹北市光明六路100號', openDays: [1, 2, 3, 4, 5], createdAt: '2026-01-01' },
  { id: 'site_2', name: '竹南日照據點', region: 'miaoli', address: '苗栗縣竹南鎮中正路200號', openDays: [1, 2, 3, 4, 5], createdAt: '2026-01-01' },
  { id: 'site_3', name: '湖口長照據點', region: 'hsinchu', address: '新竹縣湖口鄉成功路50號', openDays: [1, 3, 5], createdAt: '2026-01-01' },
  { id: 'site_4', name: '苗栗市社區據點', region: 'miaoli', address: '苗栗縣苗栗市府前路1號', openDays: [1, 2, 3, 4, 5], createdAt: '2026-01-01' }
]

export const mockVehicles: VehicleDTO[] = [
  { id: 'veh_1', plateNo: 'BZG-7915', displayName: '竹北一車', region: 'hsinchu', active: true, createdAt: '2026-01-01' },
  { id: 'veh_2', plateNo: 'ABC-1234', displayName: '竹北二車', region: 'hsinchu', active: true, createdAt: '2026-01-01' },
  { id: 'veh_3', plateNo: 'DEF-5678', displayName: '竹南1車', region: 'miaoli', active: true, createdAt: '2026-01-01' },
  { id: 'veh_4', plateNo: 'GHI-9012', displayName: '竹南2車', region: 'miaoli', active: true, createdAt: '2026-01-01' }
]

export const mockDrivers: DriverDTO[] = [
  { id: 'drv_1', name: '郭澤威', nationalId: 'G12***6465', phone: '0912-345678', email: 'driver1@ltc.example.com', active: true, createdAt: '2026-01-01', assignments: [{ id: 'asgn_1', driverId: 'drv_1', vehicleId: 'veh_1', vehicleName: '竹北一車', startDate: '2026-01-01', isPrimary: true }] },
  { id: 'drv_2', name: '林志豪', nationalId: 'J12***9988', phone: '0922-111222', email: 'driver2@ltc.example.com', active: true, createdAt: '2026-01-01', assignments: [{ id: 'asgn_2', driverId: 'drv_2', vehicleId: 'veh_2', vehicleName: '竹北二車', startDate: '2026-01-01', isPrimary: true }] },
  { id: 'drv_3', name: '陳國華', nationalId: 'K12***8177', phone: '0933-444555', email: 'driver3@ltc.example.com', active: true, createdAt: '2026-01-01', assignments: [{ id: 'asgn_3', driverId: 'drv_3', vehicleId: 'veh_4', vehicleName: '竹南2車', startDate: '2026-01-01', isPrimary: true }] }
]

export const mockCases: CaseDTO[] = [
  {
    id: 'case_1',
    code: 'C0001',
    name: '蔡曾切',
    nationalId: 'A20***9750',
    homeAddress: '苗栗縣竹南鎮大營路123號',
    region: 'miaoli',
    serviceCategory: 1,
    serviceUsageType: 2,
    claimStartDate: '2026-07-01',
    status: 'active',
    createdAt: '2026-06-15',
    updatedAt: '2026-07-01',
    activeSchedule: {
      id: 'sch_1',
      caseId: 'case_1',
      siteId: 'site_2',
      siteName: '竹南日照據點',
      effectiveFrom: '2026-07-01',
      weekdays: [1, 2, 3, 4, 5],
      tripPattern: 2,
      unitPrice: 115,
      distanceKm: 5.0,
      serviceDurationMin: 10,
      serviceCode: 'BD03',
      legs: [
        { id: 'leg_1_1', legSeq: 1, direction: 'outbound', departTime: '09:40', arriveTime: '09:50', runNo: 1, vehicleId: 'veh_4', vehicleName: '竹南2車' },
        { id: 'leg_1_2', legSeq: 2, direction: 'inbound', departTime: '16:00', arriveTime: '16:10', runNo: 1, vehicleId: 'veh_4', vehicleName: '竹南2車' }
      ]
    }
  },
  {
    id: 'case_2',
    code: 'C0002',
    name: '葉秀珍',
    nationalId: 'J22***3344',
    homeAddress: '新竹縣竹北市中正西路50號',
    region: 'hsinchu',
    serviceCategory: 1,
    serviceUsageType: 1,
    claimStartDate: '2026-07-01',
    status: 'active',
    createdAt: '2026-06-15',
    updatedAt: '2026-07-01',
    activeSchedule: {
      id: 'sch_2',
      caseId: 'case_2',
      siteId: 'site_1',
      siteName: '竹北日照中心',
      effectiveFrom: '2026-07-01',
      weekdays: [1, 2, 3, 4, 5],
      tripPattern: 4,
      unitPrice: 115,
      distanceKm: 6.5,
      serviceDurationMin: 15,
      serviceCode: 'BD03',
      legs: [
        { id: 'leg_2_1', legSeq: 1, direction: 'outbound', departTime: '08:30', runNo: 1, vehicleId: 'veh_1', vehicleName: '竹北一車' },
        { id: 'leg_2_2', legSeq: 2, direction: 'inbound', departTime: '11:30', runNo: 1, vehicleId: 'veh_1', vehicleName: '竹北一車' },
        { id: 'leg_2_3', legSeq: 3, direction: 'outbound', departTime: '13:30', runNo: 1, vehicleId: 'veh_1', vehicleName: '竹北一車' },
        { id: 'leg_2_4', legSeq: 4, direction: 'inbound', departTime: '16:30', runNo: 1, vehicleId: 'veh_1', vehicleName: '竹北一車' }
      ]
    }
  },
  {
    id: 'case_3',
    code: 'C0003',
    name: '吳𣵛桂',
    nationalId: 'H22***5566',
    homeAddress: '新竹縣竹北市福興東路二段88號',
    region: 'hsinchu',
    serviceCategory: 1,
    serviceUsageType: 2,
    claimStartDate: '2026-07-01',
    status: 'active',
    createdAt: '2026-06-20',
    updatedAt: '2026-07-01',
    activeSchedule: {
      id: 'sch_3',
      caseId: 'case_3',
      siteId: 'site_1',
      siteName: '竹北日照中心',
      effectiveFrom: '2026-07-01',
      weekdays: [2, 4],
      tripPattern: 1,
      unitPrice: 115,
      distanceKm: 4.2,
      serviceDurationMin: 10,
      serviceCode: 'BD03',
      legs: [
        { id: 'leg_3_1', legSeq: 1, direction: 'outbound', departTime: '09:10', runNo: 1, vehicleId: 'veh_1', vehicleName: '竹北一車' }
      ]
    }
  },
  {
    id: 'case_4',
    code: 'C0004',
    name: '張詹竹妹',
    nationalId: 'O20***1122',
    homeAddress: '新竹縣竹北市三民路15號',
    region: 'hsinchu',
    serviceCategory: 1,
    serviceUsageType: 2,
    claimStartDate: '2026-07-01',
    status: 'suspended',
    createdAt: '2026-06-20',
    updatedAt: '2026-07-01'
  }
]

export const mockForms: FormDTO[] = [
  { id: 'form_1', formId: 'zhubei_car_1', title: '竹北一車 (回覆)', sheetUrl: 'https://docs.google.com/spreadsheets/d/1', lastSyncedAt: '2026-08-25 14:00', totalColumns: 56, pendingColumns: 2, hasSyncAlert: false },
  { id: 'form_2', formId: 'zhunan_car_2', title: '竹南2車 (回覆)', sheetUrl: 'https://docs.google.com/spreadsheets/d/2', lastSyncedAt: '2026-08-25 14:05', totalColumns: 62, pendingColumns: 0, hasSyncAlert: false },
  { id: 'form_3', formId: 'zhubei_car_2', title: '竹北二車 (回覆)', sheetUrl: 'https://docs.google.com/spreadsheets/d/3', lastSyncedAt: '2026-08-22 09:00', totalColumns: 48, pendingColumns: 5, hasSyncAlert: true }
]

export const mockFormColumns: FormColumnDTO[] = [
  {
    id: 'col_1',
    formId: 'form_1',
    columnName: '1. 張詹竹妹 [去程]',
    columnSeq: 4,
    kind: 'ride',
    mappingStatus: 'pending',
    suggestedCaseId: 'case_4',
    suggestedCaseName: '張詹竹妹',
    suggestedLegSeq: 1,
    suggestionScore: 1.0,
    updatedAt: '2026-08-25'
  },
  {
    id: 'col_2',
    formId: 'form_1',
    columnName: '4. 葉秀珍 (4趟) [去程]',
    columnSeq: 5,
    kind: 'ride',
    mappingStatus: 'pending',
    suggestedCaseId: 'case_2',
    suggestedCaseName: '葉秀珍',
    suggestedLegSeq: 1,
    suggestionScore: 0.8,
    updatedAt: '2026-08-25'
  },
  {
    id: 'col_3',
    formId: 'form_1',
    columnName: '1. 吳𣵛桂(去程竹3) [去程]',
    columnSeq: 6,
    kind: 'ride',
    mappingStatus: 'pending',
    suggestedCaseId: 'case_3',
    suggestedCaseName: '吳𣵛桂',
    suggestedLegSeq: 1,
    suggestionScore: 0.8,
    updatedAt: '2026-08-25'
  },
  {
    id: 'col_4',
    formId: 'form_1',
    columnName: '問題回報',
    columnSeq: 40,
    kind: 'issue',
    mappingStatus: 'ignored',
    updatedAt: '2026-08-25'
  }
]

export const mockPrecheckResult: PrecheckResultDTO = {
  passed: true,
  hasErrors: false,
  hasWarnings: true,
  summary: {
    totalErrors: 0,
    totalWarnings: 2,
    totalInfos: 2
  },
  items: [
    {
      level: 'warning',
      code: 'UNREPORTED_EXPECTED_RIDES',
      message: '尚有 3 筆應搭日為「未回報」狀態',
      details: [
        { caseId: 'case_1', caseName: '蔡曾切', serviceDate: '2026-07-15', description: '07/15 回程未取得司機回報' }
      ]
    },
    {
      level: 'warning',
      code: 'CONFLICT_EXISTS',
      message: '發現 1 筆跨車回報混車衝突尚未裁決',
      details: [
        { caseId: 'case_2', caseName: '葉秀珍', serviceDate: '2026-07-20', rideId: 'ride_conflict_1', description: '竹北一車與竹北二車皆回報有坐' }
      ]
    },
    {
      level: 'info',
      code: 'CORRECTED_RECORDS_COUNT',
      message: '本月包含已人工更正紀錄共 2 筆（已留存稽核）'
    },
    {
      level: 'info',
      code: 'QUOTA_CHECK_UNAVAILABLE',
      message: '個案配給額度檢查未執行——尚未取得額度計算規則'
    }
  ]
}

export const mockExportJobs: ExportJobDTO[] = [
  {
    id: 'job_202607_01',
    jobType: 'gov_claim',
    periodYm: '11507',
    region: 'miaoli',
    mode: 'single_multi_case',
    status: 'succeeded',
    totalCases: 42,
    totalRows: 380,
    fileName: 'gov-claim-11507-miaoli.xlsx',
    fileSize: 45200,
    downloadUrl: 'https://placeholder-download.supabase.co/gov-claim-11507-miaoli.xlsx',
    createdAt: '2026-08-01 10:00:00',
    completedAt: '2026-08-01 10:00:15'
  }
]

export const mockDashboardStats: DashboardStatsDTO = {
  currentMonth: '115-07',
  totalCasesCount: 186,
  reportedTripsCount: 2450,
  unreportedVehiclesToday: 1,
  pendingConflictsCount: 2,
  pendingFormColumnsCount: 7,
  recentExports: mockExportJobs
}

export const mockAuditLogs: AuditLogDTO[] = [
  {
    id: 'audit_1',
    actorId: 'usr_admin',
    actorName: '系統管理員',
    actorRole: 'admin',
    action: 'correct',
    entityType: 'ride_records',
    entityId: 'ride_case_1_10_1',
    entityName: '蔡曾切 (2026-07-10 去程)',
    beforeData: {
      departTimeOverride: null,
      effectiveStatus: 'boarded'
    },
    afterData: {
      departTimeOverride: '10:05',
      effectiveStatus: 'boarded',
      reason: '司機填錯時間'
    },
    ipAddress: '192.168.1.100',
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
    createdAt: '2026-07-11 09:30:15'
  },
  {
    id: 'audit_2',
    actorId: 'usr_staff',
    actorName: '行政承辦',
    actorRole: 'staff',
    action: 'reveal_pii',
    entityType: 'cases',
    entityId: 'case_1',
    entityName: '蔡曾切',
    beforeData: undefined,
    afterData: undefined,
    ipAddress: '192.168.1.105',
    userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X)',
    createdAt: '2026-07-15 14:22:01'
  },
  {
    id: 'audit_3',
    actorId: 'usr_admin',
    actorName: '系統管理員',
    actorRole: 'admin',
    action: 'resolve_conflict',
    entityType: 'ride_records',
    entityId: 'ride_conflict_1',
    entityName: '葉秀珍 (2026-07-20 去程)',
    beforeData: {
      hasConflict: true,
      vehicleId: null
    },
    afterData: {
      hasConflict: false,
      vehicleId: 'veh_1',
      vehicleName: '竹北一車',
      driverName: '郭澤威'
    },
    ipAddress: '192.168.1.100',
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
    createdAt: '2026-07-21 11:05:40'
  },
  {
    id: 'audit_4',
    actorId: 'usr_admin',
    actorName: '系統管理員',
    actorRole: 'admin',
    action: 'setting_change',
    entityType: 'notification_recipients',
    entityId: 'rec_1',
    entityName: 'admin@ltc.example.com',
    beforeData: {
      active: false
    },
    afterData: {
      active: true
    },
    ipAddress: '192.168.1.100',
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
    createdAt: '2026-08-01 08:45:00'
  },
  {
    id: 'audit_5',
    actorId: 'usr_staff',
    actorName: '行政承辦',
    actorRole: 'staff',
    action: 'export',
    entityType: 'export_jobs',
    entityId: 'job_202607_01',
    entityName: 'gov-claim-11507-miaoli.xlsx',
    beforeData: undefined,
    afterData: {
      periodYm: '11507',
      totalCases: 42,
      totalRows: 380,
      fileChecksum: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855'
    },
    ipAddress: '192.168.1.105',
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
    createdAt: '2026-08-01 10:00:15'
  }
]

export const mockNotificationRecipients: NotificationRecipientDTO[] = [
  {
    id: 'rec_1',
    topic: 'missing_report',
    email: 'admin@ltc.example.com',
    displayName: '系統主管理員',
    active: true,
    createdByName: '系統管理員',
    createdAt: '2026-01-01 00:00:00'
  },
  {
    id: 'rec_2',
    topic: 'missing_report',
    email: 'staff.miaoli@ltc.example.com',
    displayName: '苗栗區行政組',
    active: true,
    createdByName: '系統管理員',
    createdAt: '2026-01-05 10:30:00'
  },
  {
    id: 'rec_3',
    topic: 'driver_leave',
    email: 'dispatch@ltc.example.com',
    displayName: '調度中心專線',
    active: true,
    createdByName: '系統管理員',
    createdAt: '2026-01-10 14:00:00'
  },
  {
    id: 'rec_4',
    topic: 'month_end',
    email: 'finance@ltc.example.com',
    displayName: '財務申報組',
    active: true,
    createdByName: '系統管理員',
    createdAt: '2026-01-15 09:15:00'
  },
  {
    id: 'rec_5',
    topic: 'export_failed',
    email: 'tech@ltc.example.com',
    displayName: '資訊維運團隊',
    active: false,
    createdByName: '系統管理員',
    createdAt: '2026-02-01 16:20:00'
  }
]

export const mockNotificationLogs: NotificationLogDTO[] = [
  {
    id: 'nlog_1',
    topic: 'missing_report',
    channel: 'email',
    recipientEmails: ['admin@ltc.example.com', 'staff.miaoli@ltc.example.com'],
    subject: '【長照交通系統】今日未回報催報通知 (2026-08-25)',
    contentSummary: '竹南2車 (回覆) 尚有 3 筆應搭乘趟次未提交回報，請儘速核對。',
    status: 'sent',
    triggeredByName: '系統定時排程 (Cloud Scheduler)',
    sentAt: '2026-08-25 18:00:02'
  },
  {
    id: 'nlog_2',
    topic: 'month_end',
    channel: 'email',
    recipientEmails: ['finance@ltc.example.com'],
    subject: '【長照交通系統】115年07月份申報資料結算提醒',
    contentSummary: '本月已達26日，目前仍有 2 筆混車衝突未裁決，請於月底前完成處理。',
    status: 'sent',
    triggeredByName: '系統定時排程 (Cloud Scheduler)',
    sentAt: '2026-07-26 09:00:00'
  },
  {
    id: 'nlog_3',
    topic: 'export_failed',
    channel: 'email',
    recipientEmails: ['tech@ltc.example.com'],
    subject: '【警報】政府申報檔案產生異常',
    contentSummary: '批次產生任務 job_fail_998 逾時失敗',
    status: 'failed',
    errorMessage: '無設定啟用收件人 (收件人為空)',
    triggeredByName: '系統事件',
    sentAt: '2026-07-20 23:15:10'
  }
]

export const mockMissingRides: MissingRideDTO[] = [
  {
    id: 'mis_1',
    caseId: 'case_1',
    caseName: '蔡曾切',
    serviceDate: '2026-08-24',
    legSeq: 2,
    direction: 'inbound',
    departTime: '16:00',
    vehicleId: 'veh_4',
    vehicleName: '竹南2車',
    driverName: '陳國華',
    daysOverdue: 1
  },
  {
    id: 'mis_2',
    caseId: 'case_2',
    caseName: '葉秀珍',
    serviceDate: '2026-08-24',
    legSeq: 4,
    direction: 'inbound',
    departTime: '16:30',
    vehicleId: 'veh_1',
    vehicleName: '竹北一車',
    driverName: '郭澤威',
    daysOverdue: 1
  },
  {
    id: 'mis_3',
    caseId: 'case_3',
    caseName: '吳𣵛桂',
    serviceDate: '2026-08-20',
    legSeq: 1,
    direction: 'outbound',
    departTime: '09:10',
    vehicleId: 'veh_1',
    vehicleName: '竹北一車',
    driverName: '郭澤威',
    daysOverdue: 5
  }
]

export const mockTripSummaryReport: TripSummaryReportDTO = {
  periodYm: '115-07',
  region: 'hsinchu',
  generatedAt: '2026-08-25 15:00:00',
  vehicles: [
    {
      vehicleId: 'veh_1',
      vehicleName: '竹北一車',
      plateNo: 'BZG-7915',
      driverName: '郭澤威',
      rows: [
        {
          caseId: 'case_2',
          caseCode: 'C0002',
          caseName: '葉秀珍',
          outboundCount: 22,
          inboundCount: 22,
          totalCount: 44
        },
        {
          caseId: 'case_3',
          caseCode: 'C0003',
          caseName: '吳𣵛桂',
          outboundCount: 9,
          inboundCount: 0,
          totalCount: 9
        },
        {
          caseId: 'case_4',
          caseCode: 'C0004',
          caseName: '張詹竹妹',
          outboundCount: 15,
          inboundCount: 15,
          totalCount: 30
        }
      ],
      subtotalOutbound: 46,
      subtotalInbound: 37,
      subtotalTotal: 83
    },
    {
      vehicleId: 'veh_2',
      vehicleName: '竹北二車',
      plateNo: 'ABC-1234',
      driverName: '林志豪',
      rows: [
        {
          caseId: 'case_5',
          caseCode: 'C0005',
          caseName: '李國盛',
          outboundCount: 10,
          inboundCount: 10,
          totalCount: 20
        },
        {
          caseId: 'case_6',
          caseCode: 'C0006',
          caseName: '陳素貞',
          outboundCount: 18,
          inboundCount: 18,
          totalCount: 36
        }
      ],
      subtotalOutbound: 28,
      subtotalInbound: 28,
      subtotalTotal: 56
    }
  ],
  grandTotalOutbound: 74,
  grandTotalInbound: 65,
  grandTotal: 139
}

export const mockMaintenanceLogs: MaintenanceLogDTO[] = [
  {
    id: 'maint_1',
    vehicleId: 'veh_1',
    vehicleName: '竹北一車',
    plateNo: 'BZG-7915',
    serviceDate: '2026-07-10',
    mileage: 52000,
    items: '五萬公里大保養、機油更換、變速箱油、煞車油',
    vendor: '竹北正大保修廠',
    cost: 8500,
    note: '更換四輪來令片，狀況良好',
    createdBy: '系統管理員',
    createdAt: '2026-07-10 17:00:00'
  },
  {
    id: 'maint_2',
    vehicleId: 'veh_4',
    vehicleName: '竹南2車',
    plateNo: 'GHI-9012',
    serviceDate: '2026-07-22',
    mileage: 38400,
    items: '定期機油更換、冷氣濾網清潔',
    vendor: '頭份順益原廠',
    cost: 3200,
    note: '胎壓偵測器校正',
    createdBy: '行政承辦',
    createdAt: '2026-07-22 16:30:00'
  }
]

export const mockAttendanceReport: MonthAttendanceReportDTO = {
  periodYm: '115-07',
  daysInMonth: 31,
  drivers: [
    {
      driverId: 'drv_1',
      driverCode: 'D0001',
      driverName: '郭澤威',
      region: 'hsinchu',
      days: {
        '2026-07-01': { date: '2026-07-01', status: 'work' },
        '2026-07-02': { date: '2026-07-02', status: 'work' },
        '2026-07-03': { date: '2026-07-03', status: 'work' },
        '2026-07-06': { date: '2026-07-06', status: 'work' },
        '2026-07-07': { date: '2026-07-07', status: 'leave', note: '家中有事請事假' },
        '2026-07-08': { date: '2026-07-08', status: 'work' }
      },
      workDays: 21,
      leaveDays: 1,
      sickDays: 0,
      offDays: 9
    },
    {
      driverId: 'drv_2',
      driverCode: 'D0002',
      driverName: '林志豪',
      region: 'hsinchu',
      days: {
        '2026-07-01': { date: '2026-07-01', status: 'work' },
        '2026-07-02': { date: '2026-07-02', status: 'work' },
        '2026-07-03': { date: '2026-07-03', status: 'work' },
        '2026-07-15': { date: '2026-07-15', status: 'sick', note: '流感發燒' }
      },
      workDays: 20,
      leaveDays: 0,
      sickDays: 1,
      offDays: 10
    },
    {
      driverId: 'drv_3',
      driverCode: 'D0003',
      driverName: '陳國華',
      region: 'miaoli',
      days: {
        '2026-07-01': { date: '2026-07-01', status: 'work' },
        '2026-07-02': { date: '2026-07-02', status: 'work' }
      },
      workDays: 22,
      leaveDays: 0,
      sickDays: 0,
      offDays: 9
    }
  ]
}

export const mockFuelLogs: FuelLogDTO[] = [
  {
    id: 'fuel_1',
    vehicleId: 'veh_1',
    vehicleName: '竹北一車',
    plateNo: 'BZG-7915',
    driverId: 'drv_1',
    driverName: '郭澤威',
    fuelDate: '2026-07-05',
    liters: 45.2,
    cost: 1450,
    createdBy: '郭澤威',
    createdAt: '2026-07-05 18:00:00'
  },
  {
    id: 'fuel_2',
    vehicleId: 'veh_1',
    vehicleName: '竹北一車',
    plateNo: 'BZG-7915',
    driverId: 'drv_1',
    driverName: '郭澤威',
    fuelDate: '2026-07-18',
    liters: 48.0,
    cost: 1530,
    createdBy: '郭澤威',
    createdAt: '2026-07-18 17:30:00'
  },
  {
    id: 'fuel_3',
    vehicleId: 'veh_4',
    vehicleName: '竹南2車',
    plateNo: 'GHI-9012',
    driverId: 'drv_3',
    driverName: '陳國華',
    fuelDate: '2026-07-12',
    liters: 40.5,
    cost: 1290,
    createdBy: '陳國華',
    createdAt: '2026-07-12 18:20:00'
  }
]

export const mockHsinchuScheduleReport: HsinchuScheduleReportDTO = {
  generatedAt: '2026-08-25 15:30:00',
  siteName: '竹北日照中心',
  vehicleName: '竹北一車',
  outbound: [
    {
      direction: 'outbound',
      runNo: 1,
      caseCode: 'C0002',
      caseName: '葉秀珍',
      note: '早去午回',
      departTime: '08:30',
      origin: '新竹縣竹北市中正西路50號',
      arriveTime: '08:45',
      destination: '竹北日照中心',
      vehicleName: '竹北一車',
      siteName: '竹北日照中心'
    },
    {
      direction: 'outbound',
      runNo: 1,
      caseCode: 'C0003',
      caseName: '吳𣵛桂',
      note: '去程竹1車',
      departTime: '08:45',
      origin: '新竹縣竹北市博愛街264號',
      arriveTime: '09:00',
      destination: '竹北日照中心',
      vehicleName: '竹北一車',
      siteName: '竹北日照中心'
    }
  ],
  inbound: [
    {
      direction: 'inbound',
      runNo: 1,
      caseCode: 'C0002',
      caseName: '葉秀珍',
      departTime: '16:30',
      origin: '竹北日照中心',
      arriveTime: '16:45',
      destination: '新竹縣竹北市中正西路50號',
      vehicleName: '竹北一車',
      siteName: '竹北日照中心'
    }
  ]
}

export const mockDashboardMetrics: DashboardMetricsDTO = {
  currentMonth: '115-07',
  totalCasesCount: 42,
  reportedTripsCount: 380,
  unreportedVehiclesToday: 1,
  pendingConflictsCount: 1,
  pendingFormColumnsCount: 2,
  attendanceDistribution: {
    workCount: 63,
    leaveCount: 2,
    sickCount: 1,
    offCount: 28,
    leavePercentage: 4.5
  },
  vehicleTripTrends: [
    { vehicleName: '竹北一車', plateNo: 'BZG-7915', tripCount: 83 },
    { vehicleName: '竹北二車', plateNo: 'ABC-1234', tripCount: 56 },
    { vehicleName: '竹南1車', plateNo: 'DEF-5678', tripCount: 112 },
    { vehicleName: '竹南2車', plateNo: 'GHI-9012', tripCount: 129 }
  ],
  claimFulfillmentRate: 98.2
}

