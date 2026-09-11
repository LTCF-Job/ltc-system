<template>
  <div class="driver-report-status-view">
    <DataTablePage
      title="接送匯報總覽"
      :description="`共 ${forms.length} 台車，顯示各車已有對應資料的月份`"
      :loading="loading"
    >
      <template #filter>
        <el-input
          v-model="searchQuery"
          placeholder="搜尋車輛"
          clearable
          style="width: 240px"
          @keyup.enter="fetchForms"
        />
        <el-button type="primary" @click="fetchForms">查詢</el-button>
        <el-button @click="handleReset">重設</el-button>
      </template>

      <template #table>
      <el-table
        :data="forms"
        border
        stripe
        row-key="id"
        style="width: 100%"
        :expand-row-keys="expandedIds"
        @expand-change="onExpandChange"
      >
        <el-table-column type="expand">
          <template #default="{ row }">
            <div class="months-detail">
              <template v-if="monthsByForm.get(row.id)?.length">
                <div
                  v-for="m in monthsByForm.get(row.id)"
                  :key="m.yearMonth"
                  class="month-item month-item--clickable"
                  @click="openMonthDetail(row as DriverReportFormDTO, m.yearMonth)"
                >
                  <span class="month-label">{{ m.yearMonth }}</span>
                  <span class="month-count">{{ m.submissionCount }} 天</span>
                  <span class="text-secondary">最後匯入 {{ formatDateTime(m.lastImportedAt, '—') }}</span>
                </div>
              </template>
              <p v-else class="text-secondary">這台車尚未有任何月份的匯入紀錄。</p>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="車輛" min-width="140" class-name="vehicle-col">
          <template #default="{ row }">
            {{ row.vehicleName }}
          </template>
        </el-table-column>

        <el-table-column label="已有資料月份" min-width="260" class-name="months-col">
          <template #default="{ row }">
            <div v-if="monthsByForm.get(row.id)?.length" class="month-tags">
              <el-tag
                v-for="m in monthsByForm.get(row.id)"
                :key="m.yearMonth"
                size="small"
                class="month-tag--clickable"
                @click="openMonthDetail(row as DriverReportFormDTO, m.yearMonth)"
              >
                {{ m.yearMonth }}（{{ m.submissionCount }}天）
              </el-tag>
            </div>
            <span v-else class="text-muted">尚未匯入</span>
          </template>
        </el-table-column>

        <el-table-column label="最後匯入時間" min-width="170" align="center" class-name="report-nowrap-col import-time-col">
          <template #default="{ row }">
            {{ formatDateTime(row.lastImportedAt, '尚未匯入') }}
          </template>
        </el-table-column>

        <template #empty>
          <div class="empty-state">
            <p>尚未建立任何車輛的接送匯報表。</p>
            <p class="text-secondary">請到「批次上傳」頁面上傳司機填寫的 .xlsx 匯報檔，系統會自動建立對應車輛的匯報表。</p>
          </div>
        </template>
      </el-table>
      </template>
    </DataTablePage>

    <!-- 單一月份匯入資料鑽取彈窗：左右主從版面，左側清單挑選日期／個案，右側顯示細節，
         取代先前的全螢幕大表格，視窗貼合內容尺寸、點遮罩或 Esc 即可關閉回主頁 -->
    <el-dialog
      v-model="monthDialogVisible"
      :title="monthDialogTitle"
      width="min(920px, 92vw)"
      destroy-on-close
      class="month-detail-dialog"
    >
      <div class="month-detail-summary">
        <span class="summary-item">
          <span class="summary-value">{{ monthSubmissions.length }}</span> 天回報明細
        </span>
        <span class="summary-divider" />
        <span class="summary-item">
          <span class="summary-value">{{ monthRideEntries.length }}</span> 筆搭乘紀錄
        </span>
      </div>

      <el-tabs v-model="monthDialogTab">
        <el-tab-pane label="逐日回報明細" name="submissions">
          <div v-loading="monthDetailLoading" class="detail-split">
            <el-empty v-if="!monthDetailLoading && !monthSubmissions.length" description="這個月沒有逐日回報資料" />
            <template v-else>
              <div class="split-list">
                <div
                  v-for="s in monthSubmissions"
                  :key="s.serviceDate"
                  class="split-list-row"
                  :class="{ active: s.serviceDate === activeServiceDate }"
                  @click="activeServiceDate = s.serviceDate"
                >
                  <span>{{ s.serviceDate }}</span>
                  <span class="split-list-badge">{{ Object.keys(s.answers || {}).length }}</span>
                </div>
              </div>
              <div class="split-detail">
                <template v-if="activeSubmission">
                  <div class="detail-head">
                    <strong>{{ activeSubmission.serviceDate }}</strong>
                    <span class="text-secondary">駕駛：{{ activeSubmission.driverNameRaw }}</span>
                  </div>
                  <p v-if="activeSubmission.remark" class="detail-remark">{{ activeSubmission.remark }}</p>
                  <div class="answers-detail">
                    <template v-if="Object.keys(activeSubmission.answers || {}).length">
                      <div v-for="group in groupAnswersByCase(activeSubmission.answers)" :key="group.key" class="answer-group">
                        <div v-for="item in group.items" :key="item.header" class="answer-row">
                          <span class="answer-key">{{ item.header }}</span>
                          <span class="answer-value">{{ item.value || '—' }}</span>
                        </div>
                      </div>
                    </template>
                    <p v-else class="text-secondary">這天沒有原始欄位資料。</p>
                  </div>
                </template>
              </div>
            </template>
          </div>
        </el-tab-pane>

        <el-tab-pane label="逐個案搭乘紀錄" name="rideEntries">
          <div v-loading="monthDetailLoading" class="detail-split">
            <el-empty v-if="!monthDetailLoading && !monthRideEntries.length" description="這個月沒有個案搭乘紀錄" />
            <template v-else>
              <div class="split-list">
                <div
                  v-for="g in caseGroups"
                  :key="g.caseId"
                  class="split-list-row"
                  :class="{ active: g.caseId === activeCaseId }"
                  @click="activeCaseId = g.caseId"
                >
                  <span>{{ g.caseName }}</span>
                  <span class="split-list-badge">{{ g.entries.length }}</span>
                </div>
              </div>
              <div class="split-detail">
                <template v-if="activeCaseGroup">
                  <div class="detail-head"><strong>{{ activeCaseGroup.caseName }}</strong></div>
                  <table class="dense-table">
                    <thead>
                      <tr>
                        <th>服務日期</th>
                        <th>趟次</th>
                        <th>回報結果</th>
                        <th>駕駛人</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="(e, i) in activeCaseGroup.entries" :key="i">
                        <td>{{ e.serviceDate }}</td>
                        <td>{{ legLabel(e.legSeq) }}</td>
                        <td>
                          <el-tag size="small" :type="e.reported === 'boarded' ? 'success' : 'info'" disable-transitions>
                            {{ e.reported === 'boarded' ? '有搭乘' : '未搭乘' }}
                          </el-tag>
                        </td>
                        <td>{{ e.driverName }}</td>
                      </tr>
                    </tbody>
                  </table>
                </template>
              </div>
            </template>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import DataTablePage from '@/components/DataTablePage.vue'
