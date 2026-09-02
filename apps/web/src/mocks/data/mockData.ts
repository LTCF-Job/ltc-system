import type {
  CaseDTO,
  RegionDTO,
  SiteDTO,
  VehicleDTO,
  DriverDTO,
  CaregiverDTO,
  DriverReportFormDTO,
  DriverReportColumnDTO,
  ExportJobDTO,
  DashboardStatsDTO,
  PrecheckResultDTO,
  AuditLogDTO,
  NotificationRecipientDTO,
  NotificationLogDTO,
  MissingRideDTO,
  IssueRideDTO,
  TripSummaryReportDTO,
  MaintenanceLogDTO,
  MonthAttendanceReportDTO,
  FuelLogDTO,
  HsinchuScheduleReportDTO,
  DashboardMetricsDTO,
  UserDTO,
  RoleDTO
} from '@/types/api'
import { DEFAULT_ROLE_PERMISSIONS } from '@/types/domain'

// 區域主檔展示資料：涵蓋全台灣 22 縣市
export const mockRegions: RegionDTO[] = [
  { id: 'reg_1', name: '新竹縣', description: '新竹縣營運區域', status: 'active', sortOrder: 1, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_2', name: '新竹市', description: '新竹市營運區域', status: 'active', sortOrder: 2, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_3', name: '苗栗縣', description: '苗栗縣營運區域', status: 'active', sortOrder: 3, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_4', name: '臺北市', description: '臺北市營運區域', status: 'active', sortOrder: 4, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_5', name: '新北市', description: '新北市營運區域', status: 'active', sortOrder: 5, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_6', name: '基隆市', description: '基隆市營運區域', status: 'active', sortOrder: 6, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_7', name: '桃園市', description: '桃園市營運區域', status: 'active', sortOrder: 7, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_8', name: '臺中市', description: '臺中市營運區域', status: 'active', sortOrder: 8, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_9', name: '彰化縣', description: '彰化縣營運區域', status: 'active', sortOrder: 9, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_10', name: '南投縣', description: '南投縣營運區域', status: 'active', sortOrder: 10, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_11', name: '雲林縣', description: '雲林縣營運區域', status: 'active', sortOrder: 11, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_12', name: '嘉義市', description: '嘉義市營運區域', status: 'active', sortOrder: 12, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_13', name: '嘉義縣', description: '嘉義縣營運區域', status: 'active', sortOrder: 13, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_14', name: '臺南市', description: '臺南市營運區域', status: 'active', sortOrder: 14, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_15', name: '高雄市', description: '高雄市營運區域', status: 'active', sortOrder: 15, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_16', name: '屏東縣', description: '屏東縣營運區域', status: 'active', sortOrder: 16, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_17', name: '宜蘭縣', description: '宜蘭縣營運區域', status: 'active', sortOrder: 17, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_18', name: '花蓮縣', description: '花蓮縣營運區域', status: 'active', sortOrder: 18, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_19', name: '臺東縣', description: '臺東縣營運區域', status: 'active', sortOrder: 19, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_20', name: '澎湖縣', description: '澎湖縣營運區域', status: 'active', sortOrder: 20, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_21', name: '金門縣', description: '金門縣營運區域', status: 'active', sortOrder: 21, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_22', name: '連江縣', description: '連江縣營運區域', status: 'active', sortOrder: 22, createdAt: '2026-01-01', updatedAt: '2026-01-01' }
]


// 據點主檔展示資料：涵蓋新竹與苗栗、不同機構型態與營業日排程
export const mockSites: SiteDTO[] = [
  { id: 'site_1', name: '竹北日照中心', region: 'hsinchu', address: '新竹縣竹北市光明六路100號', openDays: [1, 2, 3, 4, 5], createdAt: '2026-01-01' },
  { id: 'site_2', name: '竹南日照據點', region: 'miaoli', address: '苗栗縣竹南鎮中正路200號', openDays: [1, 2, 3, 4, 5], createdAt: '2026-01-01' },
  { id: 'site_3', name: '湖口長照據點', region: 'hsinchu', address: '新竹縣湖口鄉成功路50號', openDays: [1, 3, 5], createdAt: '2026-01-01' },
  { id: 'site_4', name: '苗栗市社區據點', region: 'miaoli', address: '苗栗縣苗栗市府前路1號', openDays: [1, 2, 3, 4, 5], createdAt: '2026-01-01' },
  { id: 'site_5', name: '新竹縣輔具資源中心', region: 'hsinchu', address: '新竹縣竹北市光明九路235號', openDays: [1, 2, 3, 4, 5, 6], createdAt: '2026-01-10' },
  { id: 'site_6', name: '竹南身障日間作業據點', region: 'miaoli', address: '苗栗縣竹南鎮博愛街120號', openDays: [2, 4], createdAt: '2026-02-01' }
]

// 車輛主檔展示資料：涵蓋新竹與苗栗、正常服役與定期維修停用狀態
export const mockVehicles: VehicleDTO[] = [
  { id: 'veh_1', plateNo: 'BZG-7915', displayName: '竹北一車', region: 'hsinchu', active: true, createdAt: '2026-01-01' },
  { id: 'veh_2', plateNo: 'ABC-1234', displayName: '竹北二車', region: 'hsinchu', active: true, createdAt: '2026-01-01' },
  { id: 'veh_3', plateNo: 'DEF-5678', displayName: '竹南1車', region: 'miaoli', active: true, createdAt: '2026-01-01' },
  { id: 'veh_4', plateNo: 'GHI-9012', displayName: '竹南2車', region: 'miaoli', active: true, createdAt: '2026-01-01' },
  { id: 'veh_5', plateNo: 'JKL-3456', displayName: '竹東一車', region: 'hsinchu', active: false, createdAt: '2026-02-15' },
  { id: 'veh_6', plateNo: 'MNO-7890', displayName: '苗栗市1車', region: 'miaoli', active: true, createdAt: '2026-03-01' }
]

// 司機主檔展示資料：一位司機同期只掛一台車，竹北一車由兩位司機共同駕駛，另涵蓋尚未指派車輛的離職司機
export const mockDrivers: DriverDTO[] = [
  {
    id: 'drv_1',
    name: '郭澤威',
    nationalId: 'G123456465',
    phone: '0912345678',
    email: 'driver1@ltc.example.com',
    active: true,
    createdAt: '2026-01-01',
    assignments: [
      { id: 'asgn_1', driverId: 'drv_1', vehicleId: 'veh_1', vehicleName: '竹北一車', vehiclePlateNo: 'BZG-7915', plateNo: 'BZG-7915', startDate: '2026-01-01' }
    ]
  },
  {
    id: 'drv_2',
    name: '林志豪',
    nationalId: 'J123459988',
    phone: '0922111222',
    email: 'driver2@ltc.example.com',
    active: true,
    createdAt: '2026-01-01',
    assignments: [
      { id: 'asgn_2', driverId: 'drv_2', vehicleId: 'veh_2', vehicleName: '竹北二車', vehiclePlateNo: 'ABC-1234', plateNo: 'ABC-1234', startDate: '2026-01-01' }
    ]
  },
  {
    id: 'drv_3',
    name: '陳國華',
    nationalId: 'K123458177',
    phone: '0933444555',
    email: 'driver3@ltc.example.com',
    active: true,
    createdAt: '2026-01-01',
    assignments: [
      { id: 'asgn_3', driverId: 'drv_3', vehicleId: 'veh_4', vehicleName: '竹南2車', vehiclePlateNo: 'GHI-9012', plateNo: 'GHI-9012', startDate: '2026-01-01' }
    ]
  },
  {
    id: 'drv_4',
    name: '曾建宏',
    nationalId: 'O123453321',
    phone: '0955666777',
    email: 'driver4@ltc.example.com',
    active: true,
    createdAt: '2026-02-01',
    assignments: [
      { id: 'asgn_4', driverId: 'drv_4', vehicleId: 'veh_3', vehicleName: '竹南1車', vehiclePlateNo: 'DEF-5678', plateNo: 'DEF-5678', startDate: '2026-02-01' }
    ]
  },
  {
    id: 'drv_5',
    name: '吳秀珠',
    nationalId: 'J223457788',
    phone: '0966888999',
    email: 'driver5@ltc.example.com',
    active: true,
    createdAt: '2026-03-01',
    assignments: [
      { id: 'asgn_5', driverId: 'drv_5', vehicleId: 'veh_6', vehicleName: '苗栗市1車', vehiclePlateNo: 'MNO-7890', plateNo: 'MNO-7890', startDate: '2026-03-01' }
    ]
  },
  {
    id: 'drv_6',
    name: '黃建民',
    nationalId: 'H123454455',
    phone: '0977123456',
    email: 'driver6@ltc.example.com',
    active: false,
    createdAt: '2026-01-15',
    assignments: []
  },
  {
    id: 'drv_7',
    name: '張美惠',
    nationalId: 'A223456712',
    phone: '0988222333',
    email: 'driver7@ltc.example.com',
    active: true,
    createdAt: '2026-07-01',
    assignments: [
      { id: 'asgn_7', driverId: 'drv_7', vehicleId: 'veh_1', vehicleName: '竹北一車', vehiclePlateNo: 'BZG-7915', plateNo: 'BZG-7915', startDate: '2026-07-01' }
    ]
  }
]

// 照護人員主檔展示資料：涵蓋單位已關聯、單位待關聯（siteNameRaw）、聯絡方式/備註缺漏待補齊三種狀態
export const mockCaregivers: CaregiverDTO[] = [
  {
    id: 'caregiver_1',
    siteId: 'site_1',
    siteName: '竹北日照中心',
    name: '陳小華',
    type: 'case_manager',
    contact: '0912-345-678',
    notes: '熟悉輪椅移位協助',
    createdAt: '2026-01-01',
    updatedAt: '2026-01-01'
  },
  {
    id: 'caregiver_2',
    siteNameRaw: '竹北二日照據點',
    name: '王大明',
    type: 'specialist',
    contact: '0987-654-321',
    notes: '',
    createdAt: '2026-03-01',
    updatedAt: '2026-03-01'
  },
  {
    id: 'caregiver_3',
    siteId: 'site_2',
    siteName: '竹南日照據點',
    name: '李美玲',
    type: 'case_manager',
    contact: '',
    notes: '',
    createdAt: '2026-04-01',
    updatedAt: '2026-04-01'
  }
]

// 個案主檔展示資料：涵蓋所有個案狀態、服務類別、服務機構類型與趟次型態
export const mockCases: CaseDTO[] = [
  // 1. 苗栗 / 補助 / 社區據點 / 2 趟 / 在案
  {
    id: 'case_1',
    code: 'C0001',
    name: '蔡曾切',
    nationalId: 'A202559750',
    homeAddress: '苗栗縣竹南鎮大營路123號',
    region: 'miaoli',
    serviceCategory: 1,
    serviceUsageType: 2,
    status: 'active',
    householdType: '與子女同住',
    gender: '女',
    birthDate: '1948-03-12',
    careContactRole: '個管',
    careContactName: '蔡怡君',
    registeredAddress: '苗栗縣竹南鎮大營路123號',
    siteId: 'site_2',
    siteName: '竹南日照據點',
    outboundVehicleId: 'veh_4',
    outboundVehicle: '竹南2車',
    inboundVehicleId: 'veh_4',
    inboundVehicle: '竹南2車',
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
      scheduleMode: 'monthly',
      weeklyConfigs: [
        { weekday: 1, label: '週一', tripCount: 2, departTime: '09:40', returnTime: '16:00', vehicleId: 'veh_4' },
        { weekday: 2, label: '週二', tripCount: 2, departTime: '09:40', returnTime: '16:00', vehicleId: 'veh_4' },
        { weekday: 3, label: '週三', tripCount: 2, departTime: '09:40', returnTime: '16:00', vehicleId: 'veh_4' },
        { weekday: 4, label: '週四', tripCount: 2, departTime: '09:40', returnTime: '16:00', vehicleId: 'veh_4' },
        { weekday: 5, label: '週五', tripCount: 2, departTime: '09:40', returnTime: '16:00', vehicleId: 'veh_4' },
        { weekday: 6, label: '週六', tripCount: 0, departTime: '09:40', returnTime: '16:00', vehicleId: 'veh_4' },
        { weekday: 7, label: '週日', tripCount: 0, departTime: '09:40', returnTime: '16:00', vehicleId: 'veh_4' }
      ],
      monthlyConfigs: {
        '2026-07-15': { date: '2026-07-15', tripCount: 0, note: '個案家屬通知請假' },
        '2026-07-20': { date: '2026-07-20', tripCount: 4, departTime: '08:30', returnTime: '16:30', vehicleId: 'veh_4', note: '全日活動特開四趟' }
      },
      legs: [
        { id: 'leg_1_1', legSeq: 1, direction: 'outbound', departTime: '09:40', arriveTime: '09:50', runNo: 1, vehicleId: 'veh_4', vehicleName: '竹南2車' },
        { id: 'leg_1_2', legSeq: 2, direction: 'inbound', departTime: '16:00', arriveTime: '16:10', runNo: 1, vehicleId: 'veh_4', vehicleName: '竹南2車' }
      ]
    }
  },
  // 2. 新竹 / 補助 / 社區長照機構 / 4 趟 / 在案
  {
    id: 'case_2',
    code: 'C0002',
    name: '葉秀珍',
    nationalId: 'J220123344',
    homeAddress: '新竹縣竹北市中正西路50號',
    region: 'hsinchu',
    serviceCategory: 1,
    serviceUsageType: 1,
    status: 'active',
    householdType: '獨居',
    gender: '女',
    birthDate: '1952-08-30',
    careContactRole: '照專',
    careContactName: '葉建明',
    registeredAddress: '新竹縣竹北市中正西路50號',
    siteId: 'site_1',
    siteName: '竹北日照中心',
    outboundVehicleId: 'veh_1',
    outboundVehicle: '竹北一車',
    inboundVehicleId: 'veh_1',
    inboundVehicle: '竹北一車',
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
        { id: 'leg_2_1', legSeq: 1, direction: 'outbound', departTime: '08:30', arriveTime: '08:45', runNo: 1, vehicleId: 'veh_1', vehicleName: '竹北一車' },
        { id: 'leg_2_2', legSeq: 2, direction: 'inbound', departTime: '11:30', arriveTime: '11:45', runNo: 1, vehicleId: 'veh_1', vehicleName: '竹北一車' },
        { id: 'leg_2_3', legSeq: 3, direction: 'outbound', departTime: '13:30', arriveTime: '13:45', runNo: 1, vehicleId: 'veh_1', vehicleName: '竹北一車' },
        { id: 'leg_2_4', legSeq: 4, direction: 'inbound', departTime: '16:30', arriveTime: '16:45', runNo: 1, vehicleId: 'veh_1', vehicleName: '竹北一車' }
      ]
    }
  },
  // 3. 新竹 / 補助 / 社區據點 / 1 趟 (單向去程) / 在案
  {
    id: 'case_3',
    code: 'C0003',
    name: '吳𣵛桂',
    nationalId: 'H229875566',
    homeAddress: '新竹縣竹北市福興東路二段88號',
    region: 'hsinchu',
    serviceCategory: 1,
    serviceUsageType: 2,
    status: 'active',
    householdType: '配偶同住',
    gender: '男',
    birthDate: '1945-11-05',
    careContactRole: '個管',
    careContactName: '吳嘉玲',
    registeredAddress: '新竹縣竹北市福興東路二段88號',
    siteId: 'site_1',
    siteName: '竹北日照中心',
    outboundVehicleId: 'veh_1',
    outboundVehicle: '竹北一車',
    inboundVehicleId: 'veh_1',
    inboundVehicle: '竹北一車',
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
        { id: 'leg_3_1', legSeq: 1, direction: 'outbound', departTime: '09:10', arriveTime: '09:20', runNo: 1, vehicleId: 'veh_1', vehicleName: '竹北一車' }
      ]
    }
  },
  // 4. 新竹 / 補助 / 社區據點 / 2 趟 / 暫停 (suspended)
  {
    id: 'case_4',
    code: 'C0004',
    name: '張詹竹妹',
    nationalId: 'O201121122',
    homeAddress: '新竹縣竹北市三民路15號',
    region: 'hsinchu',
    serviceCategory: 1,
    serviceUsageType: 2,
    status: 'suspended',
    householdType: '與子女同住',
    gender: '女',
    birthDate: '1950-01-18',
    careContactRole: '照專',
    careContactName: '張明宏',
    registeredAddress: '新竹縣竹北市三民路15號',
    siteId: 'site_1',
    siteName: '竹北日照中心',
    outboundVehicleId: 'veh_1',
    outboundVehicle: '竹北一車',
    inboundVehicleId: 'veh_1',
    inboundVehicle: '竹北一車',
    createdAt: '2026-06-20',
    updatedAt: '2026-07-01',
    activeSchedule: {
      id: 'sch_4',
      caseId: 'case_4',
      siteId: 'site_1',
      siteName: '竹北日照中心',
      effectiveFrom: '2026-07-01',
      weekdays: [1, 2, 3, 4, 5],
      tripPattern: 2,
      unitPrice: 115,
      distanceKm: 5.5,
      serviceDurationMin: 12,
      serviceCode: 'BD03',
      legs: [
        { id: 'leg_4_1', legSeq: 1, direction: 'outbound', departTime: '08:45', arriveTime: '09:00', runNo: 1, vehicleId: 'veh_1', vehicleName: '竹北一車' },
        { id: 'leg_4_2', legSeq: 2, direction: 'inbound', departTime: '16:15', arriveTime: '16:30', runNo: 1, vehicleId: 'veh_1', vehicleName: '竹北一車' }
      ]
    }
  },
  // 5. 新竹 / 自費 / 社區長照機構 / 2 趟 / 在案 (一三五排班)
  {
    id: 'case_5',
    code: 'C0005',
    name: '李國盛',
    nationalId: 'J123458899',
    homeAddress: '新竹縣竹北市文興路一段200號',
    region: 'hsinchu',
    serviceCategory: 2,
    serviceUsageType: 1,
    status: 'active',
    householdType: '獨居',
    gender: '男',
    birthDate: '1955-06-22',
    careContactRole: '個管',
    careContactName: '李美玲',
    registeredAddress: '新竹縣竹北市文興路一段200號',
    siteId: 'site_1',
    siteName: '竹北日照中心',
    outboundVehicleId: 'veh_2',
    outboundVehicle: '竹北二車',
    inboundVehicleId: 'veh_2',
    inboundVehicle: '竹北二車',
    createdAt: '2026-06-25',
    updatedAt: '2026-07-01',
    activeSchedule: {
      id: 'sch_5',
      caseId: 'case_5',
      siteId: 'site_1',
      siteName: '竹北日照中心',
      effectiveFrom: '2026-07-01',
      weekdays: [1, 3, 5],
      tripPattern: 2,
      unitPrice: 200,
      distanceKm: 8.0,
      serviceDurationMin: 20,
      serviceCode: 'SELF',
      legs: [
        { id: 'leg_5_1', legSeq: 1, direction: 'outbound', departTime: '09:00', arriveTime: '09:20', runNo: 1, vehicleId: 'veh_2', vehicleName: '竹北二車' },
        { id: 'leg_5_2', legSeq: 2, direction: 'inbound', departTime: '15:30', arriveTime: '15:50', runNo: 1, vehicleId: 'veh_2', vehicleName: '竹北二車' }
      ]
    }
  },
  // 6. 新竹 / 補助 / 輔具中心 / 1 趟 (單向回程) / 停案 (closed)
  {
    id: 'case_6',
    code: 'C0006',
    name: '陳素貞',
    nationalId: 'J221234411',
    homeAddress: '新竹縣竹北市縣政九路80號',
    region: 'hsinchu',
    serviceCategory: 1,
    serviceUsageType: 3,
    claimEndDate: '2026-07-31',
    status: 'closed',
    householdType: '與子女同住',
    gender: '女',
    birthDate: '1947-09-09',
    careContactRole: '照專',
    careContactName: '陳志豪',
    registeredAddress: '新竹縣竹北市縣政九路80號',
    siteId: 'site_5',
    siteName: '新竹縣輔具資源中心',
    outboundVehicleId: 'veh_2',
    outboundVehicle: '竹北二車',
    inboundVehicleId: 'veh_2',
    inboundVehicle: '竹北二車',
    createdAt: '2026-05-15',
    updatedAt: '2026-07-31',
    activeSchedule: {
      id: 'sch_6',
      caseId: 'case_6',
      siteId: 'site_5',
      siteName: '新竹縣輔具資源中心',
      effectiveFrom: '2026-06-01',
      weekdays: [3],
      tripPattern: 1,
      unitPrice: 115,
      distanceKm: 4.0,
      serviceDurationMin: 10,
      serviceCode: 'BD04',
      legs: [
        { id: 'leg_6_1', legSeq: 1, direction: 'inbound', departTime: '16:00', arriveTime: '16:15', runNo: 1, vehicleId: 'veh_2', vehicleName: '竹北二車' }
      ]
    }
  },
  // 7. 苗栗 / 補助 / 身障日間照顧服務 / 2 趟 / 在案 (週一至週六)
  {
    id: 'case_7',
    code: 'C0007',
    name: '黃天賜',
    nationalId: 'K123459900',
    homeAddress: '苗栗縣竹南鎮延平路66號',
    region: 'miaoli',
    serviceCategory: 1,
    serviceUsageType: 4,
    status: 'active',
    householdType: '與家人同住',
    gender: '男',
    birthDate: '1990-04-14',
    careContactRole: '個管',
    careContactName: '黃淑芬',
    registeredAddress: '苗栗縣竹南鎮延平路66號',
    siteId: 'site_6',
    siteName: '竹南身障日間作業據點',
    outboundVehicleId: 'veh_3',
    outboundVehicle: '竹南1車',
    inboundVehicleId: 'veh_3',
    inboundVehicle: '竹南1車',
    createdAt: '2026-06-28',
    updatedAt: '2026-07-01',
    activeSchedule: {
      id: 'sch_7',
      caseId: 'case_7',
      siteId: 'site_6',
      siteName: '竹南身障日間作業據點',
      effectiveFrom: '2026-07-01',
      weekdays: [1, 2, 3, 4, 5, 6],
      tripPattern: 2,
      unitPrice: 115,
      distanceKm: 7.2,
      serviceDurationMin: 15,
      serviceCode: 'BD03',
      legs: [
        { id: 'leg_7_1', legSeq: 1, direction: 'outbound', departTime: '08:15', arriveTime: '08:30', runNo: 1, vehicleId: 'veh_3', vehicleName: '竹南1車' },
        { id: 'leg_7_2', legSeq: 2, direction: 'inbound', departTime: '15:45', arriveTime: '16:00', runNo: 1, vehicleId: 'veh_3', vehicleName: '竹南1車' }
      ]
    }
  },
  // 8. 苗栗 / 自費 / 身障日間照顧服務 / 4 趟 / 在案
  {
    id: 'case_8',
    code: 'C0008',
    name: '彭阿土',
    nationalId: 'K102342233',
    homeAddress: '苗栗縣苗栗市中正路500號',
    region: 'miaoli',
    serviceCategory: 2,
    serviceUsageType: 4,
    status: 'active',
    householdType: '與家人同住',
    gender: '男',
    birthDate: '1985-12-01',
    careContactRole: '照專',
    careContactName: '彭美惠',
    registeredAddress: '苗栗縣苗栗市中正路500號',
    siteId: 'site_4',
    siteName: '苗栗市社區據點',
    outboundVehicleId: 'veh_6',
    outboundVehicle: '苗栗市1車',
    inboundVehicleId: 'veh_6',
    inboundVehicle: '苗栗市1車',
    createdAt: '2026-06-30',
    updatedAt: '2026-07-01',
    activeSchedule: {
      id: 'sch_8',
      caseId: 'case_8',
      siteId: 'site_4',
      siteName: '苗栗市社區據點',
      effectiveFrom: '2026-07-01',
      weekdays: [2, 4],
      tripPattern: 4,
      unitPrice: 180,
      distanceKm: 9.0,
      serviceDurationMin: 25,
      serviceCode: 'SELF',
      legs: [
        { id: 'leg_8_1', legSeq: 1, direction: 'outbound', departTime: '08:00', arriveTime: '08:25', runNo: 1, vehicleId: 'veh_6', vehicleName: '苗栗市1車' },
        { id: 'leg_8_2', legSeq: 2, direction: 'inbound', departTime: '11:00', arriveTime: '11:25', runNo: 1, vehicleId: 'veh_6', vehicleName: '苗栗市1車' },
        { id: 'leg_8_3', legSeq: 3, direction: 'outbound', departTime: '13:00', arriveTime: '13:25', runNo: 1, vehicleId: 'veh_6', vehicleName: '苗栗市1車' },
        { id: 'leg_8_4', legSeq: 4, direction: 'inbound', departTime: '16:00', arriveTime: '16:25', runNo: 1, vehicleId: 'veh_6', vehicleName: '苗栗市1車' }
      ]
    }
  },
  // 9. 新竹 / 補助 / 輔具中心 / 1 趟 (單向去程) / 在案
  {
    id: 'case_9',
    code: 'C0009',
    name: '邱美蘭',
    nationalId: 'J203456677',
    homeAddress: '新竹縣湖口鄉達生路33號',
    region: 'hsinchu',
    serviceCategory: 1,
    serviceUsageType: 3,
    status: 'active',
    householdType: '獨居',
    gender: '女',
    birthDate: '1949-02-27',
    careContactRole: '個管',
    careContactName: '邱志宏',
    registeredAddress: '新竹縣湖口鄉達生路33號',
    siteId: 'site_3',
    siteName: '湖口長照據點',
    outboundVehicleId: 'veh_2',
    outboundVehicle: '竹北二車',
    inboundVehicleId: 'veh_2',
    inboundVehicle: '竹北二車',
    createdAt: '2026-07-01',
    updatedAt: '2026-07-01',
    activeSchedule: {
      id: 'sch_9',
      caseId: 'case_9',
      siteId: 'site_3',
      siteName: '湖口長照據點',
      effectiveFrom: '2026-07-01',
      weekdays: [5],
      tripPattern: 1,
      unitPrice: 115,
      distanceKm: 3.5,
      serviceDurationMin: 10,
      serviceCode: 'BD04',
      legs: [
        { id: 'leg_9_1', legSeq: 1, direction: 'outbound', departTime: '09:30', arriveTime: '09:40', runNo: 1, vehicleId: 'veh_2', vehicleName: '竹北二車' }
      ]
    }
  },
  // 10. 苗栗 / 補助 / 社區據點 / 2 趟 / 在案
  {
    id: 'case_10',
    code: 'C0010',
    name: '林阿祥',
    nationalId: 'K124561234',
    homeAddress: '苗栗縣竹南鎮光復路88號',
    region: 'miaoli',
    serviceCategory: 1,
    serviceUsageType: 2,
    status: 'active',
    householdType: '與配偶同住',
    gender: '男',
    birthDate: '1953-10-16',
    careContactRole: '照專',
    careContactName: '林淑娟',
    registeredAddress: '苗栗縣竹南鎮光復路88號',
    siteId: 'site_2',
    siteName: '竹南日照據點',
    outboundVehicleId: 'veh_4',
    outboundVehicle: '竹南2車',
    inboundVehicleId: 'veh_4',
    inboundVehicle: '竹南2車',
    createdAt: '2026-07-01',
    updatedAt: '2026-07-01',
    activeSchedule: {
      id: 'sch_10',
      caseId: 'case_10',
      siteId: 'site_2',
      siteName: '竹南日照據點',
      effectiveFrom: '2026-07-01',
      weekdays: [1, 2, 3, 4, 5],
      tripPattern: 2,
      unitPrice: 115,
      distanceKm: 6.0,
      serviceDurationMin: 12,
      serviceCode: 'BD03',
      scheduleMode: 'by_weekday',
      weeklyConfigs: [
        { weekday: 1, label: '週一', tripCount: 2, departTime: '08:50', returnTime: '15:50', vehicleId: 'veh_4' },
        { weekday: 2, label: '週二', tripCount: 1, departTime: '08:50', returnTime: '15:50', vehicleId: 'veh_4' },
        { weekday: 3, label: '週三', tripCount: 2, departTime: '08:50', returnTime: '15:50', vehicleId: 'veh_4' },
        { weekday: 4, label: '週四', tripCount: 1, departTime: '08:50', returnTime: '15:50', vehicleId: 'veh_4' },
        { weekday: 5, label: '週五', tripCount: 2, departTime: '08:50', returnTime: '15:50', vehicleId: 'veh_4' },
        { weekday: 6, label: '週六', tripCount: 0, departTime: '08:50', returnTime: '15:50', vehicleId: 'veh_4' },
        { weekday: 7, label: '週日', tripCount: 0, departTime: '08:50', returnTime: '15:50', vehicleId: 'veh_4' }
      ],
      legs: [
        { id: 'leg_10_1', legSeq: 1, direction: 'outbound', departTime: '08:50', arriveTime: '09:05', runNo: 1, vehicleId: 'veh_4', vehicleName: '竹南2車' },
        { id: 'leg_10_2', legSeq: 2, direction: 'inbound', departTime: '15:50', arriveTime: '16:05', runNo: 1, vehicleId: 'veh_4', vehicleName: '竹南2車' }
      ]
    }
  },
  // 11. 苗栗 / 匯入資料中據點與去程車輛比對不到主檔，示範「待補建關聯」頁籤
  {
    id: 'case_11',
    code: 'C0011',
    name: '邱美玲',
    nationalId: 'K220987654',
    homeAddress: '苗栗縣頭份市中華路50號',
    region: 'miaoli',
    serviceCategory: 1,
    serviceUsageType: 2,
    status: 'active',
    householdType: '獨居',
    gender: '女',
    birthDate: '1950-05-20',
    careContactRole: '個管',
    careContactName: '邱志明',
    registeredAddress: '苗栗縣頭份市中華路50號',
    remarks: '需輪椅接送，行動不便',
    siteNameRaw: '頭份日照中心（新）',
    outboundVehicleNameRaw: '頭份1號車',
    createdAt: '2026-08-01',
    updatedAt: '2026-08-01'
  }
]

