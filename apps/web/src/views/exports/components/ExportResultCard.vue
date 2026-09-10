<template>
  <el-card shadow="never" class="job-card">
    <template #header>
      <span class="card-title">{{ title }}</span>
    </template>

    <div class="job-info">
      <span>申報年月：{{ formatRocMonthLabel(rocMonthOf(job.periodYm)) }}</span>
      <span>模式：{{ EXPORT_MODE_LABELS[job.mode] }}</span>
      <span>個案數：{{ job.totalCases ?? 0 }}</span>
      <span>申報行數：{{ job.totalRows ?? 0 }}</span>
    </div>

    <el-alert
      v-if="job.dataGaps?.length"
      type="warning"
      show-icon
      :closable="false"
      title="部分欄位資料不完整，該欄位已留白匯出"
      class="skip-alert"
    >
      <ul class="skip-list">
        <li v-for="(gap, index) in job.dataGaps" :key="index">
          {{ gap.caseName }}：{{ dataGapLabel(gap.reason) }}（{{ gap.count }} 筆）
        </li>
      </ul>
    </el-alert>

    <!-- 防禦性空狀態：後端查無資料時會回 NO_EXPORT_DATA，正常不會走到這裡 -->
    <el-empty
      v-if="!job.files?.length"
      description="這次匯出沒有產生任何檔案"
      :image-size="80"
      class="empty-result"
    >
      <p class="empty-hint">
        常見原因：該月份沒有已上車的搭乘紀錄、個案仍在待維護狀態，或混車衝突尚未裁決。
      </p>
    </el-empty>

    <!-- 直接下載：一個個案一列，由使用者自行點選，避免瀏覽器擋下連續下載 -->
    <el-table
      v-else-if="job.mode === 'direct'"
      :data="job.files || []"
      border
      stripe
      class="file-table"
    >
      <el-table-column prop="caseName" label="姓名" width="120" />
      <el-table-column prop="rowCount" label="申報行數" width="100" align="center" />
      <el-table-column prop="fileName" label="檔案名稱" min-width="200" show-overflow-tooltip />
      <el-table-column label="操作" width="120" fixed="right" align="center">
        <template #default="{ row }">
          <el-button
            link
            type="primary"
            size="small"
            :loading="downloadingCaseId === row.caseId"
            @click="emit('download-case', row as ExportJobFileDTO)"
          >
            下載
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div v-else class="zip-download">
      <el-button type="success" :loading="downloadingZip" @click="emit('download-zip')">
        <el-icon><Download /></el-icon>
        下載壓縮檔 ({{ job.zipFileName }})
      </el-button>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { Download } from '@element-plus/icons-vue'
import { useRocMonth } from '@/composables/useRocMonth'
import { EXPORT_MODE_LABELS, EXPORT_DATA_GAP_LABELS } from '@/types/domain'
import type { ExportJobDTO, ExportJobFileDTO } from '@/types/api'

withDefaults(
  defineProps<{
    job: ExportJobDTO
    downloadingCaseId?: string
    downloadingZip?: boolean
    title?: string
  }>(),
  { downloadingCaseId: '', downloadingZip: false, title: '本次匯出結果' }
)

const emit = defineEmits<{
  (e: 'download-case', file: ExportJobFileDTO): void
  (e: 'download-zip'): void
}>()

const { formatRocMonthLabel } = useRocMonth()

// 民國 5 碼（11507）轉成顯示用的 115-07
function rocMonthOf(periodYm: string): string {
  if (!periodYm || periodYm.length !== 5) return periodYm
  return `${periodYm.slice(0, 3)}-${periodYm.slice(3)}`
}

function dataGapLabel(reason: string): string {
  return EXPORT_DATA_GAP_LABELS[reason] || reason
}
</script>

<style scoped>
.job-card {
  border-radius: 8px;
}

.card-title {
  font-size: 16px;
  font-weight: bold;
  color: var(--app-primary);
}

.job-info {
  display: flex;
  gap: 20px;
  align-items: center;
  flex-wrap: wrap;
  font-weight: 500;
}

.skip-alert {
  margin-top: 12px;

  .skip-list {
    margin: 4px 0 0;
    padding-left: 18px;
  }
}

.empty-result {
  padding: 12px 0 0;

  .empty-hint {
    margin: 0;
    color: var(--app-text-secondary);
    font-size: 13px;
  }
}

.file-table {
  margin-top: 12px;
}

.zip-download {
  margin-top: 12px;
  display: flex;
  justify-content: flex-start;
}

@media (max-width: 640px) {
  .job-info {
    align-items: flex-start;
    flex-wrap: wrap;
  }
}
</style>
