<template>
  <div class="dashboard-view" v-loading="loading">
    <!-- 頂部指標統計卡片 -->
    <el-row :gutter="16" class="metrics-row">
      <el-col :span="4">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-content">
            <div class="metric-info">
              <span class="label">在案個案總數</span>
              <span class="value">{{ metrics?.totalCasesCount || 0 }}</span>
            </div>
            <div class="metric-icon bg-primary">
              <el-icon><User /></el-icon>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="5">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-content">
            <div class="metric-info">
              <span class="label">本月已回報趟數</span>
              <span class="value text-success">{{ metrics?.reportedTripsCount || 0 }}</span>
            </div>
            <div class="metric-icon bg-success">
              <el-icon><Van /></el-icon>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="5">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-content">
            <div class="metric-info">
              <span class="label">司機平均請假率</span>
              <span class="value text-warning">
                {{ (metrics?.attendanceDistribution?.leavePercentage || 0).toFixed(1) }}%
              </span>
            </div>
            <div class="metric-icon bg-warning">
              <el-icon><Calendar /></el-icon>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="5">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-content">
            <div class="metric-info">
              <span class="label">待處理混車衝突</span>
              <span class="value" :class="metrics?.pendingConflictsCount ? 'text-danger' : ''">
                {{ metrics?.pendingConflictsCount || 0 }}
              </span>
            </div>
            <div class="metric-icon bg-danger">
              <el-icon><Warning /></el-icon>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="5">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-content">
            <div class="metric-info">
              <span class="label">待對應表單欄位</span>
              <span class="value" :class="metrics?.pendingFormColumnsCount ? 'text-warning' : ''">
                {{ metrics?.pendingFormColumnsCount || 0 }}
              </span>
            </div>
            <div class="metric-icon bg-purple">
              <el-icon><Connection /></el-icon>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 視覺化 ECharts 圖表區 (M3 核心) -->
    <el-row :gutter="16" class="charts-row">
      <!-- 圖表 1: 各車月接送趟數長條圖 -->
      <el-col :span="15">
        <el-card shadow="never" class="chart-card">
          <template #header>
            <div class="chart-header">
              <span class="chart-title">各車當月接送趟數分佈</span>
              <el-tag size="small" type="primary" effect="plain">{{ metrics?.currentMonth }}</el-tag>
            </div>
          </template>
          <div class="chart-container">
            <v-chart class="echart" :option="tripTrendChartOption" autoresize />
          </div>
        </el-card>
      </el-col>

      <!-- 圖表 2: 司機出勤與請假比例環形圖 -->
      <el-col :span="9">
        <el-card shadow="never" class="chart-card">
          <template #header>
            <div class="chart-header">
              <span class="chart-title">車隊出勤與請假狀態</span>
              <el-tag size="small" type="info" effect="plain">人天分佈</el-tag>
            </div>
          </template>
          <div class="chart-container">
            <v-chart class="echart" :option="attendancePieChartOption" autoresize />
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 快捷操作與近期匯出 -->
    <el-row :gutter="16" class="main-sections">
      <el-col :span="8">
        <el-card shadow="never" class="quick-links-card">
          <template #header>
            <span class="section-title">常用快捷功能</span>
          </template>

          <div class="links-grid">
            <el-button
              type="primary"
              plain
              class="quick-btn"
              @click="$router.push('/rides')"
            >
              <el-icon><Grid /></el-icon>
              搭乘月曆矩陣
            </el-button>

            <el-button
              type="warning"
              plain
              class="quick-btn"
              @click="$router.push('/rides/issues')"
            >
              <el-icon><Warning /></el-icon>
              異常集中處理
            </el-button>

            <el-button
              type="success"
              plain
              class="quick-btn"
              @click="$router.push('/exports')"
            >
              <el-icon><Download /></el-icon>
              政府申報匯出
            </el-button>

            <el-button
              type="primary"
              plain
              class="quick-btn"
              @click="$router.push('/reports/hsinchu-schedule')"
            >
              <el-icon><Document /></el-icon>
              新竹接送時刻表
            </el-button>

            <el-button
              type="info"
              plain
              class="quick-btn"
              @click="$router.push('/vehicles/maintenance')"
            >
              <el-icon><Management /></el-icon>
              車輛保養管理
            </el-button>

            <el-button
              type="primary"
              plain
              class="quick-btn"
              @click="$router.push('/attendance')"
            >
              <el-icon><Calendar /></el-icon>
              出勤與油資登錄
            </el-button>
          </div>
        </el-card>
      </el-col>

      <el-col :span="16">
        <el-card shadow="never" class="recent-exports-card">
          <template #header>
            <div class="card-header">
              <span class="section-title">最近申報匯出紀錄</span>
              <el-link type="primary" :underline="false" @click="$router.push('/exports')">
                查看全部
              </el-link>
            </div>
          </template>

          <el-table :data="recentExports" border stripe size="small">
            <el-table-column prop="periodYm" label="申報年月" width="100" />
            <el-table-column prop="region" label="區域" width="80" align="center">
              <template #default="{ row }">
                {{ row.region ? REGION_LABELS[row.region as 'miaoli'|'hsinchu'] : '全區' }}
              </template>
            </el-table-column>
            <el-table-column prop="totalRows" label="總趟數" width="80" align="center" />
            <el-table-column prop="createdAt" label="匯出時間" min-width="150" />
            <el-table-column label="狀態" width="90" align="center">
              <template #default="{ row }">
                <el-tag size="small" :type="row.status === 'succeeded' ? 'success' : 'danger'">
                  {{ EXPORT_STATUS_LABELS[row.status as 'succeeded'|'failed'] || row.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="90" align="center">
              <template #default="{ row }">
                <el-button
                  v-if="row.status === 'succeeded' && row.downloadUrl"
                  link
                  type="primary"
                  size="small"
                  @click="downloadFile(row.downloadUrl)"
                >
                  下載
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-empty
            v-if="!recentExports || recentExports.length === 0"
            description="尚無申報匯出紀錄"
          />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  User,
  Van,
  Calendar,
  Warning,
  Connection,
  Grid,
  Download,
  Document,
  Management
} from '@element-plus/icons-vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { BarChart, PieChart } from 'echarts/charts'
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
  TitleComponent
} from 'echarts/components'

