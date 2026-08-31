<template>
  <div class="dashboard-view" v-loading="loading" aria-labelledby="dashboard-title">
    <div v-if="loading" class="sr-only" role="status" aria-live="polite">正在載入儀表板資料。</div>
    <el-alert
      v-if="metricsError || exportsError"
      type="error"
      :closable="false"
      show-icon
      role="alert"
      class="dashboard-error"
    >
      <template #title>
        {{ metricsError && exportsError ? '儀表板資料載入失敗。' : metricsError ? '儀表板指標載入失敗。' : '最近匯出紀錄載入失敗。' }}
      </template>
      <div class="dashboard-error-actions">
        <span>請重新載入後再試一次。</span>
        <el-button type="danger" plain size="small" :loading="loading" @click="fetchDashboardData">重新載入</el-button>
      </div>
    </el-alert>
    <div class="dashboard-heading">
      <div>
        <h1 id="dashboard-title">營運總覽</h1>
      </div>
      <el-button type="primary" @click="$router.push('/rides')">
        <el-icon><Grid /></el-icon>
        開啟搭乘月曆
      </el-button>
    </div>

    <!-- 頂部 5 大 KPI 指標卡 (含 Sparklines 微趨勢圖) -->
    <el-row :gutter="16" class="metrics-row">
      <!-- 1. 在案個案總數 -->
      <el-col :xs="24" :sm="12" :md="8" :lg="4" :xl="4">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-header">
            <span class="metric-label">在案個案總數</span>
          </div>
          <div class="metric-body">
            <div class="metric-val-group">
              <span class="metric-value">{{ metrics?.totalCasesCount || 0 }}</span>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 2. 本月已回報趟數 -->
      <el-col :xs="24" :sm="12" :md="8" :lg="5" :xl="5">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-header">
            <span class="metric-label">本月已回報趟數</span>
          </div>
          <div class="metric-body">
            <div class="metric-val-group">
              <span class="metric-value">{{ metrics?.reportedTripsCount || 0 }}</span>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 3. 司機平均請假率 -->
      <el-col :xs="24" :sm="12" :md="8" :lg="5" :xl="5">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-header">
            <span class="metric-label">司機平均請假率</span>
          </div>
          <div class="metric-body">
            <div class="metric-val-group">
              <span class="metric-value">
                {{ (metrics?.attendanceDistribution?.leavePercentage || 0).toFixed(1) }}%
              </span>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 4. 待處理混車衝突 (Urgent Alert) -->
      <el-col :xs="24" :sm="12" :md="8" :lg="5" :xl="5">
        <el-card shadow="hover" class="metric-card" :class="{ 'is-urgent': (metrics?.pendingConflictsCount || 0) > 0 }">
          <div class="metric-header">
            <span class="metric-label">待處理混車衝突</span>
          </div>
          <div class="metric-body">
            <div class="metric-val-group">
              <span class="metric-value" :class="metrics?.pendingConflictsCount ? 'text-danger font-urgent' : ''">
                {{ metrics?.pendingConflictsCount || 0 }}
              </span>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 5. 待對應表單欄位 -->
      <el-col :xs="24" :sm="12" :md="8" :lg="5" :xl="5">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-header">
            <span class="metric-label">待對應表單欄位</span>
          </div>
          <div class="metric-body">
            <div class="metric-val-group">
              <span class="metric-value">
                {{ metrics?.pendingFormColumnsCount || 0 }}
              </span>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 中間區域：趨勢圖表與出勤分佈 -->
    <el-row :gutter="16" class="charts-row">
      <!-- 左側：乘載度進度條 + 長條圖 (15 cols) -->
      <el-col :xs="24" :lg="15">
        <div class="left-charts-column">
          <!-- 圖表 1: 各車月接送趟數長條圖 -->
          <el-card shadow="never" class="chart-card">
            <template #header>
              <div class="chart-header">
                <div class="header-title-group">
                  <span class="chart-title">各車當月接送趟數分佈</span>
                </div>
                <el-tag size="small" type="primary" effect="light">{{ metrics?.currentMonth }}</el-tag>
              </div>
            </template>
            <p id="trip-trend-description" class="sr-only">{{ tripTrendChartAriaLabel }}</p>
            <div v-if="metrics && metrics.vehicleTripTrends?.length" class="chart-container" role="img" aria-labelledby="trip-trend-description">
              <v-chart class="echart" :option="tripTrendChartOption" autoresize />
            </div>
            <el-empty v-else description="目前沒有接送趟數資料。" />
          </el-card>
        </div>
      </el-col>

      <!-- 右側：出勤分佈 -->
      <el-col :xs="24" :lg="9">
        <div class="right-charts-column">
          <!-- 圖表 2: 司機出勤與請假狀態環形圖 -->
          <el-card shadow="never" class="chart-card">
            <template #header>
              <div class="chart-header">
                <div class="header-title-group">
                  <span class="chart-title">車隊出勤與請假狀態</span>
                </div>
              </div>
            </template>
            <p id="attendance-chart-description" class="sr-only">{{ attendancePieChartAriaLabel }}</p>
            <div v-if="metrics && attendanceTotal > 0" class="chart-container-donut" role="img" aria-labelledby="attendance-chart-description">
              <v-chart class="echart" :option="attendancePieChartOption" autoresize />
            </div>
            <el-empty v-else description="目前沒有出勤資料。" />
          </el-card>

        </div>
      </el-col>
    </el-row>

    <!-- 底部：快捷操作與近期匯出紀錄 -->
    <el-row :gutter="16" class="main-sections">
      <el-col :xs="24" :lg="8">
        <el-card shadow="never" class="quick-links-card">
          <template #header>
            <div class="header-title-group">
              <span class="section-title">常用快捷功能</span>
            </div>
          </template>

          <div class="links-grid">
            <el-button
              plain
              class="quick-btn"
              @click="$router.push('/rides')"
            >
              <span>搭乘月曆表</span>
            </el-button>

            <el-button
              plain
              class="quick-btn"
              @click="$router.push('/rides/issues')"
            >
              <span>異常集中處理</span>
            </el-button>

            <el-button
              plain
              class="quick-btn"
              @click="$router.push('/exports')"
            >
              <span>政府申報匯出</span>
            </el-button>

            <el-button
              plain
              class="quick-btn"
              @click="$router.push('/reports/hsinchu-schedule')"
            >
              <span>新竹接送時刻表</span>
            </el-button>

            <el-button
              plain
              class="quick-btn"
              @click="$router.push('/vehicles/maintenance')"
            >
              <span>車輛保養管理</span>
            </el-button>

            <el-button
              plain
              class="quick-btn"
              @click="$router.push('/attendance')"
            >
              <span>出勤與油資登錄</span>
            </el-button>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :lg="16">
        <el-card shadow="never" class="recent-exports-card">
          <template #header>
            <div class="card-header">
              <div class="header-title-group">
                <span class="section-title">最近申報匯出紀錄</span>
              </div>
              <el-link type="primary" :underline="false" class="view-all-link" @click="$router.push('/exports')">
                查看全部 <el-icon><ArrowRight /></el-icon>
              </el-link>
            </div>
          </template>

          <el-table :data="recentExports" stripe size="default" class="modern-table">
            <el-table-column prop="periodYm" label="申報年月" width="110">
              <template #default="{ row }">
                <span class="font-mono-bold">{{ row.periodYm }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="region" label="區域" width="100" align="center">
              <template #default="{ row }">
                <span>{{ row.region ? (REGION_LABELS[row.region] || row.region) : '全區' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="totalRows" label="總趟數" width="90" align="center">
              <template #default="{ row }">
                <span class="trip-count-badge">{{ row.totalRows }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="createdAt" label="匯出時間" min-width="170" align="center">
              <template #default="{ row }">
                <span class="text-muted">{{ formatDateTime(row.createdAt) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="狀態" width="110" align="center">
              <template #default="{ row }">
                <span v-if="row.status === 'succeeded'" class="status-badge badge-routine">
                  <span class="status-indicator-dot dot-routine"></span>已完成
                </span>
                <span v-else class="status-badge badge-critical">
                  <span class="status-indicator-dot dot-critical"></span>失敗
                </span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="90" align="center">
              <template #default="{ row }">
                <el-button
                  v-if="row.status === 'succeeded' && row.downloadUrl"
                  link
                  type="primary"
                  size="small"
                  class="action-download-btn"
                  @click="downloadFile(row.downloadUrl, row.fileName)"
                >
                  <el-icon><Download /></el-icon> 下載
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
import { ElMessage } from 'element-plus'
import {
  Grid,
  Download,
  ArrowRight
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
import { downloadExportFile, getDashboardStats } from '@/api/exports'
import { formatDateTime } from '@/utils/formatters'
import { downloadBlob } from '@/utils/download'
import { REGION_LABELS } from '@/types/domain'
import type { DashboardMetricsDTO, ExportJobDTO } from '@/types/api'

// 註冊 ECharts 核心元件
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
const metricsError = ref(false)
const exportsError = ref(false)

// 獲取總覽儀表板指標與最近匯出紀錄
async function fetchDashboardData() {
  loading.value = true
  metricsError.value = false
  exportsError.value = false
  metrics.value = null
  recentExports.value = []
  try {
    const [mRes, statsRes] = await Promise.allSettled([
      getDashboardMetrics(),
      getDashboardStats()
    ])
    if (mRes.status === 'fulfilled') {
      metrics.value = mRes.value
    } else {
      metricsError.value = true
    }
    if (statsRes.status === 'fulfilled') {
      recentExports.value = statsRes.value.recentExports || []
    } else {
      exportsError.value = true
    }
  } finally {
    loading.value = false
  }
}

const attendanceTotal = computed(() => {
  const att = metrics.value?.attendanceDistribution
  return att ? att.workCount + att.leaveCount + att.sickCount + att.offCount : 0
})

const tripTrendChartAriaLabel = computed(() => {
  const trends = metrics.value?.vehicleTripTrends || []
  if (!trends.length) return '各車當月接送趟數分佈，目前沒有資料。'
  return `各車當月接送趟數分佈：${trends.map(item => `${item.vehicleName}${item.tripCount} 趟`).join('、')}。`
})

const attendancePieChartAriaLabel = computed(() => {
  const att = metrics.value?.attendanceDistribution
  if (!att || attendanceTotal.value === 0) return '車隊出勤與請假狀態，目前沒有資料。'
  return `車隊出勤與請假狀態：出勤 ${att.workCount} 人、請假 ${att.leaveCount} 人、病假 ${att.sickCount} 人、休假 ${att.offCount} 人，共 ${attendanceTotal.value} 人。`
})

// 車輛趟數長條圖設定
const tripTrendChartOption = computed(() => {
  const trends = metrics.value?.vehicleTripTrends || []
  const categories = trends.map(t => t.vehicleName)
  const data = trends.map(t => t.tripCount)

  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(15, 23, 42, 0.9)',
      borderColor: '#334155',
      textStyle: { color: '#ffffff', fontSize: 12 },
      axisPointer: { type: 'shadow', shadowStyle: { color: 'rgba(255, 90, 95, 0.06)' } },
      padding: [10, 14],
      borderRadius: 8
    },
    grid: {
      left: '2%',
      right: '2%',
      bottom: '6%',
      top: '12%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: categories,
      axisLine: { lineStyle: { color: '#e2e8f0' } },
      axisTick: { show: false },
      axisLabel: { interval: 0, rotate: 15, color: '#64748b', fontSize: 11.5, fontWeight: 500 }
    },
    yAxis: {
      type: 'value',
      name: '趟數',
      nameTextStyle: { color: '#94a3b8', fontSize: 11 },
      axisLine: { show: false },
      splitLine: { lineStyle: { color: '#f1f5f9', type: 'dashed' } },
      axisLabel: { color: '#94a3b8', fontSize: 11 }
    },
    series: [
      {
        name: '搭乘趟數',
            type: 'bar',
        barWidth: '34%',
        data: data,
        itemStyle: {
          color: '#2f6fed',
          borderRadius: [6, 6, 0, 0]
        },
        emphasis: {
          itemStyle: {
            color: '#1f55c7'
          }
        }
      }
    ]
  }
})

// 出勤請假環形圖設定
const attendancePieChartOption = computed(() => {
  const att = metrics.value?.attendanceDistribution || {
    workCount: 0,
    leaveCount: 0,
    sickCount: 0,
    offCount: 0
  }

  const total = att.workCount + att.leaveCount + att.sickCount + att.offCount

  return {
    tooltip: {
      trigger: 'item',
      backgroundColor: 'rgba(15, 23, 42, 0.9)',
      borderColor: '#334155',
      textStyle: { color: '#ffffff', fontSize: 12 },
      formatter: '{b}: {c} 人天 ({d}%)',
      padding: [8, 12],
      borderRadius: 8
    },
    legend: {
      bottom: '0%',
      left: 'center',
      icon: 'circle',
      itemWidth: 8,
      itemHeight: 8,
      textStyle: { color: '#64748b', fontSize: 11.5 }
    },
    title: {
      text: `${total}`,
      subtext: '總人天',
      left: 'center',
      top: '36%',
      textStyle: { fontSize: 22, fontWeight: 'bold', color: '#0f172a' },
      subtextStyle: { fontSize: 11, color: '#94a3b8' }
    },
    series: [
      {
        name: '出勤分佈',
        type: 'pie',
        radius: ['52%', '74%'],
        center: ['50%', '45%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 8,
          borderColor: '#ffffff',
          borderWidth: 3
        },
        label: {
          show: false
        },
        data: [
          { value: att.workCount, name: '正常出勤', itemStyle: { color: '#10b981' } },
          { value: att.leaveCount, name: '事假', itemStyle: { color: '#f59e0b' } },
          { value: att.sickCount, name: '病假', itemStyle: { color: '#ef4444' } },
          { value: att.offCount, name: '休假/例假', itemStyle: { color: '#94a3b8' } }
        ]
      }
    ]
  }
})

async function downloadFile(url: string, filename?: string) {
  try {
    const blob = await downloadExportFile(url)
    downloadBlob(blob, filename || '申報匯出.xlsx')
    ElMessage.success('申報檔案下載成功')
  } catch (error: any) {
    ElMessage.error(error?.message || '下載檔案失敗')
  }
}

onMounted(() => {
  fetchDashboardData()
})
</script>

<style scoped lang="scss">
.dashboard-view {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.dashboard-error { margin-bottom: 16px; }
.dashboard-error-actions { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.sr-only {
  position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px;
  overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0;
}

.dashboard-heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  padding: 2px 2px 4px;

  h1 { font-size: 24px; line-height: 1.25; font-weight: 700; letter-spacing: -0.02em; }
}

/* 頂部 KPI 卡片樣式 */
.metrics-row {
  margin-bottom: 0;
}

.metric-card {
  border-radius: var(--app-radius-md);
  border: 1px solid var(--app-border-color);
  background-color: #ffffff;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;

  &:hover {
    border-color: var(--app-border-color);
    box-shadow: var(--app-shadow-md);
    transform: translateY(-2px);
  }

  &.is-urgent {
    border-color: #fecaca;
    background: var(--app-status-danger-bg);
  }

  :deep(.el-card__body) {
    padding: 16px 18px 12px;
  }

  .metric-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 10px;

    .metric-label {
      font-size: 11px;
      font-weight: 700;
      color: var(--app-text-secondary);
      letter-spacing: 0.05em;
      text-transform: uppercase;
    }

  }

  .metric-body {
    display: flex;
    flex-direction: column;
    gap: 8px;

    .metric-val-group {
      display: flex;
      align-items: baseline;
      justify-content: space-between;

      .metric-value {
        font-size: 24px;
        font-weight: 700;
        color: var(--app-text-primary);
        letter-spacing: -0.02em;
        line-height: 1;

        &.text-danger {
          color: var(--app-status-danger-fg);
        }
        &.font-urgent {
          font-weight: 800;
        }
      }
    }

  }

}

.status-badge {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 9999px;
  display: inline-flex;
  align-items: center;

  &.badge-critical {
    background-color: var(--app-status-danger-bg);
    color: var(--app-status-danger-fg);
  }

  &.badge-routine {
    background-color: var(--app-status-success-bg);
    color: var(--app-status-success-fg);
  }
}

.left-charts-column {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.right-charts-column {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.chart-card {
  border-radius: var(--app-radius-md);

  .chart-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .chart-container {
    height: 270px;
    width: 100%;

    .echart {
      height: 100%;
      width: 100%;
    }
  }

  .chart-container-donut {
    height: 230px;
    width: 100%;

    .echart {
      height: 100%;
      width: 100%;
    }
  }
}

.header-title-group {
  display: flex;
  align-items: center;
  gap: 8px;

  .section-title,
  .chart-title {
    font-size: 14.5px;
    font-weight: 700;
    color: var(--app-text-primary);
  }
}

.card-header-flex,
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.quick-links-card,
.recent-exports-card {
  border-radius: var(--app-radius-md);
}

.links-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;

    .quick-btn {
    width: 100%;
    justify-content: flex-start;
    height: 46px;
    font-size: 13px;
    font-weight: 600;
    margin-left: 0 !important;
    padding: 0 12px;
    border-radius: 10px;
      gap: 10px;
      transition: transform 0.18s ease-out, box-shadow 0.18s ease-out, background-color 0.18s ease-out;

      &:hover { transform: translateY(-2px); box-shadow: 0 5px 12px rgba(16, 21, 34, 0.07); }
      &:active { transform: translateY(0) scale(0.98); }

  }
}

.view-all-link {
  font-size: 12.5px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 4px;
}

.modern-table {
  .font-mono-bold {
    font-family: 'JetBrains Mono', monospace;
    font-weight: 600;
    color: var(--app-text-primary);
  }

  .trip-count-badge {
    font-weight: 600;
    color: var(--app-text-primary);
  }

  .text-muted {
    color: var(--app-text-secondary);
    font-size: 12.5px;
  }

  .action-download-btn {
    font-weight: 600;
  }
}

@media (max-width: 640px) {
  .dashboard-heading { align-items: flex-start; flex-direction: column; }
  .dashboard-heading .el-button { width: 100%; }
  .dashboard-heading h1 { font-size: 21px; }
}

@media (prefers-reduced-motion: reduce) {
  .metric-card, .alert-feed-item, .quick-btn { transition: none !important; }
  .metric-card:hover, .alert-feed-item:hover, .quick-btn:hover { transform: none; }
}
</style>
