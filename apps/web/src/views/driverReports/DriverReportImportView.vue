<template>
  <div class="driver-report-import-view">
    <el-tabs v-model="activeTab" type="border-card" class="import-tabs" @tab-change="handleTabChange">
      <el-tab-pane label="批次上傳" name="upload">
        <div class="upload-stack">
          <PageHeader
            title="批次上傳接送匯報"
            description="選擇多個 .xlsx 檔案，系統會依檔名自動比對車輛、依內容自動判斷涵蓋月份，解析完成後自動匯入。有系統推薦個案的欄位會自動套用，完全找不到對應個案的欄位會留在「待維護資料」頁籤，稍後逐一連結既有個案或建立新個案。"
          />

          <el-upload
            drag
            multiple
            :auto-upload="false"
            :show-file-list="false"
            accept=".xlsx"
            class="drop-zone"
            :disabled="dropDisabled"
            :on-change="(file: UploadFile) => onFileChange(file)"
          >
            <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
            <!-- 拖放區本身已是 role="button" 的可聚焦控制項，CTA 只做視覺呈現、
                 不另外包 el-button，避免同一個動作出現兩個 Tab 停留點 -->
            <span class="drop-cta">{{ ctaLabel }}</span>
            <p class="drop-hint">或把 .xlsx 匯報檔拖曳到這一塊</p>
            <p class="drop-hint-sub">檔名建議包含車輛名稱（例如「竹南2車 (回覆).xlsx」）</p>
          </el-upload>

          <el-alert
            v-if="contextLoadFailed"
            type="error"
            show-icon
            :closable="false"
            title="車輛與匯報表資料載入失敗，請重新整理頁面再試"
          />

          <el-alert v-else-if="hasNoVehicles" type="warning" show-icon :closable="false" title="尚未建立任何車輛">
            <template #default>
              <p class="no-vehicles-hint">批次上傳需要先有車輛才能比對匯報表，請先新增車輛。</p>
              <el-button type="warning" size="small" plain @click="router.push({ name: 'VehicleList' })">
                前往車輛管理
              </el-button>
            </template>
          </el-alert>

          <el-alert
            v-if="overlapRows.length"
            type="warning"
            show-icon
            :closable="false"
            title="以下車輛涵蓋的月份已有資料，這次會整月覆蓋"
            class="overlap-alert"
          >
            <template #default>
              <ul class="overlap-list">
                <li v-for="row in overlapRows" :key="row.key">
                  {{ row.vehicleName }}：{{ row.overlapMonths.join('、') }}
                </li>
              </ul>
              <el-checkbox v-model="overlapAcknowledged">我已確認風險，仍要覆蓋以上月份的既有資料</el-checkbox>
            </template>
          </el-alert>

          <div v-if="summary" class="result-banner" role="status">
            成功 {{ summary.succeeded }} 個檔案、共 {{ summary.importedDays }} 天，失敗 {{ summary.failed }} 個檔案
            <template v-if="summary.pendingColumns > 0">
              ，{{ summary.pendingColumns }} 個欄位找不到對應個案，已進入待維護資料
            </template>
          </div>

          <el-empty v-if="!rows.length" description="尚未選擇任何檔案" />

          <section v-else class="file-panel">
            <div class="file-panel-head">已選擇 {{ rows.length }} 個檔案</div>

            <el-table :data="rows" row-key="key" table-layout="auto" border class="file-table">
              <el-table-column label="檔案名稱" min-width="240" class-name="file-name-col">
                <template #default="{ row }">
                  <span class="file-name">{{ row.file.name }}</span>
                </template>
              </el-table-column>

              <el-table-column label="車輛" min-width="200" class-name="vehicle-col">
                <template #default="{ row }">
                  <el-select
                    v-if="row.status === 'needsVehicle'"
                    :model-value="row.vehicleId || undefined"
                    placeholder="選擇車輛"
                    filterable
                    size="small"
                    class="vehicle-select"
                    :aria-label="`${row.file.name} 選擇車輛`"
                    @change="(vehicleId: string) => onVehiclePicked(row as BatchFileRow, vehicleId)"
                  >
                    <el-option v-for="v in vehicles" :key="v.id" :label="v.displayName" :value="v.id" />
                  </el-select>
                  <span v-else class="cell-value">{{ row.vehicleName || '-' }}</span>
                </template>
              </el-table-column>

              <el-table-column label="涵蓋月份" min-width="170" class-name="months-col">
                <template #default="{ row }">
                  <span v-if="row.months.length" class="cell-value">{{ row.months.join('、') }}</span>
                  <span v-else-if="row.status === 'analyzing'" class="cell-value text-muted">解析中…</span>
                  <span v-else class="cell-value text-muted">-</span>
                </template>
              </el-table-column>

              <el-table-column label="狀態" min-width="110" class-name="status-col">
                <template #default="{ row }">
                  <StatusTag :status="row.status" preset="driverReportImportStatus" variant="chip" />
                </template>
              </el-table-column>

              <el-table-column label="說明" min-width="260" class-name="detail-col">
                <template #default="{ row }">
                  <div class="detail-cell">
                    <div class="detail-summary-line">
                      <span v-if="row.status === 'failed'" class="text-danger truncate-text" :title="row.message">
                        {{ row.message }}
                      </span>
                      <template v-else-if="row.status === 'done'">
                        <span v-if="row.importedCount > 0" class="text-regular">
                          可匯入 {{ row.importedCount }} 天
                          <span v-if="row.pendingColumnCount > 0" class="text-warning">
                            · {{ row.pendingColumnCount }} 欄待維護
                          </span>
                        </span>
                        <span v-else-if="row.pendingColumnCount > 0" class="text-warning">
                          已建立待維護欄位
                        </span>
                        <span v-else class="text-muted">沒有可寫入的搭乘資料</span>
                      </template>
                      <span v-else-if="row.overlapMonths.length" class="text-warning truncate-text" :title="`將整月覆蓋 ${row.overlapMonths.join('、')} 既有資料`">
                        將整月覆蓋 {{ row.overlapMonths.join('、') }} 既有資料
                      </span>
                      <span v-else-if="row.status === 'analyzing'" class="text-muted">解析中…</span>
                      <span v-else-if="row.status === 'needsVehicle'" class="text-warning">請先選擇對應車輛</span>
                      <span v-else class="text-muted">-</span>
                    </div>

                    <el-button
                      v-if="row.issues.length || row.status === 'failed' || row.overlapMonths.length"
                      size="small"
                      type="primary"
                      link
                      class="view-detail-btn"
                      @click="openDetailDialog(row as BatchFileRow)"
                    >
                      <el-icon class="btn-icon"><Document /></el-icon>
                      檢視說明<template v-if="row.issues.length">（{{ row.issues.length }}）</template>
                    </el-button>
                  </div>
                </template>
              </el-table-column>

              <el-table-column label="操作" width="100" align="center" fixed="right" class-name="action-col">
                <template #default="{ row }">
                  <TableRowActions>
                    <el-button link type="danger" size="small" :disabled="running" @click="removeRow(row as BatchFileRow)">
                      移除
                    </el-button>
                  </TableRowActions>
                </template>
              </el-table-column>
            </el-table>
          </section>
        </div>
      </el-tab-pane>

      <el-tab-pane name="pending">
        <template #label>
          <span>待維護資料{{ pendingColumns.length ? `（${pendingColumns.length}）` : '' }}</span>
        </template>

        <PageHeader
          title="待維護資料"
          description="這些欄位在匯入時完全比對不到既有個案，請連結既有個案或建立新個案；建立時會自動帶入原始欄位解析出的姓名。"
        />

        <el-empty v-if="!pendingLoading && pendingColumns.length === 0" description="目前沒有待維護的欄位" />

        <el-table v-else :data="pendingColumns" v-loading="pendingLoading" max-height="600" border>
          <el-table-column label="車輛／匯報表" min-width="180">
            <template #default="{ row }">
              <div>{{ row.vehicleName }}</div>
              <div class="text-secondary small">{{ row.formTitle }}</div>
            </template>
          </el-table-column>

          <el-table-column label="原始欄位名稱" min-width="220">
            <template #default="{ row }">
              <div class="raw-name">{{ row.columnHeader }}</div>
              <el-tag v-if="row.suggestionScore" size="small" type="warning">
                系統推薦：{{ row.suggestedCaseName || '無相符個案' }}
                (信心度: {{ (row.suggestionScore * 100).toFixed(0) }}%)
              </el-tag>
            </template>
          </el-table-column>

          <!-- 趟次選項文字含「第 X 趟 (去程／回程)」7 個字，120px 會被逼縮寫成省略號，
               故加寬到 150px 讓文字完全展開；個案選單同步收斂為 170px 維持欄位總寬平衡 -->
          <el-table-column label="連結既有個案" min-width="340">
            <template #default="{ row }">
              <div class="target-binding-box">
                <el-select v-model="row.editCaseId" placeholder="搜尋個案" filterable clearable style="width: 170px">
                  <el-option v-for="c in cases" :key="c.id" :label="`${c.name}${c.code ? ` (${c.code})` : ''}`" :value="c.id" />
                </el-select>
                <el-select v-model="row.editLegSeq" placeholder="趟次" style="width: 150px">
                  <el-option v-for="leg in LEG_SEQ_OPTIONS" :key="leg.value" :value="leg.value" :label="leg.label" />
                </el-select>
              </div>
            </template>
          </el-table-column>

          <!-- 「確認綁定」「新增個案」「略過此欄」三顆各 4 字操作按鈕併排，
               標準 220px 欄寬會被逼超出版面，故加寬到 300px -->
          <el-table-column label="操作" width="300" align="center" fixed="right">
            <template #default="{ row }">
              <TableRowActions>
                <el-button link type="primary" size="small" :disabled="!row.editCaseId" @click="handleBind(row)">
                  確認綁定
                </el-button>
                <el-button link type="primary" size="small" @click="openQuickCreateCase(row)">
                  新增個案
                </el-button>
                <el-button link type="primary" size="small" @click="handleIgnore(row)">
                  略過此欄
                </el-button>
              </TableRowActions>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- 新增個案並直接綁定：帶入匯報表原始欄位解析出的姓名，補完趟次即可送出 -->
    <el-dialog
      v-model="quickCreateVisible"
      title="新增個案並綁定"
      width="min(420px, calc(100vw - 32px))"
      destroy-on-close
    >
      <el-form label-width="90px">
        <el-form-item label="個案姓名" required>
          <el-input v-model="quickCreateForm.name" placeholder="個案姓名" />
        </el-form-item>
        <el-form-item label="趟次">
          <el-select v-model="quickCreateForm.legSeq" style="width: 100%">
            <el-option v-for="leg in LEG_SEQ_OPTIONS" :key="leg.value" :value="leg.value" :label="leg.label" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <DialogFooter
          confirm-text="建立並綁定"
          :loading="quickCreating"
          :confirm-disabled="!quickCreateForm.name"
          @confirm="handleQuickCreateAndBind"
          @cancel="quickCreateVisible = false"
        />
      </template>
    </el-dialog>

    <!-- 說明與解析問題檢視彈窗（內建自訂卷軸） -->
    <el-dialog
      v-model="detailDialogVisible"
      title="檔案解析說明與問題清單"
      width="min(680px, calc(100vw - 32px))"
      destroy-on-close
      class="detail-modal"
    >
      <div v-if="selectedRowForDetail" class="detail-dialog-content">
        <div class="file-info-bar">
          <div class="info-group">
            <span class="info-label">檔案名稱：</span>
            <span class="info-val file-title">{{ selectedRowForDetail.file.name }}</span>
          </div>
          <div class="info-row">
            <div class="info-item">
              <span class="info-label">對應車輛：</span>
              <span class="info-val">{{ selectedRowForDetail.vehicleName || '未指定' }}</span>
            </div>
            <div class="info-item" v-if="selectedRowForDetail.months.length">
              <span class="info-label">涵蓋月份：</span>
              <span class="info-val">{{ selectedRowForDetail.months.join('、') }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">目前狀態：</span>
              <StatusTag :status="selectedRowForDetail.status" preset="driverReportImportStatus" variant="chip" />
            </div>
          </div>
        </div>

        <!-- 錯誤告警區 -->
        <el-alert
          v-if="selectedRowForDetail.status === 'failed' && selectedRowForDetail.message"
          :title="selectedRowForDetail.message"
          type="error"
          show-icon
          :closable="false"
          class="dialog-alert"
        />

        <!-- 重複覆蓋提示 -->
        <el-alert
          v-if="selectedRowForDetail.overlapMonths.length"
          :title="`匯入時將整月覆蓋 ${selectedRowForDetail.overlapMonths.join('、')} 之既有搭乘紀錄`"
          type="warning"
          show-icon
          :closable="false"
          class="dialog-alert"
        />

        <!-- 說明與解析問題清單（具備卷軸設計） -->
        <div class="issues-box">
          <div class="issues-header">
            <span class="issues-title">詳細說明與提醒項目</span>
            <el-tag v-if="selectedRowForDetail.issues.length" size="small" type="info" round effect="light">
              共 {{ selectedRowForDetail.issues.length }} 項
            </el-tag>
          </div>

          <div class="issues-scroll-body">
            <div v-if="selectedRowForDetail.issues.length === 0" class="empty-issues text-muted">
              無其他特殊提醒或問題項目
            </div>
            <div
              v-for="(issue, index) in selectedRowForDetail.issues"
              :key="`${issue.level}-${index}-${issue.message}`"
              :class="['issue-row', `is-${issue.level}`]"
            >
              <el-tag
                size="small"
                :type="issue.level === 'error' ? 'danger' : 'warning'"
                effect="plain"
                class="issue-level-tag"
              >
                {{ issue.level === 'error' ? '錯誤' : '提醒' }}
              </el-tag>
              <span class="issue-text">{{ issue.message }}</span>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="dialog-action-footer">
          <el-button type="primary" @click="detailDialogVisible = false">關閉</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { UploadFilled, Document } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type UploadFile } from 'element-plus'
