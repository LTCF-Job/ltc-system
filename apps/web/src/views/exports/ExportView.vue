<template>
  <div class="export-view">
    <PageHeader title="政府申報匯出" />
    <!-- 匯出條件設定卡片 -->
    <el-card shadow="never" class="export-settings-card">
      <template #header>
        <span class="card-title">政府申報表匯出設定</span>
      </template>

      <el-form
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
                @change="handleRegionChange"
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

        <el-row :gutter="16">
          <el-col :xs="24" :sm="12">
            <el-form-item label="申報個案">
              <div class="case-picker">
                <el-button plain @click="caseDialogVisible = true">選擇個案</el-button>
                <span class="case-picker-summary">{{ selectedCaseSummary }}</span>
              </div>
            </el-form-item>
          </el-col>

          <el-col :xs="24" :sm="12">
            <el-form-item label="匯出檔案模式" class="mode-form-item">
              <el-radio-group v-model="form.mode">
                <el-radio value="direct">直接下載</el-radio>
                <el-radio value="zip">壓縮檔</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16">
          <el-col :span="24">
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

    <!-- 匯出結果：逐案下載清單或整包壓縮檔 -->
    <el-card v-if="currentJob" shadow="never" class="job-card">
      <template #header>
        <span class="card-title">本次匯出結果</span>
      </template>

      <div class="job-info">
        <span>申報年月：{{ formatRocMonthLabel(rocMonthOf(currentJob.periodYm)) }}</span>
        <span>模式：{{ EXPORT_MODE_LABELS[currentJob.mode] }}</span>
        <span>個案數：{{ currentJob.totalCases ?? 0 }}</span>
        <span>申報行數：{{ currentJob.totalRows ?? 0 }}</span>
      </div>

      <el-alert
        v-if="currentJob.skipped?.length"
        type="warning"
        show-icon
        :closable="false"
        title="部分趟次資料不完整，未納入申報"
        class="skip-alert"
      >
        <ul class="skip-list">
          <li v-for="(skip, index) in currentJob.skipped" :key="index">
            {{ skip.caseName }}：{{ skipReasonLabel(skip.reason) }}（{{ skip.count }} 筆）
          </li>
        </ul>
      </el-alert>

      <!-- 直接下載：一個個案一列，由使用者自行點選，避免瀏覽器擋下連續下載 -->
      <el-table
        v-if="currentJob.mode === 'direct'"
        :data="currentJob.files || []"
        border
        stripe
        class="file-table"
      >
        <el-table-column prop="caseCode" label="個案編號" width="110" align="center" />
        <el-table-column prop="caseName" label="姓名" width="120" />
        <el-table-column prop="region" label="區域" width="110" align="center">
          <template #default="{ row }">
            {{ row.region ? (REGION_LABELS[row.region] || row.region) : '—' }}
          </template>
        </el-table-column>
        <el-table-column prop="rowCount" label="申報行數" width="100" align="center" />
        <el-table-column prop="fileName" label="檔案名稱" min-width="200" show-overflow-tooltip />
        <el-table-column label="操作" width="120" fixed="right" align="center">
          <template #default="{ row }">
            <el-button
              link
              type="primary"
              size="small"
              :loading="downloadingCaseId === row.caseId"
              @click="handleDownloadCaseFile(row as ExportJobFileDTO)"
            >
              下載
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div v-else class="zip-download">
        <el-button type="success" size="large" :loading="downloadingZip" @click="handleDownloadZip">
          <el-icon><Download /></el-icon>
          下載壓縮檔 ({{ currentJob.zipFileName }})
        </el-button>
      </div>
    </el-card>

    <!-- 前置檢核結果呈現區塊 -->
    <el-card v-if="precheckResult" shadow="never" class="precheck-card">
      <template #header>
        <span class="card-title">前置檢核報告</span>
      </template>

      <PrecheckResult :result="precheckResult" />
    </el-card>

    <!-- 歷史匯出紀錄：只供查看當時匯出的個案，不重複提供檔案下載 -->
    <el-card shadow="never" class="history-card">
      <template #header>
        <span class="card-title">歷史匯出紀錄</span>
      </template>

      <el-table :data="historyJobs" border stripe v-loading="loadingHistory">
        <el-table-column prop="periodYm" label="申報年月" width="110" align="center" />
        <el-table-column prop="region" label="區域" width="100" align="center">
          <template #default="{ row }">
            {{ row.region ? (REGION_LABELS[row.region] || row.region) : '全區' }}
          </template>
        </el-table-column>
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
        <el-table-column label="狀態" width="100" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 'succeeded' ? 'success' : 'danger'">
              {{ EXPORT_STATUS_LABELS[row.status as ExportJobStatus] || row.status }}
            </el-tag>
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

    <CaseSelectDialog
      v-model="caseDialogVisible"
      title="選擇申報個案"
      confirm-text="確認選擇"
      :region="form.region"
      :initial-selected-ids="form.caseIds"
      @confirm="handleCaseSelected"
    />

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
        <el-table-column prop="caseCode" label="個案編號" width="110" align="center" />
        <el-table-column prop="caseName" label="姓名" width="120" />
        <el-table-column prop="region" label="區域" width="110" align="center">
          <template #default="{ row }">
            {{ row.region ? (REGION_LABELS[row.region] || row.region) : '—' }}
          </template>
        </el-table-column>
        <el-table-column prop="rowCount" label="申報行數" width="100" align="center" />
        <el-table-column prop="fileName" label="檔案名稱" min-width="180" show-overflow-tooltip />
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import PageHeader from '@/components/PageHeader.vue'
import CaseSelectDialog from '@/components/CaseSelectDialog.vue'
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
  downloadExportCaseFile,
  downloadExportZip
} from '@/api/exports'
import { useAuthStore } from '@/stores/auth'
import { useRocMonth } from '@/composables/useRocMonth'
import { downloadBlob } from '@/utils/download'
import {
  REGION_LABELS,
  EXPORT_STATUS_LABELS,
  EXPORT_MODE_LABELS,
  EXPORT_SKIP_REASON_LABELS
} from '@/types/domain'
import type { Region, ExportMode, ExportJobStatus } from '@/types/domain'
import type {
  PrecheckResultDTO,
  ExportJobDTO,
  ExportJobFileDTO,
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
const caseDialogVisible = ref(false)
const downloadingCaseId = ref<string>('')
const downloadingZip = ref(false)
const historyDetailVisible = ref(false)
const historyDetail = ref<ExportJobDTO | null>(null)
const loadingHistoryDetail = ref(false)

const currentRocMonth = computed(() => toRocMonth(selectedDate.value))

const form = reactive<{
  region: Region | ''
  mode: ExportMode
  caseIds: string[]
  caseNames: string[]
}>({
  region: '',
  mode: 'direct',
  caseIds: [],
  caseNames: []
})

const selectedCaseSummary = computed(() => {
  if (form.caseIds.length === 0) return '尚未選擇個案'
  const preview = form.caseNames.slice(0, 3).join('、')
  const suffix = form.caseNames.length > 3 ? ' 等' : ''
  return `已選擇 ${form.caseIds.length} 筆：${preview}${suffix}`
})

// 民國 5 碼（11507）轉成顯示用的 115-07
function rocMonthOf(periodYm: string): string {
  if (!periodYm || periodYm.length !== 5) return periodYm
  return `${periodYm.slice(0, 3)}-${periodYm.slice(3)}`
}

function skipReasonLabel(reason: string): string {
  return EXPORT_SKIP_REASON_LABELS[reason] || reason
}

// 地區是個案清單的篩選條件，改地區後既有勾選可能已不在清單內，一律清空重選
function handleRegionChange() {
  form.caseIds = []
  form.caseNames = []
}

function handleCaseSelected(cases: { id: string; name: string }[]) {
  form.caseIds = cases.map((c) => c.id)
  form.caseNames = cases.map((c) => c.name)
  caseDialogVisible.value = false
}

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
  if (form.caseIds.length === 0) {
    ElMessage.warning('請先選擇要申報的個案')
    return
  }

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
      mode: form.mode,
      caseIds: [...form.caseIds]
    }

    currentJob.value = await createExportJob(jobReq)
    ElMessage.success(`已產生 ${currentJob.value.totalCases ?? 0} 份申報檔案`)
    await fetchHistory()
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '產生申報檔失敗'))
  } finally {
    exporting.value = false
  }
}

