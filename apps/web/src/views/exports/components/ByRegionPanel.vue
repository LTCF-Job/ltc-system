<template>
  <div class="panel">
    <el-card shadow="never" class="settings-card">
      <template #header>
        <span class="card-title">依區域批次匯出申報檔</span>
      </template>

      <el-alert type="info" show-icon :closable="false" class="scope-hint">
        一次拉出所選區域底下的所有個案，逐月產出 33 欄政府申報檔（每個案每月一份），
        免去逐案勾選。區域取自個案所屬據點；尚未綁定據點或仍在待維護的個案不會納入。
      </el-alert>

      <el-form label-width="140px" :disabled="!canEdit" class="settings-form">
        <el-row :gutter="16">
          <el-col :xs="24" :sm="12">
            <el-form-item label="申報區域">
              <el-select
                v-model="selectedRegions"
                multiple
                filterable
                collapse-tags
                collapse-tags-tooltip
                placeholder="選擇區域（可多選）"
                style="width: 100%"
                :loading="loadingRegions"
              >
                <el-option
                  v-for="region in regionOptions"
                  :key="region"
                  :label="regionLabel(region)"
                  :value="region"
                />
              </el-select>
            </el-form-item>
          </el-col>

          <el-col :xs="24" :sm="12">
            <el-form-item label="申報月份 (民國)">
              <div class="month-picker">
                <el-date-picker
                  v-model="selectedMonths"
                  type="months"
                  format="YYYY-MM"
                  value-format="YYYY-MM"
                  placeholder="選擇月份（可多選）"
                  style="width: 100%"
                />
                <span class="month-summary">{{ summary }}</span>
              </div>
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

    <el-card v-if="result" shadow="never" class="result-card">
      <template #header>
        <span class="card-title">本次批次匯出結果</span>
      </template>

      <div class="result-info">
        <span>區域：{{ result.regions.join('、') }}</span>
        <span>個案數：{{ result.caseCount }}</span>
        <span>檔案總數：{{ result.totalFiles }}</span>
      </div>

      <el-table :data="result.months" border stripe class="month-table">
        <el-table-column label="申報年月" width="120" align="center">
          <template #default="{ row }">{{ formatRocMonthLabel(rocMonthOf(row.periodYm)) }}</template>
        </el-table-column>
        <el-table-column label="個案數" width="90" align="center">
          <template #default="{ row }">{{ row.job?.totalCases ?? 0 }}</template>
        </el-table-column>
        <el-table-column label="申報行數" width="100" align="center">
          <template #default="{ row }">{{ row.job?.totalRows ?? 0 }}</template>
        </el-table-column>
        <el-table-column label="狀態" min-width="200">
          <template #default="{ row }">
            <el-tag v-if="row.succeeded" size="small" type="success">已產生</el-tag>
            <span v-else class="month-error">
              <el-tag size="small" type="warning">未產生</el-tag>
              {{ row.errorMessage }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right" align="center">
          <template #default="{ row }">
            <el-button
              v-if="row.succeeded && row.job"
              link
              type="primary"
              size="small"
              :loading="downloadingJobId === row.job.id"
              @click="handleDownloadMonth(row as RegionExportMonthDTO)"
            >
              下載
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="result.batchDownloadUrl" class="batch-download">
        <el-button type="success" :loading="downloadingBatch" @click="handleDownloadBatch">
          <el-icon><Download /></el-icon>
          下載全部（{{ result.batchFileName }}）
        </el-button>
      </div>
    </el-card>

    <el-card v-if="precheckResult" shadow="never" class="precheck-card">
      <template #header>
        <span class="card-title">前置檢核報告</span>
      </template>
      <PrecheckResult :result="precheckResult" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Download } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import PrecheckResult from '@/components/PrecheckResult.vue'
import {
  precheckExport,
  createRegionExportJobs,
  downloadExportZip,
  downloadExportBatchZip
} from '@/api/exports'
import { useRegionOptions } from '@/composables/useRegionOptions'
import { useRocMonthRange } from '@/composables/useRocMonthRange'
import { useRocMonth } from '@/composables/useRocMonth'
import { downloadBlob } from '@/utils/download'
import type { PrecheckResultDTO, RegionExportResultDTO, RegionExportMonthDTO } from '@/types/api'

defineProps<{ canEdit: boolean }>()

const emit = defineEmits<{ (e: 'exported'): void }>()

