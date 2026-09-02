<template>
  <div class="missing-rides-view">
    <el-tabs v-model="activeTab" type="border-card">
      <!-- 分頁一：未回報清單 -->
      <el-tab-pane label="未回報清單" name="missing">
        <DataTablePage
          :max-width="1090"
          :loading="loadingMissing"
          :total="missingTotal"
          :page="page"
          :page-size="pageSize"
          @page-change="onPageChange"
          @size-change="onSizeChange"
        >
          <template #filter>
            <el-input
              v-model="missingQuery"
              placeholder="搜尋個案姓名／車輛／司機"
              clearable
              style="width: 240px;"
              @keyup.enter="fetchMissingRides"
            />

            <el-select
              v-model="selectedVehicle"
              placeholder="篩選車輛"
              clearable
              style="width: 160px;"
              @change="fetchMissingRides"
            >
              <el-option
                v-for="v in vehicles"
                :key="v.id"
                :label="v.displayName"
                :value="v.id"
              />
            </el-select>

            <el-button type="primary" @click="fetchMissingRides">
              查詢
            </el-button>
            <el-button @click="handleResetMissing">
              重設
            </el-button>
          </template>

          <template #actions>
            <el-button
              type="warning"
              plain
              :loading="triggering"
              @click="handleTriggerNotify"
            >
              立即執行未回報催報
            </el-button>
          </template>

          <template #table>
            <el-table :data="missingList" stripe border style="width: 100%;">
              <el-table-column prop="serviceDate" label="服務日期" width="120" sortable />
              <el-table-column prop="caseName" label="個案姓名" width="120" />
              <el-table-column label="方向 / 趟次" width="130">
                <template #default="{ row }">
                  <span>{{ (DIRECTION_LABELS as any)[row.direction] || row.direction }}</span>
                  <span style="margin-left: 6px; font-size: 13px;">第 {{ row.legSeq }} 趟</span>
                </template>
              </el-table-column>
              <el-table-column prop="departTime" label="排定出發時間" width="130" />
              <el-table-column prop="vehicleName" label="負責車輛" width="130">
                <template #default="{ row }">
                  <span>{{ row.vehicleName || '未指定' }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="driverName" label="司機" width="120" />
              <el-table-column label="逾期天數" width="120" align="center">
                <template #default="{ row }">
                  <span
                    class="overdue-days"
                    :class="(row.daysOverdue || 0) >= 3 ? 'is-danger' : 'is-warning'"
                  >
                    逾期 {{ row.daysOverdue || 0 }} 天
                  </span>
                </template>
              </el-table-column>
              <el-table-column
                v-if="authStore.can('staff')"
                label="操作"
                width="120"
                align="center"
                fixed="right"
              >
                <template #default="{ row }">
                  <TableRowActions>
                    <el-button
                      link
                      type="primary"
                      size="small"
                      @click="openReportDialog(row)"
                    >
                      人工回報
                    </el-button>
                  </TableRowActions>
                </template>
              </el-table-column>
            </el-table>
          </template>
        </DataTablePage>
      </el-tab-pane>

      <!-- 分頁二：催報通知歷史 -->
      <el-tab-pane label="催報通知歷史" name="history">
        <DataTablePage
          :max-width="1400"
          :loading="loadingLogs"
          :total="logsTotal"
          :page="logPage"
          :page-size="logPageSize"
          @page-change="onLogPageChange"
          @size-change="onLogSizeChange"
        >
          <template #filter>
            <el-input
              v-model="logQuery"
              placeholder="搜尋標題／收件人／來源"
              clearable
              style="width: 240px;"
              @keyup.enter="fetchNotificationLogs"
            />

            <el-select
              v-model="selectedTopic"
              placeholder="通知類型"
              clearable
              style="width: 160px;"
              @change="fetchNotificationLogs"
            >
              <el-option
                v-for="(label, key) in NOTIFICATION_TOPIC_LABELS"
                :key="key"
                :label="label"
                :value="key"
              />
            </el-select>

            <el-button type="primary" @click="fetchNotificationLogs">
              查詢
            </el-button>
            <el-button @click="handleResetLogs">
              重設
            </el-button>
          </template>

          <template #actions>
            <el-button plain @click="fetchNotificationLogs">
              重新整理
            </el-button>
          </template>

          <template #table>
            <el-table :data="logList" stripe border table-layout="auto" style="width: 100%;">
              <el-table-column prop="sentAt" label="發送時間" width="170" sortable align="center">
                <template #default="{ row }">
                  <span>{{ formatDateTime(row.sentAt) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="主題" min-width="140">
                <template #default="{ row }">
                  <span class="topic-label">{{ (NOTIFICATION_TOPIC_LABELS as any)[row.topic] || row.topic }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="subject" label="信件標題" min-width="220" show-overflow-tooltip />
              <el-table-column label="收件人清單" min-width="200" show-overflow-tooltip>
                <template #default="{ row }">
                  <span v-if="row.recipientEmails && row.recipientEmails.length">
                    {{ row.recipientEmails.join(', ') }}
                  </span>
                  <span v-else-if="row.target">{{ row.target }}</span>
                  <el-tag v-else type="danger" size="small">無設定收件人</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="triggeredByName" label="觸發來源" min-width="140" class-name="trigger-source-col" />
              <el-table-column label="狀態" width="100" align="center">
                <template #default="{ row }">
                  <el-tag :type="row.status === 'sent' || row.success ? 'success' : 'danger'" size="small">
                    {{ row.status === 'sent' || row.success ? '發送成功' : '失敗' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="errorMessage" label="備註 / 失敗原因" min-width="180" show-overflow-tooltip>
                <template #default="{ row }">
                  <span v-if="row.errorMessage || row.error" style="color: var(--app-status-danger-fg);">{{ row.errorMessage || row.error }}</span>
                  <span v-else class="text-secondary">{{ row.contentSummary || row.body || '-' }}</span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="90" align="center" fixed="right">
                <template #default="{ row }">
                  <TableRowActions>
                    <el-button
                      link
                      type="info"
                      size="small"
                      @click="openLogDetail(row)"
                    >
                      詳情
                    </el-button>
                  </TableRowActions>
                </template>
              </el-table-column>
            </el-table>
          </template>
        </DataTablePage>
      </el-tab-pane>
    </el-tabs>

    <!-- 通知歷史詳情對話框 -->
    <el-dialog
      v-model="logDetailVisible"
      title="通知發送詳情"
      width="min(820px, calc(100vw - 32px))"
      destroy-on-close
    >
      <el-descriptions v-if="currentLogRow" class="log-detail-descriptions" :column="1" border size="small">
        <el-descriptions-item label="發送時間">{{ formatDateTime(currentLogRow.sentAt) }}</el-descriptions-item>
        <el-descriptions-item label="主題">
          {{ (NOTIFICATION_TOPIC_LABELS as any)[currentLogRow.topic] || currentLogRow.topic }}
        </el-descriptions-item>
        <el-descriptions-item label="通知管道">{{ currentLogRow.channel }}</el-descriptions-item>
        <el-descriptions-item label="信件標題">{{ currentLogRow.subject }}</el-descriptions-item>
        <el-descriptions-item label="收件人清單">
          <span v-if="currentLogRow.recipientEmails && currentLogRow.recipientEmails.length">
            {{ currentLogRow.recipientEmails.join(', ') }}
          </span>
          <span v-else-if="(currentLogRow as any).target">{{ (currentLogRow as any).target }}</span>
          <el-tag v-else type="danger" size="small">無設定收件人</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="觸發來源">{{ currentLogRow.triggeredByName || '-' }}</el-descriptions-item>
        <el-descriptions-item label="狀態">
          <el-tag :type="currentLogRow.status === 'sent' || (currentLogRow as any).success ? 'success' : 'danger'" size="small">
            {{ currentLogRow.status === 'sent' || (currentLogRow as any).success ? '發送成功' : '失敗' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item v-if="currentLogRow.errorMessage || (currentLogRow as any).error" label="失敗原因">
          <span style="color: var(--app-status-danger-fg);">{{ currentLogRow.errorMessage || (currentLogRow as any).error }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="完整內容">
          <div class="log-detail-content" :class="{ 'is-expanded': logContentExpanded }">
            {{ currentLogRow.contentSummary || (currentLogRow as any).body || '-' }}
          </div>
          <el-button
            v-if="isLogContentLong"
            link
            type="primary"
            size="small"
            class="log-detail-toggle"
            @click="logContentExpanded = !logContentExpanded"
          >
            {{ logContentExpanded ? '收合' : '展開完整內容' }}
          </el-button>
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>

    <!-- 人工輸入回報內容對話框 -->
    <el-dialog
      v-model="reportDialogVisible"
      :title="`人工輸入回報 — ${currentReportRow?.caseName || ''} (${currentReportRow?.serviceDate || ''})`"
      width="min(600px, calc(100vw - 32px))"
      destroy-on-close
    >
      <div v-if="currentReportRow" class="report-dialog-body dialog-scroll-form">
        <el-descriptions :column="2" border size="small" class="mb-3">
          <el-descriptions-item label="個案姓名">{{ currentReportRow.caseName }}</el-descriptions-item>
          <el-descriptions-item label="服務日期">{{ currentReportRow.serviceDate }}</el-descriptions-item>
          <el-descriptions-item label="方向與趟次">
            <span>{{ (DIRECTION_LABELS as any)[currentReportRow.direction] || currentReportRow.direction }}</span>
            <span style="margin-left: 6px;">第 {{ currentReportRow.legSeq }} 趟</span>
          </el-descriptions-item>
          <el-descriptions-item label="排定出發時間">{{ currentReportRow.departTime || '-' }}</el-descriptions-item>
        </el-descriptions>

        <el-form
          ref="reportFormRef"
          :model="reportForm"
          label-width="110px"
          label-position="right"
          style="margin-top: 16px;"
        >
          <el-form-item label="實際搭乘狀態" required>
            <el-radio-group v-model="reportForm.effectiveStatus">
              <el-radio value="boarded">有坐</el-radio>
              <el-radio value="absent">沒坐</el-radio>
            </el-radio-group>
          </el-form-item>

          <el-form-item label="實際承載車輛">
            <el-select
              v-model="reportForm.vehicleId"
              :placeholder="isReportAbsent ? '沒坐無須選擇車輛' : '請選擇承載車輛'"
              :disabled="isReportAbsent"
              filterable
              clearable
              style="width: 100%;"
            >
              <el-option
                v-for="v in vehicles"
                :key="v.id"
                :label="`${v.displayName} (${v.plateNo})`"
                :value="v.id"
              />
            </el-select>
          </el-form-item>

          <el-form-item label="實際駕駛司機">
            <el-select
              v-model="reportForm.driverId"
              :placeholder="isReportAbsent ? '沒坐無須選擇司機' : '請選擇駕駛司機'"
              :disabled="isReportAbsent"
              filterable
              clearable
              style="width: 100%;"
            >
              <el-option
                v-for="d in drivers"
                :key="d.id"
                :label="d.name"
                :value="d.id"
              />
            </el-select>
          </el-form-item>

          <el-form-item label="出發時間更正">
            <el-time-picker
              v-model="reportForm.departTimeOverride"
              format="HH:mm"
              value-format="HH:mm"
              :placeholder="isReportAbsent ? '沒坐無出發時間' : '預設沿用排班時間'"
              :disabled="isReportAbsent"
              style="width: 100%;"
            />
          </el-form-item>

          <el-form-item label="服務時長 (分)">
            <el-input-number
              v-model="reportForm.durationMinOverride"
              :min="1"
              :max="240"
              :placeholder="isReportAbsent ? '沒坐無服務時長' : '預設 10 分鐘'"
              :disabled="isReportAbsent"
              style="width: 100%;"
            />
          </el-form-item>

          <el-form-item label="不申報 AA09">
            <el-switch
              v-model="reportForm.notClaimedAa09"
              :disabled="isReportAbsent"
            />
          </el-form-item>

          <el-form-item label="回報備註 / 原因">
            <div class="reason-quick-tags">
              <el-tag
                v-for="reason in QUICK_REPORT_REASONS"
                :key="reason"
                class="reason-tag"
                effect="plain"
                @click="reportForm.reason = reason"
              >
                {{ reason }}
              </el-tag>
            </div>
            <el-input
              v-model="reportForm.reason"
              type="textarea"
              :rows="2"
              placeholder="可填寫口頭回報說明、補登原因等（選填）"
              style="margin-top: 6px;"
            />
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <DialogFooter :loading="savingReport" @confirm="handleSubmitReport" @cancel="reportDialogVisible = false" />
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Edit } from '@element-plus/icons-vue'
import DialogFooter from '@/components/DialogFooter.vue'
import DataTablePage from '@/components/DataTablePage.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import { formatDateTime } from '@/utils/formatters'
import { listMissingRides, submitManualRideReport } from '@/api/rides'
import { listVehicles, listDrivers } from '@/api/masters'
import { listNotificationLogs, triggerMissingReportsCheck } from '@/api/notifications'
import { useAuthStore } from '@/stores/auth'
import type { MissingRideDTO, NotificationLogDTO, VehicleDTO, DriverDTO, ManualReportRideRequest } from '@/types/api'
import { DIRECTION_LABELS, NOTIFICATION_TOPIC_LABELS } from '@/types/domain'

const authStore = useAuthStore()
const route = useRoute()
const activeTab = ref<'missing' | 'history'>('missing')

// 未回報清單資料
const missingQuery = ref('')
const missingList = ref<MissingRideDTO[]>([])
const loadingMissing = ref(false)
const missingTotal = ref(0)
const page = ref(1)
const pageSize = ref(20)
const selectedVehicle = ref<string | undefined>(undefined)
const vehicles = ref<VehicleDTO[]>([])
const drivers = ref<DriverDTO[]>([])

// 人工回報對話框狀態
const reportDialogVisible = ref(false)
const savingReport = ref(false)
const currentReportRow = ref<MissingRideDTO | null>(null)

const QUICK_REPORT_REASONS = [
  '司機口頭回報',
  '電話確認已搭乘',
  '司機忘記送出表單',
  '家屬確認已接送',
  '代班司機回報',
  '請假補登'
]

const reportForm = reactive<ManualReportRideRequest>({
  caseId: '',
  serviceDate: '',
  legSeq: 1,
  effectiveStatus: 'boarded',
  vehicleId: '',
  driverId: '',
  departTimeOverride: null,
  durationMinOverride: 10,
  notClaimedAa09: false,
  reason: '司機口頭回報'
})

const isReportAbsent = computed(() => reportForm.effectiveStatus === 'absent')

watch(
  () => reportForm.effectiveStatus,
  (newStatus, oldStatus) => {
    if (newStatus === 'absent') {
      reportForm.vehicleId = ''
      reportForm.driverId = ''
      reportForm.departTimeOverride = null
      reportForm.durationMinOverride = null
      reportForm.notClaimedAa09 = false
    } else if (oldStatus === 'absent' && newStatus === 'boarded' && currentReportRow.value) {
      reportForm.vehicleId = currentReportRow.value.vehicleId || ''
      if (currentReportRow.value.driverName && drivers.value.length > 0) {
        const matched = drivers.value.find((d) => d.name === currentReportRow.value?.driverName)
        reportForm.driverId = matched ? matched.id : ''
      }
      reportForm.departTimeOverride = currentReportRow.value.departTime || null
      reportForm.durationMinOverride = 10
      reportForm.notClaimedAa09 = false
    }
  }
)

// 通知歷史資料
const logQuery = ref('')
const logList = ref<NotificationLogDTO[]>([])
const loadingLogs = ref(false)
const logsTotal = ref(0)
const logPage = ref(1)
const logPageSize = ref(20)
const selectedTopic = ref<string | undefined>(undefined)

const triggering = ref(false)

// 通知歷史詳情對話框狀態
const logDetailVisible = ref(false)
const currentLogRow = ref<NotificationLogDTO | null>(null)
const logContentExpanded = ref(false)
const isLogContentLong = computed(() => {
  const content = currentLogRow.value?.contentSummary || (currentLogRow.value as any)?.body || ''
  return content.length > 120
})

function openLogDetail(row: any) {
  currentLogRow.value = row
  logContentExpanded.value = false
  logDetailVisible.value = true
}

async function fetchVehicles() {
  try {
    const res = await listVehicles({ active: true, pageSize: 100 })
    vehicles.value = res.data
  } catch (error) {
    // handled by interceptor
  }
}

async function fetchDrivers() {
  try {
    const res = await listDrivers({ active: true, pageSize: 100 })
    drivers.value = res.data
  } catch (error) {
    // handled by interceptor
  }
}

async function fetchMissingRides() {
  loadingMissing.value = true
  try {
    const res = await listMissingRides({
      page: page.value,
      pageSize: pageSize.value,
      vehicleId: selectedVehicle.value,
      q: missingQuery.value || undefined
    })
    missingList.value = res.data || []
    missingTotal.value = res.meta?.total || res.data?.length || 0
  } finally {
    loadingMissing.value = false
  }
}

function handleResetMissing() {
  missingQuery.value = ''
  selectedVehicle.value = undefined
  page.value = 1
  fetchMissingRides()
}

async function fetchNotificationLogs() {
  loadingLogs.value = true
  try {
    const res = await listNotificationLogs({
      page: logPage.value,
      pageSize: logPageSize.value,
      topic: selectedTopic.value,
      q: logQuery.value || undefined
    })
    logList.value = res.data
    logsTotal.value = res.meta?.total || res.data.length
  } finally {
    loadingLogs.value = false
  }
}

function handleResetLogs() {
  logQuery.value = ''
  selectedTopic.value = undefined
  logPage.value = 1
  fetchNotificationLogs()
}

function onPageChange(p: number) {
  page.value = p
  fetchMissingRides()
}

function onSizeChange(size: number) {
  pageSize.value = size
  page.value = 1
  fetchMissingRides()
}

function onLogPageChange(p: number) {
  logPage.value = p
  fetchNotificationLogs()
}

function onLogSizeChange(size: number) {
  logPageSize.value = size
  logPage.value = 1
  fetchNotificationLogs()
}

async function openReportDialog(row: any) {
  currentReportRow.value = row
  reportForm.id = row.id
  reportForm.caseId = row.caseId
  reportForm.serviceDate = row.serviceDate
  reportForm.legSeq = row.legSeq
  reportForm.effectiveStatus = 'boarded'
  reportForm.vehicleId = row.vehicleId || ''

  // 自動依司機姓名配對 driverId
  if (row.driverName && drivers.value.length > 0) {
    const matched = drivers.value.find((d) => d.name === row.driverName)
    reportForm.driverId = matched ? matched.id : ''
  } else {
    reportForm.driverId = ''
  }

  reportForm.departTimeOverride = row.departTime || null
  reportForm.durationMinOverride = 10
  reportForm.notClaimedAa09 = false
  reportForm.reason = '司機口頭回報'

  if (drivers.value.length === 0) {
    await fetchDrivers()
    if (row.driverName) {
      const matched = drivers.value.find((d) => d.name === row.driverName)
      if (matched) reportForm.driverId = matched.id
    }
  }

  reportDialogVisible.value = true
}

async function handleSubmitReport() {
  if (!currentReportRow.value) return

  savingReport.value = true
  try {
    await submitManualRideReport({
      id: reportForm.id,
      caseId: reportForm.caseId,
      serviceDate: reportForm.serviceDate,
      legSeq: reportForm.legSeq,
      effectiveStatus: reportForm.effectiveStatus,
      vehicleId: isReportAbsent.value ? undefined : (reportForm.vehicleId || undefined),
      driverId: isReportAbsent.value ? undefined : (reportForm.driverId || undefined),
      departTimeOverride: isReportAbsent.value ? undefined : (reportForm.departTimeOverride || undefined),
      durationMinOverride: isReportAbsent.value ? undefined : (reportForm.durationMinOverride || undefined),
      notClaimedAa09: isReportAbsent.value ? false : reportForm.notClaimedAa09,
      reason: reportForm.reason || undefined
    })

    ElMessage.success('已成功補登回報內容')
    reportDialogVisible.value = false
    await fetchMissingRides()
  } catch (err: any) {
    ElMessage.error(err?.message || '儲存回報失敗')
  } finally {
    savingReport.value = false
  }
}

async function handleTriggerNotify() {
  try {
    await ElMessageBox.confirm(
      '確定要立即執行未回報檢核並發送催報通知？此動作會寄發電子郵件給通知收件人。',
      '執行確認',
      {
        confirmButtonText: '確定發送',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    triggering.value = true
    const res = await triggerMissingReportsCheck()
    ElMessage.success(res.message || '未回報催報通知已成功送出！')
    await fetchNotificationLogs()
  } catch (err: any) {
    if (err !== 'cancel') {
      ElMessage.error(err?.message || '發送失敗')
    }
  } finally {
    triggering.value = false
  }
}

// 前置檢核報告的「查看未回報」會帶 ?q=個案姓名 進來，預填搜尋條件讓清單直接只剩該個案
onMounted(() => {
  const q = route.query.q
  if (typeof q === 'string' && q) missingQuery.value = q

  fetchVehicles()
  fetchDrivers()
  fetchMissingRides()
  fetchNotificationLogs()
})
</script>

<style scoped>
.missing-rides-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.log-detail-descriptions :deep(.el-descriptions__body .el-descriptions__table .el-descriptions__cell) {
  font-size: var(--app-font-md);
}

.log-detail-content {
  max-height: 120px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-word;
}

.log-detail-content.is-expanded {
  max-height: none;
}

.log-detail-toggle {
  margin-top: 4px;
}

.text-secondary {
  color: var(--app-text-secondary);
  font-size: 13px;
}

.topic-label {
  white-space: nowrap;
}

:deep(.trigger-source-col .cell) {
  white-space: nowrap;
}

.overdue-days {
  font-weight: 600;
  font-size: var(--app-font-sm);

  &.is-danger {
    color: var(--app-status-danger-fg);
  }

  &.is-warning {
    color: var(--app-status-warning-fg);
  }
}

.report-dialog-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.reason-quick-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.reason-tag {
  cursor: pointer;
  user-select: none;
  &:hover {
    background-color: var(--app-primary-light);
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
