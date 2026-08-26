<template>
  <div class="trip-summary-view">
    <!-- 篩選與操作列 -->
    <el-card class="filter-card" shadow="never">
      <el-row :gutter="16" justify="space-between" align="middle">
        <el-col :span="18">
          <div class="filter-wrapper" style="display: flex; gap: 8px; align-items: center;">
            <el-date-picker
              v-model="queryMonth"
              type="month"
              value-format="YYYY-MM"
              placeholder="選擇申報月份"
              style="width: 150px;"
              @change="fetchReport"
            />

            <el-input
              v-model="queryKeyword"
              placeholder="搜尋車輛／車牌／司機"
              clearable
              style="width: 200px;"
              @keyup.enter="fetchReport"
            />

            <el-select
              v-model="queryRegion"
              placeholder="選擇區域"
              clearable
              filterable
              style="width: 140px;"
              @change="fetchReport"
            >
              <el-option
                v-for="(label, key) in REGION_LABELS"
                :key="key"
                :label="label"
                :value="key"
              />
            </el-select>

            <el-select
              v-model="queryVehicle"
              placeholder="指定車輛"
              clearable
              style="width: 150px;"
              @change="fetchReport"
            >
              <el-option
                v-for="v in vehicleOptions"
                :key="v.id"
                :label="v.displayName"
                :value="v.id"
              />
            </el-select>

            <el-button type="primary" icon="Search" @click="fetchReport">
              查詢報表
            </el-button>
            <el-button @click="handleReset">
              重設
            </el-button>
          </div>
        </el-col>
        <el-col :span="6" class="actions-col">
          <el-button
            type="success"
            icon="Download"
            :loading="exporting"
            @click="handleExportExcel"
          >
            匯出 Excel 趟數表
          </el-button>
        </el-col>
      </el-row>
    </el-card>

    <!-- 總覽資料統計卡片 -->
    <el-row :gutter="16">
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-title">全期去程趟數合計</div>
          <div class="stat-number outbound-color">{{ reportData?.grandTotalOutbound || 0 }} <span>趟</span></div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-title">全期回程趟數合計</div>
          <div class="stat-number inbound-color">{{ reportData?.grandTotalInbound || 0 }} <span>趟</span></div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="stat-card highlight">
          <div class="stat-title">全期車輛總趟數</div>
          <div class="stat-number total-color">{{ reportData?.grandTotal || 0 }} <span>趟</span></div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 各車輛趟數明細區塊 -->
    <div v-loading="loading" class="tables-wrapper">
      <template v-if="reportData && reportData.vehicles && reportData.vehicles.length">
        <el-card
          v-for="veh in reportData.vehicles"
          :key="veh.vehicleId"
          class="vehicle-card"
          shadow="never"
        >
          <template #header>
            <div class="veh-header">
              <div class="veh-title">
                <el-icon><Van /></el-icon>
                <span class="veh-name">{{ veh.vehicleName }}</span>
                <el-tag size="small" type="info">{{ veh.plateNo }}</el-tag>
                <span v-if="veh.driverName" class="veh-driver">主要司機：{{ veh.driverName }}</span>
              </div>
              <div class="veh-subtotal">
                小計：去程 <b>{{ veh.subtotalOutbound }}</b> 趟 / 回程 <b>{{ veh.subtotalInbound }}</b> 趟 / 合計 <b class="total-text">{{ veh.subtotalTotal }}</b> 趟
              </div>
            </div>
          </template>

          <el-table :data="veh.rows" border stripe style="width: 100%;">
            <el-table-column prop="caseCode" label="個案編號" width="120" />
            <el-table-column prop="caseName" label="個案姓名" width="140" />
            <el-table-column prop="outboundCount" label="去程趟數" align="right" width="150">
              <template #default="{ row }">
                <span class="count-cell">{{ row.outboundCount }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="inboundCount" label="回程趟數" align="right" width="150">
              <template #default="{ row }">
                <span class="count-cell">{{ row.inboundCount }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="totalCount" label="個人趟數合計" align="right">
              <template #default="{ row }">
                <b class="count-cell total-text">{{ row.totalCount }}</b>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </template>

      <el-empty v-else-if="!loading" description="此月份與篩選條件下無任何搭乘紀錄趟數資料" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Van, Download, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getTripSummaryReport, exportTripSummaryExcel } from '@/api/reports'
import { listVehicles } from '@/api/masters'
import type { TripSummaryReportDTO, VehicleDTO } from '@/types/api'
import { REGION_LABELS } from '@/types/domain'

const queryMonth = ref('2026-07')
const queryKeyword = ref('')
const queryRegion = ref<string | undefined>(undefined)
const queryVehicle = ref<string | undefined>(undefined)
const vehicleOptions = ref<VehicleDTO[]>([])

const loading = ref(false)
const exporting = ref(false)
const reportData = ref<TripSummaryReportDTO | null>(null)

async function fetchVehicleOptions() {
  try {
    const res = await listVehicles()
    vehicleOptions.value = res.data
  } catch (error) {
    // handled
  }
}

async function fetchReport() {
  loading.value = true
  try {
    const res = await getTripSummaryReport({
      periodYm: queryMonth.value,
      region: queryRegion.value,
      vehicleId: queryVehicle.value,
      q: queryKeyword.value || undefined
    })
    reportData.value = res
  } catch (error: any) {
    ElMessage.error(error?.message || '載入車輛趟數表失敗')
  } finally {
    loading.value = false
  }
}

function handleReset() {
  queryKeyword.value = ''
  queryRegion.value = undefined
  queryVehicle.value = undefined
  fetchReport()
}

async function handleExportExcel() {
  exporting.value = true
  try {
    const blob = await exportTripSummaryExcel({
      periodYm: queryMonth.value,
      region: queryRegion.value,
      vehicleId: queryVehicle.value
    })
    const downloadUrl = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = downloadUrl
    link.download = `車輛趟數表-${queryMonth.value}.xlsx`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(downloadUrl)
    ElMessage.success('趟數表已成功匯出！')
  } catch (error: any) {
    ElMessage.error(error?.message || '匯出 Excel 失敗')
  } finally {
    exporting.value = false
  }
}

onMounted(() => {
  fetchVehicleOptions()
  fetchReport()
})
</script>

<style scoped>
.trip-summary-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.filter-card {
  border-radius: 8px;
}

.filter-wrapper {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}

.actions-col {
  display: flex;
  justify-content: flex-end;
}

.stat-card {
  border-radius: 8px;
  text-align: center;
  padding: 12px 0;

  .stat-title {
    font-size: 14px;
    color: var(--el-text-color-secondary);
    margin-bottom: 8px;
  }

  .stat-number {
    font-size: 28px;
    font-weight: bold;

    span {
      font-size: 14px;
      font-weight: normal;
      color: var(--el-text-color-secondary);
      margin-left: 4px;
    }
  }

  .outbound-color {
    color: var(--el-color-primary);
  }

  .inbound-color {
    color: var(--el-color-success);
  }

  .total-color {
    color: #1d5b79;
  }

  &.highlight {
    background: linear-gradient(135deg, #f0f7fa 0%, #e1eff5 100%);
  }
}

.tables-wrapper {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.vehicle-card {
  border-radius: 8px;

  .veh-header {
    display: flex;
    justify-content: space-between;
    align-items: center;

    .veh-title {
      display: flex;
      align-items: center;
      gap: 10px;
      font-size: 16px;
      font-weight: bold;
      color: #1d5b79;

      .veh-driver {
        font-size: 13px;
        font-weight: normal;
        color: var(--el-text-color-secondary);
        margin-left: 8px;
      }
    }

    .veh-subtotal {
      font-size: 14px;
      color: var(--el-text-color-regular);

      .total-text {
        color: var(--el-color-primary);
        font-size: 16px;
      }
    }
  }
}

.count-cell {
  font-family: monospace;
  font-size: 14px;
}

.total-text {
  color: var(--el-color-primary);
}
</style>