const { regionOptions, siteCountByRegion, loadingRegions, refreshRegionOptions } = useRegionOptions()
const { selectedMonths, periodYms, summary } = useRocMonthRange()
const { formatRocMonthLabel } = useRocMonth()

const selectedRegions = ref<string[]>([])
const checking = ref(false)
const exporting = ref(false)
const precheckResult = ref<PrecheckResultDTO | null>(null)
const result = ref<RegionExportResultDTO | null>(null)
const downloadingJobId = ref<string>('')
const downloadingBatch = ref(false)

function regionLabel(region: string): string {
  const count = siteCountByRegion.value[region] ?? 0
  return `${region}（${count} 個據點）`
}

function rocMonthOf(periodYm: string): string {
  if (!periodYm || periodYm.length !== 5) return periodYm
  return `${periodYm.slice(0, 3)}-${periodYm.slice(3)}`
}

async function runPrecheck(): Promise<PrecheckResultDTO | null> {
  checking.value = true
  try {
    const res = await precheckExport({
      periodYms: [...periodYms.value],
      regions: [...selectedRegions.value]
    })
    precheckResult.value = res
    return res
  } catch {
    return null
  } finally {
    checking.value = false
  }
}

function validateSelection(): boolean {
  if (selectedRegions.value.length === 0) {
    ElMessage.warning('請先選擇要申報的區域')
    return false
  }
  if (periodYms.value.length === 0) {
    ElMessage.warning('請先選擇要申報的月份')
    return false
  }
  return true
}

async function handleRunPrecheck() {
  if (!validateSelection()) return
  await runPrecheck()
}

async function handleStartExport() {
  if (!validateSelection()) return

  const report = await runPrecheck()
  if (!report) return

  if (report.hasErrors) {
    ElMessage.error('前置檢核存在未裁決的混車衝突，無法執行匯出，請先完成裁決。')
    return
  }

  if (report.hasWarnings) {
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
    result.value = await createRegionExportJobs({
      regions: [...selectedRegions.value],
      periodYms: [...periodYms.value]
    })
    ElMessage.success(`已產生 ${result.value.totalFiles} 份申報檔案`)
    emit('exported')
  } catch {
    // 全域攔截器負責顯示 API 錯誤（含查無可申報資料）。
  } finally {
    exporting.value = false
  }
}

async function handleDownloadMonth(row: RegionExportMonthDTO) {
  if (!row.job) return
  downloadingJobId.value = row.job.id
  try {
    const blob = await downloadExportZip(row.job.id)
    downloadBlob(blob, row.job.zipFileName || `gov-claim-${row.periodYm}.zip`)
    ElMessage.success('壓縮檔下載成功')
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  } finally {
    downloadingJobId.value = ''
  }
}

async function handleDownloadBatch() {
  if (!result.value) return
  const jobIds = result.value.months
    .filter((month) => month.succeeded && month.job)
    .map((month) => month.job!.id)
  if (jobIds.length === 0) return

  downloadingBatch.value = true
  try {
    const blob = await downloadExportBatchZip(jobIds)
    downloadBlob(blob, result.value.batchFileName || 'gov-claim.zip')
    ElMessage.success('壓縮檔下載成功')
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  } finally {
    downloadingBatch.value = false
  }
}

onMounted(() => {
  refreshRegionOptions().catch(() => {
    // 全域攔截器負責顯示 API 錯誤；區域選項為空時使用者會看到空選單。
  })
})
</script>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.settings-card,
.result-card,
.precheck-card {
  border-radius: 8px;
}

.card-title {
  font-size: 16px;
  font-weight: bold;
  color: var(--app-primary);
}

.scope-hint {
  margin-bottom: 16px;
}

.month-picker {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 100%;

  .month-summary {
    color: var(--app-text-secondary);
    font-size: 13px;
  }
}

.action-buttons {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 12px;
}

.result-info {
  display: flex;
  gap: 20px;
  align-items: center;
  flex-wrap: wrap;
  font-weight: 500;
}

.month-table {
  margin-top: 12px;
}

.month-error {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--app-text-secondary);
}

.batch-download {
  margin-top: 12px;
  display: flex;
  justify-content: flex-start;
}

@media (max-width: 640px) {
  .action-buttons {
    justify-content: flex-start;
  }

  .result-info {
    align-items: flex-start;
  }
}
</style>