// 司機接送匯報表展示資料：一台車一份匯報表，涵蓋已完成對應與仍有待對應欄位兩種狀態
export const mockDriverReportForms: DriverReportFormDTO[] = [
  {
    id: 'form_1',
    vehicleId: 'veh_1',
    vehicleName: '竹北一車',
    title: '竹北一車接送匯報',
    region: 'hsinchu',
    lastImportedAt: '2026-08-25 14:00:00',
    totalColumns: 56,
    mappedColumns: 53,
    pendingColumns: 3,
    submissionCount: 42,
    status: 'active'
  },
  {
    id: 'form_2',
    vehicleId: 'veh_2',
    vehicleName: '竹南2車',
    title: '竹南2車接送匯報',
    region: 'miaoli',
    lastImportedAt: '2026-08-25 14:05:00',
    totalColumns: 53,
    mappedColumns: 53,
    pendingColumns: 0,
    submissionCount: 106,
    status: 'active'
  },
  {
    id: 'form_3',
    vehicleId: 'veh_3',
    vehicleName: '竹北二車',
    title: '竹北二車接送匯報',
    region: 'hsinchu',
    lastImportedAt: '2026-08-22 09:00:00',
    totalColumns: 48,
    mappedColumns: 43,
    pendingColumns: 5,
    submissionCount: 30,
    status: 'active'
  },
  {
    id: 'form_4',
    vehicleId: 'veh_4',
    vehicleName: '苗栗市1車',
    title: '苗栗市1車接送匯報',
    region: 'miaoli',
    lastImportedAt: null,
    totalColumns: 0,
    mappedColumns: 0,
    pendingColumns: 0,
    submissionCount: 0,
    status: 'active'
  }
]

