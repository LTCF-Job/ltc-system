<template>
  <el-dialog
    v-model="visible"
    :title="`匯入司機接送匯報 — ${form?.title || ''}`"
    width="960px"
    destroy-on-close
    :before-close="handleClose"
  >
    <!-- 第一步：上傳檔案 -->
    <div v-if="!preview" class="upload-section">
      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="匯入前請先下載該車範本填寫，確認欄位順序正確後再上傳。"
      >
        <template #default>
          <div class="template-action">
            <span>
              欄位順序固定為：民國日期、駕駛人、各個案趟次欄、備註。
              個案趟次欄只填「有坐」或「沒坐」，留白視為未回報。
            </span>
            <el-button
              type="primary"
              plain
              size="small"
              :loading="downloadingTemplate"
              style="margin-top: 8px"
              @click="handleDownloadTemplate"
            >
              <el-icon><Download /></el-icon>
              下載該車匯報範本
            </el-button>
          </div>
        </template>
      </el-alert>

      <el-upload
        drag
        action="#"
        :auto-upload="false"
        :limit="1"
        :on-change="handleFileChange"
        accept=".xlsx"
        style="width: 100%; margin-top: 16px;"
      >
        <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
        <div class="el-upload__text">
          拖曳 Excel 檔案至此，或 <em>點選上傳</em>
        </div>
        <template #tip>
          <div class="el-upload__tip">僅支援 .xlsx 格式之匯報表</div>
        </template>
      </el-upload>

      <div class="dialog-footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button
          type="primary"
          :disabled="!selectedFile"
          :loading="analyzing"
          @click="startDryRun"
        >
          開始解析與預覽
        </el-button>
      </div>
    </div>

    <!-- 第二步：欄位對應就地確認與匯報列預覽 -->
    <div v-else class="preview-section">
      <div class="stats-bar" role="status">
        <el-tag type="info">匯報天數：{{ preview.totalRows }}</el-tag>
        <el-tag type="success">可匯入：{{ preview.validRows }}</el-tag>
        <el-tag v-if="preview.errorRows > 0" type="danger">日期錯誤：{{ preview.errorRows }}</el-tag>
        <el-tag v-if="preview.warningRows > 0" type="warning">需確認：{{ preview.warningRows }}</el-tag>
        <el-tag :type="unresolvedColumnCount > 0 ? 'warning' : 'success'">
          待對應欄位：{{ unresolvedColumnCount }} / {{ preview.columns.length }}
        </el-tag>
      </div>

      <el-alert
        v-if="unresolvedColumnCount > 0"
        type="warning"
        show-icon
        :closable="false"
        title="尚有欄位未對應到個案，未對應的欄位不會寫入搭乘紀錄。"
        description="請於下方逐欄選擇個案與趟次，或將該欄標記為略過。對應結果會一併儲存，下次匯入同一份匯報表時自動沿用。"
        style="margin-bottom: 12px;"
      />

      <el-tabs v-model="activeTab">
        <el-tab-pane name="columns" :label="`欄位對應 (${preview.columns.length})`">
          <el-table :data="preview.columns" max-height="360" border size="small">
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
                <el-tag size="small" :type="statusTagType(decisions[row.columnHeader].mappingStatus)">
                  {{ MAPPING_STATUS_LABELS[decisions[row.columnHeader].mappingStatus] }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100" align="center">
              <template #default="{ row }">
                <el-button
                  link
                  size="small"
                  :type="decisions[row.columnHeader].mappingStatus === 'ignored' ? 'primary' : 'info'"
                  @click="toggleIgnore(row.columnHeader)"
                >
                  {{ decisions[row.columnHeader].mappingStatus === 'ignored' ? '取消略過' : '略過此欄' }}
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane name="rows" :label="`每日匯報 (${preview.previewRows.length})`">
          <el-table
            :data="preview.previewRows"
            max-height="360"
            border
            size="small"
            :row-class-name="rowClassName"
          >
            <el-table-column prop="reportDate" label="民國日期" width="110" />
            <el-table-column prop="serviceDate" label="服務日期" width="120" />
            <el-table-column label="駕駛人" width="140">
              <template #default="{ row }">
                <span>{{ row.driverRaw || '（未填）' }}</span>
                <el-tag v-if="row.driverId" size="small" type="success" class="ml-1">已比對</el-tag>
                <el-tag v-else-if="row.driverRaw" size="small" type="warning" class="ml-1">未比對</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="有坐" width="70" align="center" prop="boardedCount" />
            <el-table-column label="沒坐" width="70" align="center" prop="absentCount" />
            <el-table-column prop="remark" label="備註" min-width="160" show-overflow-tooltip />
            <el-table-column label="檢核" min-width="200">
              <template #default="{ row }">
                <span v-if="row.errorMessage" class="text-danger">{{ row.errorMessage }}</span>
                <span v-else-if="row.warningMessage" class="text-warning">{{ row.warningMessage }}</span>
                <span v-else class="text-muted">—</span>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>

      <div class="dialog-footer">
        <el-button @click="resetToUpload">重新選擇檔案</el-button>
        <el-button
          type="primary"
          :disabled="preview.validRows === 0 || mappedColumnCount === 0"
          :loading="submitting"
          @click="confirmImport"
        >
          匯入 ({{ preview.validRows }} 天)
        </el-button>
      </div>

      <div v-if="commitResult" class="result-list" role="status">
        <h4>
          匯入結果：寫入 {{ commitResult.importedRows }} 天，
          產生 {{ commitResult.rideRecordRows }} 筆搭乘紀錄，
          略過 {{ commitResult.skippedRows.length }} 天
        </h4>
        <ul v-if="commitResult.skippedRows.length">
          <li v-for="row in commitResult.skippedRows" :key="row.rowIndex">
            第 {{ row.rowIndex }} 列（{{ row.reportDate }}）：{{ row.reasons.join('；') }}
          </li>
        </ul>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { UploadFilled, Download } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { UploadFile } from 'element-plus'
import {
  dryRunImportDriverReport,
  commitImportDriverReport,
  downloadDriverReportTemplate
} from '@/api/driverReports'
import { listCases } from '@/api/cases'
import { LEG_SEQ_OPTIONS } from './legOptions'
import { MAPPING_STATUS_LABELS, type MappingStatus } from '@/types/domain'
import type {
  CaseDTO,
  DriverReportCommitResultDTO,
  DriverReportFormDTO,
  DriverReportPreviewDTO
} from '@/types/api'

const emit = defineEmits<{ (e: 'success'): void }>()

interface ColumnDecisionState {
  mappingStatus: MappingStatus
  caseId?: string
  legSeq?: number
}

const visible = ref(false)
const form = ref<DriverReportFormDTO | null>(null)
const selectedFile = ref<File | null>(null)
const analyzing = ref(false)
const submitting = ref(false)
const downloadingTemplate = ref(false)
const preview = ref<DriverReportPreviewDTO | null>(null)
const commitResult = ref<DriverReportCommitResultDTO | null>(null)
const cases = ref<CaseDTO[]>([])
const activeTab = ref('columns')
// 以欄位表頭為鍵：欄號會隨個案增減而位移，表頭才是後端認得的穩定識別
const decisions = ref<Record<string, ColumnDecisionState>>({})

const unresolvedColumnCount = computed(
  () => Object.values(decisions.value).filter((d) => d.mappingStatus === 'pending').length
)
const mappedColumnCount = computed(
  () => Object.values(decisions.value).filter((d) => d.mappingStatus === 'mapped').length
)

function statusTagType(status: MappingStatus) {
  if (status === 'mapped') return 'success'
  if (status === 'pending') return 'warning'
  return 'info'
}

function rowClassName({ row }: { row: { errorMessage?: string; warningMessage?: string } }) {
  if (row.errorMessage) return 'row-error'
  if (row.warningMessage) return 'row-warning'
  return ''
}

function open(target: DriverReportFormDTO) {
  form.value = target
  selectedFile.value = null
  preview.value = null
  commitResult.value = null
  decisions.value = {}
  activeTab.value = 'columns'
  visible.value = true
}

function close() {
  resetToUpload()
  visible.value = false
}

function resetToUpload() {
  preview.value = null
  commitResult.value = null
  selectedFile.value = null
  decisions.value = {}
}

function handleClose(done: () => void) {
  resetToUpload()
  done()
}

function handleFileChange(uploadFile: UploadFile) {
  selectedFile.value = uploadFile.raw ?? null
}

async function handleDownloadTemplate() {
  if (!form.value) return
  downloadingTemplate.value = true
  try {
    const blob = await downloadDriverReportTemplate(form.value.id)
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${form.value.vehicleName}接送匯報範本.xlsx`
    link.click()
    URL.revokeObjectURL(url)
  } finally {
    downloadingTemplate.value = false
  }
}

// 解析失敗的原因已由 API client 以通知呈現，這裡吞掉 rejection 讓畫面停在上傳步驟
async function startDryRun() {
  if (!selectedFile.value || !form.value) return
  analyzing.value = true
  try {
    const res = await dryRunImportDriverReport(form.value.id, selectedFile.value)
    preview.value = res
    decisions.value = Object.fromEntries(
      res.columns.map((c) => [
        c.columnHeader,
        {
          mappingStatus: c.mappingStatus,
          caseId: c.caseId || c.suggestedCaseId || undefined,
          legSeq: c.legSeq || c.suggestedLegSeq || undefined
        }
      ])
    )
    activeTab.value = res.unmappedColumns > 0 ? 'columns' : 'rows'
  } catch {
    preview.value = null
  } finally {
    analyzing.value = false
  }
}

// 選好個案即視為已對應：趟次留白時沿用推薦值，避免使用者被迫多點一次
function onCaseChange(columnHeader: string) {
  const decision = decisions.value[columnHeader]
  if (!decision) return
  if (decision.caseId) {
    decision.mappingStatus = 'mapped'
    if (!decision.legSeq) {
      const column = preview.value?.columns.find((c) => c.columnHeader === columnHeader)
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

async function confirmImport() {
  if (!selectedFile.value || !form.value) return
  submitting.value = true
  try {
    const payload = Object.entries(decisions.value).map(([columnHeader, d]) => ({
      columnHeader,
      mappingStatus: d.mappingStatus,
      caseId: d.mappingStatus === 'mapped' ? d.caseId : null,
      legSeq: d.mappingStatus === 'mapped' ? d.legSeq : null
    }))
    const result = await commitImportDriverReport(form.value.id, selectedFile.value, payload)
    commitResult.value = result
    ElMessage.success(`已匯入 ${result.importedRows} 天的接送匯報`)
    emit('success')
  } catch {
    // 失敗原因已由 API client 以通知呈現；保留預覽讓使用者修正對應後再送出
  } finally {
    submitting.value = false
  }
}

async function loadCases() {
  const res = await listCases({ pageSize: 200 })
  cases.value = res.data
}
loadCases()

defineExpose({ open, close })
</script>

<style scoped>
.upload-section {
  display: flex;
  flex-direction: column;
  padding: 8px 0 0;
}

.template-action {
  display: flex;
  flex-direction: column;
}

.stats-bar {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}

.column-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.column-header {
  font-weight: 500;
}

.column-meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.ml-1 {
  margin-left: 4px;
}

.text-danger {
  color: var(--el-color-danger);
}

.text-warning {
  color: var(--el-color-warning);
}

.text-muted {
  color: var(--el-text-color-placeholder);
}

.result-list {
  margin-top: 12px;
  padding: 8px 12px;
  background-color: var(--el-color-info-light-9);
  border-radius: 4px;
  font-size: 13px;
}

.result-list ul {
  padding-left: 20px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 20px;
}

:deep(.row-error) {
  background-color: var(--el-color-danger-light-9) !important;
}

:deep(.row-warning) {
  background-color: var(--el-color-warning-light-9) !important;
}
</style>