import { useRouter } from 'vue-router'
import {
  createDriverReportForm,
  commitImportDriverReport,
  dryRunImportDriverReport,
  listDriverReportColumns,
  listDriverReportForms,
  listDriverReportImportedMonths,
  updateColumnMapping
} from '@/api/driverReports'
import { listCases, createCase } from '@/api/cases'
import { listVehicles } from '@/api/masters'
import PageHeader from '@/components/PageHeader.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import DialogFooter from '@/components/DialogFooter.vue'
import StatusTag from '@/components/StatusTag.vue'
import { resolveErrorMessage } from '@/api/errorCodes'
import { toColumnDecisionPayload, type ColumnDecisionMap } from './columnDecisions'
import { LEG_SEQ_OPTIONS } from './legOptions'
import type {
  CaseDTO,
  DriverReportColumnDTO,
  DriverReportFormDTO,
  DriverReportImportedMonthDTO,
  DriverReportPreviewDTO,
  VehicleDTO
} from '@/types/api'

type EditableColumn = DriverReportColumnDTO & { editCaseId?: string; editLegSeq?: number }

const router = useRouter()
const activeTab = ref<'upload' | 'pending'>('upload')

function apiErrorCode(error: unknown): string | undefined {
  return (error as { response?: { data?: { error?: { code?: string } } } })?.response?.data?.error?.code
}

