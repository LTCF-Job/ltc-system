<template>
  <el-table :data="columns" :max-height="maxHeight" border size="small">
    <el-table-column prop="columnIndex" label="#" width="50" align="center" />
    <el-table-column label="匯報表欄位" min-width="220">
      <template #default="{ row }">
        <div class="column-cell">
          <span class="column-header">{{ row.columnHeader }}</span>
          <span class="column-meta">
            有坐 {{ row.boardedCount }} 天 / 沒坐 {{ row.absentCount }} 天
          </span>
        </div>
      </template>
    </el-table-column>
    <el-table-column label="對應個案" min-width="200">
      <template #default="{ row }">
        <el-select
          v-model="decisions[row.columnHeader].caseId"
          :aria-label="`${row.columnHeader} 對應個案`"
          placeholder="選擇個案"
          filterable
          clearable
          style="width: 100%"
          @change="onCaseChange(row.columnHeader)"
        >
          <el-option
            v-for="c in cases"
            :key="c.id"
            :label="`${c.name} (${c.code})`"
            :value="c.id"
          />
        </el-select>
      </template>
    </el-table-column>
    <el-table-column label="趟次" width="160">
      <template #default="{ row }">
        <el-select
          v-model="decisions[row.columnHeader].legSeq"
          :aria-label="`${row.columnHeader} 趟次`"
          placeholder="選擇趟次"
          style="width: 100%"
        >
          <el-option
            v-for="leg in LEG_SEQ_OPTIONS"
            :key="leg.value"
            :value="leg.value"
            :label="leg.label"
          />
        </el-select>
      </template>
    </el-table-column>
    <el-table-column label="狀態" width="130" align="center">
      <template #default="{ row }">
        <StatusTag :status="decisions[row.columnHeader].mappingStatus" preset="fieldMappingStatus" />
      </template>
    </el-table-column>
    <el-table-column label="操作" width="100" align="center">
      <template #default="{ row }">
        <el-button
          link
          type="primary"
          size="small"
          @click="toggleIgnore(row.columnHeader)"
        >
          {{ decisions[row.columnHeader].mappingStatus === 'ignored' ? '取消略過' : '略過此欄' }}
        </el-button>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
import { LEG_SEQ_OPTIONS } from './legOptions'
import type { ColumnDecisionMap } from './columnDecisions'
import StatusTag from '@/components/StatusTag.vue'
import type { CaseDTO, DriverReportPreviewColumn } from '@/types/api'

const props = defineProps<{
  columns: DriverReportPreviewColumn[]
  cases: CaseDTO[]
  maxHeight?: string | number
}>()

// 決定物件由呼叫端持有：單車對話框與批次頁各自管理自己的送出時機
const decisions = defineModel<ColumnDecisionMap>('decisions', { required: true })

// 選好個案即視為已對應：趟次留白時沿用推薦值，避免使用者被迫多點一次
function onCaseChange(columnHeader: string) {
  const decision = decisions.value[columnHeader]
  if (!decision) return
  if (decision.caseId) {
    decision.mappingStatus = 'mapped'
    if (!decision.legSeq) {
      const column = props.columns.find((c) => c.columnHeader === columnHeader)
      decision.legSeq = column?.suggestedLegSeq || 1
    }
  } else {
    decision.mappingStatus = 'pending'
  }
}

function toggleIgnore(columnHeader: string) {
  const decision = decisions.value[columnHeader]
  if (!decision) return
  if (decision.mappingStatus === 'ignored') {
    decision.mappingStatus = decision.caseId ? 'mapped' : 'pending'
  } else {
    decision.mappingStatus = 'ignored'
  }
}
</script>

<style scoped>
.column-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.column-header {
  font-weight: 500;
}

.column-meta {
  font-size: var(--app-font-xs);
  color: var(--app-text-muted);
}
</style>