// 系統使用者展示資料：涵蓋系統管理員、調度員、司機、檢視者
export const mockUsers: UserDTO[] = [
  {
    id: 'usr_admin',
    email: 'admin@ltc.example.com',
    displayName: '系統管理員 (王大明)',
    role: 'admin',
    phone: '0912-111-222',
    status: 'active',
    customPermissions: null,
    lastLoginAt: '2026-08-26 09:15:00',
    createdAt: '2026-01-01 08:00:00'
  },
  {
    id: 'usr_viewer_1',
    email: 'viewer@ltc.example.com',
    displayName: '主管檢視者 (林督導)',
    role: 'viewer',
    phone: '0944-777-888',
    status: 'active',
    customPermissions: null,
    lastLoginAt: '2026-08-24 16:20:00',
    createdAt: '2026-02-15 10:00:00'
  }
]

// 角色身分展示資料：包含系統內建核心角色與示範自訂角色
export const mockRoles: RoleDTO[] = [
  {
    id: 'role_admin',
    key: 'admin',
    name: '系統管理員',
    description: '具備全系統最高權限，可管理使用者帳號、角色、稽核紀錄與所有主檔及申報功能。',
    tagType: 'danger',
    isSystem: true,
    permissions: JSON.parse(JSON.stringify(DEFAULT_ROLE_PERMISSIONS.admin)),
    createdAt: '2026-01-01 00:00:00',
    updatedAt: '2026-01-01 00:00:00'
  },
  {
    id: 'role_viewer',
    key: 'viewer',
    name: '檢視者',
    description: '僅具備全系統營運資料之唯讀檢視權限，無法進行任何新增、修改或刪除操作。',
    tagType: 'info',
    isSystem: true,
    permissions: JSON.parse(JSON.stringify(DEFAULT_ROLE_PERMISSIONS.viewer)),
    createdAt: '2026-01-01 00:00:00',
    updatedAt: '2026-01-01 00:00:00'
  },
  {
    id: 'role_auditor',
    key: 'auditor',
    name: '外部稽核督導',
    description: '專門負責政府評鑑抽查、搭乘日誌校驗與系統操作稽核之唯讀進階身分。',
    tagType: 'warning',
    isSystem: false,
    permissions: {
      dashboard: { view: true, edit: false },
      masters_regions: { view: true, edit: false },
      masters_cases: { view: true, edit: false },
      masters_sites: { view: true, edit: false },
      masters_vehicles: { view: true, edit: false },
      masters_drivers: { view: true, edit: false },
      driver_reports: { view: true, edit: false },
      driver_report_mappings: { view: true, edit: false },
      rides_calendar: { view: true, edit: false },
      rides_issues: { view: true, edit: false },
      rides_missing: { view: true, edit: false },
      reports_trip_summary: { view: true, edit: false },
      reports_hsinchu_schedule: { view: true, edit: false },
      vehicles_maintenance: { view: true, edit: false },
      attendance_fuel: { view: true, edit: false },
      exports: { view: true, edit: false },
      audit_logs: { view: true, edit: false },
      settings_notifications: { view: true, edit: false },
      settings_users: { view: false, edit: false },
      settings_roles: { view: false, edit: false }
    },
    createdAt: '2026-02-20 11:30:00',
    updatedAt: '2026-02-20 11:30:00'
  }
]

