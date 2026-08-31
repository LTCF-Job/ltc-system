<template>
  <div class="export-view">
    <PageHeader title="政府申報匯出" />
    <!-- 匯出條件設定卡片 -->
    <el-card shadow="never" class="export-settings-card">
      <template #header>
        <span class="card-title">政府申報表匯出設定</span>
      </template>

      <el-form
        ref="formRef"
        :model="form"
        label-width="140px"
        :disabled="!authStore.can('staff')"
      >
        <el-row :gutter="16">
          <el-col :xs="24" :sm="12">
            <el-form-item label="申報年月 (民國)">
              <div class="roc-month-picker">
                <el-date-picker
                  v-model="selectedDate"
                  type="month"
                  format="YYYY-MM"
                  value-format="YYYY-MM"
                  placeholder="選擇月份"
                  style="width: 160px"
                  :clearable="false"
                />
                <span class="roc-label">{{ formatRocMonthLabel(currentRocMonth) }}</span>
              </div>
            </el-form-item>
          </el-col>

          <el-col :xs="24" :sm="12">
            <el-form-item label="申報地區">
              <el-select
                v-model="form.region"
                placeholder="全部地區"
                clearable
                filterable
                style="width: 180px"
              >
                <el-option label="全部地區" value="" />
                <el-option
                  v-for="(label, key) in REGION_LABELS"
                  :key="key"
                  :label="label"
                  :value="key"
                />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16" align="middle" class="export-mode-row">
          <el-col :xs="24" :sm="12">
            <el-form-item label="匯出檔案模式" class="mode-form-item">
              <el-radio-group v-model="form.mode">
                <el-radio value="single_multi_case">單檔多案 (.xlsx)</el-radio>
                <el-radio value="case_per_file">一案一檔壓縮包 (.zip)</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>

          <el-col :xs="24" :sm="12">
            <div v-if="authStore.can('staff')" class="action-buttons">
              <el-button
                plain
                :loading="checking"
                @click="handleRunPrecheck"
              >
                執行前置檢核
              </el-button>

              <el-button
                type="primary"
                :loading="exporting"
                @click="handleStartExport"
              >
                <el-icon><Download /></el-icon>
                開始產生申報檔
              </el-button>
            </div>
          </el-col>
        </el-row>
      </el-form>
    </el-card>

    <!-- 前置檢核結果呈現區塊 -->
    <el-card v-if="precheckResult" shadow="never" class="precheck-card">
      <template #header>
        <span class="card-title">前置檢核報告</span>
      </template>

      <PrecheckResult :result="precheckResult" />
    </el-card>

    <!-- 匯出進度與下載中指示 -->
    <el-card v-if="currentJob" shadow="never" class="job-card">
      <template #header>
        <span class="card-title">當前匯出工作進度</span>
      </template>

      <div class="job-status-container">
        <div class="job-info">
          <span>工作編號：{{ currentJob.id }}</span>
          <span>模式：{{ currentJob.mode === 'single_multi_case' ? '單檔多案' : '一案一檔 ZIP' }}</span>
          <el-tag :type="currentJob.status === 'succeeded' ? 'success' : (currentJob.status === 'failed' ? 'danger' : 'warning')">
            {{ EXPORT_STATUS_LABELS[currentJob.status] }}
          </el-tag>
        </div>

        <el-progress
          :percentage="progressPercent"
          :status="currentJob.status === 'succeeded' ? 'success' : (currentJob.status === 'failed' ? 'exception' : '')"
          :indeterminate="currentJob.status === 'running'"
          style="margin: 16px 0;"
        />

        <div v-if="currentJob.status === 'succeeded' && currentJob.downloadUrl" class="download-box">
          <el-button type="success" size="large" @click="downloadFile(currentJob.downloadUrl)">
            <el-icon><Download /></el-icon>
            下載已簽章檔案 ({{ currentJob.fileName || 'gov-claim.xlsx' }})
          </el-button>
        </div>

        <div v-if="currentJob.status === 'failed'" class="error-box">
          <el-alert
            type="error"
            show-icon
            :closable="false"
            title="匯出失敗"
            :description="currentJob.errorMessage || '請檢查前置檢核結果後重試；若仍無法匯出，請聯絡管理員。'"
          />
        </div>
      </div>
    </el-card>

    <!-- 歷史匯出紀錄 -->
    <el-card shadow="never" class="history-card">
      <template #header>
        <span class="card-title">歷史匯出紀錄</span>
      </template>

      <el-table :data="historyJobs" border stripe v-loading="loadingHistory">
        <el-table-column prop="periodYm" label="申報年月" width="110" />
        <el-table-column prop="region" label="區域" width="100" align="center">
          <template #default="{ row }">
            {{ row.region ? (REGION_LABELS[row.region] || row.region) : '全區' }}
          </template>
        </el-table-column>
        <el-table-column label="模式" width="140">
          <template #default="{ row }">
            {{ row.mode === 'single_multi_case' ? '單檔多案' : '一案一檔 ZIP' }}
          </template>
        </el-table-column>
        <el-table-column prop="totalCases" label="個案數" width="90" align="center" />
        <el-table-column prop="totalRows" label="總趟數" width="90" align="center" />
        <el-table-column prop="createdAt" label="產生時間" width="170" align="center">
          <template #default="{ row }">
            <span>{{ formatDateTime(row.createdAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="狀態" width="100" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 'succeeded' ? 'success' : 'danger'">
              {{ EXPORT_STATUS_LABELS[row.status as 'succeeded'|'failed'] }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="130" fixed="right" align="center">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 'succeeded' && row.downloadUrl"
              link
              type="primary"
              size="small"
              @click="downloadFile(row.downloadUrl)"
            >
              下載檔案
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import PageHeader from '@/components/PageHeader.vue'
import { Download } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { resolveErrorMessage } from '@/api/errorCodes'
import PrecheckResult from '@/components/PrecheckResult.vue'
import { formatDateTime } from '@/utils/formatters'
import {
  precheckExport,
  createExportJob,
  getExportJob,
  listExportJobs,
  downloadExportFile
} from '@/api/exports'
import { useAuthStore } from '@/stores/auth'
import { useRocMonth } from '@/composables/useRocMonth'
import { downloadBlob } from '@/utils/download'
import {
  REGION_LABELS,
  EXPORT_STATUS_LABELS
} from '@/types/domain'
import type {
  PrecheckResultDTO,
  ExportJobDTO,
  CreateExportJobRequest
} from '@/types/api'

const authStore = useAuthStore()
const { toRocMonth, toRocPeriodYm, formatRocMonthLabel } = useRocMonth()

const selectedDate = ref<string>('2026-07')
const checking = ref(false)
const exporting = ref(false)
const precheckResult = ref<PrecheckResultDTO | null>(null)
const currentJob = ref<ExportJobDTO | null>(null)
const historyJobs = ref<ExportJobDTO[]>([])
const loadingHistory = ref(false)

let pollTimer: ReturnType<typeof setInterval> | null = null

const currentRocMonth = computed(() => {
  return toRocMonth(selectedDate.value)
})

const form = reactive<{
  region: 'miaoli' | 'hsinchu' | ''
  mode: 'single_multi_case' | 'case_per_file'
}>({
  region: '',
  mode: 'single_multi_case'
})

const progressPercent = computed(() => {
  if (!currentJob.value) return 0
  if (currentJob.value.status === 'succeeded') return 100
  if (currentJob.value.status === 'failed') return 100
  return 60
})

async function handleRunPrecheck() {
  checking.value = true
  try {
    const res = await precheckExport({
      periodYm: toRocPeriodYm(selectedDate.value),
      region: form.region || undefined
    })
    precheckResult.value = res
  } finally {
    checking.value = false
  }
}

async function handleStartExport() {
  // 先自動跑一次檢核
  await handleRunPrecheck()

  if (precheckResult.value?.hasErrors) {
    ElMessage.error('前置檢核存在阻斷性錯誤，無法執行匯出，請先修正問題。')
    return
  }

  if (precheckResult.value?.hasWarnings) {
    await ElMessageBox.confirm(
      '本次匯出含有警告事項未處理，確定仍要繼續執行匯出？',
      '匯出警告確認',
      {
        confirmButtonText: '繼續匯出',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
  }

  exporting.value = true
  try {
    const jobReq: CreateExportJobRequest = {
      jobType: 'gov_claim',
      periodYm: toRocPeriodYm(selectedDate.value),
      region: form.region || undefined,
      mode: form.mode
    }

    const job = await createExportJob(jobReq)
    currentJob.value = job
    startPolling(job.id)
  } finally {
    exporting.value = false
  }
}

function startPolling(jobId: string) {
  if (pollTimer) clearInterval(pollTimer)
  pollTimer = setInterval(async () => {
    try {
      const res = await getExportJob(jobId)
      currentJob.value = res
      if (res.status === 'succeeded' || res.status === 'failed') {
        if (pollTimer) clearInterval(pollTimer)
        fetchHistory()
      }
    } catch (err: any) {
      if (pollTimer) clearInterval(pollTimer)
      // 輪詢請求本身失敗（非匯出工作失敗），需明確呈現，避免進度條永遠停在「進行中」
      if (currentJob.value) currentJob.value.status = 'failed'
      ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '查詢匯出進度失敗'))
    }
  }, 2000)
}

async function downloadFile(url: string) {
  try {
    const blob = await downloadExportFile(url)
    downloadBlob(blob, currentJob.value?.fileName || `gov-claim-${selectedDate.value}.xlsx`)
    ElMessage.success('申報檔案下載成功')
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '下載檔案失敗'))
  }
}

async function fetchHistory() {
  loadingHistory.value = true
  try {
    const res = await listExportJobs({ pageSize: 10 })
    historyJobs.value = res.data
  } finally {
    loadingHistory.value = false
  }
}

onMounted(() => {
  fetchHistory()
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style scoped>
.export-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.export-settings-card, .precheck-card, .job-card, .history-card {
  border-radius: 8px;
}

.card-title {
  font-size: 16px;
  font-weight: bold;
  color: var(--el-color-primary);
}

.roc-month-picker {
  display: flex;
  align-items: center;
  gap: 10px;

  .roc-label {
    font-weight: bold;
    color: var(--el-color-primary);
  }
}

.mode-form-item {
  margin-bottom: 0;
}

.action-buttons {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 12px;
}

.job-status-container {
  display: flex;
  flex-direction: column;

  .job-info {
    display: flex;
    gap: 20px;
    align-items: center;
    font-weight: 500;
  }

  .download-box {
    margin-top: 12px;
    display: flex;
    justify-content: flex-end;
  }

  .error-box {
    margin-top: 12px;
  }
}

@media (max-width: 640px) {
  .roc-month-picker,
  .action-buttons,
  .job-status-container .job-info {
    align-items: flex-start;
    flex-wrap: wrap;
  }

  .action-buttons,
  .job-status-container .download-box {
    justify-content: flex-start;
  }
}
</style>
