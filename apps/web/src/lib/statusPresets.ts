// 集中管理狀態顏色，取代各頁各寫一份三元式或硬寫色碼
// 新增狀態值前先確認語意屬於哪個 preset，preset 內找不到情境才新增一組

export type StatusVariant = 'success' | 'warning' | 'danger' | 'info' | 'neutral'

export interface StatusPresetEntry {
  label: string
  variant: StatusVariant
}

export type StatusPresetMap = Record<string, StatusPresetEntry>

// 個案狀態：在案／暫停／停案。停案是終止狀態不是錯誤，一律用 neutral（灰）
export const CASE_STATUS_PRESET: StatusPresetMap = {
  active: { label: '在案', variant: 'success' },
  suspended: { label: '暫停', variant: 'warning' },
  closed: { label: '停案', variant: 'neutral' }
}

// 啟用／停用（地區、車輛、司機、照護人員共用）
export const ACTIVE_STATE_PRESET: StatusPresetMap = {
  active: { label: '啟用中', variant: 'success' },
  inactive: { label: '已停用', variant: 'neutral' }
}

// 在職／離職（司機專用文案，語意同上）
export const EMPLOYMENT_STATE_PRESET: StatusPresetMap = {
  active: { label: '在職中', variant: 'success' },
  inactive: { label: '已離職', variant: 'neutral' }
}

// 司機接送匯報表批次匯入流程狀態
export const BATCH_IMPORT_STATUS_PRESET: StatusPresetMap = {
  idle: { label: '待處理', variant: 'info' },
  analyzing: { label: '解析中', variant: 'info' },
  ready: { label: '可匯入', variant: 'success' },
  needsMapping: { label: '需處理', variant: 'warning' },
  importing: { label: '匯入中', variant: 'info' },
  imported: { label: '已匯入', variant: 'success' },
  failed: { label: '失敗', variant: 'danger' }
}

// 欄位對應狀態（FieldMappingView / DriverReportColumnMappingTable 共用）
export const FIELD_MAPPING_STATUS_PRESET: StatusPresetMap = {
  mapped: { label: '已對應', variant: 'success' },
  pending: { label: '待對應', variant: 'warning' },
  ignored: { label: '已略過', variant: 'info' },
  unavailable: { label: '不適用', variant: 'info' }
}

// 完成／失敗（匯出紀錄、通知寄送紀錄共用）
export const COMPLETION_STATUS_PRESET: StatusPresetMap = {
  success: { label: '已完成', variant: 'success' },
  failed: { label: '失敗', variant: 'danger' },
  pending: { label: '處理中', variant: 'info' }
}

// 系統角色
export const ROLE_PRESET: StatusPresetMap = {
  admin: { label: '管理員', variant: 'danger' },
  dispatcher: { label: '排班人員', variant: 'info' },
  staff: { label: '承辦人員', variant: 'success' },
  driver: { label: '司機', variant: 'warning' },
  viewer: { label: '檢視者', variant: 'neutral' }
}

export const STATUS_PRESETS = {
  caseStatus: CASE_STATUS_PRESET,
  activeState: ACTIVE_STATE_PRESET,
  employmentState: EMPLOYMENT_STATE_PRESET,
  batchImportStatus: BATCH_IMPORT_STATUS_PRESET,
  fieldMappingStatus: FIELD_MAPPING_STATUS_PRESET,
  completionStatus: COMPLETION_STATUS_PRESET,
  role: ROLE_PRESET
} as const

export type StatusPresetName = keyof typeof STATUS_PRESETS

export function resolveStatusEntry(presetName: StatusPresetName, status: string): StatusPresetEntry {
  return STATUS_PRESETS[presetName][status] || { label: status, variant: 'neutral' }
}
