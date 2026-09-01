<template>
  <div class="driver-report-import-view">
    <el-tabs v-model="activeTab" type="border-card" class="import-tabs" @tab-change="handleTabChange">
      <el-tab-pane label="批次上傳" name="upload">
        <div class="upload-split">
          <aside class="upload-side">
            <el-upload
              drag
              multiple
              :auto-upload="false"
              :show-file-list="false"
              accept=".xlsx"
              class="drop-zone"
              :disabled="running || contextLoading"
              :on-change="(file: UploadFile) => onFileChange(file)"
            >
              <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
              <p class="drop-hint">
                {{ contextLoading ? '正在載入車輛資料…' : '將 .xlsx 匯報檔拖曳到此，或點選選擇檔案' }}
              </p>
              <p class="drop-hint-sub">檔名建議包含車輛名稱（例如「竹南2車 (回覆).xlsx」）</p>
            </el-upload>

            <div v-if="rows.length" class="file-count-card">
              <div class="file-count-num">{{ rows.length }}</div>
              <div class="file-count-label">個檔案待處理</div>
            </div>

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

            <el-button
              type="primary"
              class="submit-btn"
              :loading="running"
              :disabled="!canImport"
              @click="runImport"
            >
              開始解析與匯入
            </el-button>
          </aside>

          <section class="upload-main">
            <PageHeader
              title="批次上傳接送匯報"
              description="拖曳或選擇多個 .xlsx 檔案，系統會依檔名自動比對車輛、依內容自動判斷涵蓋月份，解析後立即匯入。有系統推薦個案的欄位會自動套用，完全找不到對應個案的欄位會留在「待維護資料」頁籤，稍後逐一連結既有個案或建立新個案。"
            />

            <div v-if="summary" class="result-banner" role="status">
              成功 {{ summary.succeeded }} 個檔案、共 {{ summary.importedDays }} 天，失敗 {{ summary.failed }} 個檔案
              <template v-if="summary.pendingColumns > 0">
                ，{{ summary.pendingColumns }} 個欄位找不到對應個案，已進入待維護資料
              </template>
            </div>

            <el-empty v-if="!rows.length" description="尚未選擇任何檔案" />

            <div v-else class="file-cards">
              <article v-for="row in rows" :key="row.key" class="file-card">
                <div class="file-card-head">
                  <span class="file-name">{{ row.file.name }}</span>
                  <StatusTag :status="row.status" preset="driverReportImportStatus" variant="chip" />
                </div>

                <el-select
                  v-if="row.status === 'needsVehicle'"
                  :model-value="row.vehicleId || undefined"
                  placeholder="選擇車輛"
                  filterable
                  size="small"
                  :aria-label="`${row.file.name} 選擇車輛`"
                  @change="(vehicleId: string) => onVehiclePicked(row, vehicleId)"
                >
                  <el-option v-for="v in vehicles" :key="v.id" :label="v.displayName" :value="v.id" />
                </el-select>
                <div v-else class="file-card-meta">
                  <span>{{ row.vehicleName }}</span>
                  <span v-if="row.months.length">{{ row.months.join('、') }}</span>
                  <span v-else-if="row.status === 'analyzing'" class="text-muted">解析中…</span>
                </div>

                <div v-if="row.overlapMonths.length" class="file-card-overlap">
                  將整月覆蓋 {{ row.overlapMonths.join('、') }} 既有資料
                </div>

                <div v-if="row.status === 'done'" class="file-card-result">
                  可匯入 {{ row.importedCount }} 天
                  <span v-if="row.pendingColumnCount > 0" class="text-warning">
                    · {{ row.pendingColumnCount }} 欄待維護
                  </span>
                </div>
                <div v-else-if="row.status === 'failed'" class="file-card-result text-danger">{{ row.message }}</div>

                <button class="file-card-remove" type="button" :disabled="running" @click="removeRow(row)">
                  移除
                </button>
              </article>
            </div>
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

        <el-table v-else :data="pendingColumns" v-loading="pendingLoading" border>
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
                  <el-option v-for="c in cases" :key="c.id" :label="`${c.name} (${c.code})`" :value="c.id" />
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
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { UploadFilled } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type UploadFile } from 'element-plus'
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
import { toColumnDecisionPayload, type ColumnDecisionMap } from './columnDecisions'
import { LEG_SEQ_OPTIONS } from './legOptions'
import type {
  CaseDTO,
  DriverReportColumnDTO,
  DriverReportFormDTO,
  DriverReportImportedMonthDTO,
  VehicleDTO
} from '@/types/api'

