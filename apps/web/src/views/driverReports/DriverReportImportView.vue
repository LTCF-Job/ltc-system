<template>
  <div class="driver-report-import-view">
    <el-tabs v-model="activeTab" type="border-card" class="import-tabs" @tab-change="handleTabChange">
      <el-tab-pane label="批次上傳" name="upload">
        <div class="upload-stack">
          <PageHeader
            title="批次上傳接送匯報"
            description="選擇多個 .xlsx 檔案，系統會依檔名自動比對車輛代稱、依內容自動判斷涵蓋月份，解析完成後自動匯入。有系統推薦個案的欄位會自動套用，完全找不到對應個案的欄位會留在「待維護資料」頁籤，稍後逐一連結既有個案或建立新個案。"
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
            <p class="drop-hint-sub">檔名建議包含車輛代稱（例如「竹南2車 (回覆).xlsx」）</p>
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

              <el-table-column label="代稱" min-width="200" class-name="vehicle-col">
                <template #default="{ row }">
                  <el-select
                    v-if="row.status === 'needsVehicle'"
                    :model-value="row.vehicleId || undefined"
                    placeholder="選擇代稱"
                    filterable
                    size="small"
                    class="vehicle-select"
                    :aria-label="`${row.file.name} 選擇代稱`"
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
                        <span
                          :class="rowResultPending(row as BatchFileRow) ? 'text-warning' : 'text-regular'"
                          class="truncate-text"
                          :title="rowResultText(row as BatchFileRow)"
                        >
                          {{ rowResultText(row as BatchFileRow) }}
                        </span>
                      </template>
                      <span v-else-if="row.status === 'analyzing'" class="text-muted">解析中…</span>
                      <span v-else-if="row.status === 'needsVehicle'" class="text-warning">請先選擇對應代稱</span>
                      <span v-else class="text-muted">-</span>
                    </div>

                    <el-button
                      v-if="row.issues.length || row.status === 'failed'"
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
                    <el-button
                      v-if="canRetryRow(row as BatchFileRow)"
                      link
                      type="warning"
                      size="small"
                      :disabled="running"
                      @click="retryRow(row as BatchFileRow)"
                    >
                      重試失敗月份
                    </el-button>
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
          <span>待維護資料{{ pendingTabCount ? `（${pendingTabCount}）` : '' }}</span>
        </template>

        <PageHeader
          title="待維護資料"
          description="以下每一列是一筆匯報日期的提交紀錄，展開可看到這一列有哪些欄位比對不到個案、駕駛人是否比對不到司機主檔；可連結既有資料或建立新資料。個案與司機綁定完成後系統都會立即用當初上傳的資料補寫搭乘紀錄，不需要重新上傳檔案。"
        />

        <el-empty v-if="!reviewLoading && submissionReviews.length === 0" description="目前沒有待處理的匯報列" />

        <el-table
          ref="submissionReviewTableRef"
          v-else
          :data="submissionReviews"
          v-loading="reviewLoading"
          row-key="submissionId"
          max-height="600"
          border
          @expand-change="handleSubmissionReviewExpandChange"
        >
          <el-table-column type="expand">
            <template #default="{ row }">
              <div class="review-detail">
                <div v-for="issue in row.caseIssues" :key="issue.id" class="review-issue-row">
                  <div class="review-issue-desc">
                    <el-tag size="small" type="warning">個案未比對</el-tag>
                    <span class="raw-name">{{ issue.columnHeader }}</span>
                    <span v-if="issue.suggestionScore" class="text-secondary small">
                      系統推薦：{{ issue.suggestedCaseName || '無相符個案' }}
                      (信心度: {{ (issue.suggestionScore * 100).toFixed(0) }}%)
                    </span>
                  </div>
                  <div class="target-binding-box">
                    <el-select v-model="issue.editCaseId" placeholder="搜尋個案" filterable clearable style="width: 170px">
                      <el-option v-for="c in cases" :key="c.id" :label="c.name" :value="c.id" />
                    </el-select>
                    <el-select v-model="issue.editLegSeq" placeholder="趟次" style="width: 150px">
                      <el-option v-for="leg in LEG_SEQ_OPTIONS" :key="leg.value" :value="leg.value" :label="leg.label" />
                    </el-select>
                    <TableRowActions>
                      <el-button link type="primary" size="small" :disabled="!issue.editCaseId" @click="handleBindCase(issue)">
                        確認綁定
                      </el-button>
                      <el-button link type="primary" size="small" @click="openQuickCreateCase(issue)">
                        新增個案並綁定
                      </el-button>
                      <el-button link type="danger" size="small" @click="handleIgnoreCase(issue)">
                        忽略此筆
                      </el-button>
                    </TableRowActions>
                  </div>
                </div>

                <div v-if="row.driverIssue" class="review-issue-row">
                  <div class="review-issue-desc">
                    <el-tag size="small" type="danger">駕駛人未比對</el-tag>
                    <span class="raw-name">{{ row.driverIssue.driverNameRaw }}</span>
                  </div>
                  <div class="target-binding-box">
                    <el-select v-model="row.editDriverId" placeholder="搜尋司機" filterable clearable style="width: 170px">
                      <el-option v-for="d in drivers" :key="d.id" :label="d.name" :value="d.id" />
                    </el-select>
                    <TableRowActions>
                      <el-button link type="primary" size="small" :disabled="!row.editDriverId" @click="handleBindDriver(row as SubmissionReviewRow)">
                        確認綁定
                      </el-button>
                      <el-button link type="primary" size="small" @click="openQuickCreateDriver(row as SubmissionReviewRow)">
                        新增司機並綁定
                      </el-button>
                      <el-button link type="danger" size="small" @click="handleIgnoreSubmission(row as SubmissionReviewRow)">
                        忽略此筆
                      </el-button>
                    </TableRowActions>
                  </div>
                </div>

                <div v-for="conflict in row.rowConflicts" :key="conflict.id" class="review-issue-row">
                  <div class="review-issue-desc">
                    <el-tag size="small" type="danger">與既有資料衝突</el-tag>
                    <span class="raw-name">{{ conflict.caseName }}（第 {{ conflict.legSeq }} 趟）</span>
                    <span class="text-secondary small">
                      既有：{{ conflict.previousReported === 'boarded' ? '有坐' : '沒坐' }}
                      / {{ conflict.previousDriverName || '無司機' }}
                      　新上傳：{{ conflict.newReported === 'boarded' ? '有坐' : '沒坐' }}
                      / {{ conflict.newDriverName || '無司機' }}
                    </span>
                  </div>
                  <div class="target-binding-box">
                    <TableRowActions>
                      <el-button link type="primary" size="small" @click="handleResolveRowConflict(conflict, true)">
                        採用新資料
                      </el-button>
                      <el-button link size="small" @click="handleResolveRowConflict(conflict, false)">
                        保留原資料
                      </el-button>
                      <el-button link type="danger" size="small" @click="handleIgnoreRowConflict(conflict)">
                        忽略此筆
                      </el-button>
                    </TableRowActions>
                  </div>
                </div>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="服務日期" width="120">
            <template #default="{ row }">{{ row.serviceDate }}</template>
          </el-table-column>

          <el-table-column label="車輛／匯報表" min-width="180">
            <template #default="{ row }">
              <div>{{ row.vehicleName }}</div>
              <div class="text-secondary small">{{ row.formTitle }}</div>
            </template>
          </el-table-column>

          <el-table-column label="待處理問題" width="150" align="center">
            <template #default="{ row }">
              <el-tag type="warning">{{ issueCount(row as SubmissionReviewRow) }} 個問題待處理</el-tag>
            </template>
          </el-table-column>
        </el-table>

        <PageHeader
          title="出勤待維護"
          description="以下每一列是匯報比對到的司機，當天已有跟系統匯入判斷不同的人工出勤登記；系統不會自動覆蓋人工判斷，請選擇要保留原本的人工登記，還是改採這次匯入判斷的出勤結果。"
          class="attendance-conflict-header"
        />

        <el-empty
          v-if="!attendanceConflictLoading && attendanceConflicts.length === 0"
          description="目前沒有待處理的出勤衝突"
        />

        <el-table v-else :data="attendanceConflicts" v-loading="attendanceConflictLoading" row-key="id" border>
          <el-table-column label="司機" prop="driverName" width="140" />
          <el-table-column label="日期" prop="recordDate" width="120" />
          <el-table-column label="人工登記狀態" width="140" align="center">
            <template #default="{ row }">
              <el-tag type="info" effect="plain">{{ attendanceStatusLabel(row.existingStatus) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="匯入判斷狀態" width="140" align="center">
            <template #default="{ row }">
              <el-tag type="success" effect="plain">{{ attendanceStatusLabel(row.importedStatus) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" min-width="220">
            <template #default="{ row }">
              <TableRowActions>
                <el-button link type="primary" size="small" @click="handleResolveAttendanceConflict(row as AttendanceConflictDTO, 'keep_manual')">
                  保留人工登記
                </el-button>
                <el-button link type="warning" size="small" @click="handleResolveAttendanceConflict(row as AttendanceConflictDTO, 'use_import')">
                  改採匯入結果
                </el-button>
                <el-button link type="danger" size="small" @click="handleIgnoreAttendanceConflict(row as AttendanceConflictDTO)">
                  忽略此筆
                </el-button>
              </TableRowActions>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- 新增個案並綁定：跟個案清單頁的「新增個案基本資料」共用同一個元件與 API -->
    <CaseCreateDialog
      v-model="quickCreateCaseVisible"
      :prefill-name="quickCreateCaseTarget?.cleanedName"
      @created="handleCaseCreatedFromPending"
    />

    <!-- 新增司機並綁定：跟司機管理頁的「新增司機」共用同一個元件與 API -->
    <DriverCreateDialog
      v-model="quickCreateDriverVisible"
      :prefill-name="quickCreateDriverTarget?.driverIssue?.driverNameRaw"
      @created="handleDriverCreatedFromPending"
    />

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
              <span class="info-label">對應代稱：</span>
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

        <div v-if="selectedRowForDetail.months.length" class="month-status-box">
          <div class="issues-header">
            <span class="issues-title">月份處理狀態</span>
            <span class="text-secondary small">成功月份不會因重試再次寫入</span>
          </div>
          <div class="month-status-list">
            <el-tag
              v-for="month in selectedRowForDetail.months"
              :key="month"
              :type="monthStatusType(selectedRowForDetail.monthStates[month])"
              effect="plain"
            >
              {{ month }}：{{ monthStatusLabel(selectedRowForDetail.monthStates[month]) }}
            </el-tag>
          </div>
          <div v-if="canRetryRow(selectedRowForDetail)" class="month-retry-hint">
            <el-button type="warning" plain size="small" :disabled="running" @click="retryRow(selectedRowForDetail)">
              僅重試失敗月份
            </el-button>
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
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { UploadFilled, Document } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type TableInstance, type UploadFile } from 'element-plus'
import { useRouter } from 'vue-router'
import {
  createDriverReportForm,
  commitImportDriverReport,
  dryRunImportDriverReport,
  listDriverReportForms,
  listSubmissionReview,
  matchPendingColumnsByName,
  updateColumnMapping,
  bindPendingDriver,
  resolveRowConflict,
  ignoreDriverReportColumn,
  ignoreDriverReportSubmission,
  ignoreRowConflict
} from '@/api/driverReports'
import { listAttendanceConflicts, resolveAttendanceConflict, ignoreAttendanceConflict } from '@/api/attendance'
import { listAllCases } from '@/api/cases'
import { listAllVehicles, listAllDrivers } from '@/api/masters'
import PageHeader from '@/components/PageHeader.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import DialogFooter from '@/components/DialogFooter.vue'
import StatusTag from '@/components/StatusTag.vue'
import CaseCreateDialog from '@/components/cases/CaseCreateDialog.vue'
import DriverCreateDialog from '@/components/masters/DriverCreateDialog.vue'
import { resolveApiErrorMessage, NETWORK_ERROR_MESSAGE, TIMEOUT_ERROR_MESSAGE } from '@/api/errorCodes'
import { toColumnDecisionPayload, type ColumnDecisionMap } from './columnDecisions'
import { describeImportResult, hasPendingWork } from './importSummary'
import { LEG_SEQ_OPTIONS } from './legOptions'
import type {
  ApiError,
  AttendanceConflictDTO,
  CaseDTO,
  DriverDTO,
  DriverReportColumnDTO,
  DriverReportFormDTO,
  DriverReportPreviewDTO,
  DriverReportCommitResultDTO,
  RowConflictDTO,
  SubmissionReviewDTO,
  VehicleDTO
} from '@/types/api'

type EditableCaseIssue = DriverReportColumnDTO & { editCaseId?: string; editLegSeq?: number }
type SubmissionReviewRow = Omit<SubmissionReviewDTO, 'caseIssues'> & {
  caseIssues: EditableCaseIssue[]
  editDriverId?: string
}

const router = useRouter()
const activeTab = ref<'upload' | 'pending'>('upload')

// ---- 批次上傳 ----

type RowStatus = 'needsVehicle' | 'queued' | 'analyzing' | 'processing' | 'done' | 'failed'
type MonthImportStatus = 'pending' | 'processing' | 'succeeded' | 'failed'
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
  monthStates: Record<string, MonthImportStatus>
  monthMessages: Record<string, string>
  importedCount: number
  rideRecordCount: number
  reaffirmedCount: number
  conflictCount: number
  backfilledCount: number
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
const summary = ref<{ succeeded: number; failed: number; importedDays: number; pendingColumns: number } | null>(null)
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
// 拖曳區在資料載完前停用，避免車輛清單還是空的時就跑自動比對而誤判成「待選車輛」
const dropDisabled = computed(
  () => running.value || contextLoading.value || contextLoadFailed.value || vehicles.value.length === 0
)
// 資料真的載入成功、只是車輛清單為空時，要跟「還在載入」「載入失敗」分開提示，
// 否則使用者只會看到一塊選不出任何選項的下拉選單、不知道該去哪裡處理
const hasNoVehicles = computed(
  () => !contextLoading.value && !contextLoadFailed.value && vehicles.value.length === 0
)

const formByVehicle = computed(() => new Map(forms.value.map((f) => [f.vehicleId, f.id])))
// 已解析完成、還沒匯入的檔案。已匯入或失敗的列不再納入，否則自動匯入會反覆重跑同一批
const pendingRows = computed(() => rows.value.filter((r) => r.vehicleId && r.status === 'queued'))
const canImport = computed(
  () => !running.value && analyzePending.value === 0 && pendingRows.value.length > 0
)

// 沒有送出按鈕：整批解析完就自動匯入。等 analyzePending 歸零才觸發，讓一次拖入的多個檔案併成一批。
// 涵蓋月份已有資料時不再攔截確認：逐列比對本來就不覆蓋，值不同的會進待維護等使用者裁決。
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
    monthStates: {},
    monthMessages: {},
    importedCount: 0,
    rideRecordCount: 0,
    reaffirmedCount: 0,
    conflictCount: 0,
    backfilledCount: 0,
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
  row.months = []
  row.monthStates = {}
  row.monthMessages = {}
  enqueueAnalyze(row)
}

function removeRow(row: BatchFileRow) {
  rows.value = rows.value.filter((r) => r.key !== row.key)
}

const formCreationByVehicle = new Map<string, Promise<string>>()
// 同一個表單、月份與檔案若因重複觸發同時送出，只保留一個前端請求；後端沒有檔案層級的
// 重複判斷，同月併發是由 LockDriverReportImport 的 advisory lock 擋下。
const activeMonthImports = new Map<string, Promise<DriverReportCommitResultDTO>>()
// 不同檔案寫入同一表單月份時也要在前端排隊，避免結果互相競速。
const monthImportLocks = new Map<string, Promise<void>>()

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

function activeMonthImportKey(formId: string, row: BatchFileRow, month: string): string {
  return [formId, month, row.file.name, row.file.size, row.file.lastModified].join('::')
}

function commitMonthOnce(
  formId: string,
  row: BatchFileRow,
  payload: ReturnType<typeof toColumnDecisionPayload>,
  month: string
): Promise<DriverReportCommitResultDTO> {
  const key = activeMonthImportKey(formId, row, month)
  const active = activeMonthImports.get(key)
  if (active) return active

  const lockKey = `${formId}::${month}`
  const previous = monthImportLocks.get(lockKey) ?? Promise.resolve()
  let release!: () => void
  const current = new Promise<void>((resolve) => {
    release = resolve
  })
  monthImportLocks.set(lockKey, current)

  const request = (async () => {
    await previous
    try {
      return await commitImportDriverReport(formId, row.file, payload, month)
    } finally {
      release()
      if (monthImportLocks.get(lockKey) === current) monthImportLocks.delete(lockKey)
    }
  })()
  activeMonthImports.set(key, request)
  const clear = () => {
    if (activeMonthImports.get(key) === request) activeMonthImports.delete(key)
  }
  void request.then(clear, clear)
  return request
}

function failedMonths(row: BatchFileRow): string[] {
  return row.months.filter((month) => row.monthStates[month] === 'failed')
}

function canRetryRow(row: BatchFileRow): boolean {
  return !running.value && failedMonths(row).length > 0
}

function monthStatusLabel(status: MonthImportStatus | undefined): string {
  if (status === 'succeeded') return '成功'
  if (status === 'processing') return '處理中'
  if (status === 'failed') return '失敗'
  return '待處理'
}

function rowResultCounts(row: BatchFileRow) {
  return {
    importedDays: row.importedCount,
    rideRecords: row.rideRecordCount,
    reaffirmed: row.reaffirmedCount,
    conflicts: row.conflictCount,
    backfilled: row.backfilledCount,
    pendingColumns: row.pendingColumnCount
  }
}

function rowResultText(row: BatchFileRow): string {
  return describeImportResult(rowResultCounts(row))
}

function rowResultPending(row: BatchFileRow): boolean {
  return hasPendingWork(rowResultCounts(row))
}

function monthStatusType(status: MonthImportStatus | undefined): 'success' | 'warning' | 'danger' | 'info' {
  if (status === 'succeeded') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'processing') return 'warning'
  return 'info'
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

// analyzeRow 只做「解析出涵蓋月份與是否已有既有資料」的預覽，不寫入任何資料；
// 真正的欄位對應與 commit 交給 processRow，兩者都各自 dry-run 一次，換取程式碼單純。
async function analyzeRow(row: BatchFileRow) {
  if (row.status === 'needsVehicle') return
  row.status = 'analyzing'
  try {
    const formId = formByVehicle.value.get(row.vehicleId) ?? ''
    row.formId = formId
    if (!formId) {
      row.months = []
      row.monthStates = {}
      row.monthMessages = {}
      row.issues = []
      row.status = 'queued'
      row.message = '尚未建立匯報表，將於匯入時建立'
      return
    }
    const preview = await dryRunImportDriverReport(formId, row.file)
    row.issues = collectPreviewIssues(preview)
    const months = computeMonths(preview.previewRows)
    row.months = months
    row.monthStates = Object.fromEntries(months.map((month) => [month, 'pending' as MonthImportStatus]))
    row.monthMessages = {}
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

    row.months = months
    row.monthStates = Object.fromEntries(
      months.map((month) => [month, row.monthStates[month] ?? 'pending'])
    ) as Record<string, MonthImportStatus>
    row.monthMessages = Object.fromEntries(
      months
        .filter((month) => row.monthMessages[month])
        .map((month) => [month, row.monthMessages[month]])
    )
    const payload = toColumnDecisionPayload(decisions)
    for (const month of months) {
      if (row.monthStates[month] === 'succeeded') continue

      row.monthStates[month] = 'processing'
      try {
        const result = await commitMonthOnce(formId, row, payload, month)
        row.importedCount += result.importedRows
        row.rideRecordCount += result.rideRecordRows
        row.reaffirmedCount += result.reaffirmedRows
        row.conflictCount += result.pendingConflictRows
        row.backfilledCount += result.backfilledRows
        row.monthStates[month] = 'succeeded'
        row.monthMessages[month] = ''
        row.issues.push(
          ...result.skippedRows.flatMap((item) =>
            item.reasons.map((reason) => ({
              level: 'error' as const,
              message: `第 ${item.rowIndex} 列${item.reportDate ? `（${item.reportDate}）` : ''}：${reason}`
            }))
          ),
          ...(result.warnings ?? []).map((item) => ({ level: 'warning' as const, message: formatPreviewIssue(item) })),
          ...(result.pendingConflictRows > 0
            ? [{ level: 'warning' as const, message: `${month}：${result.pendingConflictRows} 筆與既有資料不同，已進入待維護等待選擇` }]
            : [])
        )
      } catch (error) {
        row.monthStates[month] = 'failed'
        row.monthMessages[month] = rowErrorMessage(error)
        row.issues.push({ level: 'error', message: `${month}：${row.monthMessages[month]}` })
      }
    }
    row.issues = row.issues.filter(
      (issue, index, all) =>
        all.findIndex((candidate) => candidate.level === issue.level && candidate.message === issue.message) === index
    )
    const failed = failedMonths(row)
    row.status = failed.length > 0 ? 'failed' : 'done'
    row.message = failed.length > 0 ? `${failed.length} 個月份匯入失敗，可只重試失敗月份` : ''
    row.pendingColumnCount = Object.values(decisions).filter((d) => d.mappingStatus === 'pending').length
  } catch (error) {
    row.status = 'failed'
    row.message = rowErrorMessage(error)
  }
}

async function retryRow(row: BatchFileRow) {
  if (!canRetryRow(row)) return
  running.value = true
  try {
    await processRow(row)
    refreshImportSummary()
  } finally {
    running.value = false
  }
}

function rowErrorMessage(error: unknown): string {
  const response = (error as { response?: { data?: { error?: ApiError } } })?.response
  const detail = response?.data?.error
  if (detail?.details?.length) {
    return detail.details
      .map((d) => `${d.field ? `【${IMPORT_FIELD_LABELS[d.field] || d.field}】` : ''}${d.reason}`)
      .join('；')
  }
  if (detail) return resolveApiErrorMessage(detail, '匯入失敗，請確認檔案內容')

  // 走到這裡代表請求沒拿到後端回應。原始 error.message 是 axios 或瀏覽器的技術字串
  // （如 "Network Error"、"timeout of 30000ms exceeded"），不能直接顯示給使用者。
  const timedOut = (error as { code?: string })?.code === 'ECONNABORTED'
  return timedOut ? TIMEOUT_ERROR_MESSAGE : NETWORK_ERROR_MESSAGE
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

    const importSummary = refreshImportSummary()
    const succeeded = targets.filter((r) => r.status === 'done')
    if (succeeded.length) await handleUploadSuccess({ pendingColumns: importSummary.pendingColumns })
  } finally {
    running.value = false
  }
}

function refreshImportSummary(): { succeeded: number; failed: number; importedDays: number; pendingColumns: number } {
  const processed = rows.value.filter((row) => row.status === 'done' || row.status === 'failed')
  const succeeded = processed.filter((row) => row.status === 'done')
  const result = {
    succeeded: succeeded.length,
    failed: processed.length - succeeded.length,
    importedDays: processed.reduce((sum, row) => sum + row.importedCount, 0),
    pendingColumns: succeeded.reduce((sum, row) => sum + row.pendingColumnCount, 0)
  }
  summary.value = result
  return result
}

// 分頁端點在 0 筆結果時回傳 data: null，一律預設空陣列，避免後續 detectVehicle 等處
// 的 .filter／.map 對 null 直接丟出未捕捉例外。
async function loadUploadContext() {
  try {
    const [vehiclePage, formList] = await Promise.all([
      listAllVehicles({ status: 'active' }),
      listDriverReportForms()
    ])
    vehicles.value = vehiclePage ?? []
    forms.value = formList ?? []
  } catch {
    contextLoadFailed.value = true
  } finally {
    contextLoading.value = false
  }
}

async function loadCases() {
  try {
    cases.value = await listAllCases()
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  }
}

// ---- 待維護資料 ----

const cases = ref<CaseDTO[]>([])
const drivers = ref<DriverDTO[]>([])
const submissionReviews = ref<SubmissionReviewRow[]>([])
const reviewLoading = ref(false)

// el-table 的展開狀態綁在資料列的物件參考上；每次連結/略過都會整批重打
// fetchSubmissionReview 換新陣列，若不自己記住已展開的 submissionId 並在資料回來後
// 重新展開，使用者連續處理同一列的多個待維護欄位時，畫面會每按一次就收闔一次。
const submissionReviewTableRef = ref<TableInstance>()
const expandedSubmissionIds = ref(new Set<string>())

// el-table 的 expand-change 型別是聯集（一般表格傳展開列陣列，樹狀表格傳布林值），
// 這張表不是樹狀結構，實際上永遠會收到陣列，只是型別上要涵蓋另一種簽名。
function handleSubmissionReviewExpandChange(
  _row: SubmissionReviewRow,
  expandedRows: SubmissionReviewRow[] | boolean
) {
  if (!Array.isArray(expandedRows)) return
  expandedSubmissionIds.value = new Set(expandedRows.map((r) => r.submissionId))
}

async function restoreSubmissionReviewExpansion() {
  await nextTick()
  const table = submissionReviewTableRef.value
  if (!table) return
  for (const row of submissionReviews.value) {
    if (expandedSubmissionIds.value.has(row.submissionId)) table.toggleRowExpansion(row, true)
  }
}

const quickCreateCaseVisible = ref(false)
const quickCreateCaseTarget = ref<EditableCaseIssue | null>(null)
const quickCreateDriverVisible = ref(false)
const quickCreateDriverTarget = ref<SubmissionReviewRow | null>(null)

const attendanceConflicts = ref<AttendanceConflictDTO[]>([])
const attendanceConflictLoading = ref(false)

const ATTENDANCE_STATUS_LABELS: Record<string, string> = {
  work: '出勤 (O)',
  leave: '事假 (事)',
  sick: '病假 (病)',
  off: '休假 (休)'
}

function attendanceStatusLabel(status: string): string {
  return ATTENDANCE_STATUS_LABELS[status] || status
}

function issueCount(row: SubmissionReviewRow): number {
  return row.caseIssues.length + (row.driverIssue ? 1 : 0) + (row.rowConflicts?.length ?? 0)
}

// pendingTabCount 是頁籤上顯示的總數字，個案／駕駛人待維護列與出勤衝突是兩個獨立區塊，
// 各自的數量都算「待處理」，加總才是使用者實際要處理的項目數。
const pendingTabCount = computed(() => submissionReviews.value.length + attendanceConflicts.value.length)

async function loadDrivers() {
  try {
    drivers.value = await listAllDrivers({ status: 'active' })
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  }
}

// 切換至待維護頁籤時主動重新整理清單，確保顯示最新待處理項目
async function handleTabChange(name: string | number) {
  if (name === 'pending') {
    await fetchSubmissionReview()
    await fetchAttendanceConflicts()
  }
}

// 上傳完成後若有欄位進入待維護，詢問是否直接切過去處理，比照個案管理匯入完成後的提示模式
async function handleUploadSuccess(result: { pendingColumns: number }) {
  if (result.pendingColumns === 0) return
  await fetchSubmissionReview()
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

async function fetchSubmissionReview() {
  reviewLoading.value = true
  try {
    const reviews = await listSubmissionReview()
    submissionReviews.value = reviews.map((r) => ({
      ...r,
      caseIssues: r.caseIssues.map((c) => ({ ...c, editCaseId: c.suggestedCaseId || '', editLegSeq: c.suggestedLegSeq || 1 })),
      editDriverId: ''
    }))
    await restoreSubmissionReviewExpansion()
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  } finally {
    reviewLoading.value = false
  }
}

async function handleBindCase(issue: EditableCaseIssue) {
  try {
    const { backfilledRows } = await updateColumnMapping(issue.id, {
      caseId: issue.editCaseId,
      legSeq: issue.editLegSeq,
      mappingStatus: 'mapped'
    })
    ElMessage.success(`已將「${issue.columnHeader}」成功綁定，補寫 ${backfilledRows} 筆搭乘紀錄`)
    await fetchSubmissionReview()
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  }
}

// 以下四支「忽略此筆」都是直接刪除待維護資料列。重新匯入同一份檔案時該筆會再次出現，
// 所以確認文案一律說明這件事，避免使用者以為忽略等於永久靜音。
async function confirmIgnore(message: string): Promise<boolean> {
  try {
    await ElMessageBox.confirm(`${message}若之後重新匯入同一份檔案，這筆仍會再次出現。`, '忽略確認', {
      confirmButtonText: '忽略並刪除',
      cancelButtonText: '取消',
      type: 'warning',
      confirmButtonClass: 'el-button--danger'
    })
    return true
  } catch {
    return false
  }
}

async function handleIgnoreCase(issue: EditableCaseIssue) {
  if (!(await confirmIgnore(`確定要忽略欄位「${issue.columnHeader}」？將直接刪除這筆欄位對應資料。`))) return
  try {
    await ignoreDriverReportColumn(issue.id)
    ElMessage.success(`已忽略「${issue.columnHeader}」`)
    await fetchSubmissionReview()
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  }
}

async function handleIgnoreSubmission(row: SubmissionReviewRow) {
  const label = row.driverIssue?.driverNameRaw ?? row.serviceDate
  if (!(await confirmIgnore(`確定要忽略「${label}」這筆匯報列？將直接刪除該筆匯報資料。`))) return
  try {
    await ignoreDriverReportSubmission(row.submissionId)
    ElMessage.success(`已忽略「${label}」`)
    await fetchSubmissionReview()
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  }
}

async function handleIgnoreRowConflict(conflict: RowConflictDTO) {
  if (!(await confirmIgnore(`確定要忽略「${conflict.caseName}」第 ${conflict.legSeq} 趟的衝突？既有搭乘資料維持原值不變。`))) return
  try {
    await ignoreRowConflict(conflict.id)
    ElMessage.success('已忽略該筆衝突')
    await fetchSubmissionReview()
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  }
}

async function handleIgnoreAttendanceConflict(row: AttendanceConflictDTO) {
  if (!(await confirmIgnore(`確定要忽略「${row.driverName}」${row.recordDate} 的出勤衝突？出勤紀錄維持原本的人工登記。`))) return
  try {
    await ignoreAttendanceConflict(row.id)
    attendanceConflicts.value = attendanceConflicts.value.filter((c) => c.id !== row.id)
    ElMessage.success('已忽略該筆出勤衝突')
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  }
}

async function fetchAttendanceConflicts() {
  attendanceConflictLoading.value = true
  try {
    attendanceConflicts.value = await listAttendanceConflicts()
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  } finally {
    attendanceConflictLoading.value = false
  }
}

async function handleResolveAttendanceConflict(conflict: AttendanceConflictDTO, choice: 'keep_manual' | 'use_import') {
  try {
    await resolveAttendanceConflict(conflict.id, { choice })
    ElMessage.success(
      choice === 'keep_manual'
        ? `已保留「${conflict.driverName}」${conflict.recordDate} 的人工出勤登記`
        : `已將「${conflict.driverName}」${conflict.recordDate} 改採匯入判斷的出勤結果`
    )
    attendanceConflicts.value = attendanceConflicts.value.filter((c) => c.id !== conflict.id)
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  }
}

async function handleResolveRowConflict(conflict: RowConflictDTO, useNew: boolean) {
  try {
    await resolveRowConflict(conflict.id, { useNew })
    ElMessage.success(useNew ? `已採用「${conflict.caseName}」最新上傳的資料` : `已保留「${conflict.caseName}」原有的資料`)
    await fetchSubmissionReview()
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  }
}

async function handleBindDriver(row: SubmissionReviewRow) {
  if (!row.driverIssue || !row.editDriverId) return
  try {
    const { affectedCount } = await bindPendingDriver({
      driverNameRaw: row.driverIssue.driverNameRaw,
      driverId: row.editDriverId
    })
    ElMessage.success(`已完成司機綁定，共回填 ${affectedCount} 筆搭乘紀錄`)
    await fetchSubmissionReview()
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  }
}

// openQuickCreateCase 帶入匯報表原始欄位解析出的姓名，讓使用者只需補完其他欄位即可建立並直接綁定
function openQuickCreateCase(issue: EditableCaseIssue) {
  quickCreateCaseTarget.value = issue
  quickCreateCaseVisible.value = true
}

async function handleCaseCreatedFromPending(created: CaseDTO) {
  cases.value = [...cases.value, created]
  const target = quickCreateCaseTarget.value
  if (target) {
    try {
      const { backfilledRows } = await updateColumnMapping(target.id, {
        caseId: created.id,
        legSeq: target.editLegSeq || 1,
        mappingStatus: 'mapped'
      })
      ElMessage.success(`已建立個案「${created.name}」並完成綁定，補寫 ${backfilledRows} 筆搭乘紀錄`)
    } catch {
      // 全域攔截器負責顯示 API 錯誤。
    }
  }
  await fetchSubmissionReview()
  await promptRelatedCaseIssues(created, target)
}

// 新建個案後，掃描目前待維護清單裡其他姓名相符（含近似）的欄位，詢問是否一併連結到同一個案
async function promptRelatedCaseIssues(created: CaseDTO, triggerTarget: EditableCaseIssue | null) {
  let matches: DriverReportColumnDTO[] = []
  try {
    matches = await matchPendingColumnsByName(created.name)
  } catch {
    return
  }

  const candidates = matches.filter((m) => !triggerTarget || m.id !== triggerTarget.id)
  if (candidates.length === 0) return

  try {
    await ElMessageBox.confirm(
      `待維護清單中還有這些欄位的姓名疑似也是「${created.name}」：${candidates.map((c) => c.columnHeader).join('、')}，是否一併連結到這個個案？`,
      '發現疑似同一人',
      { confirmButtonText: '一併連結', cancelButtonText: '不用，我自己處理', type: 'info' }
    )
  } catch {
    return
  }

  let boundCount = 0
  for (const m of candidates) {
    try {
      await updateColumnMapping(m.id, {
        caseId: created.id,
        legSeq: m.suggestedLegSeq || 1,
        mappingStatus: 'mapped'
      })
      boundCount++
    } catch {
      // 全域攔截器負責顯示 API 錯誤。
    }
  }
  if (boundCount > 0) {
    ElMessage.success(`已一併連結 ${boundCount} 個欄位到「${created.name}」`)
    await fetchSubmissionReview()
  }
}

function openQuickCreateDriver(row: SubmissionReviewRow) {
  quickCreateDriverTarget.value = row
  quickCreateDriverVisible.value = true
}

async function handleDriverCreatedFromPending(created: DriverDTO) {
  drivers.value = [...drivers.value, created]
  const target = quickCreateDriverTarget.value
  if (target?.driverIssue) {
    try {
      const { affectedCount } = await bindPendingDriver({
        driverNameRaw: target.driverIssue.driverNameRaw,
        driverId: created.id
      })
      ElMessage.success(`已建立司機「${created.name}」並完成綁定，共回填 ${affectedCount} 筆搭乘紀錄`)
    } catch {
      // 全域攔截器負責顯示 API 錯誤。
    }
  }
  await fetchSubmissionReview()
}

onMounted(() => {
  void loadCases()
  void loadDrivers()
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

.no-vehicles-hint {
  margin: 0 0 var(--app-space-2);
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

.review-detail {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px 16px;
}

.attendance-conflict-header {
  margin-top: 24px;
}

.review-issue-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid var(--app-border-light, #ebeef5);
}

.review-issue-row:last-child {
  border-bottom: none;
}

.review-issue-desc {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
</style>
