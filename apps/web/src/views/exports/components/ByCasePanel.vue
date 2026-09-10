<template>
  <div class="panel">
    <el-card shadow="never" class="settings-card">
      <template #header>
        <span class="card-title">政府申報表匯出設定</span>
      </template>

      <el-form :model="form" label-width="140px" :disabled="!canEdit">
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
            <div v-if="canEdit" class="action-buttons">
              <el-button plain :loading="checking" @click="handleRunPrecheck">執行前置檢核</el-button>
              <el-button type="primary" :loading="exporting" @click="handleStartExport">
                <el-icon><Download /></el-icon>
                開始產生申報檔
              </el-button>
            </div>
          </el-col>
        </el-row>
      </el-form>
    </el-card>

    <ExportResultCard
      v-if="currentJob"
      :job="currentJob"
      :downloading-case-id="downloadingCaseId"
      :downloading-zip="downloadingZip"
      @download-case="handleDownloadCaseFile"
      @download-zip="handleDownloadZip"
    />

    <el-card v-if="precheckResult" shadow="never" class="precheck-card">
      <template #header>
        <span class="card-title">前置檢核報告</span>
      </template>
      <PrecheckResult :result="precheckResult" />
    </el-card>

    <CaseSelectDialog
      v-model="caseDialogVisible"
      title="選擇申報個案"
      confirm-text="確認選擇"
      :initial-selected-ids="form.caseIds"
      @confirm="handleCaseSelected"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import CaseSelectDialog from '@/components/CaseSelectDialog.vue'
import PrecheckResult from '@/components/PrecheckResult.vue'
import ExportResultCard from './ExportResultCard.vue'
import { Download } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { currentLocalMonth } from '@/utils/formatters'
import {
  precheckExport,
  createExportJob,
  downloadExportCaseFile,
  downloadExportZip
} from '@/api/exports'
import { useRocMonth } from '@/composables/useRocMonth'
import { downloadBlob } from '@/utils/download'
import type { ExportMode } from '@/types/domain'
import type {
  PrecheckResultDTO,
  ExportJobDTO,
  ExportJobFileDTO,
  CreateExportJobRequest
} from '@/types/api'

defineProps<{ canEdit: boolean }>()

const emit = defineEmits<{ (e: 'exported'): void }>()

const { toRocMonth, toRocPeriodYm, formatRocMonthLabel } = useRocMonth()

const selectedDate = ref<string>(currentLocalMonth())
const checking = ref(false)
const exporting = ref(false)
const precheckResult = ref<PrecheckResultDTO | null>(null)
const currentJob = ref<ExportJobDTO | null>(null)
const caseDialogVisible = ref(false)
const downloadingCaseId = ref<string>('')
const downloadingZip = ref(false)

const currentRocMonth = computed(() => toRocMonth(selectedDate.value))

const form = reactive<{ mode: ExportMode; caseIds: string[]; caseNames: string[] }>({
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

function handleCaseSelected(cases: { id: string; name: string }[]) {
  form.caseIds = cases.map((c) => c.id)
  form.caseNames = cases.map((c) => c.name)
  caseDialogVisible.value = false
}

// 回傳檢核結果而非只寫進 ref：呼叫端要能分辨「檢核失敗」與「檢核通過但有警告」。
async function runPrecheck(): Promise<PrecheckResultDTO | null> {
  checking.value = true
  try {
    const res = await precheckExport({
      periodYm: toRocPeriodYm(selectedDate.value),
      caseIds: [...form.caseIds]
    })
    precheckResult.value = res
    return res
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
    return null
  } finally {
    checking.value = false
  }
}

async function handleRunPrecheck() {
  // caseIds 為空時後端會把檢核範圍放大成整月全部個案，結果與使用者以為的不同。
  if (form.caseIds.length === 0) {
    ElMessage.warning('請先選擇要申報的個案')
    return
  }
  await runPrecheck()
}

async function handleStartExport() {
  if (form.caseIds.length === 0) {
    ElMessage.warning('請先選擇要申報的個案')
    return
  }

  const result = await runPrecheck()
  if (!result) return

  if (result.hasErrors) {
    ElMessage.error('前置檢核存在未裁決的混車衝突，無法執行匯出，請先完成裁決。')
    return
  }

  if (result.hasWarnings) {
    // 使用者按「取消」時 ElMessageBox 會 reject，不接住會變成未處理的 promise 錯誤。
    const confirmed = await ElMessageBox.confirm(
      '本次匯出有個案資料不完整，缺少的欄位會留白匯出，確定仍要繼續執行匯出？',
      '匯出警告確認',
      { confirmButtonText: '繼續匯出', cancelButtonText: '取消', type: 'warning' }
    )
      .then(() => true)
      .catch(() => false)
    if (!confirmed) return
  }

  exporting.value = true
  try {
    const jobReq: CreateExportJobRequest = {
      jobType: 'gov_claim',
      periodYm: toRocPeriodYm(selectedDate.value),
      mode: form.mode,
      caseIds: [...form.caseIds]
    }

    currentJob.value = await createExportJob(jobReq)
    ElMessage.success(
      `已產生 ${currentJob.value.totalCases ?? 0} 份申報檔案（共 ${currentJob.value.totalRows ?? 0} 列）`
    )
    emit('exported')
  } catch {
    // 全域攔截器負責顯示 API 錯誤（含查無可申報資料）。
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
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
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
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  } finally {
    downloadingZip.value = false
  }
}
</script>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.settings-card,
.precheck-card {
  border-radius: 8px;
}

.card-title {
  font-size: 16px;
  font-weight: bold;
  color: var(--app-primary);
}

.roc-month-picker {
  display: flex;
  align-items: center;
  gap: 10px;

  .roc-label {
    font-weight: bold;
    color: var(--app-primary);
  }
}

.case-picker {
  display: flex;
  align-items: center;
  gap: 10px;

  .case-picker-summary {
    color: var(--app-text-secondary);
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

@media (max-width: 640px) {
  .roc-month-picker,
  .case-picker,
  .action-buttons {
    align-items: flex-start;
    flex-wrap: wrap;
  }

  .action-buttons {
    justify-content: flex-start;
  }
}
</style>