// 匯報欄位對應展示資料：表頭沿用實際匯報檔的寫法，涵蓋 pending／mapped／ignored 三種狀態
export const mockDriverReportColumns: DriverReportColumnDTO[] = [
  {
    id: 'col_1',
    formId: 'form_1',
    formTitle: '竹北一車接送匯報',
    vehicleName: '竹北一車',
    columnIndex: 3,
    columnHeader: '1.張詹竹妹 [去程]',
    cleanedName: '張詹竹妹',
    kind: 'ride',
    mappingStatus: 'pending',
    caseId: null,
    caseName: null,
    legSeq: null,
    suggestedCaseId: 'case_4',
    suggestedCaseName: '張詹竹妹',
    suggestedLegSeq: 1,
    suggestionScore: 1.0
  },
  {
    id: 'col_2',
    formId: 'form_1',
    formTitle: '竹北一車接送匯報',
    vehicleName: '竹北一車',
    columnIndex: 4,
    columnHeader: '4.葉秀珍 (4趟) [去程]',
    cleanedName: '葉秀珍',
    kind: 'ride',
    mappingStatus: 'pending',
    caseId: null,
    caseName: null,
    legSeq: null,
    suggestedCaseId: 'case_2',
    suggestedCaseName: '葉秀珍',
    suggestedLegSeq: 1,
    suggestionScore: 0.85
  },
  {
    id: 'col_3',
    formId: 'form_1',
    formTitle: '竹北一車接送匯報',
    vehicleName: '竹北一車',
    columnIndex: 5,
    columnHeader: '1.吳𣵛桂(去程竹3) [去程]',
    cleanedName: '吳𣵛桂',
    kind: 'ride',
    mappingStatus: 'mapped',
    caseId: 'case_3',
    caseName: '吳𣵛桂',
    legSeq: 1,
    suggestedCaseId: 'case_3',
    suggestedCaseName: '吳𣵛桂',
    suggestedLegSeq: 1,
    suggestionScore: 0.8
  },
  {
    id: 'col_4',
    formId: 'form_1',
    formTitle: '竹北一車接送匯報',
    vehicleName: '竹北一車',
    columnIndex: 6,
    columnHeader: '1.吳𣵛桂(去程竹3) [回程]',
    cleanedName: '吳𣵛桂',
    kind: 'ride',
    mappingStatus: 'mapped',
    caseId: 'case_3',
    caseName: '吳𣵛桂',
    legSeq: 2,
    suggestedCaseId: 'case_3',
    suggestedCaseName: '吳𣵛桂',
    suggestedLegSeq: 2,
    suggestionScore: 0.8
  },
  {
    id: 'col_5',
    formId: 'form_3',
    formTitle: '竹北二車接送匯報',
    vehicleName: '竹北二車',
    columnIndex: 12,
    columnHeader: '新進未知個案-測試欄位',
    cleanedName: '新進未知個案-測試欄位',
    kind: 'unknown',
    mappingStatus: 'ignored',
    caseId: null,
    caseName: null,
    legSeq: null,
    suggestedCaseId: null,
    suggestedCaseName: null,
    suggestedLegSeq: null,
    suggestionScore: 0.3
  }
]

