<template>
  <div class="dashboard-view" v-loading="loading">
    <!-- 頂部指標統計卡片 -->
    <el-row :gutter="16" class="metrics-row">
      <el-col :span="6">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-content">
            <div class="metric-info">
              <span class="label">在案個案總數</span>
              <span class="value">{{ stats?.totalCasesCount || 0 }}</span>
            </div>
            <div class="metric-icon bg-primary">
              <el-icon><User /></el-icon>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="6">
        <el-col :span="24">
          <el-card shadow="hover" class="metric-card">
            <div class="metric-content">
              <div class="metric-info">
                <span class="label">本月已回報搭乘趟數</span>
                <span class="value">{{ stats?.reportedTripsCount || 0 }}</span>
              </div>
              <div class="metric-icon bg-success">
                <el-icon><Check /></el-icon>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-col>

      <el-col :span="6">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-content">
            <div class="metric-info">
              <span class="label">待處理混車衝突</span>
              <span class="value text-danger">{{ stats?.pendingConflictsCount || 0 }}</span>
            </div>
            <div class="metric-icon bg-danger">
              <el-icon><Warning /></el-icon>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :span="6">
        <el-card shadow="hover" class="metric-card">
          <div class="metric-content">
            <div class="metric-info">
              <span class="label">待對應表單欄位</span>
              <span class="value text-warning">{{ stats?.pendingFormColumnsCount || 0 }}</span>
            </div>
            <div class="metric-icon bg-warning">
              <el-icon><Connection /></el-icon>
            </div>
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

          <div class="links-list">
            <el-button
              type="primary"
              plain
              class="quick-btn"
              @click="$router.push('/rides')"
            >
              <el-icon><Grid /></el-icon>
              檢視搭乘月曆矩陣
            </el-button>

            <el-button
              type="warning"
              plain
              class="quick-btn"
              @click="$router.push('/rides/issues')"
            >
              <el-icon><Warning /></el-icon>
              處理混車與未回報異常
            </el-button>

            <el-button
              type="success"
              plain
              class="quick-btn"
              @click="$router.push('/exports')"
            >
              <el-icon><Download /></el-icon>
              產生政府申報表 (33欄)
            </el-button>

            <el-button
              type="info"
              plain
              class="quick-btn"
              @click="$router.push('/forms/mappings')"
            >
              <el-icon><Connection /></el-icon>
              維護表單欄位對應
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

          <el-table :data="stats?.recentExports || []" border stripe size="small">
            <el-table-column prop="periodYm" label="申報年月" width="100" />
            <el-table-column prop="region" label="區域" width="80">
              <template #default="{ row }">
                {{ row.region ? REGION_LABELS[row.region as 'miaoli'|'hsinchu'] : '全區' }}
              </template>
            </el-table-column>
            <el-table-column prop="totalRows" label="總趟數" width="80" align="center" />
            <el-table-column prop="createdAt" label="匯出時間" min-width="150" />
            <el-table-column label="狀態" width="90" align="center">
              <template #default="{ row }">
                <el-tag size="small" :type="row.status === 'succeeded' ? 'success' : 'danger'">
                  {{ EXPORT_STATUS_LABELS[row.status as 'succeeded'|'failed'] }}
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
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  User,
  Check,
  Warning,
  Connection,
  Grid,
  Download
} from '@element-plus/icons-vue'
import { getDashboardStats } from '@/api/exports'
import { REGION_LABELS, EXPORT_STATUS_LABELS } from '@/types/domain'
import type { DashboardStatsDTO } from '@/types/api'

const loading = ref(false)
const stats = ref<DashboardStatsDTO | null>(null)

async function fetchStats() {
  loading.value = true
  try {
    stats.value = await getDashboardStats()
  } finally {
    loading.value = false
  }
}

function downloadFile(url: string) {
  window.open(url, '_blank')
}

onMounted(() => {
  fetchStats()
})
</script>

<style scoped>
.dashboard-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.metrics-row {
  margin-bottom: 4px;
}

.metric-card {
  border-radius: 8px;

  .metric-content {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 6px 0;

    .metric-info {
      display: flex;
      flex-direction: column;
      gap: 4px;

      .label {
        font-size: 13px;
        color: var(--el-text-color-secondary);
      }

      .value {
        font-size: 24px;
        font-weight: bold;
        color: var(--el-text-color-primary);

        &.text-danger {
          color: var(--el-color-danger);
        }
        &.text-warning {
          color: var(--el-color-warning);
        }
      }
    }

    .metric-icon {
      width: 48px;
      height: 48px;
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      color: #ffffff;
      font-size: 22px;

      &.bg-primary {
        background-color: var(--el-color-primary);
      }
      &.bg-success {
        background-color: var(--el-color-success);
      }
      &.bg-danger {
        background-color: var(--el-color-danger);
      }
      &.bg-warning {
        background-color: var(--el-color-warning);
      }
    }
  }
}

.main-sections {
  margin-top: 8px;
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

.links-list {
  display: flex;
  flex-direction: column;
  gap: 12px;

  .quick-btn {
    width: 100%;
    justify-content: flex-start;
    height: 44px;
    font-size: 14px;
    margin-left: 0 !important;
  }
}
</style>