// ---- 批次上傳 ----

type RowStatus = 'needsVehicle' | 'queued' | 'analyzing' | 'processing' | 'done' | 'failed'
type RowIssue = { level: 'error' | 'warning'; message: string }

const IMPORT_FIELD_LABELS: Record<string, string> = {
  file: '檔案',
  columnDecisions: '欄位對應',
  民國日期: '民國日期',
  駕駛人: '駕駛人',
  備註: '備註'
}

interface BatchFileRow {
  key: string
  file: File
  vehicleId: string
  vehicleName: string
  formId: string
  status: RowStatus
  months: string[]
  overlapMonths: string[]
  importedCount: number
  pendingColumnCount: number
  message: string
  issues: RowIssue[]
}

const MAX_CONCURRENT = 3

const running = ref(false)
const contextLoading = ref(true)
const contextLoadFailed = ref(false)
const rows = ref<BatchFileRow[]>([])
const vehicles = ref<VehicleDTO[]>([])
const forms = ref<DriverReportFormDTO[]>([])
const importedMonths = ref<DriverReportImportedMonthDTO[]>([])
const summary = ref<{ succeeded: number; failed: number; importedDays: number; pendingColumns: number } | null>(null)
const overlapAcknowledged = ref(false)
// 排隊中 + 執行中的解析數，供自動匯入判斷「整批都解析完了沒」
const analyzePending = ref(0)