// 申報前置檢核展示資料：涵蓋錯誤 (阻擋項)、警告 (可強制執行) 與資訊 (提示)
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
        { caseId: 'case_1', caseName: '蔡曾切', serviceDate: '2026-07-15', description: '07/15 回程未取得司機回報' },
        { caseId: 'case_2', caseName: '葉秀珍', serviceDate: '2026-07-24', description: '07/24 第4趟未取得司機回報' }
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
      message: '本月包含已人工更正紀錄共 2 筆（已留存完整稽核軌跡）'
    },
    {
      level: 'info',
      code: 'QUOTA_CHECK_UNAVAILABLE',
      message: '個案配給額度檢查未執行——系統依現有實際排班趟次核算'
    }
  ]
}

// 匯出工作展示資料：政府申報匯出一律一個個案一個月一份檔案，涵蓋直接下載、壓縮檔與失敗三種情境
export const mockExportJobs: ExportJobDTO[] = [
  {
    id: 'job_11507_direct',
    jobType: 'gov_claim',
    periodYm: '11507',
    region: 'hsinchu',
    mode: 'direct',
    status: 'succeeded',
    totalCases: 2,
    totalRows: 78,
    files: [
      {
        caseId: 'case_2',
        caseCode: 'C0002',
        caseName: '葉秀珍',
        region: 'hsinchu',
        rowCount: 39,
        fileName: '葉秀珍11507.xlsx',
        downloadUrl: '/api/v1/exports/job_11507_direct/files/case_2/download'
      },
      {
        caseId: 'case_3',
        caseCode: 'C0003',
        caseName: '吳𣵛桂',
        region: 'hsinchu',
        rowCount: 39,
        fileName: '吳𣵛桂11507.xlsx',
        downloadUrl: '/api/v1/exports/job_11507_direct/files/case_3/download'
      }
    ],
    createdAt: '2026-08-01T10:00:00+08:00',
    completedAt: '2026-08-01T10:00:15+08:00'
  },
  {
    id: 'job_11507_zip',
    jobType: 'gov_claim',
    periodYm: '11507',
    region: 'miaoli',
    mode: 'zip',
    status: 'succeeded',
    totalCases: 1,
    totalRows: 39,
    files: [
      {
        caseId: 'case_1',
        caseCode: 'C0001',
        caseName: '蔡曾切',
        region: 'miaoli',
        rowCount: 39,
        fileName: '蔡曾切11507.xlsx',
        downloadUrl: '/api/v1/exports/job_11507_zip/files/case_1/download'
      }
    ],
    zipFileName: 'gov-claim-miaoli-11507.zip',
    downloadUrl: '/api/v1/exports/job_11507_zip/download',
    createdAt: '2026-08-01T10:05:00+08:00',
    completedAt: '2026-08-01T10:05:08+08:00'
  },
  {
    id: 'job_11508_failed',
    jobType: 'gov_claim',
    periodYm: '11508',
    region: 'hsinchu',
    mode: 'direct',
    status: 'failed',
    totalCases: 0,
    totalRows: 0,
    errorMessage: '指定條件下沒有可申報的搭乘紀錄',
    createdAt: '2026-08-25T15:45:00+08:00',
    completedAt: '2026-08-25T15:45:04+08:00'
  }
]

// 儀表板統計展示資料
export const mockDashboardStats: DashboardStatsDTO = {
  currentMonth: '115-07',
  totalCasesCount: 186,
  reportedTripsCount: 2450,
  unreportedVehiclesToday: 1,
  pendingConflictsCount: 1,
  pendingFormColumnsCount: 4,
  recentExports: mockExportJobs
}

