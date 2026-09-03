import type { CreateVehicleRequest } from '@/types/api'

// 四個日期欄位的標籤、驗證與民國換算提示完全一致，集中宣告避免逐欄重複
export const VEHICLE_DATE_FIELDS = [
  { prop: 'compulsoryInsuranceExpiry', label: '強制責任險' },
  { prop: 'passengerInsuranceExpiry', label: '乘客責任險' },
  { prop: 'thirdPartyInsuranceExpiry', label: '第三人責任險' },
  { prop: 'lastInspectionDate', label: '前次檢驗日期' }
] as const

export type VehicleDateField = (typeof VEHICLE_DATE_FIELDS)[number]['prop']

/** 產生一份空白的車輛表單值，供新增與重設編輯狀態使用。 */
export function emptyVehicleForm(): CreateVehicleRequest {
  return {
    plateNo: '',
    displayName: '',
    siteId: '',
    brand: '',
    model: '',
    manufactureYm: '',
    compulsoryInsuranceExpiry: '',
    passengerInsuranceExpiry: '',
    thirdPartyInsuranceExpiry: '',
    lastInspectionDate: '',
    wheelchairAccessible: true,
    status: 'active'
  }
}

/** 車輛表單的必填規則：除車號與所屬單位外皆非必填，與後端 VehicleWriteFields 的 binding:"required" 一致。 */
export const vehicleFormRules = {
  plateNo: [
    { required: true, message: '請輸入車號', trigger: 'blur' },
    { pattern: /^[A-Z0-9]{2,4}-[A-Z0-9]{2,4}$/, message: '車號格式錯誤 (例如 BZG-7915)', trigger: 'blur' }
  ],
  siteId: [{ required: true, message: '請選擇所屬單位', trigger: 'change' }]
}

/** 將車輛表單資料之選填空字串過濾為 null，確保送往後端之 JSON 符合 NULL 契約而非空白字串。 */
export function sanitizeVehiclePayload<T extends CreateVehicleRequest>(form: T): T {
  return {
    ...form,
    plateNo: form.plateNo?.trim() || '',
    displayName: form.displayName?.trim() || null,
    siteId: form.siteId || '',
    brand: form.brand?.trim() || null,
    model: form.model?.trim() || null,
    manufactureYm: form.manufactureYm?.trim() || null,
    compulsoryInsuranceExpiry: form.compulsoryInsuranceExpiry?.trim() || null,
    passengerInsuranceExpiry: form.passengerInsuranceExpiry?.trim() || null,
    thirdPartyInsuranceExpiry: form.thirdPartyInsuranceExpiry?.trim() || null,
    lastInspectionDate: form.lastInspectionDate?.trim() || null
  }
}