const detailDialogVisible = ref(false)
const selectedRowForDetail = ref<BatchFileRow | null>(null)

function openDetailDialog(row: BatchFileRow) {
  selectedRowForDetail.value = row
  detailDialogVisible.value = true
}

let rowSeq = 0

// 拖放區停用時要說明原因，否則使用者只看得到一塊按不動的區域
const ctaLabel = computed(() => {
  if (contextLoadFailed.value) return '車輛資料載入失敗'
  if (contextLoading.value) return '正在載入車輛資料…'
  if (running.value) return '正在匯入…'
  if (vehicles.value.length === 0) return '尚未建立車輛'
  return '選擇 .xlsx 檔案'
})
const dropDisabled = computed(
  () => running.value || contextLoading.value || contextLoadFailed.value || vehicles.value.length === 0
)
// 資料真的載入成功、只是車輛清單為空時，要跟「還在載入」「載入失敗」分開提示，
// 否則使用者只會看到一塊選不出任何選項的下拉選單、不知道該去哪裡處理
const hasNoVehicles = computed(
  () => !contextLoading.value && !contextLoadFailed.value && vehicles.value.length === 0
)

const formByVehicle = computed(() => new Map(forms.value.map((f) => [f.vehicleId, f.id])))
const importedByKey = computed(
  () => new Map(importedMonths.value.map((m) => [`${m.formId}::${m.yearMonth}`, m]))
)
// 已解析完成、還沒匯入的檔案。已匯入或失敗的列不再納入，否則自動匯入會反覆重跑同一批
const pendingRows = computed(() => rows.value.filter((r) => r.vehicleId && r.status === 'queued'))
const overlapRows = computed(() => pendingRows.value.filter((r) => r.overlapMonths.length > 0))
const canImport = computed(
  () =>
    !running.value &&
    analyzePending.value === 0 &&
    pendingRows.value.length > 0 &&
    (overlapRows.value.length === 0 || overlapAcknowledged.value)
)

// 沒有送出按鈕：整批解析完就自動匯入。等 analyzePending 歸零才觸發，讓一次拖入的多個檔案併成一批；
// 命中既有月份時停在這裡等使用者勾選確認風險，勾完 canImport 再次轉真才續跑。
watch(canImport, (ready) => {
  if (ready) void runImport()
})

function stripExtension(name: string): string {
  return name.replace(/\.[^./\\]+$/, '')
}

function normalizeForMatch(text: string): string {
  return text.replace(/[\s()（）]/g, '')
}

// detectVehicle 以檔名內容比對車輛顯示名稱；唯一命中才視為自動判斷成功，
// 沒命中或命中多輛都交由使用者手動選擇，避免猜錯車輛覆蓋錯資料。
function detectVehicle(fileName: string, list: VehicleDTO[]): VehicleDTO | 'ambiguous' | null {
  const normalized = normalizeForMatch(stripExtension(fileName))
  const matches = list.filter((v) => normalized.includes(normalizeForMatch(v.displayName)))
  if (matches.length === 1) return matches[0]
  if (matches.length > 1) return 'ambiguous'
  return null
}

function onFileChange(file: UploadFile) {
  const raw = file.raw
  if (!raw) return
  if (rows.value.some((r) => r.file.name === raw.name && r.file.size === raw.size)) return

  const detected = detectVehicle(raw.name, vehicles.value)
  const vehicle = detected && detected !== 'ambiguous' ? detected : null
  const row: BatchFileRow = {
    key: `row_${++rowSeq}`,
    file: raw,
    vehicleId: vehicle?.id ?? '',
    vehicleName: vehicle?.displayName ?? '',
    formId: vehicle ? formByVehicle.value.get(vehicle.id) ?? '' : '',
    status: vehicle ? 'queued' : 'needsVehicle',
    months: [],
    overlapMonths: [],
    importedCount: 0,
    pendingColumnCount: 0,
    message: '',
    issues: []
  }
  rows.value.push(row)
  if (vehicle) enqueueAnalyze(rows.value[rows.value.length - 1])
}

function onVehiclePicked(row: BatchFileRow, vehicleId: string) {
  const vehicle = vehicles.value.find((v) => v.id === vehicleId)
  if (!vehicle) return
  row.vehicleId = vehicle.id
  row.vehicleName = vehicle.displayName
  row.formId = formByVehicle.value.get(vehicle.id) ?? ''
  row.status = 'queued'
  enqueueAnalyze(row)
}

function removeRow(row: BatchFileRow) {
  rows.value = rows.value.filter((r) => r.key !== row.key)
}

const formCreationByVehicle = new Map<string, Promise<string>>()