import {
  listDriverReportForms,
  listDriverReportImportedMonths,
  getDriverReportMonthDetail
} from '@/api/driverReports'
import { formatDateTime } from '@/utils/formatters'
import { LEG_SEQ_OPTIONS } from './legOptions'
import type {
  DriverReportFormDTO,
  DriverReportImportedMonthDTO,
  DriverReportMonthSubmissionDTO,
  DriverReportMonthRideEntryDTO
} from '@/types/api'

const forms = ref<DriverReportFormDTO[]>([])
const importedMonths = ref<DriverReportImportedMonthDTO[]>([])
const searchQuery = ref('')
const loading = ref(false)
const expandedIds = ref<string[]>([])

// 依 formId 分組並依月份新到舊排序，供展開列與月份標籤欄共用
const monthsByForm = computed(() => {
  const grouped = new Map<string, DriverReportImportedMonthDTO[]>()
  for (const m of importedMonths.value) {
    const list = grouped.get(m.formId) ?? []
    list.push(m)
    grouped.set(m.formId, list)
  }
  for (const list of grouped.values()) list.sort((a, b) => b.yearMonth.localeCompare(a.yearMonth))
  return grouped
})

async function fetchForms() {
  loading.value = true
  try {
    const [formList, months] = await Promise.all([
      listDriverReportForms({ q: searchQuery.value || undefined }),
      listDriverReportImportedMonths()
    ])
    forms.value = formList
    importedMonths.value = months
  } finally {
    loading.value = false
  }
}

function handleReset() {
  searchQuery.value = ''
  fetchForms()
}

function onExpandChange(_row: unknown, expanded: unknown) {
  if (!Array.isArray(expanded)) return
  expandedIds.value = (expanded as DriverReportFormDTO[]).map((f) => f.id)
}