// 系統操作紀錄展示資料：涵蓋登入 (login)、主檔 CUD、事後補報 (manual_report)、更正 (correct)、衝突裁決 (resolve_conflict)、匯出 (export) 與設定變更
export const mockAuditLogs: AuditLogDTO[] = [
  {
    id: 'audit_manual_report_demo',
    actorId: 'usr_staff',
    actorName: '行政承辦',
    actorRole: 'staff',
    action: 'manual_report',
    entityType: 'ride_records',
    entityId: 'c87eefa5-8b94-4362-b6e2-aa564f52080a',
    beforeData: undefined,
    afterData: {
      caseId: 'case_1',
      caseName: '蔡曾切',
      serviceDate: '2026-08-28',
      legSeq: 1,
      effectiveStatus: 'boarded',
      departTimeOverride: '09:15',
      durationMinOverride: 15,
      notClaimedAa09: false,
      reason: '司機電話回報補登'
    },
    ipAddress: '192.168.1.105',
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
    createdAt: '2026-08-28 09:20:00'
  },
  {
    id: 'audit_0',
    actorId: 'usr_admin',
    actorName: '系統管理員',
    actorRole: 'admin',
    action: 'login',
    entityType: 'auth',
    entityId: 'usr_admin',
    entityName: '系統管理員 (admin@ltc.example.com)',
    beforeData: undefined,
    afterData: {
      email: 'admin@ltc.example.com',
      role: 'admin',
      loginTime: '2026-08-26 09:15:00'
    },
    ipAddress: '192.168.1.100',
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
    createdAt: '2026-08-26 09:15:00'
  },
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
    beforeData: { active: false },
    afterData: { active: true },
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
  },
  {
    id: 'audit_6',
    actorId: 'usr_admin',
    actorName: '系統管理員',
    actorRole: 'admin',
    action: 'create',
    entityType: 'cases',
    entityId: 'case_10',
    entityName: '林阿祥',
    beforeData: undefined,
    afterData: { code: 'C0010', region: 'miaoli', status: 'active' },
    ipAddress: '192.168.1.100',
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
    createdAt: '2026-07-01 09:00:00'
  },
  {
    id: 'audit_7',
    actorId: 'usr_staff',
    actorName: '行政承辦',
    actorRole: 'staff',
    action: 'update',
    entityType: 'vehicles',
    entityId: 'veh_5',
    entityName: '竹東一車',
    beforeData: { active: true },
    afterData: { active: false, reason: '進廠大保養' },
    ipAddress: '192.168.1.105',
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
    createdAt: '2026-08-20 14:10:00'
  },
  {
    id: 'audit_8',
    actorId: 'usr_admin',
    actorName: '系統管理員',
    actorRole: 'admin',
    action: 'delete',
    entityType: 'sites',
    entityId: 'site_old_0',
    entityName: '已停用舊服務站點',
    beforeData: { status: 'inactive' },
    afterData: undefined,
    ipAddress: '192.168.1.100',
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
    createdAt: '2026-08-10 11:30:00'
  },
  {
    id: 'audit_9',
    actorId: 'usr_staff',
    actorName: '行政承辦',
    actorRole: 'staff',
    action: 'import',
    entityType: 'cases',
    entityId: 'import_batch_1',
    entityName: '新進個案名冊批次匯入 (共14筆)',
    beforeData: undefined,
    afterData: { importedCount: 14, errorCount: 1 },
    ipAddress: '192.168.1.105',
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
    createdAt: '2026-08-15 16:40:00'
  }
]

// 通知收件人展示資料：自訂外部信箱
export const mockNotificationRecipients: NotificationRecipientDTO[] = [
  {
    id: 'rec_1',
    topic: 'missing_report',
    email: 'admin@ltc.example.com',
    displayName: '系統管理組',
    active: true,
    createdByName: '系統管理員',
    createdAt: '2026-01-01 00:00:00'
  },
  {
    id: 'rec_2',
    topic: 'missing_report',
    email: 'dispatcher@ltc.example.com',
    displayName: '苗栗調度中心',
    active: true,
    createdByName: '系統管理員',
    createdAt: '2026-01-05 10:30:00'
  },
  {
    id: 'rec_3',
    topic: 'driver_leave',
    email: 'dispatch_lead@ltc.example.com',
    displayName: '調度組長',
    active: true,
    createdByName: '系統管理員',
    createdAt: '2026-01-10 14:00:00'
  },
  {
    id: 'rec_4',
    topic: 'month_end',
    email: 'supervisor@gov.example.tw',
    displayName: '長照督導辦公室',
    active: true,
    createdByName: '系統管理員',
    createdAt: '2026-01-15 09:15:00'
  },
  {
    id: 'rec_5',
    topic: 'export_failed',
    email: 'tech@ltc.example.com',
    displayName: '外部資訊維運團隊',
    active: true,
    createdByName: '系統管理員',
    createdAt: '2026-02-01 16:20:00'
  }
]

// 異常搭乘集中處理展示資料：涵蓋混車衝突待裁決 (conflict)、應搭未回報清單 (unreported) 與表單匯入異常 (import_error)
export const mockIssueRides: IssueRideDTO[] = [
  // 1. 混車衝突待裁決 (conflict)
  {
    id: 'ride_conflict_1',
    caseId: 'case_2',
    caseName: '葉秀珍',
    serviceDate: '2026-08-20',
    legSeq: 1,
    issueType: 'conflict',
    hasConflict: true,
    description: '竹北一車與竹北二車皆於 08:30 回報「有坐」，需指定正確承載車輛與司機',
    vehicles: ['竹北一車', '竹北二車']
  },
  {
    id: 'ride_conflict_2',
    caseId: 'case_1',
    caseName: '蔡曾切',
    serviceDate: '2026-08-22',
    legSeq: 2,
    issueType: 'conflict',
    hasConflict: true,
    description: '竹南1車與竹南2車於下午回程皆勾選載送，司機填報紀錄衝突',
    vehicles: ['竹南1車', '竹南2車']
  },
  {
    id: 'ride_conflict_3',
    caseId: 'case_5',
    caseName: '李國盛',
    serviceDate: '2026-08-25',
    legSeq: 1,
    issueType: 'conflict',
    hasConflict: true,
    description: '排定竹北二車承載，但竹北一車表單亦提交該員搭乘，需人工確認實際接送車輛',
    vehicles: ['竹北二車', '竹北一車']
  },
  {
    id: 'ride_conflict_4',
    caseId: 'case_8',
    caseName: '彭阿土',
    serviceDate: '2026-08-26',
    legSeq: 3,
    issueType: 'conflict',
    hasConflict: true,
    description: '苗栗市1車與竹南2車下午第 3 趟重複載送回報，跨車排程衝突',
    vehicles: ['苗栗市1車', '竹南2車']
  },

  // 2. 應搭未回報清單 (unreported)
  {
    id: 'ride_unrep_1',
    caseId: 'case_1',
    caseName: '蔡曾切',
    serviceDate: '2026-08-15',
    legSeq: 2,
    issueType: 'unreported',
    hasConflict: false,
    description: '08/15 第 2 趟（回程）司機尚未提交表單回覆',
    vehicles: ['竹南2車']
  },
  {
    id: 'ride_unrep_2',
    caseId: 'case_2',
    caseName: '葉秀珍',
    serviceDate: '2026-08-24',
    legSeq: 4,
    issueType: 'unreported',
    hasConflict: false,
    description: '08/24 第 4 趟（回程）司機未提交回覆，已發送催報提醒',
    vehicles: ['竹北一車']
  },
  {
    id: 'ride_unrep_3',
    caseId: 'case_3',
    caseName: '吳𣵛桂',
    serviceDate: '2026-08-22',
    legSeq: 1,
    issueType: 'unreported',
    hasConflict: false,
    description: '08/22 第 1 趟（去程）排班應搭，司機忘記填寫回報表',
    vehicles: ['竹北一車']
  },
  {
    id: 'ride_unrep_4',
    caseId: 'case_7',
    caseName: '黃天賜',
    serviceDate: '2026-08-20',
    legSeq: 1,
    issueType: 'unreported',
    hasConflict: false,
    description: '08/20 第 1 趟（去程）竹南1車未見回報日誌，逾期 5 天',
    vehicles: ['竹南1車']
  },
  {
    id: 'ride_unrep_5',
    caseId: 'case_10',
    caseName: '林阿祥',
    serviceDate: '2026-08-25',
    legSeq: 2,
    issueType: 'unreported',
    hasConflict: false,
    description: '08/25 第 2 趟（回程）竹南2車未提交回報，待行政人員確認',
    vehicles: ['竹南2車']
  },

  // 3. 表單匯入異常 (import_error)
  {
    id: 'err_1',
    caseId: 'case_unknown',
    caseName: '去程到08/21',
    serviceDate: '2026-08-21',
    legSeq: 1,
    issueType: 'import_error',
    hasConflict: false,
    description: '搭乘欄填寫非標準字串「去程到08/21」，無法自動解析為有坐/沒坐',
    vehicles: ['竹北一車'],
    rawPayload: '{"vehicle":"竹北一車","serviceDate":"2026-08-21","leg":1,"field":"搭乘","rawValue":"去程到08/21","submittedAt":"2026-08-21T08:12:00+08:00"}'
  },
  {
    id: 'err_2',
    caseId: 'case_unknown',
    caseName: '家屬臨時改下午一點接',
    serviceDate: '2026-08-23',
    legSeq: 2,
    issueType: 'import_error',
    hasConflict: false,
    description: '回報欄位填入說明文字「家屬臨時改下午一點接」，無法對應標準時間或搭乘狀態欄位',
    vehicles: ['竹南1車'],
    rawPayload: '{"vehicle":"竹南1車","serviceDate":"2026-08-23","leg":2,"field":"回程時間","rawValue":"家屬臨時改下午一點接","submittedAt":"2026-08-23T13:05:00+08:00"}'
  },
  {
    id: 'err_3',
    caseId: 'case_2',
    caseName: '葉秀珍 (重複時間戳記)',
    serviceDate: '2026-08-24',
    legSeq: 1,
    issueType: 'import_error',
    hasConflict: false,
    description: '同一司機於 1 分鐘內重複提交兩次相異狀態表單（第一次勾沒坐，第二次勾有坐）',
    vehicles: ['竹北一車'],
    rawPayload: '[{"submittedAt":"2026-08-24T09:30:00+08:00","status":"沒坐"},{"submittedAt":"2026-08-24T09:31:00+08:00","status":"有坐"}]'
  },
  {
    id: 'err_4',
    caseId: 'case_unknown',
    caseName: '未知個案「王大明」',
    serviceDate: '2026-08-25',
    legSeq: 1,
    issueType: 'import_error',
    hasConflict: false,
    description: '表單回報名冊中包含未建檔之個案姓名「王大明」，無法自動關聯個案主檔代碼',
    vehicles: ['苗栗市1車'],
    rawPayload: '{"vehicle":"苗栗市1車","serviceDate":"2026-08-25","leg":1,"field":"個案姓名","rawValue":"王大明","submittedAt":"2026-08-25T07:50:00+08:00"}'
  },
  {
    id: 'err_5',
    caseId: 'case_unknown',
    caseName: '日期格式異常 (115/8/26 下午)',
    serviceDate: '2026-08-26',
    legSeq: 1,
    issueType: 'import_error',
    hasConflict: false,
    description: '表單服務日期填寫為「115/8/26 下午」，非標準 ISO 日期格式致系統解析失敗',
    vehicles: ['竹北二車'],
    rawPayload: '{"vehicle":"竹北二車","field":"服務日期","rawValue":"115/8/26 下午","submittedAt":"2026-08-26T10:00:00+08:00"}'
  }
]