async function handleDownloadCaseFile(file: ExportJobFileDTO) {
  if (!currentJob.value) return
  downloadingCaseId.value = file.caseId
  try {
    const blob = await downloadExportCaseFile(currentJob.value.id, file.caseId)
    downloadBlob(blob, file.fileName)
    ElMessage.success(`${file.fileName} 下載成功`)
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '下載檔案失敗'))
  } finally {
    downloadingCaseId.value = ''
  }
}

async function handleDownloadZip() {
  if (!currentJob.value) return
  downloadingZip.value = true
  try {
    const blob = await downloadExportZip(currentJob.value.id)
    downloadBlob(blob, currentJob.value.zipFileName || 'gov-claim.zip')
    ElMessage.success('壓縮檔下載成功')
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '下載壓縮檔失敗'))
  } finally {
    downloadingZip.value = false
  }
}

async function openHistoryDetail(row: ExportJobDTO) {
  historyDetailVisible.value = true
  historyDetail.value = null
  loadingHistoryDetail.value = true
  try {
    historyDetail.value = await getExportJob(row.id)
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '載入匯出明細失敗'))
  } finally {
    loadingHistoryDetail.value = false
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

.case-picker {
  display: flex;
  align-items: center;
  gap: 10px;

  .case-picker-summary {
    color: var(--el-text-color-secondary);
    font-size: 13px;
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

.file-table {
  margin-top: 12px;
}

.zip-download {
  margin-top: 12px;
  display: flex;
  justify-content: flex-start;
}

.history-detail-meta {
  display: flex;
  gap: 20px;
  flex-wrap: wrap;
  margin-bottom: 12px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

@media (max-width: 640px) {
  .roc-month-picker,
  .case-picker,
  .action-buttons,
  .job-info {
    align-items: flex-start;
    flex-wrap: wrap;
  }

  .action-buttons {
    justify-content: flex-start;
  }
}
</style>