// 月份鑽取彈窗：點某台車某個月份的標籤後，載入該月完整匯入資料
const monthDialogVisible = ref(false)
const monthDialogTab = ref<'submissions' | 'rideEntries'>('submissions')
const monthDialogTitle = ref('')
const monthDetailLoading = ref(false)
const monthSubmissions = ref<DriverReportMonthSubmissionDTO[]>([])
const monthRideEntries = ref<DriverReportMonthRideEntryDTO[]>([])

// 逐個案搭乘紀錄：左右主從版面，左側依個案姓名分組清單，右側顯示所選個案的所有搭乘紀錄
interface CaseGroup {
  caseId: string
  caseName: string
  entries: DriverReportMonthRideEntryDTO[]
}

function buildCaseGroups(entries: DriverReportMonthRideEntryDTO[]): CaseGroup[] {
  const groups = new Map<string, CaseGroup>()
  for (const e of entries) {
    let g = groups.get(e.caseId)
    if (!g) {
      g = { caseId: e.caseId, caseName: e.caseName, entries: [] }
      groups.set(e.caseId, g)
    }
    g.entries.push(e)
  }
  const result = Array.from(groups.values())
  for (const g of result) {
    g.entries.sort((a, b) => (a.serviceDate < b.serviceDate ? -1 : a.serviceDate > b.serviceDate ? 1 : a.legSeq - b.legSeq))
  }
  result.sort((a, b) => a.caseName.localeCompare(b.caseName, 'zh-Hant'))
  return result
}

const caseGroups = computed(() => buildCaseGroups(monthRideEntries.value))
const activeCaseId = ref('')
const activeCaseGroup = computed(() => caseGroups.value.find((g) => g.caseId === activeCaseId.value))

const activeServiceDate = ref('')
const activeSubmission = computed(() => monthSubmissions.value.find((s) => s.serviceDate === activeServiceDate.value))

function legLabel(legSeq: number): string {
  return LEG_SEQ_OPTIONS.find((opt) => opt.value === legSeq)?.label ?? `第 ${legSeq} 趟`
}

interface AnswerItem {
  header: string
  value: string
  directionOrder: number
}

interface AnswerGroup {
  key: string
  seq: number
  name: string
  items: AnswerItem[]
}

// 原始欄位標題格式為「<序號>.<個案姓名>...[去程/回程]」；序號僅代表原始表單的欄位順序，
// 並非個案專屬（不同個案可能共用同一個序號），故實際分組須以「個案姓名」為準，
// 讓同一個個案的去程／回程資料顯示在一起，再依序號排序整體呈現順序。
function parseHeaderMeta(header: string): { seq: number; name: string; directionOrder: number } {
  const seqMatch = header.match(/^\s*(\d+)[.、．\s]+/)
  const seq = seqMatch ? Number(seqMatch[1]) : Number.MAX_SAFE_INTEGER
  const withoutSeq = header.replace(/^\s*\d+[.、．\s]+/, '')
  const directionMatch = withoutSeq.match(/\[(去程|回程)\]\s*$/)
  const directionOrder = directionMatch ? (directionMatch[1] === '去程' ? 0 : 1) : 2
  const name = withoutSeq.replace(/\[(去程|回程)\]\s*$/, '').trim()
  return { seq, name, directionOrder }
}

function groupAnswersByCase(answers: Record<string, string>): AnswerGroup[] {
  const groups = new Map<string, AnswerGroup>()
  for (const [header, value] of Object.entries(answers || {})) {
    const { seq, name, directionOrder } = parseHeaderMeta(header)
    const key = name || `_ungrouped_${header}`
    let group = groups.get(key)
    if (!group) {
      group = { key, seq, name, items: [] }
      groups.set(key, group)
    } else if (seq < group.seq) {
      group.seq = seq
    }
    group.items.push({ header, value, directionOrder })
  }
  const result = Array.from(groups.values())
  for (const group of result) {
    group.items.sort((a, b) => a.directionOrder - b.directionOrder)
  }
  result.sort((a, b) => a.seq - b.seq || a.name.localeCompare(b.name, 'zh-Hant'))
  return result
}

