<template>
  <div class="missing-rides-view">
    <el-tabs v-model="activeTab" type="border-card">
      <!-- 分頁一：未回報清單 -->
      <el-tab-pane label="未回報清單" name="missing">
        <DataTablePage
          :loading="loadingMissing"
          :total="missingTotal"
          :page="page"
          :page-size="pageSize"
          @page-change="onPageChange"
          @size-change="onSizeChange"
        >
          <template #filter>
            <el-select
              v-model="selectedVehicle"
              placeholder="篩選車輛"
              clearable
              style="width: 180px;"
              @change="fetchMissingRides"
            >
              <el-option
                v-for="v in vehicles"
                :key="v.id"
                :label="v.displayName"
                :value="v.id"
              />
            </el-select>
          </template>

          <template #actions>
            <el-button
              type="warning"
              icon="Bell"
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
                  <el-tag :type="row.direction === 'outbound' ? 'primary' : 'success'" size="small">
                    {{ (DIRECTION_LABELS as any)[row.direction] || row.direction }}
                  </el-tag>
                  <span style="margin-left: 6px; font-size: 13px;">第 {{ row.legSeq }} 趟</span>
                </template>
              </el-table-column>
              <el-table-column prop="departTime" label="排定出發時間" width="130" />
              <el-table-column prop="vehicleName" label="負責車輛" width="130">
                <template #default="{ row }">
                  <el-tag effect="plain" type="info">{{ row.vehicleName || '未指定' }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="driverName" label="司機" width="120" />
              <el-table-column label="逾期天數" width="120" align="center">
                <template #default="{ row }">
                  <el-tag
                    :type="(row.daysOverdue || 0) >= 3 ? 'danger' : 'warning'"
                    effect="dark"
                    size="small"
                  >
                    逾期 {{ row.daysOverdue || 0 }} 天
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="狀態" min-width="120">
                <template #default>
                  <el-tag type="danger" size="small">司機未提交回覆</el-tag>
                </template>
              </el-table-column>
            </el-table>
          </template>
        </DataTablePage>
      </el-tab-pane>

      <!-- 分頁二：催報通知歷史 -->
      <el-tab-pane label="催報通知歷史" name="history">
        <DataTablePage
          :loading="loadingLogs"
          :total="logsTotal"
          :page="logPage"
          :page-size="logPageSize"
          @page-change="onLogPageChange"
          @size-change="onLogSizeChange"
        >
          <template #filter>
            <el-select
              v-model="selectedTopic"
              placeholder="通知類型"
              clearable
              style="width: 180px;"
              @change="fetchNotificationLogs"
            >
              <el-option
                v-for="(label, key) in NOTIFICATION_TOPIC_LABELS"
                :key="key"
                :label="label"
                :value="key"
              />
            </el-select>
          </template>

          <template #actions>
            <el-button icon="Refresh" @click="fetchNotificationLogs">
              重新整理
            </el-button>
          </template>

          <template #table>
            <el-table :data="logList" stripe border style="width: 100%;">
              <el-table-column prop="sentAt" label="發送時間" width="170" sortable />
              <el-table-column label="主題" width="140">
                <template #default="{ row }">
                  <el-tag type="info">{{ (NOTIFICATION_TOPIC_LABELS as any)[row.topic] || row.topic }}</el-tag>
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
              <el-table-column prop="triggeredByName" label="觸發來源" width="180" />
              <el-table-column label="狀態" width="100" align="center">
                <template #default="{ row }">
                  <el-tag :type="row.status === 'sent' || row.success ? 'success' : 'danger'" effect="dark" size="small">
                    {{ row.status === 'sent' || row.success ? '發送成功' : '失敗' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="errorMessage" label="備註 / 失敗原因" min-width="180">
                <template #default="{ row }">
                  <span v-if="row.errorMessage || row.error" style="color: var(--el-color-danger);">{{ row.errorMessage || row.error }}</span>
                  <span v-else class="text-secondary">{{ row.contentSummary || row.body || '-' }}</span>
                </template>
              </el-table-column>
            </el-table>
          </template>


        </DataTablePage>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import DataTablePage from '@/components/DataTablePage.vue'
import { listMissingRides } from '@/api/rides'
import { listVehicles } from '@/api/masters'
import { listNotificationLogs, triggerMissingReportsCheck } from '@/api/notifications'
import type { MissingRideDTO, NotificationLogDTO, VehicleDTO } from '@/types/api'
import { DIRECTION_LABELS, NOTIFICATION_TOPIC_LABELS } from '@/types/domain'

const activeTab = ref<'missing' | 'history'>('missing')

// 未回報清單資料
const missingList = ref<MissingRideDTO[]>([])
const loadingMissing = ref(false)
const missingTotal = ref(0)
const page = ref(1)
const pageSize = ref(20)
const selectedVehicle = ref<string | undefined>(undefined)
const vehicles = ref<VehicleDTO[]>([])

// 通知歷史資料
const logList = ref<NotificationLogDTO[]>([])
const loadingLogs = ref(false)
const logsTotal = ref(0)
const logPage = ref(1)
const logPageSize = ref(20)
const selectedTopic = ref<string | undefined>(undefined)

const triggering = ref(false)

async function fetchVehicles() {
  try {
    const res = await listVehicles()
    vehicles.value = res.data
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
      vehicleId: selectedVehicle.value
    })
    missingList.value = res.data
    missingTotal.value = res.meta?.total || res.data.length
  } finally {
    loadingMissing.value = false
  }
}

async function fetchNotificationLogs() {
  loadingLogs.value = true
  try {
    const res = await listNotificationLogs({
      page: logPage.value,
      pageSize: logPageSize.value,
      topic: selectedTopic.value
    })
    logList.value = res.data
    logsTotal.value = res.meta?.total || res.data.length
  } finally {
    loadingLogs.value = false
  }
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

onMounted(() => {
  fetchVehicles()
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

.text-secondary {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
</style>
