import type {
  CaseDTO,
  RegionDTO,
  SiteDTO,
  VehicleDTO,
  DriverDTO,
  FormDTO,
  FormColumnDTO,
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
  DashboardMetricsDTO,
  UserDTO,
  RoleDTO
} from '@/types/api'
import { DEFAULT_ROLE_PERMISSIONS } from '@/types/domain'

// 區域主檔展示資料：涵蓋全台灣 22 縣市
export const mockRegions: RegionDTO[] = [
  { id: 'reg_1', code: 'hsinchu', name: '新竹縣', description: '新竹縣營運區域', status: 'active', sortOrder: 1, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_2', code: 'hsinchu_city', name: '新竹市', description: '新竹市營運區域', status: 'active', sortOrder: 2, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_3', code: 'miaoli', name: '苗栗縣', description: '苗栗縣營運區域', status: 'active', sortOrder: 3, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_4', code: 'taipei', name: '臺北市', description: '臺北市營運區域', status: 'active', sortOrder: 4, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_5', code: 'new_taipei', name: '新北市', description: '新北市營運區域', status: 'active', sortOrder: 5, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_6', code: 'keelung', name: '基隆市', description: '基隆市營運區域', status: 'active', sortOrder: 6, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_7', code: 'taoyuan', name: '桃園市', description: '桃園市營運區域', status: 'active', sortOrder: 7, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_8', code: 'taichung', name: '臺中市', description: '臺中市營運區域', status: 'active', sortOrder: 8, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_9', code: 'changhua', name: '彰化縣', description: '彰化縣營運區域', status: 'active', sortOrder: 9, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_10', code: 'nantou', name: '南投縣', description: '南投縣營運區域', status: 'active', sortOrder: 10, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_11', code: 'yunlin', name: '雲林縣', description: '雲林縣營運區域', status: 'active', sortOrder: 11, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_12', code: 'chiayi_city', name: '嘉義市', description: '嘉義市營運區域', status: 'active', sortOrder: 12, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_13', code: 'chiayi', name: '嘉義縣', description: '嘉義縣營運區域', status: 'active', sortOrder: 13, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_14', code: 'tainan', name: '臺南市', description: '臺南市營運區域', status: 'active', sortOrder: 14, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_15', code: 'kaohsiung', name: '高雄市', description: '高雄市營運區域', status: 'active', sortOrder: 15, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_16', code: 'pingtung', name: '屏東縣', description: '屏東縣營運區域', status: 'active', sortOrder: 16, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_17', code: 'yilan', name: '宜蘭縣', description: '宜蘭縣營運區域', status: 'active', sortOrder: 17, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_18', code: 'hualien', name: '花蓮縣', description: '花蓮縣營運區域', status: 'active', sortOrder: 18, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_19', code: 'taitung', name: '臺東縣', description: '臺東縣營運區域', status: 'active', sortOrder: 19, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_20', code: 'penghu', name: '澎湖縣', description: '澎湖縣營運區域', status: 'active', sortOrder: 20, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_21', code: 'kinmen', name: '金門縣', description: '金門縣營運區域', status: 'active', sortOrder: 21, createdAt: '2026-01-01', updatedAt: '2026-01-01' },
  { id: 'reg_22', code: 'lienchiang', name: '連江縣', description: '連江縣營運區域', status: 'active', sortOrder: 22, createdAt: '2026-01-01', updatedAt: '2026-01-01' }
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

// 司機主檔展示資料：涵蓋主責車輛指派、支援調度指派、在職與留職停薪狀態
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
      { id: 'asgn_1', driverId: 'drv_1', vehicleId: 'veh_1', vehicleName: '竹北一車', vehiclePlateNo: 'BZG-7915', plateNo: 'BZG-7915', startDate: '2026-01-01', isPrimary: true }
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
      { id: 'asgn_2', driverId: 'drv_2', vehicleId: 'veh_2', vehicleName: '竹北二車', vehiclePlateNo: 'ABC-1234', plateNo: 'ABC-1234', startDate: '2026-01-01', isPrimary: true },
      { id: 'asgn_2_sub', driverId: 'drv_2', vehicleId: 'veh_1', vehicleName: '竹北一車', vehiclePlateNo: 'BZG-7915', plateNo: 'BZG-7915', startDate: '2026-07-01', isPrimary: false }
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
      { id: 'asgn_3', driverId: 'drv_3', vehicleId: 'veh_4', vehicleName: '竹南2車', vehiclePlateNo: 'GHI-9012', plateNo: 'GHI-9012', startDate: '2026-01-01', isPrimary: true }
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
      { id: 'asgn_4', driverId: 'drv_4', vehicleId: 'veh_3', vehicleName: '竹南1車', vehiclePlateNo: 'DEF-5678', plateNo: 'DEF-5678', startDate: '2026-02-01', isPrimary: true }
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
      { id: 'asgn_5', driverId: 'drv_5', vehicleId: 'veh_6', vehicleName: '苗栗市1車', vehiclePlateNo: 'MNO-7890', plateNo: 'MNO-7890', startDate: '2026-03-01', isPrimary: true }
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
    phone: '0912345678',
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
  // 2. 新竹 / 補助 / 社區長照機構 / 4 趟 / 在案
  {
    id: 'case_2',
    code: 'C0002',
    name: '葉秀珍',
    nationalId: 'J220123344',
    phone: '0922333444',
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
    phone: '0933555666',
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
    phone: '0955777888',
    homeAddress: '新竹縣竹北市三民路15號',
    region: 'hsinchu',
    serviceCategory: 1,
    serviceUsageType: 2,
    claimStartDate: '2026-07-01',
    status: 'suspended',
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
    phone: '0966999000',
    homeAddress: '新竹縣竹北市文興路一段200號',
    region: 'hsinchu',
    serviceCategory: 2,
    serviceUsageType: 1,
    claimStartDate: '2026-07-01',
    status: 'active',
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
    phone: '0977111222',
    homeAddress: '新竹縣竹北市縣政九路80號',
    region: 'hsinchu',
    serviceCategory: 1,
    serviceUsageType: 3,
    claimStartDate: '2026-06-01',
    claimEndDate: '2026-07-31',
    status: 'closed',
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
    phone: '0988333444',
    homeAddress: '苗栗縣竹南鎮延平路66號',
    region: 'miaoli',
    serviceCategory: 1,
    serviceUsageType: 4,
    claimStartDate: '2026-07-01',
    status: 'active',
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
    phone: '0911555666',
    homeAddress: '苗栗縣苗栗市中正路500號',
    region: 'miaoli',
    serviceCategory: 2,
    serviceUsageType: 4,
    claimStartDate: '2026-07-01',
    status: 'active',
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
    phone: '0928777888',
    homeAddress: '新竹縣湖口鄉達生路33號',
    region: 'hsinchu',
    serviceCategory: 1,
    serviceUsageType: 3,
    claimStartDate: '2026-07-01',
    status: 'active',
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
    phone: '0937999000',
    homeAddress: '苗栗縣竹南鎮光復路88號',
    region: 'miaoli',
    serviceCategory: 1,
    serviceUsageType: 2,
    claimStartDate: '2026-07-01',
    status: 'active',
    createdAt: '2026-07-01',
    updatedAt: '2026-07-01',
    activeSchedule: {
      id: 'sch_10',
      caseId: 'case_10',
      siteId: 'site_2',
      siteName: '竹南日照據點',
      effectiveFrom: '2026-07-01',
      weekdays: [1, 3, 5],
      tripPattern: 2,
      unitPrice: 115,
      distanceKm: 6.0,
      serviceDurationMin: 12,
      serviceCode: 'BD03',
      legs: [
        { id: 'leg_10_1', legSeq: 1, direction: 'outbound', departTime: '08:50', arriveTime: '09:05', runNo: 1, vehicleId: 'veh_4', vehicleName: '竹南2車' },
        { id: 'leg_10_2', legSeq: 2, direction: 'inbound', departTime: '15:50', arriveTime: '16:05', runNo: 1, vehicleId: 'veh_4', vehicleName: '竹南2車' }
      ]
    }
  }
]