async function openMonthDetail(form: DriverReportFormDTO, yearMonth: string) {
  monthDialogTitle.value = `${form.vehicleName} — ${yearMonth} 匯入資料`
  monthDialogTab.value = 'submissions'
  monthDialogVisible.value = true
  monthDetailLoading.value = true
  monthSubmissions.value = []
  monthRideEntries.value = []
  try {
    const detail = await getDriverReportMonthDetail(form.id, yearMonth)
    monthSubmissions.value = detail.submissions
    monthRideEntries.value = detail.rideEntries
    activeServiceDate.value = detail.submissions[0]?.serviceDate ?? ''
    activeCaseId.value = buildCaseGroups(detail.rideEntries)[0]?.caseId ?? ''
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
    monthDialogVisible.value = false
  } finally {
    monthDetailLoading.value = false
  }
}

onMounted(fetchForms)
</script>

<style scoped>
.driver-report-status-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.text-secondary {
  color: var(--app-text-secondary);
  font-size: 13px;
}

:deep(.report-nowrap-col .cell) {
  white-space: nowrap;
}

:deep(.vehicle-col .cell) {
  white-space: nowrap;
  min-width: 140px;
}

:deep(.months-col .cell) {
  min-width: 260px;
}

:deep(.import-time-col .cell) {
  min-width: 170px;
}

.month-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.months-detail {
  padding: 8px 24px 12px;
}

.month-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 0;
  font-size: 13px;
}

.month-label {
  font-weight: 500;
  min-width: 70px;
}

.month-count {
  color: var(--app-status-success-fg);
}

.empty-state {
  padding: 24px 0;
  line-height: 1.8;
}

.month-tag--clickable {
  cursor: pointer;
}

.month-item--clickable {
  cursor: pointer;
  border-radius: 4px;
}

.month-item--clickable:hover {
  background: var(--app-bg-muted, #f8fafc);
}

.month-detail-summary {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 2px 4px 14px;
  font-size: 13px;
  color: var(--app-text-secondary);
}

.summary-item .summary-value {
  font-weight: 600;
  color: var(--app-text-primary, #1f2933);
  font-size: 15px;
}

.summary-divider {
  width: 1px;
  height: 12px;
  background: var(--app-border-color, #e2e8f0);
}

/* 月份鑽取彈窗左右主從版面：左側依日期／個案分組的清單，右側顯示所選項目的細節，
   取代先前的全螢幕大表格，避免捲動距離過長，也不用切換頁籤才能比對不同天/不同個案 */
.detail-split {
  display: flex;
  gap: 12px;
  height: 60vh;
}

.split-list {
  width: 220px;
  flex-shrink: 0;
  border: 1px solid var(--app-border-color, #e2e8f0);
  border-radius: var(--app-radius-xs, 8px);
  overflow-y: auto;
}

.split-list-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 14px;
  font-size: 13px;
  cursor: pointer;
  border-bottom: 1px solid var(--app-border-light, #f0f2f4);
}

.split-list-row:hover {
  background: var(--app-bg-muted, #f8fafc);
}

.split-list-row.active {
  background: var(--app-status-info-bg, #ddf8fb);
  color: var(--app-status-info-fg, #00788a);
  font-weight: 600;
}

.split-list-badge {
  color: var(--app-text-secondary);
  font-size: 12px;
}

.split-list-row.active .split-list-badge {
  color: inherit;
}

.split-detail {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  border: 1px solid var(--app-border-color, #e2e8f0);
  border-radius: var(--app-radius-xs, 8px);
  padding: 14px 18px;
}

.detail-head {
  display: flex;
  align-items: baseline;
  gap: 10px;
  margin-bottom: 8px;
}

.detail-remark {
  color: var(--app-status-warning-fg);
  font-size: 13px;
  margin: 0 0 10px;
}

.dense-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.dense-table th,
.dense-table td {
  text-align: left;
  padding: 6px 10px;
  border-bottom: 1px solid var(--app-border-light, #f0f2f4);
}

.dense-table th {
  color: var(--app-text-secondary);
  font-weight: 500;
}

/* 逐日回報明細右側細節面板：原始欄位以鍵值對呈現，取代擠在單一儲存格內的長列表。
   每個 .answer-group 對應同一個個案（依序號分組），整組作為一個網格項目，
   確保去程／回程不會被欄位換行拆散到不同區塊。 */
.answers-detail {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 8px 20px;
  align-items: start;
}

.answer-group {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px 10px;
  border-radius: 6px;
  background: var(--app-bg-muted, #f8fafc);
}

.answer-row {
  display: flex;
  gap: 8px;
  font-size: 12px;
  line-height: 1.6;
}

.answer-key {
  flex-shrink: 0;
  color: var(--app-text-secondary);
}

.answer-value {
  color: var(--app-text-primary, #1f2933);
  word-break: break-all;
}
</style>
