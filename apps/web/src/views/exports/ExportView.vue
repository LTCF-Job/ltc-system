<template>
  <div class="export-view">
    <PageHeader title="政府申報匯出" />

    <!-- 三種選取模式共用同一頁：逐案勾選、以據點統計趟數、以區域批次產申報檔 -->
    <el-tabs v-model="activeTab" class="export-tabs">
      <el-tab-pane label="逐案勾選" name="by-case">
        <ByCasePanel :can-edit="canEdit" @exported="fetchHistory" />
      </el-tab-pane>
      <el-tab-pane label="以據點（趟數彙總）" name="by-site">
        <BySitePanel :can-edit="canEdit" />
      </el-tab-pane>
      <el-tab-pane label="以區域（批次申報檔）" name="by-region">
        <ByRegionPanel :can-edit="canEdit" @exported="fetchHistory" />
      </el-tab-pane>
    </el-tabs>

    <!-- 歷史匯出紀錄：只供查看當時匯出的個案，不重複提供檔案下載 -->
    <el-card shadow="never" class="history-card">
      <template #header>
        <span class="card-title">歷史匯出紀錄</span>
      </template>

      <el-table :data="historyJobs" border stripe v-loading="loadingHistory">
        <el-table-column prop="periodYm" label="申報年月" width="110" align="center" />
        <el-table-column label="模式" width="120" align="center">
          <template #default="{ row }">{{ EXPORT_MODE_LABELS[row.mode as ExportMode] || row.mode }}</template>
        </el-table-column>
        <el-table-column prop="totalCases" label="個案數" width="90" align="center" />
        <el-table-column prop="totalRows" label="申報行數" width="100" align="center" />
        <el-table-column prop="createdByName" label="操作人員" width="140" align="center">
          <template #default="{ row }">{{ row.createdByName || '—' }}</template>
        </el-table-column>
        <el-table-column prop="createdAt" label="產生時間" width="180" align="center">
          <template #default="{ row }">
            <span>{{ formatDateTime(row.createdAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="狀態" min-width="180" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 'succeeded' ? 'success' : 'danger'">
              {{ EXPORT_STATUS_LABELS[row.status as ExportJobStatus] || row.status }}
            </el-tag>
            <span v-if="row.errorMessage" class="history-error">{{ row.errorMessage }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right" align="center">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 'succeeded'"
              link
              type="primary"
              size="small"
              @click="openHistoryDetail(row as ExportJobDTO)"
            >
              檢視個案
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog
      v-model="historyDetailVisible"
      title="該次匯出的個案清單"
      width="min(720px, calc(100vw - 32px))"
    >
      <div v-if="historyDetail" class="history-detail-meta">
        <span>申報年月：{{ formatRocMonthLabel(rocMonthOf(historyDetail.periodYm)) }}</span>
        <span>模式：{{ EXPORT_MODE_LABELS[historyDetail.mode] }}</span>
        <span>產生時間：{{ formatDateTime(historyDetail.createdAt) }}</span>
      </div>
      <el-table
        v-loading="loadingHistoryDetail"
        :data="historyDetail?.files || []"
        border
        stripe
        max-height="360px"
      >
        <el-table-column prop="caseName" label="姓名" width="120" />
        <el-table-column prop="rowCount" label="申報行數" width="100" align="center" />
        <el-table-column prop="fileName" label="檔案名稱" min-width="180" show-overflow-tooltip />
      </el-table>

      <template #footer>
        <el-button @click="historyDetailVisible = false">關閉</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import PageHeader from '@/components/PageHeader.vue'
import ByCasePanel from './components/ByCasePanel.vue'
import BySitePanel from './components/BySitePanel.vue'
import ByRegionPanel from './components/ByRegionPanel.vue'
import { formatDateTime } from '@/utils/formatters'
import { getExportJob, listExportJobs } from '@/api/exports'
import { useAuthStore } from '@/stores/auth'
import { useRocMonth } from '@/composables/useRocMonth'
import { EXPORT_STATUS_LABELS, EXPORT_MODE_LABELS } from '@/types/domain'
import type { ExportMode, ExportJobStatus } from '@/types/domain'
import type { ExportJobDTO } from '@/types/api'

const authStore = useAuthStore()
const { formatRocMonthLabel } = useRocMonth()

const canEdit = computed(() => authStore.hasPermission('exports', 'edit'))

const activeTab = ref<'by-case' | 'by-site' | 'by-region'>('by-case')
const historyJobs = ref<ExportJobDTO[]>([])
const loadingHistory = ref(false)
const historyDetailVisible = ref(false)
const historyDetail = ref<ExportJobDTO | null>(null)
const loadingHistoryDetail = ref(false)

// 民國 5 碼（11507）轉成顯示用的 115-07
function rocMonthOf(periodYm: string): string {
  if (!periodYm || periodYm.length !== 5) return periodYm
  return `${periodYm.slice(0, 3)}-${periodYm.slice(3)}`
}

async function openHistoryDetail(row: ExportJobDTO) {
  historyDetailVisible.value = true
  historyDetail.value = null
  loadingHistoryDetail.value = true
  try {
    historyDetail.value = await getExportJob(row.id)
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  } finally {
    loadingHistoryDetail.value = false
  }
}

async function fetchHistory() {
  loadingHistory.value = true
  try {
    const res = await listExportJobs({ pageSize: 10 })
    historyJobs.value = res.data
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  } finally {
    loadingHistory.value = false
  }
}

onMounted(() => {
  fetchHistory()
})
</script>

<style scoped>
.export-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.export-tabs {
  :deep(.el-tabs__content) {
    overflow: visible;
  }
}

.history-card {
  border-radius: 8px;
}

.card-title {
  font-size: 16px;
  font-weight: bold;
  color: var(--app-primary);
}

.history-error {
  margin-left: 8px;
  color: var(--app-text-secondary);
  font-size: 13px;
}

.history-detail-meta {
  display: flex;
  gap: 20px;
  flex-wrap: wrap;
  margin-bottom: 12px;
  color: var(--app-text-secondary);
  font-size: 13px;
}
</style>