// 通知發送歷史日誌：涵蓋全部 4 種主題、成功與失敗狀態
export const mockNotificationLogs: NotificationLogDTO[] = [
  {
    id: 'nlog_1',
    topic: 'missing_report',
    channel: 'email',
    recipientEmails: ['admin@ltc.example.com', 'staff.miaoli@ltc.example.com'],
    subject: '【長照交通系統】今日未回報催報通知 (2026-08-26)',
    contentSummary: '竹南2車、竹北一車尚有 2 筆今日應搭乘趟次未提交回報，請司機或調度員儘速核對。',
    status: 'sent',
    triggeredByName: '系統定時排程 (Cloud Scheduler)',
    sentAt: '2026-08-26 18:00:02'
  },
  {
    id: 'nlog_2',
    topic: 'missing_report',
    channel: 'email',
    recipientEmails: ['dispatcher@ltc.example.com', 'dispatch_lead@ltc.example.com'],
    subject: '【長照交通系統】手動催報執行通知 (2026-08-26)',
    contentSummary: '調度員手動執行全車隊未回報檢核，已發送未回報提醒，共計 12 筆待回報項目。',
    status: 'sent',
    triggeredByName: '當前操作人員 (手動觸發)',
    sentAt: '2026-08-26 11:30:15'
  },
  {
    id: 'nlog_3',
    topic: 'month_end',
    channel: 'email',
    recipientEmails: ['finance@ltc.example.com', 'supervisor@gov.example.tw'],
    subject: '【長照交通系統】115年08月份申報資料結算提醒 (月底即將截數)',
    contentSummary: '本月營運日即將結束，目前尚有 4 筆混車衝突待裁決與 12 筆未回報，請於月底前完成處理以利申報匯出。',
    status: 'sent',
    triggeredByName: '系統定時排程 (Cloud Scheduler)',
    sentAt: '2026-08-26 09:00:00'
  },
  {
    id: 'nlog_4',
    topic: 'driver_leave',
    channel: 'email',
    recipientEmails: ['dispatch@ltc.example.com', 'dispatch_lead@ltc.example.com'],
    subject: '【調度通報】司機 林志豪 於 2026-08-27 請事假一日',
    contentSummary: '竹北二車需安排代班司機（郭澤威）或跨車支援調整。',
    status: 'sent',
    triggeredByName: '出勤登記作業',
    sentAt: '2026-08-25 17:15:00'
  },
  {
    id: 'nlog_5',
    topic: 'missing_report',
    channel: 'email',
    recipientEmails: ['admin@ltc.example.com'],
    subject: '【長照交通系統】今日未回報催報通知 (2026-08-25)',
    contentSummary: '竹北一車、竹北二車尚有 2 筆應搭乘趟次未提交回報。',
    status: 'sent',
    triggeredByName: '系統定時排程 (Cloud Scheduler)',
    sentAt: '2026-08-25 18:00:01'
  },
  {
    id: 'nlog_6',
    topic: 'export_failed',
    channel: 'email',
    recipientEmails: ['tech@ltc.example.com'],
    subject: '【警報】空白保養表產生異常',
    contentSummary: '批次產生任務 job_202608_04 模板套用失敗',
    status: 'failed',
    errorMessage: '模板樣式套用異常：工作表名稱「保養紀錄表」衝突',
    triggeredByName: '匯出非同步任務',
    sentAt: '2026-08-25 15:45:05'
  },
  {
    id: 'nlog_7',
    topic: 'driver_leave',
    channel: 'email',
    recipientEmails: ['dispatch@ltc.example.com'],
    subject: '【調度通報】司機 曾建宏 於 2026-08-21 請病假一日',
    contentSummary: '竹南1車當日出勤異動，已指派陳國華司機跨車支援。',
    status: 'sent',
    triggeredByName: '出勤登記作業',
    sentAt: '2026-08-20 18:30:00'
  },
  {
    id: 'nlog_8',
    topic: 'missing_report',
    channel: 'email',
    recipientEmails: ['staff.miaoli@ltc.example.com'],
    subject: '【長照交通系統】苗栗區未回報趟次催報警示',
    contentSummary: '竹南1車、苗栗市1車逾期逾 3 天之未回報趟次共 3 筆，請承辦人員電話確認。',
    status: 'sent',
    triggeredByName: '系統定時排程 (Cloud Scheduler)',
    sentAt: '2026-08-20 08:30:00'
  },
  {
    id: 'nlog_9',
    topic: 'export_failed',
    channel: 'email',
    recipientEmails: ['tech@ltc.example.com', 'admin@ltc.example.com'],
    subject: '【警報】新竹縣申報表批次匯出連線逾時',
    contentSummary: '申報批次匯出 job_202607_02 產生過程發生 Google API 額度限制連線逾時',
    status: 'failed',
    errorMessage: 'Google Sheets API Rate Limit Exceeded (HTTP 429)',
    triggeredByName: '申報匯出排程',
    sentAt: '2026-08-01 10:30:22'
  },
  {
    id: 'nlog_10',
    topic: 'month_end',
    channel: 'email',
    recipientEmails: ['finance@ltc.example.com'],
    subject: '【長照交通系統】115年07月份申報資料結算提醒',
    contentSummary: '本月已達 26 日，目前仍有 1 筆混車衝突未裁決，請於月底前完成處理。',
    status: 'sent',
    triggeredByName: '系統定時排程 (Cloud Scheduler)',
    sentAt: '2026-07-26 09:00:00'
  },
  {
    id: 'nlog_11',
    topic: 'driver_leave',
    channel: 'email',
    recipientEmails: ['dispatch@ltc.example.com'],
    subject: '【調度通報】司機 郭澤威 於 2026-07-07 請事假一日',
    contentSummary: '竹北一車需安排代班司機或跨車支援調整。',
    status: 'sent',
    triggeredByName: '出勤登記作業',
    sentAt: '2026-07-06 17:30:00'
  },
  {
    id: 'nlog_12',
    topic: 'missing_report',
    channel: 'email',
    recipientEmails: ['driver.invalid@ltc.example.com'],
    subject: '【長照交通系統】未回報通知發送失敗警報',
    contentSummary: '嘗試發送未回報提醒至司機信箱 driver.invalid@ltc.example.com 失敗',
    status: 'failed',
    errorMessage: 'SMTP 550: 5.1.1 User unknown / 郵件伺服器退信',
    triggeredByName: '系統定時排程 (Cloud Scheduler)',
    sentAt: '2026-07-02 18:00:05'
  }
]

