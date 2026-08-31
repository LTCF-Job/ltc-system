import type { MappingStatus } from '@/types/domain'
import type { DriverReportColumnDecision } from '@/types/api'

// 匯入預覽階段對單一欄位的對應決定；以欄位表頭為鍵，欄號會隨個案增減位移
export interface ColumnDecisionState {
  mappingStatus: MappingStatus
  caseId?: string
  legSeq?: number
}

export type ColumnDecisionMap = Record<string, ColumnDecisionState>

// toColumnDecisionPayload 組出送交後端的對應決定；只有已對應的欄位帶個案與趟次。
export function toColumnDecisionPayload(decisions: ColumnDecisionMap): DriverReportColumnDecision[] {
  return Object.entries(decisions).map(([columnHeader, d]) => ({
    columnHeader,
    mappingStatus: d.mappingStatus,
    caseId: d.mappingStatus === 'mapped' ? d.caseId : null,
    legSeq: d.mappingStatus === 'mapped' ? d.legSeq : null
  }))
}