async function ensureForm(row: BatchFileRow): Promise<string> {
  const known = formByVehicle.value.get(row.vehicleId)
  if (known) {
    row.formId = known
    return known
  }
  let creating = formCreationByVehicle.get(row.vehicleId)
  if (!creating) {
    creating = createDriverReportForm({
      vehicleId: row.vehicleId,
      title: `${row.vehicleName}接送匯報`
    }).then((created) => {
      forms.value = [...forms.value, created]
      return created.id
    })
    creating.catch(() => formCreationByVehicle.delete(row.vehicleId))
    formCreationByVehicle.set(row.vehicleId, creating)
  }
  const formId = await creating
  rows.value.forEach((r) => {
    if (r.vehicleId === row.vehicleId) r.formId = formId
  })
  return formId
}

function computeMonths(previewRows: Array<{ errorMessage?: string; serviceDate: string }>): string[] {
  const months = new Set<string>()
  for (const r of previewRows) {
    if (!r.errorMessage && r.serviceDate) months.add(r.serviceDate.slice(0, 7))
  }
  return [...months].sort()
}

function collectPreviewIssues(preview: Pick<DriverReportPreviewDTO, 'errors' | 'warnings'>): RowIssue[] {
  return [
    ...preview.errors.map((item) => ({ level: 'error' as const, message: formatPreviewIssue(item) })),
    ...preview.warnings.map((item) => ({ level: 'warning' as const, message: formatPreviewIssue(item) }))
  ]
}

function formatPreviewIssue(item: { rowIndex: number; field?: string; message: string }): string {
  const field = item.field ? IMPORT_FIELD_LABELS[item.field] || item.field : ''
  if (item.rowIndex === 1 && (item.field === '備註' || item.field === '問題回報' || item.field === 'remark')) {
    return `【表頭欄位】${item.message}`
  }
  return `第 ${item.rowIndex} 列${field ? `【${field}】` : ''}：${item.message}`
}

// 有系統推薦個案的欄位自動視為已對應直接匯入；完全沒有推薦、比對不到個案的欄位
// 留在 pending，交由使用者稍後於「待維護資料」逐一連結既有個案或建立新個案。
function buildAutoDecisions(
  columns: Array<{
    columnHeader: string
    mappingStatus: string
    caseId?: string
    legSeq?: number
    suggestedCaseId?: string
    suggestedLegSeq?: number
  }>
): ColumnDecisionMap {
  const decisions: ColumnDecisionMap = {}
  for (const c of columns) {
    if (c.mappingStatus === 'mapped' && c.caseId && c.legSeq) {
      decisions[c.columnHeader] = { mappingStatus: 'mapped', caseId: c.caseId, legSeq: c.legSeq }
    } else if (c.suggestedCaseId && c.suggestedLegSeq) {
      decisions[c.columnHeader] = { mappingStatus: 'mapped', caseId: c.suggestedCaseId, legSeq: c.suggestedLegSeq }
    } else {
      decisions[c.columnHeader] = { mappingStatus: 'pending' }
    }
  }
  return decisions
}

// 逐檔獨立的並發佇列：新檔案隨拖入即排入解析，不用等前一批解析完才看得到涵蓋月份
let activeAnalyses = 0
const analyzeQueue: BatchFileRow[] = []

function enqueueAnalyze(row: BatchFileRow) {
  if (analyzeQueue.includes(row)) return
  analyzeQueue.push(row)
  analyzePending.value++
  pumpAnalyzeQueue()
}

function pumpAnalyzeQueue() {
  while (activeAnalyses < MAX_CONCURRENT && analyzeQueue.length) {
    const row = analyzeQueue.shift()!
    activeAnalyses++
    analyzeRow(row).finally(() => {
      activeAnalyses--
      analyzePending.value--
      pumpAnalyzeQueue()
    })
  }
}

// analyzeRow 只做「解析出涵蓋月份與是否覆蓋既有資料」的預覽，不寫入任何資料；
// 真正的欄位對應與 commit 交給 processRow，兩者都各自 dry-run 一次，換取程式碼單純。
async function analyzeRow(row: BatchFileRow) {
  if (row.status === 'needsVehicle') return
  row.status = 'analyzing'
  try {
    const formId = await ensureForm(row)
    const preview = await dryRunImportDriverReport(formId, row.file)
    row.issues = collectPreviewIssues(preview)
    const months = computeMonths(preview.previewRows)
    row.months = months
    row.overlapMonths = months.filter((m) => importedByKey.value.has(`${formId}::${m}`))
    row.status = 'queued'
  } catch (error) {
    row.status = 'failed'
    row.message = rowErrorMessage(error)
  }
}