// 未回報搭乘清單：涵蓋多天逾期 (1天至12天)、各車輛與司機
export const mockMissingRides: MissingRideDTO[] = [
  {
    id: 'mis_1',
    caseId: 'case_1',
    caseName: '蔡曾切',
    serviceDate: '2026-08-26',
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
    serviceDate: '2026-08-26',
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
    serviceDate: '2026-08-25',
    legSeq: 1,
    direction: 'outbound',
    departTime: '09:10',
    vehicleId: 'veh_1',
    vehicleName: '竹北一車',
    driverName: '郭澤威',
    daysOverdue: 2
  },
  {
    id: 'mis_4',
    caseId: 'case_5',
    caseName: '李國盛',
    serviceDate: '2026-08-25',
    legSeq: 2,
    direction: 'inbound',
    departTime: '15:30',
    vehicleId: 'veh_2',
    vehicleName: '竹北二車',
    driverName: '林志豪',
    daysOverdue: 2
  },
  {
    id: 'mis_5',
    caseId: 'case_7',
    caseName: '黃天賜',
    serviceDate: '2026-08-24',
    legSeq: 1,
    direction: 'outbound',
    departTime: '08:15',
    vehicleId: 'veh_3',
    vehicleName: '竹南1車',
    driverName: '曾建宏',
    daysOverdue: 3
  },
  {
    id: 'mis_6',
    caseId: 'case_4',
    caseName: '張詹竹妹',
    serviceDate: '2026-08-24',
    legSeq: 2,
    direction: 'inbound',
    departTime: '16:15',
    vehicleId: 'veh_1',
    vehicleName: '竹北一車',
    driverName: '郭澤威',
    daysOverdue: 3
  },
  {
    id: 'mis_7',
    caseId: 'case_10',
    caseName: '林阿祥',
    serviceDate: '2026-08-22',
    legSeq: 1,
    direction: 'outbound',
    departTime: '08:50',
    vehicleId: 'veh_4',
    vehicleName: '竹南2車',
    driverName: '陳國華',
    daysOverdue: 5
  },
  {
    id: 'mis_8',
    caseId: 'case_9',
    caseName: '邱美蘭',
    serviceDate: '2026-08-22',
    legSeq: 1,
    direction: 'outbound',
    departTime: '09:30',
    vehicleId: 'veh_2',
    vehicleName: '竹北二車',
    driverName: '林志豪',
    daysOverdue: 5
  },
  {
    id: 'mis_9',
    caseId: 'case_8',
    caseName: '彭阿土',
    serviceDate: '2026-08-20',
    legSeq: 2,
    direction: 'inbound',
    departTime: '11:00',
    vehicleId: 'veh_6',
    vehicleName: '苗栗市1車',
    driverName: '吳秀珠',
    daysOverdue: 7
  },
  {
    id: 'mis_10',
    caseId: 'case_8',
    caseName: '彭阿土',
    serviceDate: '2026-08-20',
    legSeq: 4,
    direction: 'inbound',
    departTime: '16:00',
    vehicleId: 'veh_6',
    vehicleName: '苗栗市1車',
    driverName: '吳秀珠',
    daysOverdue: 7
  },
  {
    id: 'mis_11',
    caseId: 'case_1',
    caseName: '蔡曾切',
    serviceDate: '2026-08-18',
    legSeq: 1,
    direction: 'outbound',
    departTime: '09:40',
    vehicleId: 'veh_4',
    vehicleName: '竹南2車',
    driverName: '陳國華',
    daysOverdue: 9
  },
  {
    id: 'mis_12',
    caseId: 'case_2',
    caseName: '葉秀珍',
    serviceDate: '2026-08-15',
    legSeq: 2,
    direction: 'inbound',
    departTime: '11:30',
    vehicleId: 'veh_1',
    vehicleName: '竹北一車',
    driverName: '郭澤威',
    daysOverdue: 12
  }
]

// 車輛趟數表報表資料
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
          outboundCount: 12,
          inboundCount: 12,
          totalCount: 24
        },
        {
          caseId: 'case_6',
          caseCode: 'C0006',
          caseName: '陳素貞',
          outboundCount: 0,
          inboundCount: 4,
          totalCount: 4
        },
        {
          caseId: 'case_9',
          caseCode: 'C0009',
          caseName: '邱美蘭',
          outboundCount: 4,
          inboundCount: 0,
          totalCount: 4
        }
      ],
      subtotalOutbound: 16,
      subtotalInbound: 16,
      subtotalTotal: 32
    }
  ],
  grandTotalOutbound: 62,
  grandTotalInbound: 53,
  grandTotal: 115
}

// 車輛保養與維修紀錄：涵蓋大保養、定期小保養、耗材更換與費用
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
  },
  {
    id: 'maint_3',
    vehicleId: 'veh_2',
    vehicleName: '竹北二車',
    plateNo: 'ABC-1234',
    serviceDate: '2026-08-05',
    mileage: 64100,
    items: '更換前兩輪普利司通輪胎、四輪定位',
    vendor: '新竹金弘笙保養廠',
    cost: 6800,
    note: '輪胎磨耗達安全指示線，已安全更新',
    createdBy: '系統管理員',
    createdAt: '2026-08-05 18:00:00'
  },
  {
    id: 'maint_4',
    vehicleId: 'veh_6',
    vehicleName: '苗栗市1車',
    plateNo: 'MNO-7890',
    serviceDate: '2026-08-18',
    mileage: 21500,
    items: '兩萬公里定保、更換機油芯與雨刷片',
    vendor: '苗栗國榮汽車',
    cost: 2900,
    note: '全車底盤防鏽檢查正常',
    createdBy: '行政承辦',
    createdAt: '2026-08-18 15:20:00'
  }
]

// 司機月出勤展示資料：涵蓋 4 種出勤狀態 (work 出勤, leave 事假, sick 病假, off 休假) 與請假備註
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
        '2026-07-08': { date: '2026-07-08', status: 'work' },
        '2026-07-09': { date: '2026-07-09', status: 'work' },
        '2026-07-10': { date: '2026-07-10', status: 'work' }
      },
      workDays: 21,
      leaveDays: 1,
      sickDays: 0,
      offDays: 9,
      absentDays: 0
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
        '2026-07-15': { date: '2026-07-15', status: 'sick', note: '流感發燒請病假' }
      },
      workDays: 20,
      leaveDays: 0,
      sickDays: 1,
      offDays: 10,
      absentDays: 0
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
      offDays: 9,
      absentDays: 0
    },
    {
      driverId: 'drv_4',
      driverCode: 'D0004',
      driverName: '曾建宏',
      region: 'miaoli',
      days: {
        '2026-07-01': { date: '2026-07-01', status: 'work' }
      },
      workDays: 22,
      leaveDays: 0,
      sickDays: 0,
      offDays: 9,
      absentDays: 0
    }
  ]
}

// 車輛油資登錄紀錄
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
  },
  {
    id: 'fuel_4',
    vehicleId: 'veh_2',
    vehicleName: '竹北二車',
    plateNo: 'ABC-1234',
    driverId: 'drv_2',
    driverName: '林志豪',
    fuelDate: '2026-07-25',
    liters: 50.1,
    cost: 1600,
    createdBy: '林志豪',
    createdAt: '2026-07-25 18:40:00'
  }
]

// 新竹接送時刻表展示資料：涵蓋去程與回程站點路線
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
    },
    {
      direction: 'outbound',
      runNo: 1,
      caseCode: 'C0004',
      caseName: '張詹竹妹',
      note: '一般接送',
      departTime: '09:00',
      origin: '新竹縣竹北市三民路15號',
      arriveTime: '09:15',
      destination: '竹北日照中心',
      vehicleName: '竹北一車',
      siteName: '竹北日照中心'
    }
  ],
  inbound: [
    {
      direction: 'inbound',
      runNo: 1,
      caseCode: 'C0004',
      caseName: '張詹竹妹',
      note: '回程竹1車',
      departTime: '16:00',
      origin: '竹北日照中心',
      arriveTime: '16:15',
      destination: '新竹縣竹北市三民路15號',
      vehicleName: '竹北一車',
      siteName: '竹北日照中心'
    },
    {
      direction: 'inbound',
      runNo: 1,
      caseCode: 'C0002',
      caseName: '葉秀珍',
      note: '下班接回',
      departTime: '16:30',
      origin: '竹北日照中心',
      arriveTime: '16:45',
      destination: '新竹縣竹北市中正西路50號',
      vehicleName: '竹北一車',
      siteName: '竹北日照中心'
    }
  ]
}

// 儀表板趨勢與指標
export const mockDashboardMetrics: DashboardMetricsDTO = {
  currentMonth: '115-07',
  totalCasesCount: 42,
  reportedTripsCount: 380,
  unreportedVehiclesToday: 1,
  pendingConflictsCount: 1,
  pendingFormColumnsCount: 4,
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
    { vehicleName: '竹南2車', plateNo: 'GHI-9012', tripCount: 129 },
    { vehicleName: '苗栗市1車', plateNo: 'MNO-7890', tripCount: 68 }
  ],
  claimFulfillmentRate: 98.2
}

// 司機接送匯報已匯入的服務日期，依匯報表分組；用純物件（非 Map/Set）儲存，
// 讓 demoStore 的 JSON 快照／還原機制能正確持久化，重新整理頁面後不遺失
export const mockDriverReportImportedDates: Record<string, string[]> = {}

// 司機接送匯報每個 (匯報表, 月份) 各自的最後匯入時間，對應後端的 max(submitted_at)
export const mockDriverReportLastImportByMonth: Record<string, string> = {}
