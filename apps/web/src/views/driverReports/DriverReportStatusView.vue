<template>
  <div class="driver-report-status-view">
    <DataTablePage
      title="接送匯報總覽"
      :description="`共 ${forms.length} 台車，顯示各車已有對應資料的月份`"
      :max-width="700"
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
        table-layout="auto"
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

        <el-table-column label="最後匯入時間" width="170" align="center">
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

    <!-- 單一月份匯入資料鑽取彈窗：逐日回報明細與逐個案搭乘紀錄兩個頁籤 -->
    <el-dialog
      v-model="monthDialogVisible"
      :title="monthDialogTitle"
      width="min(920px, calc(100vw - 32px))"
      destroy-on-close
    >
      <el-tabs v-model="monthDialogTab">
        <el-tab-pane label="逐日回報明細" name="submissions">
          <div v-loading="monthDetailLoading" class="month-detail-scroll-body">
            <el-empty v-if="!monthDetailLoading && !monthSubmissions.length" description="這個月沒有逐日回報資料" />
            <el-table v-else :data="monthSubmissions" border stripe table-layout="auto" style="width: 100%">
              <el-table-column label="服務日期" prop="serviceDate" width="120" />
              <el-table-column label="駕駛人（原始）" prop="driverNameRaw" width="140" />
              <el-table-column label="備註" prop="remark" min-width="140" />
              <el-table-column label="原始欄位內容" min-width="280">
                <template #default="{ row }">
                  <div class="answers-list">
                    <span v-for="(value, header) in row.answers" :key="header" class="answer-item">
                      {{ header }}：{{ value }}
                    </span>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <el-tab-pane label="逐個案搭乘紀錄" name="rideEntries">
          <div v-loading="monthDetailLoading" class="month-detail-scroll-body">
            <el-empty v-if="!monthDetailLoading && !monthRideEntries.length" description="這個月沒有個案搭乘紀錄" />
            <el-table v-else :data="monthRideEntries" border stripe table-layout="auto" style="width: 100%">
              <el-table-column label="個案" prop="caseName" min-width="120" />
              <el-table-column label="趟次" width="130">
                <template #default="{ row }">
                  {{ legLabel(row.legSeq) }}
                </template>
              </el-table-column>
              <el-table-column label="服務日期" prop="serviceDate" width="120" />
              <el-table-column label="回報結果" width="100">
                <template #default="{ row }">
                  {{ row.reported === 'boarded' ? '有搭乘' : '未搭乘' }}
                </template>
              </el-table-column>
              <el-table-column label="駕駛人" prop="driverName" min-width="120" />
            </el-table>
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
import { resolveErrorMessage } from '@/api/errorCodes'
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

function legLabel(legSeq: number): string {
  return LEG_SEQ_OPTIONS.find((opt) => opt.value === legSeq)?.label ?? `第 ${legSeq} 趟`
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
  } catch (error) {
    const code = (error as { response?: { data?: { error?: { code?: string } } } })?.response?.data?.error?.code
    ElMessage.error(resolveErrorMessage(code, '載入該月匯入資料失敗'))
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

:deep(.vehicle-col .cell) {
  white-space: nowrap;
  min-width: 140px;
}

:deep(.months-col .cell) {
  min-width: 260px;
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

/* 鑽取彈窗內的表格捲動區，避免長清單撐開整個對話框 */
.month-detail-scroll-body {
  max-height: 420px;
  overflow-y: auto;
}

.answers-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 12px;
}

.answer-item {
  color: var(--app-text-secondary);
}
</style>