async function processRow(row: BatchFileRow) {
  row.status = 'processing'
  row.message = ''
  try {
    const formId = await ensureForm(row)
    const preview = await dryRunImportDriverReport(formId, row.file)
    row.issues = collectPreviewIssues(preview)
    const decisions = buildAutoDecisions(preview.columns)
    const months = computeMonths(preview.previewRows)

    if (months.length === 0) {
      row.status = 'failed'
      row.message = row.issues.length ? '檔案沒有可寫入的日期，請依下列原因修正後重新上傳' : '檔案內沒有可匯入的日期'
      return
    }

    const payload = toColumnDecisionPayload(decisions)
    let importedRows = 0
    for (const month of months) {
      const result = await commitImportDriverReport(formId, row.file, payload, month)
      importedRows += result.importedRows
      row.issues.push(
        ...result.skippedRows.flatMap((item) =>
          item.reasons.map((reason) => ({
            level: 'error' as const,
            message: `第 ${item.rowIndex} 列${item.reportDate ? `（${item.reportDate}）` : ''}：${reason}`
          }))
        ),
        ...(result.warnings ?? []).map((item) => ({ level: 'warning' as const, message: formatPreviewIssue(item) }))
      )
    }
    row.issues = row.issues.filter(
      (issue, index, all) =>
        all.findIndex((candidate) => candidate.level === issue.level && candidate.message === issue.message) === index
    )
    row.status = 'done'
    row.importedCount = importedRows
    row.pendingColumnCount = Object.values(decisions).filter((d) => d.mappingStatus === 'pending').length
  } catch (error) {
    row.status = 'failed'
    row.message = rowErrorMessage(error)
  }
}

function rowErrorMessage(error: unknown): string {
  const detail = (error as { response?: { data?: { error?: { details?: Array<{ field?: string; reason: string }>; message?: string } } } })
    ?.response?.data?.error
  if (detail?.details?.length) {
    return detail.details
      .map((d) => `${d.field ? `【${IMPORT_FIELD_LABELS[d.field] || d.field}】` : ''}${d.reason}`)
      .join('；')
  }
  const code = apiErrorCode(error)
  if (code) return resolveErrorMessage(code, '匯入失敗，請確認檔案內容')
  if (error instanceof Error && error.message) return `解析發生錯誤：${error.message}`
  return '匯入失敗，請確認檔案內容'
}

async function runWithLimit<T>(items: T[], handler: (item: T) => Promise<void>) {
  const queue = [...items]
  const workers = Array.from({ length: Math.min(MAX_CONCURRENT, queue.length) }, async () => {
    for (let item = queue.shift(); item !== undefined; item = queue.shift()) {
      try {
        await handler(item)
      } catch {
        // 已由呼叫端寫回該列狀態
      }
    }
  })
  await Promise.all(workers)
}

async function runImport() {
  const targets = pendingRows.value
  if (!targets.length || !canImport.value) return

  running.value = true
  summary.value = null
  try {
    await runWithLimit(targets, processRow)

    const succeeded = targets.filter((r) => r.status === 'done')
    const result = {
      succeeded: succeeded.length,
      failed: targets.length - succeeded.length,
      importedDays: succeeded.reduce((sum, r) => sum + r.importedCount, 0),
      pendingColumns: succeeded.reduce((sum, r) => sum + r.pendingColumnCount, 0)
    }
    summary.value = result
    importedMonths.value = await listDriverReportImportedMonths().catch(() => importedMonths.value)
    overlapAcknowledged.value = false
    if (succeeded.length) await handleUploadSuccess({ pendingColumns: result.pendingColumns })
  } finally {
    running.value = false
  }
}

// 拖曳區在資料載完前停用，避免車輛清單還是空的時就跑自動比對而誤判成「待選車輛」。
// 分頁端點（vehicles）在 0 筆結果時後端回傳 data: null，這裡一律預設空陣列，
// 避免後續 detectVehicle 等處的 .filter／.map 對 null 直接丟出未捕捉例外。
async function loadUploadContext() {
  try {
    const [vehiclePage, formList, months] = await Promise.all([
      listVehicles({ pageSize: 200, status: 'active' }),
      listDriverReportForms(),
      listDriverReportImportedMonths()
    ])
    vehicles.value = vehiclePage.data ?? []
    forms.value = formList ?? []
    importedMonths.value = months ?? []
  } catch (error) {
    contextLoadFailed.value = true
    ElMessage.error(resolveErrorMessage(apiErrorCode(error), '載入車輛與匯報表資料失敗，請重新整理頁面再試'))
  } finally {
    contextLoading.value = false
  }
}

async function loadCases() {
  try {
    const res = await listCases({ pageSize: 200 })
    cases.value = res.data ?? []
  } catch (error) {
    ElMessage.error(resolveErrorMessage(apiErrorCode(error), '載入個案清單失敗，請重新整理頁面再試'))
  }
}

// ---- 待維護資料 ----

const cases = ref<CaseDTO[]>([])
const pendingColumns = ref<EditableColumn[]>([])
const pendingLoading = ref(false)
const quickCreateVisible = ref(false)
const quickCreating = ref(false)
const quickCreateForm = ref<{ columnId: string; name: string; legSeq: number }>({
  columnId: '',
  name: '',
  legSeq: 1
})

// 切換至待維護頁籤時主動重新整理清單，確保顯示最新待處理項目
async function handleTabChange(name: string | number) {
  if (name === 'pending') {
    await fetchPending()
  }
}

// 上傳完成後若有欄位進入待維護，詢問是否直接切過去處理，比照個案管理匯入完成後的提示模式
async function handleUploadSuccess(result: { pendingColumns: number }) {
  if (result.pendingColumns === 0) return
  await fetchPending()
  try {
    await ElMessageBox.confirm(
      `本次匯入有 ${result.pendingColumns} 個欄位找不到對應個案，已列入「待維護資料」，是否立即前往處理？`,
      '批次上傳完成',
      { confirmButtonText: '前往待維護', cancelButtonText: '稍後再說', type: 'info' }
    )
    activeTab.value = 'pending'
  } catch {
    // 使用者選擇稍後再說：留在上傳結果查看
  }
}

