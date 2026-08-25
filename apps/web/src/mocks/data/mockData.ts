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
  PrecheckResultDTO
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
