<template>
  <div class="batch-import-view">
    <el-card shadow="never" class="batch-card">
      <template #header>
        <PageHeader
          title="上傳接送匯報"
          description="每一列代表一輛車在指定月份的接送紀錄。先選擇檔案並點選試算檢查，確認無誤後即可儲存。新上傳的檔案會直接取代該車該月份的所有舊資料；檔案內只能包含指定月份的日期。尚未建立匯報表的車輛會在第一次上傳時自動建立。"
        >
          <template #actions>
            <label class="month-field">
              <span class="month-label">匯入月份</span>
              <el-date-picker
                v-model="selectedMonths"
                type="months"
                placeholder="選擇一個或多個月份"
                value-format="YYYY-MM"
                format="YYYY-MM"
                style="width: 320px"
              />
            </label>
            <el-button plain :disabled="!analyzableRows.length" :loading="analyzing" @click="analyzeAll">
              全部試算 ({{ analyzableRows.length }})
            </el-button>
            <el-button
              v-if="canEdit"
              type="primary"
              :disabled="!importableRows.length"
              :loading="importing"
              @click="confirmAndImport"
            >
              確認匯入 ({{ importableRows.length }})
            </el-button>
          </template>
        </PageHeader>
      </template>

      <div v-if="!rows.length" class="empty-state">
        <el-empty :description="normalizedMonths.length ? '沒有啟用中的車輛' : '請先選擇要匯入的月份'" />
      </div>

      <el-table
        v-else
        :data="rows"
        row-key="key"
        border
        :expand-row-keys="expandedKeys"
        v-loading="loading"
        style="margin-top: 16px"
        @expand-change="onExpandChange"
      >
        <el-table-column type="expand">
          <template #default="scope">
            <div class="row-detail">
              <p v-if="!asRow(scope.row).preview" class="text-muted">
                尚未試算，展開後可在試算完成時就地對應欄位。
              </p>
              <template v-else>
                <p class="detail-title">
                  {{ asRow(scope.row).vehicleName }} · {{ asRow(scope.row).yearMonth }} 欄位對應
                  （待對應 {{ pendingCount(asRow(scope.row)) }} /
                  {{ asRow(scope.row).preview!.columns.length }}）
                </p>
                <DriverReportColumnMappingTable
                  v-model:decisions="asRow(scope.row).decisions"
                  :columns="asRow(scope.row).preview!.columns"
                  :cases="cases"
                  max-height="300"
                />
              </template>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="車輛" min-width="150">
          <template #default="{ row }">
            <span>{{ row.vehicleName }}</span>
            <el-tag v-if="!row.formId" size="small" type="info" class="ml-1">尚未建表</el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="yearMonth" label="月份" width="110" />

        <el-table-column label="已匯入" width="190">
          <template #default="{ row }">
            <template v-if="row.importedCount > 0">
              <div>{{ row.importedCount }} 天</div>
              <div class="text-muted small">{{ formatDateTime(row.lastImportedAt) }}</div>
            </template>
            <span v-else class="text-muted">尚未匯入</span>
          </template>
        </el-table-column>

        <el-table-column label="上傳檔案" min-width="230">
          <template #default="{ row }">
            <div class="file-cell">
              <el-upload
                :auto-upload="false"
                :show-file-list="false"
                accept=".xlsx"
                :on-change="(file: UploadFile) => onFileChange(asRow(row), file)"
              >
                <el-button
                  link
                  type="primary"
                  size="small"
                  :disabled="!canEdit"
                  :aria-label="`${row.vehicleName} ${row.yearMonth} ${row.file ? '重新選擇檔案' : '選擇 .xlsx 檔案'}`"
                >
                  {{ row.file ? '重新選擇' : '選擇 .xlsx' }}
                </el-button>
              </el-upload>
              <span v-if="row.file" class="file-name">{{ row.file.name }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="試算結果" min-width="220">
          <template #default="{ row }">
            <template v-if="row.preview">
              <div>可匯入 {{ row.preview.validRows }} 天</div>
              <div v-if="row.preview.errorRows > 0" class="text-danger small">
                日期錯誤 {{ row.preview.errorRows }} 天
              </div>
              <div v-if="pendingCount(asRow(row)) > 0" class="text-warning small">
                待對應欄位 {{ pendingCount(asRow(row)) }} 欄
              </div>
            </template>
            <span v-else-if="row.message" class="text-danger">{{ row.message }}</span>
            <span v-else class="text-muted">—</span>
          </template>
        </el-table-column>

        <el-table-column label="狀態" width="120" align="center">
          <template #default="{ row }">
            <StatusTag :status="effectiveStatus(asRow(row))" preset="batchImportStatus" />
          </template>
        </el-table-column>

        <el-table-column label="操作" width="140" align="center" fixed="right">
          <template #default="{ row }">
            <el-button
              link
              type="primary"
              size="small"
              :disabled="!row.file || row.status === 'analyzing' || row.status === 'importing'"
              :aria-label="`試算 ${row.vehicleName} ${row.yearMonth}`"
              @click="analyzeRow(asRow(row))"
            >
              試算
            </el-button>
            <el-button
              v-if="row.preview"
              link
              type="primary"
              size="small"
              :aria-label="`${expandedKeys.includes(row.key) ? '收合' : '展開'} ${row.vehicleName} ${row.yearMonth} 的欄位對應`"
              @click="toggleExpand(asRow(row))"
            >
              {{ expandedKeys.includes(row.key) ? '收合' : '對應' }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="importSummary" class="result-list" role="status">
        <h2 class="result-title">
          匯入結果：成功 {{ importSummary.succeeded }} 列，失敗 {{ importSummary.failed }} 列
        </h2>
        <ul v-if="failedRows.length">
          <li v-for="row in failedRows" :key="row.key">
            {{ row.vehicleName }} · {{ row.yearMonth }}：{{ row.message }}
          </li>
        </ul>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, type UploadFile } from 'element-plus'
import {
  createDriverReportForm,
  commitImportDriverReport,
  dryRunImportDriverReport,
  listDriverReportForms,
  listDriverReportImportedMonths
} from '@/api/driverReports'
import { listVehicles } from '@/api/masters'
import { listCases } from '@/api/cases'
import { useAuthStore } from '@/stores/auth'
import PageHeader from '@/components/PageHeader.vue'
import StatusTag from '@/components/StatusTag.vue'
import { formatDateTime } from '@/utils/formatters'
import DriverReportColumnMappingTable from './DriverReportColumnMappingTable.vue'
import {
  countByStatus,
  createColumnDecisions,
  toColumnDecisionPayload,
  type ColumnDecisionMap
} from './columnDecisions'
import type { CaseDTO, DriverReportPreviewDTO } from '@/types/api'

type BatchRowStatus = 'idle' | 'analyzing' | 'ready' | 'needsMapping' | 'importing' | 'imported' | 'failed'

interface BatchRow {
  key: string
  vehicleId: string
  vehicleName: string
  formId: string
  yearMonth: string
  importedCount: number
  lastImportedAt: string
  file: File | null
  status: BatchRowStatus
  preview: DriverReportPreviewDTO | null
  decisions: ColumnDecisionMap
  message: string
}

// 逐列各自請求，不做批次 API：單列失敗不影響其他列。並發上限避免一次打出數十個請求。
const MAX_CONCURRENT_REQUESTS = 3

const authStore = useAuthStore()
const route = useRoute()
const router = useRouter()
const canEdit = computed(() => authStore.hasPermission('driver_reports', 'edit'))

function getCurrentYearMonth(): string {
  const now = new Date()
  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  return `${year}-${month}`
}

function parseMonthsQuery(raw: unknown): string[] {
  const value = Array.isArray(raw) ? raw[0] : raw
  if (typeof value !== 'string') return [getCurrentYearMonth()]
  const parsed = value.split(',').filter((m) => /^\d{4}-(0[1-9]|1[0-2])$/.test(m.trim()))
  return parsed.length > 0 ? parsed : [getCurrentYearMonth()]
}

// 月份寫進 query，讓「哪幾個月要補傳」這件事可以直接分享連結；預設帶入當前月份
const selectedMonths = ref<string[]>(parseMonthsQuery(route.query.months))

const rows = ref<BatchRow[]>([])
const cases = ref<CaseDTO[]>([])
const expandedKeys = ref<string[]>([])
const loading = ref(false)
const analyzing = ref(false)
const importing = ref(false)
const importSummary = ref<{ succeeded: number; failed: number } | null>(null)

const analyzableRows = computed(() =>
  rows.value.filter((r) => r.file && r.status !== 'analyzing' && r.status !== 'importing')
)
const importableRows = computed(() => rows.value.filter((r) => effectiveStatus(r) === 'ready'))
const failedRows = computed(() => rows.value.filter((r) => r.status === 'failed'))

function pendingCount(row: BatchRow): number {
  return countByStatus(row.decisions, 'pending')
}

// 「需處理／可匯入」由目前的對應決定推導，使用者在展開的表格改完就立即生效；
// 其餘狀態是請求生命週期，只由呼叫端設定。
function effectiveStatus(row: BatchRow): BatchRowStatus {
  if (row.status !== 'idle') return row.status
  if (!row.preview) return 'idle'
  return pendingCount(row) > 0 ? 'needsMapping' : 'ready'
}

// 月份選擇可能重複或亂序，去重後排序，讓同一台車同一個月只出現一次
const normalizedMonths = computed(() => [...new Set(selectedMonths.value ?? [])].sort())

async function buildRows() {
  if (!normalizedMonths.value.length) {
    rows.value = []
    return
  }

  loading.value = true
  try {
    const [vehiclePage, forms, months] = await Promise.all([
      listVehicles({ pageSize: 200, active: true }),
      listDriverReportForms(),
      listDriverReportImportedMonths()
    ])

    const formByVehicle = new Map(forms.map((f) => [f.vehicleId, f]))
    const importedByKey = new Map(months.map((m) => [`${m.formId}::${m.yearMonth}`, m]))
    const previous = new Map(rows.value.map((r) => [r.key, r]))

    rows.value = vehiclePage.data.flatMap((vehicle) =>
      normalizedMonths.value.map((yearMonth) => {
        const key = `${vehicle.id}::${yearMonth}`
        const kept = previous.get(key)
        const form = formByVehicle.get(vehicle.id)
        const formId = kept?.formId || form?.id || ''
        const imported = formId ? importedByKey.get(`${formId}::${yearMonth}`) : undefined

        // 保留使用者已選的檔案與試算結果，但已匯入統計一律採用剛取回的值：
        // 沿用舊快照會讓別人剛匯入的月份被判定為「尚未匯入」而略過覆蓋確認
        if (kept) {
          kept.formId = formId
          kept.importedCount = imported?.submissionCount ?? 0
          kept.lastImportedAt = imported?.lastImportedAt ?? ''
          return kept
        }

        return {
          key,
          vehicleId: vehicle.id,
          vehicleName: vehicle.displayName,
          formId,
          yearMonth,
          importedCount: imported?.submissionCount ?? 0,
          lastImportedAt: imported?.lastImportedAt ?? '',
          file: null,
          status: 'idle' as BatchRowStatus,
          preview: null,
          decisions: {},
          message: ''
        }
      })
    )
  } finally {
    loading.value = false
  }
}

function onFileChange(row: BatchRow, file: UploadFile) {
  row.file = file.raw ?? null
  row.preview = null
  row.decisions = {}
  row.message = ''
  row.status = 'idle'
}

function toggleExpand(row: BatchRow) {
  expandedKeys.value = expandedKeys.value.includes(row.key)
    ? expandedKeys.value.filter((k) => k !== row.key)
    : [...expandedKeys.value, row.key]
}

function onExpandChange(_row: unknown, expanded: unknown) {
  if (!Array.isArray(expanded)) return
  expandedKeys.value = (expanded as BatchRow[]).map((r) => r.key)
}

// el-table 的 slot 僅提供泛型 DefaultRow，統一在此轉回本頁的列型別
function asRow(row: unknown): BatchRow {
  return row as BatchRow
}

// 同一台車的多個月份會共用一次建表：一台車只能有一份匯報表，各列各自建會撞唯一索引
const formCreationByVehicle = new Map<string, Promise<string>>()

// 沒有匯報表的車輛先建表再匯入；名稱沿用清單頁的「{車輛}接送匯報」慣例
async function ensureForm(row: BatchRow): Promise<string> {
  if (row.formId) return row.formId

  let creating = formCreationByVehicle.get(row.vehicleId)
  if (!creating) {
    creating = createDriverReportForm({
      vehicleId: row.vehicleId,
      title: `${row.vehicleName}接送匯報`
    }).then((created) => created.id)
    // 建表失敗不留下已 reject 的 promise，否則同一台車的其他列永遠拿到同一個舊錯誤
    creating.catch(() => formCreationByVehicle.delete(row.vehicleId))
    formCreationByVehicle.set(row.vehicleId, creating)
  }

  const formId = await creating
  rows.value.forEach((r) => {
    if (r.vehicleId === row.vehicleId) r.formId = formId
  })
  return formId
}

async function analyzeRow(row: BatchRow) {
  if (!row.file) return
  row.status = 'analyzing'
  row.message = ''
  const hadForm = Boolean(row.formId)
  try {
    const formId = await ensureForm(row)
    // 建表可能撞上別處剛建好的同一台車而拿回既有那份，它可能已經有這個月的資料
    // 重讀失敗不該讓這一列變成試算失敗，最多是覆蓋提示沿用舊值
    if (!hadForm) await refreshImportedMonths().catch(() => undefined)
    const preview = await dryRunImportDriverReport(formId, row.file, row.yearMonth)
    row.preview = preview
    row.decisions = createColumnDecisions(preview.columns)
    row.status = 'idle'
    // 未對應的欄位不會寫入搭乘紀錄，直接展開讓使用者就地處理
    if (pendingCount(row) > 0 && !expandedKeys.value.includes(row.key)) {
      expandedKeys.value = [...expandedKeys.value, row.key]
    }
  } catch (error) {
    row.preview = null
    row.decisions = {}
    row.status = 'failed'
    row.message = rowErrorMessage(error)
  }
}

async function commitRow(row: BatchRow) {
  // 排在並發佇列後面的列，可能在等待期間被使用者改回未對應；送出前重新確認一次
  if (pendingCount(row) > 0) {
    row.status = 'failed'
    row.message = '仍有欄位未對應，未寫入'
    return
  }
  if (!row.file || !row.formId) {
    row.status = 'failed'
    row.message = '缺少上傳檔案或匯報表，未寫入'
    return
  }
  row.status = 'importing'
  try {
    const result = await commitImportDriverReport(
      row.formId,
      row.file,
      toColumnDecisionPayload(row.decisions),
      row.yearMonth
    )
    row.status = 'imported'
    row.importedCount = result.importedRows
    row.message = ''
  } catch (error) {
    row.status = 'failed'
    row.message = rowErrorMessage(error)
  }
}

async function analyzeAll() {
  analyzing.value = true
  try {
    await runWithLimit(analyzableRows.value, analyzeRow)
  } finally {
    analyzing.value = false
  }
}

async function confirmAndImport() {
  const targets = importableRows.value
  if (!targets.length) return

  const overwrites = targets.filter((r) => r.importedCount > 0)
  if (overwrites.length) {
    const detail = overwrites
      .map((r) => `${r.vehicleName} ${r.yearMonth}：既有 ${r.importedCount} 天`)
      .join('\n')
    try {
      await ElMessageBox.confirm(
        `以下 ${overwrites.length} 列該月已匯入過，這次會整月覆蓋：\n${detail}`,
        '確認覆蓋已匯入的月份',
        { type: 'warning', confirmButtonText: '確認送出', cancelButtonText: '取消' }
      )
    } catch {
      return
    }
  }

  importing.value = true
  importSummary.value = null
  try {
    await runWithLimit(targets, commitRow)
    const succeeded = targets.filter((r) => r.status === 'imported').length
    importSummary.value = { succeeded, failed: targets.length - succeeded }
    if (succeeded) ElMessage.success(`已匯入 ${succeeded} 列接送匯報`)
    // 重讀失敗不推翻已完成的匯入，只是下一次重傳的覆蓋提示會沿用舊值
    await refreshImportedMonths().catch(() => undefined)
  } finally {
    importing.value = false
  }
}

// 匯入後重讀已匯入月份，讓後續重傳仍能正確跳出覆蓋確認
async function refreshImportedMonths() {
  const months = await listDriverReportImportedMonths()
  const importedByKey = new Map(months.map((m) => [`${m.formId}::${m.yearMonth}`, m]))
  rows.value.forEach((row) => {
    const imported = row.formId ? importedByKey.get(`${row.formId}::${row.yearMonth}`) : undefined
    row.importedCount = imported?.submissionCount ?? 0
    row.lastImportedAt = imported?.lastImportedAt ?? ''
  })
}

// runWithLimit 以固定並發上限逐列送出；單列失敗由各自的 handler 記錄，不中斷其他列
async function runWithLimit<T>(items: T[], handler: (item: T) => Promise<void>) {
  const queue = [...items]
  const workers = Array.from({ length: Math.min(MAX_CONCURRENT_REQUESTS, queue.length) }, async () => {
    for (let item = queue.shift(); item !== undefined; item = queue.shift()) {
      // 逐列獨立：handler 內部已各自處理錯誤，這裡再兜一層避免任何漏網的例外
      // 讓整個 worker 提前結束、把佇列剩下的列留在原狀
      try {
        await handler(item)
      } catch {
        // 已由 analyzeRow／commitRow 寫回該列狀態
      }
    }
  })
  await Promise.all(workers)
}

// API client 已用通知呈現錯誤；這裡另外取出原因寫回該列，讓失敗停在它自己那一行
function rowErrorMessage(error: unknown): string {
  const detail = (error as { response?: { data?: { error?: { details?: Array<{ reason: string }>; message?: string } } } })
    ?.response?.data?.error
  if (detail?.details?.length) return detail.details.map((d) => d.reason).join('；')
  return detail?.message || '匯入失敗，請確認檔案內容'
}

watch(normalizedMonths, (months) => {
  const next = months.join(',')
  if ((route.query.months ?? '') !== next) {
    router.replace({ query: { ...route.query, months: next || undefined } })
  }
  // 讀取失敗已由 API client 以通知呈現，這裡吞掉 rejection 讓表格停在空狀態
  buildRows().catch(() => undefined)
}, { immediate: true })

onMounted(async () => {
  const res = await listCases({ pageSize: 200 })
  cases.value = res.data
})
</script>

<style scoped>
.batch-import-view {
  display: flex;
  flex-direction: column;
  gap: var(--app-space-4);
}

.month-field {
  display: flex;
  align-items: center;
  gap: var(--app-space-2);
}

.month-label {
  font-size: var(--app-font-md);
  color: var(--app-text-secondary);
}

.row-detail {
  padding: var(--app-space-2) var(--app-space-4) var(--app-space-3);
}

.detail-title {
  margin: 0 0 var(--app-space-2);
  font-size: var(--app-font-sm);
  font-weight: 500;
}

.file-cell {
  display: flex;
  align-items: center;
  gap: var(--app-space-2);
  flex-wrap: wrap;
}

.file-name {
  font-size: var(--app-font-xs);
  color: var(--app-text-secondary);
  word-break: break-all;
}

.ml-1 {
  margin-left: var(--app-space-1);
}

.small {
  font-size: var(--app-font-xs);
}

.text-danger {
  color: var(--app-status-danger-fg);
}

.text-warning {
  color: var(--app-status-warning-fg);
}

.text-muted {
  color: var(--app-text-muted);
}

.empty-state {
  padding: var(--app-space-6) 0;
}

.result-list {
  margin-top: var(--app-space-3);
  padding: var(--app-space-2) var(--app-space-3);
  background-color: var(--app-status-info-bg);
  border-radius: var(--app-radius-xs);
  font-size: var(--app-font-sm);
}

.result-title {
  margin: 0;
  font-size: var(--app-font-md);
  font-weight: 600;
}

.result-list ul {
  padding-left: var(--app-space-4);
  margin: var(--app-space-1) 0 0;
}
</style>