type EditableColumn = DriverReportColumnDTO & { editCaseId?: string; editLegSeq?: number }

const activeTab = ref<'upload' | 'pending'>('upload')

// ---- 批次上傳 ----

type RowStatus = 'needsVehicle' | 'queued' | 'analyzing' | 'processing' | 'done' | 'failed'

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
}

const MAX_CONCURRENT = 3

const running = ref(false)
const contextLoading = ref(true)
const rows = ref<BatchFileRow[]>([])
const vehicles = ref<VehicleDTO[]>([])
const forms = ref<DriverReportFormDTO[]>([])
const importedMonths = ref<DriverReportImportedMonthDTO[]>([])
const summary = ref<{ succeeded: number; failed: number; importedDays: number; pendingColumns: number } | null>(null)
const overlapAcknowledged = ref(false)

let rowSeq = 0

const formByVehicle = computed(() => new Map(forms.value.map((f) => [f.vehicleId, f.id])))
const importedByKey = computed(
  () => new Map(importedMonths.value.map((m) => [`${m.formId}::${m.yearMonth}`, m]))
)
const processableRows = computed(() => rows.value.filter((r) => r.vehicleId && r.status !== 'processing' && r.status !== 'analyzing'))
const overlapRows = computed(() => rows.value.filter((r) => r.overlapMonths.length > 0))
const canImport = computed(
  () => !running.value && processableRows.value.length > 0 && (overlapRows.value.length === 0 || overlapAcknowledged.value)
)

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
    message: ''
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
  if (!analyzeQueue.includes(row)) analyzeQueue.push(row)
  pumpAnalyzeQueue()
}