async function fetchPending() {
  pendingLoading.value = true
  try {
    const cols = await listDriverReportColumns({ mappingStatus: 'pending' })
    pendingColumns.value = cols.map((c) => ({
      ...c,
      editCaseId: c.suggestedCaseId || '',
      editLegSeq: c.suggestedLegSeq || 1
    }))
  } catch (error) {
    ElMessage.error(resolveErrorMessage(apiErrorCode(error), '載入待維護清單失敗'))
  } finally {
    pendingLoading.value = false
  }
}

async function handleBind(row: any) {
  try {
    await updateColumnMapping(row.id, {
      caseId: row.editCaseId,
      legSeq: row.editLegSeq,
      mappingStatus: 'mapped'
    })
    ElMessage.success(`已將「${row.columnHeader}」成功綁定`)
    await fetchPending()
  } catch (error) {
    ElMessage.error(resolveErrorMessage(apiErrorCode(error), '綁定失敗'))
  }
}

async function handleIgnore(row: any) {
  try {
    await updateColumnMapping(row.id, { mappingStatus: 'ignored' })
    ElMessage.info(`已略過「${row.columnHeader}」`)
    await fetchPending()
  } catch (error) {
    ElMessage.error(resolveErrorMessage(apiErrorCode(error), '略過失敗'))
  }
}

// openQuickCreateCase 帶入匯報表原始欄位解析出的姓名，讓使用者只需補趟次即可建立並直接綁定
function openQuickCreateCase(row: any) {
  quickCreateForm.value = { columnId: row.id, name: row.cleanedName, legSeq: row.editLegSeq || 1 }
  quickCreateVisible.value = true
}

async function handleQuickCreateAndBind() {
  quickCreating.value = true
  try {
    const created = await createCase({
      name: quickCreateForm.value.name
    })
    cases.value = [...cases.value, created]
    await updateColumnMapping(quickCreateForm.value.columnId, {
      caseId: created.id,
      legSeq: quickCreateForm.value.legSeq,
      mappingStatus: 'mapped'
    })
    ElMessage.success(`已建立個案「${created.name}」並完成綁定`)
    quickCreateVisible.value = false
    await fetchPending()
  } catch (error) {
    ElMessage.error(resolveErrorMessage(apiErrorCode(error), '新增個案或綁定失敗'))
  } finally {
    quickCreating.value = false
  }
}

onMounted(() => {
  void loadCases()
  void loadUploadContext()
})
</script>

<style scoped>
.driver-report-import-view {
  display: flex;
  flex-direction: column;
}

/* 上傳頁採上下堆疊：拖放區 → 檔案清單。max-width 讓版面不撐滿整頁寬度，靠左對齊 */
.upload-stack {
  max-width: 1100px;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: var(--app-space-4);
}

.drop-zone :deep(.el-upload),
.drop-zone :deep(.el-upload-dragger) {
  width: 100%;
}

/* 整塊拖放區就是這一頁唯一的動作，原本另外擺一顆分離的「選擇檔案」按鈕，
   使用者看不出兩者是同一件事。改成把 CTA 收進區塊中央，並把邊框加深、
   補上 hover 與 focus 回饋，讓這塊本身就看得出來可以按 */
.drop-zone :deep(.el-upload-dragger) {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: var(--app-space-8) var(--app-space-4);
  border-width: 2px;
  border-color: var(--app-primary);
  border-color: color-mix(in srgb, var(--app-primary) 32%, var(--app-surface));
  background: var(--app-surface);
  transition: border-color 0.15s ease, background-color 0.15s ease;
}

.drop-zone :deep(.el-upload:not(.is-disabled) .el-upload-dragger:hover),
.drop-zone :deep(.el-upload:not(.is-disabled) .el-upload-dragger.is-dragover) {
  border-color: var(--app-primary);
  background: var(--app-primary-light);
}

.drop-zone :deep(.el-upload:focus-visible) {
  outline: 2px solid var(--app-primary);
  outline-offset: 2px;
  border-radius: var(--app-radius-md);
}

.drop-zone :deep(.el-icon--upload) {
  margin-bottom: var(--app-space-2);
  color: var(--app-primary);
}

/* 按鈕外觀但不是可聚焦元素：可點的控制項是外層拖放區，避免同一動作有兩個 Tab 停留點 */
.drop-cta {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 220px;
  height: 46px;
  padding: 0 var(--app-space-6);
  border-radius: var(--app-radius-sm);
  background: var(--app-primary);
  color: #ffffff;
  font-size: var(--app-font-lg);
  font-weight: 600;
  box-shadow: var(--app-shadow-sm);
  transition: background-color 0.15s ease, transform 0.15s ease;
}

.drop-zone :deep(.el-upload:not(.is-disabled) .el-upload-dragger:hover) .drop-cta {
  background: var(--app-primary-dark);
  transform: translateY(-1px);
}

.drop-zone :deep(.el-upload.is-disabled) .drop-cta {
  background: var(--app-text-muted);
  box-shadow: none;
}

.drop-hint {
  margin: var(--app-space-3) 0 0;
  font-size: var(--app-font-md);
  color: var(--app-text-regular);
}

.drop-hint-sub {
  margin: var(--app-space-1) 0 0;
  font-size: var(--app-font-xs);
  color: var(--app-text-muted);
}

