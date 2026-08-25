<template>
  <div class="ride-calendar-view">
    <!-- 篩選與月份切換列 -->
    <el-card shadow="never" class="filter-card">
      <el-row :gutter="16" justify="space-between" align="middle">
        <el-col :span="18" class="filter-inputs">
          <div class="month-picker-wrapper">
            <span class="label">查詢月份：</span>
            <el-date-picker
              v-model="selectedDate"
              type="month"
              format="YYYY-MM"
              value-format="YYYY-MM"
              placeholder="選擇月份"
              style="width: 140px"
              :clearable="false"
              @change="fetchMatrix"
            />
            <span class="roc-badge">
              {{ formatRocMonthLabel(currentRocMonth) }}
            </span>
          </div>

          <el-select
            v-model="regionFilter"
            placeholder="全部區域"
            clearable
            style="width: 130px"
            @change="fetchMatrix"
          >
            <el-option label="全部區域" value="" />
            <el-option label="苗栗" value="miaoli" />
            <el-option label="新竹" value="hsinchu" />
          </el-select>

          <el-input
            v-model="searchQuery"
            placeholder="搜尋個案姓名／編號"
            clearable
            style="width: 200px"
            @keyup.enter="fetchMatrix"
          />

          <el-button type="primary" @click="fetchMatrix">查詢</el-button>
        </el-col>

        <el-col :span="6" class="actions-col">
          <el-button type="warning" plain @click="$router.push('/rides/issues')">
            <el-icon><Warning /></el-icon>
            異常集中處理 (衝突/未回報)
          </el-button>
        </el-col>
      </el-row>
    </el-card>

    <!-- 狀態圖例說明列 -->
    <el-card shadow="never" class="legend-card">
      <div class="legend-items">
        <span class="legend-title">搭乘圖例：</span>
        <span class="legend-item"><span class="mark status-boarded">√</span> 有坐 (Boarded)</span>
        <span class="legend-item"><span class="mark status-absent">／</span> 沒坐 (Absent)</span>
        <span class="legend-item"><span class="mark status-unreported">?</span> 未回報 (Unreported)</span>
        <span class="legend-item"><span class="mark status-conflict">!</span> 混車衝突 (Conflict)</span>
        <span class="legend-item"><span class="mark is-corrected">●</span> 已人工更正 (帶右上圓點)</span>
        <span class="legend-item"><span class="mark status-non-scheduled"></span> 非應搭日</span>
      </div>
    </el-card>

    <!-- 搭乘矩陣表格 -->
    <el-card shadow="never" class="matrix-card">
      <el-table
        :data="matrixData?.cases || []"
        border
        v-loading="loading"
        height="calc(100vh - 300px)"
        style="width: 100%"
        class="calendar-table"
      >
        <!-- 凍結左側個案資訊欄位 -->
        <el-table-column
          prop="caseName"
          label="個案姓名"
          width="110"
          fixed="left"
        >
          <template #default="{ row }">
            <el-link
              type="primary"
              :underline="false"
              @click="$router.push(`/cases/${row.caseId}`)"
            >
              {{ row.caseName }}
            </el-link>
          </template>
        </el-table-column>

        <el-table-column
          prop="region"
          label="區域"
          width="70"
          fixed="left"
          align="center"
        >
          <template #default="{ row }">
            <el-tag size="small" :type="row.region === 'miaoli' ? 'warning' : 'primary'">
              {{ REGION_LABELS[row.region as 'miaoli' | 'hsinchu'] }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column
          prop="tripPattern"
          label="趟數"
          width="70"
          fixed="left"
          align="center"
        >
          <template #default="{ row }">
            <span>{{ row.tripPattern }} 趟</span>
          </template>
        </el-table-column>

        <!-- 當月動態日期欄位 (1 ~ 31 日) -->
        <el-table-column
          v-for="day in daysInMonth"
          :key="day"
          :label="`${day}`"
          min-width="46"
          align="center"
        >
          <template #default="{ row }">
            <template v-if="getCell(row, day)">
              <!-- 該日若是應搭日 -->
              <div v-if="getCell(row, day).isExpected" class="cell-legs-container">
                <el-tooltip
                  v-for="record in getCell(row, day).records"
                  :key="record.id"
                  placement="top"
                  :show-after="300"
                >
                  <template #content>
                    <div class="cell-tooltip">
                      <div><strong>{{ record.serviceDate }} 第 {{ record.legSeq }} 趟 ({{ record.direction === 'outbound' ? '去程' : '回程' }})</strong></div>
                      <div>實際狀態：{{ record.effectiveStatus === 'boarded' ? '有坐' : (record.effectiveStatus === 'absent' ? '沒坐' : '未回報') }}</div>
                      <div>車輛：{{ record.vehicleName || '預設車輛' }}</div>
                      <div>司機：{{ record.driverName || '主要司機' }}</div>
                      <div v-if="record.hasConflict" style="color: #ff4d4f;">⚠️ 發現跨車回報衝突！</div>
                      <div v-if="record.correctedAt" style="color: #409eff;">✏️ 已由 {{ record.correctedByName }} 於 {{ record.correctedAt }} 更正</div>
                    </div>
                  </template>

                  <div
                    class="calendar-cell"
                    :class="[
                      `status-${record.hasConflict ? 'conflict' : record.effectiveStatus}`,
                      { 'is-corrected': !!record.correctedAt }
                    ]"
                    @click="openCorrection(record)"
                  >
                    {{ record.hasConflict ? '!' : (record.effectiveStatus === 'boarded' ? '√' : (record.effectiveStatus === 'absent' ? '／' : '?')) }}
                  </div>
                </el-tooltip>
              </div>

              <!-- 該日為非應搭日或假日 -->
              <div
                v-else
                class="calendar-cell status-non-scheduled"
                :title="getCell(row, day).holidayName || '非排定搭乘日'"
              />
            </template>
            <div v-else class="calendar-cell status-non-scheduled" />
          </template>


        </el-table-column>
      </el-table>
    </el-card>

    <!-- 搭乘紀錄更正抽屜面板 -->
    <RideCorrectionDrawer
      ref="drawerRef"
      @updated="fetchMatrix"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Warning } from '@element-plus/icons-vue'
