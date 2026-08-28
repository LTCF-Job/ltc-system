<template>
  <div class="hsinchu-schedule-view" v-loading="loading">
    <!-- 頂部操作與篩選列 (列印時隱藏) -->
    <el-card shadow="never" class="filter-card no-print">
      <div class="header-content">
        <div class="title-section">
          <h2 class="page-title">新竹接送時刻表</h2>
          <span class="subtitle">依趟序與預計出發時間排定之新竹區長照交通接送順序</span>
        </div>

        <div class="action-section">
          <el-input
            v-model="searchQuery"
            placeholder="搜尋個案姓名／代碼／地址"
            clearable
            style="width: 220px"
            @keyup.enter="fetchSchedule"
          />

          <el-select
            v-model="selectedSiteId"
            placeholder="全部據點"
            clearable
            style="width: 150px"
            @change="fetchSchedule"
          >
            <el-option
              v-for="site in sites"
              :key="site.id"
              :label="site.name"
              :value="site.id"
            />
          </el-select>

          <el-select
            v-model="selectedVehicleId"
            placeholder="全部車輛"
            clearable
            style="width: 150px"
            @change="fetchSchedule"
          >
            <el-option
              v-for="veh in vehicles"
              :key="veh.id"
              :label="veh.displayName"
              :value="veh.id"
            />
          </el-select>

          <el-button type="primary" icon="Search" @click="fetchSchedule">
            查詢
          </el-button>
          <el-button @click="handleReset">
            重設
          </el-button>

          <el-button type="success" @click="handleExportExcel" :loading="exporting">
            <el-icon><Download /></el-icon>
            匯出 Excel
          </el-button>

          <el-button type="info" @click="handlePrint">
            <el-icon><Printer /></el-icon>
            列印 (A4 橫式)
          </el-button>
        </div>
      </div>
    </el-card>

    <!-- 列印專用報表標頭 (僅在列印與預覽時呈現) -->
    <div class="print-header">
      <h1 class="print-title">長照交通接送 新竹區搭車順序時刻表</h1>
      <div class="print-meta">
        <span>產生時間：{{ formatDateTime(scheduleData?.generatedAt) }}</span>
        <span>區域：新竹區</span>
      </div>
    </div>

    <!-- 去程時段表格 -->
    <el-card shadow="never" class="schedule-card">
      <template #header>
        <div class="section-badge bg-outbound">
          <el-icon><Right /></el-icon>
          <span>去程接送順序</span>
        </div>
      </template>

      <el-table
        :data="scheduleData?.outbound || []"
        border
        stripe
        size="small"
        :cell-class-name="getCellClass"
      >
        <el-table-column label="趟次" width="90" align="center">
          <template #default="{ row }">
            <span>第 {{ row.runNo }} 趟</span>
          </template>
        </el-table-column>
        <el-table-column prop="caseCode" label="個案編號" width="100" />
        <el-table-column prop="caseName" label="個案姓名" width="120">
          <template #default="{ row }">
            <span class="font-bold">{{ row.caseName }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="note" label="備註" min-width="140">
          <template #default="{ row }">
            {{ row.note || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="departTime" label="出發時間" width="95" align="center">
          <template #default="{ row }">
            <span>{{ row.departTime }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="origin" label="出發地 (住家)" min-width="180" />
        <el-table-column prop="arriveTime" label="抵達時間" width="95" align="center">
          <template #default="{ row }">
            {{ row.arriveTime || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="destination" label="目的地 (據點)" min-width="180" />
        <el-table-column prop="vehicleName" label="承接車輛" width="110" align="center" />
      </el-table>
      <el-empty
        v-if="!scheduleData?.outbound || scheduleData.outbound.length === 0"
        description="無去程排班資料"
      />
    </el-card>

    <!-- 回程時段表格 -->
    <el-card shadow="never" class="schedule-card mt-4">
      <template #header>
        <div class="section-badge bg-inbound">
          <el-icon><Back /></el-icon>
          <span>回程接送順序</span>
        </div>
      </template>

      <el-table
        :data="scheduleData?.inbound || []"
        border
        stripe
        size="small"
        :cell-class-name="getCellClass"
      >
        <el-table-column label="趟次" width="90" align="center">
          <template #default="{ row }">
            <span>第 {{ row.runNo }} 趟</span>
          </template>
        </el-table-column>
        <el-table-column prop="caseCode" label="個案編號" width="100" />
        <el-table-column prop="caseName" label="個案姓名" width="120">
          <template #default="{ row }">
            <span class="font-bold">{{ row.caseName }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="note" label="備註" min-width="140">
          <template #default="{ row }">
            {{ row.note || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="departTime" label="出發時間" width="95" align="center">
          <template #default="{ row }">
            <span>{{ row.departTime }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="origin" label="出發地 (據點)" min-width="180" />
        <el-table-column prop="arriveTime" label="抵達時間" width="95" align="center">
          <template #default="{ row }">
            {{ row.arriveTime || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="destination" label="目的地 (住家)" min-width="180" />
        <el-table-column prop="vehicleName" label="承接車輛" width="110" align="center" />
      </el-table>
      <el-empty
        v-if="!scheduleData?.inbound || scheduleData.inbound.length === 0"
        description="無回程排班資料"
      />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  Refresh,
  Download,
  Printer,
  Right,
  Back
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { resolveErrorMessage } from '@/api/errorCodes'
import { getHsinchuSchedule, exportHsinchuScheduleExcel } from '@/api/reports'
import { listSites, listVehicles } from '@/api/masters'
import { formatDateTime } from '@/utils/formatters'
import type { HsinchuScheduleReportDTO, SiteDTO, VehicleDTO } from '@/types/api'
import { downloadBlob } from '@/utils/download'

const loading = ref(false)
const exporting = ref(false)
const scheduleData = ref<HsinchuScheduleReportDTO | null>(null)

const sites = ref<SiteDTO[]>([])
const vehicles = ref<VehicleDTO[]>([])
const searchQuery = ref('')
const selectedSiteId = ref<string>()
const selectedVehicleId = ref<string>()

async function fetchFilterOptions() {
  try {
    const [siteRes, vehRes] = await Promise.all([
      listSites({ pageSize: 100 }),
      listVehicles({ pageSize: 100 })
    ])
    sites.value = siteRes.data.filter(s => s.region === 'hsinchu')
    vehicles.value = vehRes.data.filter(v => v.region === 'hsinchu')
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '載入篩選條件失敗'))
  }
}

async function fetchSchedule() {
  loading.value = true
  try {
    scheduleData.value = await getHsinchuSchedule({
      siteId: selectedSiteId.value,
      vehicleId: selectedVehicleId.value,
      q: searchQuery.value || undefined
    })
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '查詢新竹接送時刻表失敗'))
  } finally {
    loading.value = false
  }
}

function handleReset() {
  searchQuery.value = ''
  selectedSiteId.value = undefined
  selectedVehicleId.value = undefined
  fetchSchedule()
}

async function handleExportExcel() {
  exporting.value = true
  try {
    const blob = await exportHsinchuScheduleExcel({
      siteId: selectedSiteId.value,
      vehicleId: selectedVehicleId.value
    })
    downloadBlob(blob, 'hsinchu-schedule.xlsx')
    ElMessage.success('時刻表 Excel 匯出成功')
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '匯出時刻表失敗'))
  } finally {
    exporting.value = false
  }
}

function handlePrint() {
  window.print()
}

function getCellClass() {
  return ''
}

onMounted(async () => {
  await fetchFilterOptions()
  await fetchSchedule()
})
</script>

<style scoped>
.hsinchu-schedule-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;

  .title-section {
    .page-title {
      font-size: 18px;
      font-weight: bold;
      color: var(--el-text-color-primary);
      margin: 0 0 4px 0;
    }

    .subtitle {
      font-size: 13px;
      color: var(--el-text-color-secondary);
    }
  }

  .action-section {
    display: flex;
    align-items: center;
    gap: 10px;
  }
}

.section-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  border-radius: 6px;
  font-weight: 600;
  font-size: 13.5px;
  background-color: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  border: 1px solid var(--el-border-color-light);

  &.bg-outbound {
    border-left: 3px solid var(--el-color-primary);
  }

  &.bg-inbound {
    border-left: 3px solid var(--el-color-warning);
  }
}

.print-header {
  display: none;
}

.font-bold {
  font-weight: 600;
}

.mt-4 {
  margin-top: 16px;
}

/* A4 橫式列印專用樣式規範 */
@media print {
  @page {
    size: A4 landscape;
    margin: 10mm;
  }

  :global(body) {
    background-color: #ffffff !important;
  }

  :global(.aside-menu),
  :global(.layout-header),
  .no-print {
    display: none !important;
  }

  .print-header {
    display: block;
    text-align: center;
    margin-bottom: 12px;

    .print-title {
      font-size: 18px;
      font-weight: bold;
      margin: 0 0 6px 0;
    }

    .print-meta {
      font-size: 12px;
      display: flex;
      justify-content: space-between;
      color: #666;
    }
  }

  .schedule-card {
    border: none !important;
    box-shadow: none !important;
    margin-bottom: 16px;
  }

  :global(.el-table) {
    font-size: 11px !important;
    border: 1px solid #333 !important;
  }

  :global(.el-table th),
  :global(.el-table td) {
    padding: 4px !important;
    color: #000 !important;
    border-color: #333 !important;
  }
}
</style>
