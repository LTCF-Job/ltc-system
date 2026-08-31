import type { MappingStatus } from '@/types/domain'
import type { DriverReportColumnDecision, DriverReportPreviewColumn } from '@/types/api'

// 匯入預覽階段對單一欄位的對應決定；以欄位表頭為鍵，欄號會隨個案增減位移
export interface ColumnDecisionState {
  mappingStatus: MappingStatus
  caseId?: string
  legSeq?: number
}

export type ColumnDecisionMap = Record<string, ColumnDecisionState>

// createColumnDecisions 由預覽欄位建立初始決定：已對應者沿用既有設定，未對應者帶入推薦值。
export function createColumnDecisions(columns: DriverReportPreviewColumn[]): ColumnDecisionMap {
  return Object.fromEntries(
    columns.map((c) => [
      c.columnHeader,
      {
        mappingStatus: c.mappingStatus,
        caseId: c.caseId || c.suggestedCaseId || undefined,
        legSeq: c.legSeq || c.suggestedLegSeq || undefined
      }
    ])
  )
}

// toColumnDecisionPayload 組出送交後端的對應決定；只有已對應的欄位帶個案與趟次。
export function toColumnDecisionPayload(decisions: ColumnDecisionMap): DriverReportColumnDecision[] {
  return Object.entries(decisions).map(([columnHeader, d]) => ({
    columnHeader,
    mappingStatus: d.mappingStatus,
    caseId: d.mappingStatus === 'mapped' ? d.caseId : null,
    legSeq: d.mappingStatus === 'mapped' ? d.legSeq : null
  }))
}

export function countByStatus(decisions: ColumnDecisionMap, status: MappingStatus): number {
  return Object.values(decisions).filter((d) => d.mappingStatus === status).length
}