import { getDashboardMetrics } from '@/api/dashboard'
import { getDashboardStats } from '@/api/exports'
import { REGION_LABELS, EXPORT_STATUS_LABELS } from '@/types/domain'
import type { DashboardMetricsDTO, ExportJobDTO } from '@/types/api'

// 註冊 ECharts 元件
use([
  CanvasRenderer,
  BarChart,
  PieChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  TitleComponent
])

const loading = ref(false)
const metrics = ref<DashboardMetricsDTO | null>(null)
const recentExports = ref<ExportJobDTO[]>([])

async function fetchDashboardData() {
  loading.value = true
  try {
    const [mRes, statsRes] = await Promise.allSettled([
      getDashboardMetrics(),
      getDashboardStats()
    ])
    if (mRes.status === 'fulfilled') {
      metrics.value = mRes.value
    }
    if (statsRes.status === 'fulfilled') {
      recentExports.value = statsRes.value.recentExports || []
    }
  } finally {
    loading.value = false
  }
}

// 車輛趟數長條圖設定
const tripTrendChartOption = computed(() => {
  const trends = metrics.value?.vehicleTripTrends || []
  const categories = trends.map(t => t.vehicleName)
  const data = trends.map(t => t.tripCount)

  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '8%',
      top: '8%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: categories,
      axisLabel: { interval: 0, rotate: 20, color: '#666' }
    },
    yAxis: {
      type: 'value',
      name: '趟數',
      axisLabel: { color: '#666' }
    },
    series: [
      {
        name: '搭乘趟數',
        type: 'bar',
        barWidth: '40%',
        data: data,
        itemStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: '#3498db' },
              { offset: 1, color: '#1d5b79' }
            ]
          },
          borderRadius: [4, 4, 0, 0]
        }
      }
    ]
  }
})

// 出勤請假圓餅/環形圖設定
const attendancePieChartOption = computed(() => {
  const att = metrics.value?.attendanceDistribution || {
    workCount: 0,
    leaveCount: 0,
    sickCount: 0,
    offCount: 0
  }

  return {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: {c} 人天 ({d}%)'
    },
    legend: {
      bottom: '2%',
      left: 'center'
    },
    series: [
      {
        name: '出勤分佈',
        type: 'pie',
        radius: ['45%', '70%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 6,
          borderColor: '#fff',
          borderWidth: 2
        },
        label: {
          show: false,
          position: 'center'
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 16,
            fontWeight: 'bold'
          }
        },
        data: [
          { value: att.workCount, name: '正常出勤', itemStyle: { color: '#67c23a' } },
          { value: att.leaveCount, name: '事假', itemStyle: { color: '#e6a23c' } },
          { value: att.sickCount, name: '病假', itemStyle: { color: '#f56c6c' } },
          { value: att.offCount, name: '休假/例假', itemStyle: { color: '#909399' } }
        ]
      }
    ]
  }
})

function downloadFile(url: string) {
  window.open(url, '_blank')
}

onMounted(() => {
  fetchDashboardData()
})
</script>

<style scoped>
.dashboard-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.metrics-row {
  margin-bottom: 2px;
}

.metric-card {
  border-radius: 8px;

  .metric-content {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 4px 0;

    .metric-info {
      display: flex;
      flex-direction: column;
      gap: 4px;

      .label {
        font-size: 13px;
        color: var(--el-text-color-secondary);
        white-space: nowrap;
      }

      .value {
        font-size: 22px;
        font-weight: bold;
        color: var(--el-text-color-primary);

        &.text-success {
          color: var(--el-color-success);
        }
        &.text-warning {
          color: var(--el-color-warning);
        }
        &.text-danger {
          color: var(--el-color-danger);
        }
      }
    }

    .metric-icon {
      width: 44px;
      height: 44px;
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      color: #ffffff;
      font-size: 20px;

      &.bg-primary {
        background-color: var(--el-color-primary);
      }
      &.bg-success {
        background-color: var(--el-color-success);
      }
      &.bg-warning {
        background-color: var(--el-color-warning);
      }
      &.bg-danger {
        background-color: var(--el-color-danger);
      }
      &.bg-purple {
        background-color: #8e44ad;
      }
    }
  }
}

.charts-row {
  margin-top: 4px;
}

.chart-card {
  border-radius: 8px;

  .chart-header {
    display: flex;
    justify-content: space-between;
    align-items: center;

    .chart-title {
      font-size: 15px;
      font-weight: bold;
      color: var(--el-color-primary);
    }
  }

  .chart-container {
    height: 280px;
    width: 100%;

    .echart {
      height: 100%;
      width: 100%;
    }
  }
}

.main-sections {
  margin-top: 4px;
}

.quick-links-card, .recent-exports-card {
  border-radius: 8px;
}

.section-title {
  font-size: 15px;
  font-weight: bold;
  color: var(--el-color-primary);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.links-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;

  .quick-btn {
    width: 100%;
    justify-content: flex-start;
    height: 42px;
    font-size: 13px;
    margin-left: 0 !important;
  }
}
</style>