function pumpAnalyzeQueue() {
  while (activeAnalyses < MAX_CONCURRENT && analyzeQueue.length) {
    const row = analyzeQueue.shift()!
    activeAnalyses++
    analyzeRow(row).finally(() => {
      activeAnalyses--
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
    const decisions = buildAutoDecisions(preview.columns)
    const months = computeMonths(preview.previewRows)

    if (months.length === 0) {
      row.status = 'failed'
      row.message = '檔案內沒有可匯入的日期'
      return
    }

    const payload = toColumnDecisionPayload(decisions)
    let importedRows = 0
    for (const month of months) {
      const result = await commitImportDriverReport(formId, row.file, payload, month)
      importedRows += result.importedRows
    }
    row.status = 'done'
    row.importedCount = importedRows
    row.pendingColumnCount = Object.values(decisions).filter((d) => d.mappingStatus === 'pending').length
  } catch (error) {
    row.status = 'failed'
    row.message = rowErrorMessage(error)
  }
}

function rowErrorMessage(error: unknown): string {
  const detail = (error as { response?: { data?: { error?: { details?: Array<{ reason: string }>; message?: string } } } })
    ?.response?.data?.error
  if (detail?.details?.length) return detail.details.map((d) => d.reason).join('；')
  return detail?.message || '匯入失敗，請確認檔案內容'
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
  const targets = processableRows.value
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

// 拖曳區在資料載完前停用，避免車輛清單還是空的時就跑自動比對而誤判成「待選車輛」
async function loadUploadContext() {
  try {
    const [vehiclePage, formList, months] = await Promise.all([
      listVehicles({ pageSize: 200, active: true }),
      listDriverReportForms(),
      listDriverReportImportedMonths()
    ])
    vehicles.value = vehiclePage.data
    forms.value = formList
    importedMonths.value = months
  } finally {
    contextLoading.value = false
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

// 待維護頁籤首次切入時才拉取清單，避免上傳頁多打一次 API
async function handleTabChange(name: string | number) {
  if (name === 'pending' && pendingColumns.value.length === 0 && !pendingLoading.value) {
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
  } finally {
    pendingLoading.value = false
  }
}

async function handleBind(row: any) {
  await updateColumnMapping(row.id, {
    caseId: row.editCaseId,
    legSeq: row.editLegSeq,
    mappingStatus: 'mapped'
  })
  ElMessage.success(`已將「${row.columnHeader}」成功綁定`)
  fetchPending()
}

async function handleIgnore(row: any) {
  await updateColumnMapping(row.id, { mappingStatus: 'ignored' })
  ElMessage.info(`已略過「${row.columnHeader}」`)
  fetchPending()
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
      name: quickCreateForm.value.name,
      serviceCategory: 1,
      serviceUsageType: 2
    })
    cases.value = [...cases.value, created]
    await updateColumnMapping(quickCreateForm.value.columnId, {
      caseId: created.id,
      legSeq: quickCreateForm.value.legSeq,
      mappingStatus: 'mapped'
    })
    ElMessage.success(`已建立個案「${created.name}」並完成綁定`)
    quickCreateVisible.value = false
    fetchPending()
  } finally {
    quickCreating.value = false
  }
}

onMounted(async () => {
  const [res] = await Promise.all([listCases({ pageSize: 200 }), loadUploadContext()])
  cases.value = res.data
})
</script>

<style scoped>
.driver-report-import-view {
  display: flex;
  flex-direction: column;
}

.upload-split {
  display: grid;
  grid-template-columns: 260px 1fr;
  gap: var(--app-space-4);
  align-items: start;
}

.upload-side {
  position: sticky;
  top: 0;
  background: var(--app-surface);
  border: 1px solid var(--app-border-color);
  border-radius: var(--app-radius-md);
  box-shadow: var(--app-shadow-sm);
  padding: var(--app-space-4);
  display: flex;
  flex-direction: column;
  gap: var(--app-space-3);
}

.drop-zone :deep(.el-upload) {
  width: 100%;
}

.drop-zone :deep(.el-upload-dragger) {
  width: 100%;
  padding: var(--app-space-4);
}

.drop-hint {
  margin: 0;
  font-size: var(--app-font-sm);
}

.drop-hint-sub {
  margin: var(--app-space-1) 0 0;
  font-size: var(--app-font-xs);
  color: var(--app-text-secondary);
}

.file-count-card {
  text-align: center;
  padding: var(--app-space-3);
  background: var(--app-primary-light);
  border-radius: var(--app-radius-sm);
}

.file-count-num {
  font-size: var(--app-font-2xl);
  font-weight: 700;
  color: var(--app-primary-dark);
}

.file-count-label {
  font-size: var(--app-font-xs);
  color: var(--app-primary-dark);
}

.overlap-alert {
  margin: 0;
}

.overlap-list {
  margin: 0 0 var(--app-space-2);
  padding-left: var(--app-space-4);
}

.submit-btn {
  width: 100%;
}

.upload-main {
  display: flex;
  flex-direction: column;
  gap: var(--app-space-4);
  min-width: 0;
}

.result-banner {
  padding: var(--app-space-3) var(--app-space-4);
  background: var(--app-status-info-bg);
  color: var(--app-status-info-fg);
  border-radius: var(--app-radius-sm);
  font-size: var(--app-font-sm);
  font-weight: 500;
}

.file-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: var(--app-space-3);
}

.file-card {
  position: relative;
  background: var(--app-surface);
  border: 1px solid var(--app-border-color);
  border-radius: var(--app-radius-md);
  box-shadow: var(--app-shadow-sm);
  padding: var(--app-space-3);
  display: flex;
  flex-direction: column;
  gap: var(--app-space-2);
}

.file-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--app-space-2);
}

.file-name {
  font-size: var(--app-font-sm);
  font-weight: 500;
  color: var(--app-text-primary);
  word-break: break-all;
}

.file-card-meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: var(--app-font-xs);
  color: var(--app-text-secondary);
}

.file-card-overlap {
  font-size: var(--app-font-xs);
  color: var(--app-status-warning-fg);
  font-weight: 600;
}

.file-card-result {
  font-size: var(--app-font-xs);
  color: var(--app-text-secondary);
}

.file-card-remove {
  align-self: flex-end;
  border: none;
  background: none;
  color: var(--app-status-danger-fg);
  font-size: var(--app-font-xs);
  cursor: pointer;
}

.file-card-remove:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.small {
  font-size: var(--app-font-xs);
}

.text-secondary {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.text-muted {
  color: var(--app-text-muted);
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