import RideCorrectionDrawer from './RideCorrectionDrawer.vue'
import { getRideCalendarMatrix } from '@/api/rides'
import { useRocMonth } from '@/composables/useRocMonth'
import { REGION_LABELS } from '@/types/domain'
import type { RideCalendarMatrixDTO, CaseRideCalendarRowDTO, RideRecordDTO } from '@/types/api'

const { toRocMonth, formatRocMonthLabel } = useRocMonth()

const selectedDate = ref<string>('2026-07')
const regionFilter = ref<string>('')
const searchQuery = ref<string>('')
const loading = ref(false)
const matrixData = ref<RideCalendarMatrixDTO | null>(null)
const drawerRef = ref<InstanceType<typeof RideCorrectionDrawer>>()

const currentRocMonth = computed(() => {
  return toRocMonth(selectedDate.value)
})

const daysInMonth = computed(() => {
  if (!selectedDate.value) return 31
  const [year, month] = selectedDate.value.split('-').map(Number)
  return new Date(year, month, 0).getDate()
})

async function fetchMatrix() {
  loading.value = true
  try {
    const res = await getRideCalendarMatrix({
      month: currentRocMonth.value,
      region: regionFilter.value || undefined,
      q: searchQuery.value || undefined
    })
    matrixData.value = res
  } finally {
    loading.value = false
  }
}

function getCell(row: any, day: number) {
  const dayKey = `${selectedDate.value}-${String(day).padStart(2, '0')}`
  return row.days[dayKey]
}

function openCorrection(record: RideRecordDTO) {
  drawerRef.value?.open(record)
}

onMounted(() => {
  fetchMatrix()
})
</script>

<style scoped>
.ride-calendar-view {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.filter-card, .legend-card, .matrix-card {
  border-radius: 8px;
}

.filter-inputs {
  display: flex;
  align-items: center;
  gap: 16px;

  .month-picker-wrapper {
    display: flex;
    align-items: center;
    gap: 8px;

    .label {
      font-size: 14px;
      font-weight: 500;
    }

    .roc-badge {
      font-weight: bold;
      color: var(--el-color-primary);
      font-size: 15px;
    }
  }
}

.actions-col {
  display: flex;
  justify-content: flex-end;
}

.legend-card {
  padding: 4px 0;

  .legend-items {
    display: flex;
    align-items: center;
    gap: 16px;
    flex-wrap: wrap;
    font-size: 13px;

    .legend-title {
      font-weight: bold;
      color: var(--el-text-color-primary);
    }

    .legend-item {
      display: inline-flex;
      align-items: center;
      gap: 6px;

      .mark {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 22px;
        height: 22px;
        border-radius: 4px;
        font-weight: bold;
        font-size: 13px;
      }
    }
  }
}

.matrix-card {
  padding: 0;
}

.cell-legs-container {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.cell-tooltip {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
}
</style>