@media (prefers-reduced-motion: reduce) {
  .drop-zone :deep(.el-upload-dragger),
  .drop-cta {
    transition: none;
  }

  .drop-zone :deep(.el-upload:not(.is-disabled) .el-upload-dragger:hover) .drop-cta {
    transform: none;
  }
}

.overlap-alert {
  margin: 0;
}

.no-vehicles-hint {
  margin: 0 0 var(--app-space-2);
}

.overlap-list {
  margin: 0 0 var(--app-space-2);
  padding-left: var(--app-space-4);
}

.result-banner {
  padding: var(--app-space-3) var(--app-space-4);
  background: var(--app-status-info-bg);
  color: var(--app-status-info-fg);
  border-radius: var(--app-radius-sm);
  font-size: var(--app-font-sm);
  font-weight: 500;
}

/* 這張表格不走 DataTablePage，容器自己補水平捲動，避免內容超寬時撐破整頁 */
.file-panel {
  border: 1px solid var(--app-border-color);
  border-radius: var(--app-radius-md);
  background: var(--app-surface);
  box-shadow: var(--app-shadow-sm);
  overflow-x: auto;
}

.file-panel-head {
  padding: var(--app-space-3) var(--app-space-4);
  font-size: var(--app-font-sm);
  font-weight: 600;
  color: var(--app-text-primary);
  border-bottom: 1px solid var(--app-border-color);
}

/* table-layout="auto" 下每一欄都要自己鎖 nowrap 與 min-width，否則欄寬吃緊時會逐字換行 */
.file-table :deep(.file-name-col .cell) {
  white-space: nowrap;
  min-width: 240px;
}

.file-table :deep(.vehicle-col .cell) {
  white-space: nowrap;
  min-width: 200px;
}

.file-table :deep(.months-col .cell) {
  white-space: nowrap;
  min-width: 170px;
}

.file-table :deep(.status-col .cell) {
  white-space: nowrap;
  min-width: 110px;
}

.file-table :deep(.detail-col .cell) {
  min-width: 280px;
}

.detail-cell {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 260px;
}

.detail-summary-line {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}

.truncate-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: inline-block;
  max-width: 100%;
  vertical-align: bottom;
}

.view-detail-btn {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 0 4px;
  font-size: 13px;
  font-weight: 500;
}

.view-detail-btn .btn-icon {
  font-size: 14px;
}

.file-info-bar {
  background: var(--app-bg-muted, #f8fafc);
  border: 1px solid var(--app-border-color);
  border-radius: var(--app-radius-sm, 6px);
  padding: 12px 16px;
  margin-bottom: 12px;
}

.info-group {
  margin-bottom: 8px;
  font-size: 14px;
}

.info-row {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: center;
  font-size: 13px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.info-label {
  color: var(--app-text-secondary, #64748b);
}

.info-val {
  color: var(--app-text-primary, #0f172a);
  font-weight: 500;
}

.file-title {
  font-weight: 600;
  color: var(--app-color-primary, #2563eb);
}

.dialog-alert {
  margin-bottom: 12px;
}

.issues-box {
  border: 1px solid var(--app-border-color);
  border-radius: var(--app-radius-sm, 6px);
  background: var(--app-surface, #ffffff);
  overflow: hidden;
}

.issues-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 14px;
  background: var(--app-bg-muted, #f8fafc);
  border-bottom: 1px solid var(--app-border-color);
}

.issues-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--app-text-primary, #1e293b);
}

/* 自訂滑順卷軸設計，防止長清單撐開對話框或整頁 */
.issues-scroll-body {
  max-height: 360px;
  overflow-y: auto;
  padding: 10px 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.issues-scroll-body::-webkit-scrollbar {
  width: 6px;
}

.issues-scroll-body::-webkit-scrollbar-track {
  background: transparent;
}

.issues-scroll-body::-webkit-scrollbar-thumb {
  background: var(--app-border-color-dark, #cbd5e1);
  border-radius: 4px;
}

.issues-scroll-body::-webkit-scrollbar-thumb:hover {
  background: var(--app-text-muted, #94a3b8);
}

.issue-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 4px;
  background: #ffffff;
  border: 1px solid #f1f5f9;
  font-size: 13px;
  line-height: 1.5;
}

.issue-row.is-error {
  background: #fef2f2;
  border-color: #fecaca;
  color: #991b1b;
}

.issue-row.is-warning {
  background: #fffbeb;
  border-color: #fef3c7;
  color: #92400e;
}

.issue-level-tag {
  flex-shrink: 0;
  margin-top: 1px;
}

.issue-text {
  flex: 1;
  word-break: break-all;
}

.empty-issues {
  padding: 24px;
  text-align: center;
  font-size: 13px;
}

.dialog-action-footer {
  display: flex;
  justify-content: flex-end;
}

.file-table :deep(.action-col .cell) {
  white-space: nowrap;
  min-width: 100px;
}

.cell-value {
  white-space: nowrap;
}

.file-name {
  font-weight: 500;
  color: var(--app-text-primary);
  white-space: nowrap;
}

.vehicle-select {
  width: 180px;
}

.small {
  font-size: var(--app-font-xs);
}

.text-secondary {
  color: var(--app-text-secondary);
  font-size: 13px;
}

.text-warning {
  color: var(--app-status-warning-fg);
}

.text-danger {
  color: var(--app-status-danger-fg);
}

.raw-name {
  font-weight: 500;
  margin-bottom: 4px;
}

.target-binding-box {
  display: flex;
  gap: 8px;
  align-items: center;
}
</style>