// Google 表單展示資料：涵蓋正常同步與需要對帳警示狀態、多分頁與已同步月份
export const mockForms: FormDTO[] = [
  {
    id: 'form_1',
    formId: 'zhubei_car_1',
    title: '竹北一車每日接送回報表',
    sheetUrl: 'https://docs.google.com/spreadsheets/d/1BxiMVs0XRmG1uY1b/edit',
    vehicleId: 'veh_1',
    vehicleName: '竹北一車',
    region: 'hsinchu',
    sheetTabs: ['8月回報', '7月回報', '工作表1'],
    activeTab: '8月回報',
    syncedMonths: ['2026-07', '2026-08'],
    lastSyncedAt: '2026-08-25 14:00',
    totalColumns: 56,
    pendingColumns: 3,
    hasSyncAlert: false
  },
  {
    id: 'form_2',
    formId: 'zhunan_car_2',
    title: '竹南2車每日接送回報表',
    sheetUrl: 'https://docs.google.com/spreadsheets/d/2BxiMVs0XRmG2uY2c/edit',
    vehicleId: 'veh_2',
    vehicleName: '竹南2車',
    region: 'miaoli',
    sheetTabs: ['8月回報', '7月回報'],
    activeTab: '8月回報',
    syncedMonths: ['2026-07'],
    lastSyncedAt: '2026-08-25 14:05',
    totalColumns: 62,
    pendingColumns: 0,
    hasSyncAlert: false
  },
  {
    id: 'form_3',
    formId: 'zhubei_car_2',
    title: '竹北二車每日接送回報表',
    sheetUrl: 'https://docs.google.com/spreadsheets/d/3BxiMVs0XRmG3uY3d/edit',
    vehicleId: 'veh_3',
    vehicleName: '竹北二車',
    region: 'hsinchu',
    sheetTabs: ['8月回報', '去程回報', '回程回報'],
    activeTab: '8月回報',
    syncedMonths: ['2026-07'],
    lastSyncedAt: '2026-08-22 09:00',
    totalColumns: 48,
    pendingColumns: 5,
    hasSyncAlert: true
  },
  {
    id: 'form_4',
    formId: 'miaoli_car_1',
    title: '苗栗市1車每日接送回報表',
    sheetUrl: 'https://docs.google.com/spreadsheets/d/4BxiMVs0XRmG4uY4e/edit',
    vehicleId: 'veh_4',
    vehicleName: '苗栗市1車',
    region: 'miaoli',
    sheetTabs: ['8月回報', '7月回報'],
    activeTab: '8月回報',
    syncedMonths: ['2026-07', '2026-08'],
    lastSyncedAt: '2026-08-25 15:30',
    totalColumns: 36,
    pendingColumns: 1,
    hasSyncAlert: false
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
    id: 'usr_dispatcher_1',
    email: 'dispatcher@ltc.example.com',
    displayName: '調度員 (李調度)',
    role: 'dispatcher',
    phone: '0922-333-444',
    status: 'active',
    customPermissions: null,
    lastLoginAt: '2026-08-26 08:45:00',
    createdAt: '2026-01-10 09:00:00'
  },
  {
    id: 'usr_driver_1',
    email: 'driver@ltc.example.com',
    displayName: '司機 (張司機)',
    role: 'driver',
    phone: '0933-555-666',
    status: 'active',
    customPermissions: {
      attendance_fuel: { view: true, edit: true },
      vehicles_maintenance: { view: true, edit: true }
    },
    lastLoginAt: '2026-08-25 17:30:00',
    createdAt: '2026-02-01 08:30:00'
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
    id: 'role_dispatcher',
    key: 'dispatcher',
    name: '調度員',
    description: '負責日常派車、個案管理、搭乘月曆排程、異常處理、表單同步與申報資料匯出。',
    tagType: 'primary',
    isSystem: true,
    permissions: JSON.parse(JSON.stringify(DEFAULT_ROLE_PERMISSIONS.dispatcher)),
    createdAt: '2026-01-01 00:00:00',
    updatedAt: '2026-01-01 00:00:00'
  },
  {
    id: 'role_driver',
    key: 'driver',
    name: '司機',
    description: '負責每日出勤登錄、車輛維修紀錄填寫與個人接送趟次狀況檢視。',
    tagType: 'success',
    isSystem: true,
    permissions: JSON.parse(JSON.stringify(DEFAULT_ROLE_PERMISSIONS.driver)),
    createdAt: '2026-01-01 00:00:00',
    updatedAt: '2026-01-01 00:00:00'
  },
  {
    id: 'role_staff',
    key: 'staff',
    name: '行政人員',
    description: '負責行政文書、主檔維護、搭乘資料校對及一般報表產出。',
    tagType: 'primary',
    isSystem: true,
    permissions: JSON.parse(JSON.stringify(DEFAULT_ROLE_PERMISSIONS.staff)),
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
      forms_sync: { view: true, edit: false },
      forms_mappings: { view: true, edit: false },
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

// 智慧欄位對應展示資料：涵蓋 4 種欄位種類 (meta, ride, issue, unknown) 與 3 種對應狀態 (pending, mapped, ignored)
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
    suggestionScore: 0.85,
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
    columnName: '問題回報與備註說明',
    columnSeq: 40,
    kind: 'issue',
    mappingStatus: 'ignored',
    updatedAt: '2026-08-25'
  },
  {
    id: 'col_5',
    formId: 'form_1',
    columnName: '時間戳記 (Timestamp)',
    columnSeq: 1,
    kind: 'meta',
    mappingStatus: 'mapped',
    updatedAt: '2026-08-01'
  },
  {
    id: 'col_6',
    formId: 'form_3',
    columnName: '新進未知個案-測試欄位',
    columnSeq: 12,
    kind: 'unknown',
    mappingStatus: 'pending',
    suggestionScore: 0.3,
    updatedAt: '2026-08-25'
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

// 匯出工作展示資料：涵蓋所有 4 種報表類型 (gov_claim, trip_summary, hsinchu_schedule, maintenance_blank) 與所有 4 種狀態
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
  },
  {
    id: 'job_202607_02',
    jobType: 'trip_summary',
    periodYm: '11507',
    region: 'hsinchu',
    mode: 'single_multi_case',
    status: 'succeeded',
    totalCases: 38,
    totalRows: 139,
    fileName: 'trip-summary-11507-hsinchu.xlsx',
    fileSize: 32100,
    downloadUrl: 'https://placeholder-download.supabase.co/trip-summary-11507-hsinchu.xlsx',
    createdAt: '2026-08-01 10:05:00',
    completedAt: '2026-08-01 10:05:08'
  },
  {
    id: 'job_202607_03',
    jobType: 'hsinchu_schedule',
    periodYm: '11508',
    region: 'hsinchu',
    mode: 'single_multi_case',
    status: 'running',
    totalCases: 15,
    fileName: 'hsinchu-schedule-11508.xlsx',
    createdAt: '2026-08-25 16:00:00'
  },
  {
    id: 'job_202607_04',
    jobType: 'maintenance_blank',
    periodYm: '11508',
    region: 'hsinchu',
    mode: 'single_multi_case',
    status: 'failed',
    errorMessage: '模板樣式套用異常：工作表名稱衝突',
    fileName: 'maintenance-blank-11508.xlsx',
    createdAt: '2026-08-25 15:45:00',
    completedAt: '2026-08-25 15:45:04'
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

// 系統操作紀錄展示資料：涵蓋登入 (login)、主檔 CUD、更正 (correct)、衝突裁決 (resolve_conflict)、匯出 (export) 與設定變更
export const mockAuditLogs: AuditLogDTO[] = [
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

// 通知發送歷史日誌：涵蓋全部 4 種主題、成功與失敗狀態
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
    contentSummary: '本月已達26日，目前仍有 1 筆混車衝突未裁決，請於月底前完成處理。',
    status: 'sent',
    triggeredByName: '系統定時排程 (Cloud Scheduler)',
    sentAt: '2026-07-26 09:00:00'
  },
  {
    id: 'nlog_3',
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
    id: 'nlog_4',
    topic: 'export_failed',
    channel: 'email',
    recipientEmails: ['tech@ltc.example.com'],
    subject: '【警報】空白保養表產生異常',
    contentSummary: '批次產生任務 job_202607_04 模板套用失敗',
    status: 'failed',
    errorMessage: '模板樣式套用異常：工作表名稱衝突',
    triggeredByName: '匯出非同步任務',
    sentAt: '2026-08-25 15:45:05'
  }
]

// 未回報搭乘清單：涵蓋多天逾期 (1天、3天、5天、10天) 與多台車輛
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
    serviceDate: '2026-08-22',
    legSeq: 1,
    direction: 'outbound',
    departTime: '09:10',
    vehicleId: 'veh_1',
    vehicleName: '竹北一車',
    driverName: '郭澤威',
    daysOverdue: 3
  },
  {
    id: 'mis_4',
    caseId: 'case_7',
    caseName: '黃天賜',
    serviceDate: '2026-08-20',
    legSeq: 1,
    direction: 'outbound',
    departTime: '08:15',
    vehicleId: 'veh_3',
    vehicleName: '竹南1車',
    driverName: '曾建宏',
    daysOverdue: 5
  },
  {
    id: 'mis_5',
    caseId: 'case_8',
    caseName: '彭阿土',
    serviceDate: '2026-08-15',
    legSeq: 2,
    direction: 'inbound',
    departTime: '11:00',
    vehicleId: 'veh_6',
    vehicleName: '苗栗市1車',
    driverName: '吳秀珠',
    daysOverdue: 10
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
        '2026-07-15': { date: '2026-07-15', status: 'sick', note: '流感發燒請病假' }
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
      offDays: 9
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
