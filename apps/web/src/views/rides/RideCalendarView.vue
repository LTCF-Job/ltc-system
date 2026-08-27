<template>
  <div class="ride-calendar-view">
    <!-- 篩選與月份切換列 -->
    <el-card shadow="never" class="filter-card">
      <el-row :gutter="16" justify="space-between" align="middle">
        <el-col :span="18" class="filter-inputs">
          <div class="month-picker-wrapper">
            <span class="label">查詢月份：</span>
            <el-button
              :icon="ArrowLeft"
              circle
              size="small"
              title="上一月"
              aria-label="上一月"
              @click="changeMonth(-1)"
            />
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
            <el-button
              :icon="ArrowRight"
              circle
              size="small"
              title="下一月"
              aria-label="下一月"
              @click="changeMonth(1)"
            />
          </div>

          <el-select
            v-model="regionFilter"
            placeholder="全部區域"
            clearable
            filterable
            style="width: 140px"
            @change="fetchMatrix"
          >
            <el-option label="全部區域" value="" />
            <el-option
              v-for="(label, key) in REGION_LABELS"
              :key="key"
              :label="label"
              :value="key"
            />
          </el-select>

          <el-input
            v-model="searchQuery"
            placeholder="搜尋個案姓名／編號"
            clearable
            style="width: 200px"
            @keyup.enter="fetchMatrix"
          />

          <el-button type="primary" :icon="Search" @click="fetchMatrix">查詢</el-button>
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
        <span class="legend-item"><span class="mark status-boarded"><el-icon><Check /></el-icon></span> 有坐 (Boarded)</span>
        <span class="legend-item"><span class="mark status-absent"><el-icon><Close /></el-icon></span> 沒坐 (Absent)</span>
        <span class="legend-item"><span class="mark status-unreported"><el-icon><QuestionFilled /></el-icon></span> 未回報 (Unreported)</span>
        <span class="legend-item"><span class="mark status-conflict"><el-icon><WarningFilled /></el-icon></span> 混車衝突 (Conflict)</span>
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
          width="85"
          fixed="left"
          align="center"
        >
          <template #default="{ row }">
            <span class="inline-value">
              {{ REGION_LABELS[row.region] || row.region }}
            </span>
          </template>
        </el-table-column>

        <el-table-column
          prop="tripPattern"
          label="趟數"
          width="76"
          fixed="left"
          align="center"
        >
          <template #default="{ row }">
            <el-tag
              v-if="getTripPatternDisplay(row) === '自訂'"
              size="small"
              type="info"
              effect="plain"
              class="custom-trip-tag"
            >
              自訂
            </el-tag>
            <span v-else class="trip-count-text">
              {{ getTripPatternDisplay(row) }}
            </span>
          </template>
        </el-table-column>

        <!-- 當月動態日期欄位 (1 ~ 31 日) -->
        <el-table-column
          v-for="day in daysInMonth"
          :key="day"
          :label="isHoliday(day) ? `${day} ★` : `${day}`"
          min-width="46"
          align="center"
        >
          <template #default="{ row }">
            <div class="cell-legs-container">
              <template v-for="slot in getDaySlots(row, day)" :key="slot.legSeq">
                <!-- 該趟次已有搭乘紀錄 -->
                <el-tooltip
                  v-if="slot.record"
                  placement="top"
                  :show-after="300"
                >
                  <template #content>
                    <div class="cell-tooltip">
                      <div><strong>{{ slot.record.serviceDate }} 第 {{ slot.record.legSeq }} 趟 ({{ slot.record.direction === 'outbound' ? '去程' : '回程' }})</strong></div>
                      <div>實際狀態：{{ slot.record.effectiveStatus === 'boarded' ? '有坐' : (slot.record.effectiveStatus === 'absent' ? '沒坐' : '未回報') }}</div>
                      <template v-if="slot.record.effectiveStatus === 'boarded'">
                        <div>車輛：{{ slot.record.vehicleName || '預設車輛' }}</div>
                        <div>司機：{{ slot.record.driverName || '主要司機' }}</div>
                      </template>
                      <template v-else-if="slot.record.effectiveStatus === 'absent'">
                        <div>車輛：-（沒坐）</div>
                        <div>司機：-（沒坐）</div>
                      </template>
                      <template v-else>
                        <div>車輛：{{ slot.record.vehicleName || '未指定' }}</div>
                        <div>司機：{{ slot.record.driverName || '未指定' }}</div>
                      </template>
                      <div v-if="slot.record.hasConflict" style="color: #ff4d4f;">⚠️ 發現跨車回報衝突！</div>
                      <div v-if="slot.record.correctedAt" style="color: #409eff;">✏️ 已由 {{ slot.record.correctedByName }} 於 {{ formatDateTime(slot.record.correctedAt) }} 更正</div>
                    </div>
                  </template>

                  <div
                    class="calendar-cell"
                    :class="[
                      `status-${slot.record.hasConflict ? 'conflict' : slot.record.effectiveStatus}`,
                      { 'is-corrected': !!slot.record.correctedAt }
                    ]"
                    @click="openCorrection(slot.record)"
                  >
                    <el-icon v-if="slot.record.hasConflict"><WarningFilled /></el-icon>
                    <el-icon v-else-if="slot.record.effectiveStatus === 'boarded'"><Check /></el-icon>
                    <el-icon v-else-if="slot.record.effectiveStatus === 'absent'"><Close /></el-icon>
                    <el-icon v-else-if="slot.record.effectiveStatus === 'unreported'"><QuestionFilled /></el-icon>
                  </div>
                </el-tooltip>

                <!-- 該趟次尚無紀錄之空白槽位（依趟數顯示，點選直接設定該趟紀錄） -->
                <el-tooltip
                  v-else
                  :content="`點選設定 第 ${slot.legSeq} 趟 (${slot.direction}) 搭乘記錄`"
                  placement="top"
                  :show-after="300"
                >
                  <div
                    class="calendar-cell status-non-scheduled"
                    @click="openManualEntry(row, day, slot.legSeq)"
                  >
                    <el-icon class="add-icon"><Plus /></el-icon>
                  </div>
                </el-tooltip>
              </template>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 搭乘紀錄更正抽屜面板 -->
    <RideCorrectionDrawer
      ref="drawerRef"
      @updated="fetchMatrix"
    />

    <!-- 人工填寫搭乘紀錄對話框 -->
    <RideManualEntryDialog
      ref="manualEntryDialogRef"
      @saved="fetchMatrix"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  Warning,
  WarningFilled,
  Check,
  Close,
  QuestionFilled,
  Search,
  Plus,
  ArrowLeft,
  ArrowRight
} from '@element-plus/icons-vue'
import RideCorrectionDrawer from './RideCorrectionDrawer.vue'
import RideManualEntryDialog from './RideManualEntryDialog.vue'
import { getRideCalendarMatrix } from '@/api/rides'
import { listHolidays } from '@/api/holidays'
import { formatDateTime } from '@/utils/formatters'
import { useRocMonth } from '@/composables/useRocMonth'
import { REGION_LABELS } from '@/types/domain'
import type { RideCalendarMatrixDTO, CaseRideCalendarRowDTO, RideRecordDTO } from '@/types/api'

const { toRocMonth } = useRocMonth()

const selectedDate = ref<string>('2026-07')
const regionFilter = ref<string>('')
const searchQuery = ref<string>('')
const loading = ref(false)
const matrixData = ref<RideCalendarMatrixDTO | null>(null)
const holidayMap = ref<Record<string, { name: string; isDayOff?: boolean }>>({})
const drawerRef = ref<InstanceType<typeof RideCorrectionDrawer>>()
const manualEntryDialogRef = ref<InstanceType<typeof RideManualEntryDialog>>()

const currentRocMonth = computed(() => {
  return toRocMonth(selectedDate.value)
})

const daysInMonth = computed(() => {
  if (!selectedDate.value) return 31
  const [year, month] = selectedDate.value.split('-').map(Number)
  return new Date(year, month, 0).getDate()
})

function changeMonth(delta: number) {
  if (!selectedDate.value) {
    selectedDate.value = '2026-07'
  }
  const [yearStr, monthStr] = selectedDate.value.split('-')
  let year = parseInt(yearStr, 10)
  let month = parseInt(monthStr, 10)

  month += delta
  if (month < 1) {
    month = 12
    year -= 1
  } else if (month > 12) {
    month = 1
    year += 1
  }

  selectedDate.value = `${year}-${String(month).padStart(2, '0')}`
  fetchMatrix()
}

async function fetchMatrix() {
  loading.value = true
  try {
    const [res, holidayResponse] = await Promise.all([getRideCalendarMatrix({
      month: currentRocMonth.value,
      region: regionFilter.value || undefined,
      q: searchQuery.value || undefined
    }), listHolidays({ startDate: `${selectedDate.value}-01`, endDate: `${selectedDate.value}-${String(daysInMonth.value).padStart(2, '0')}`, region: regionFilter.value || undefined })])
    matrixData.value = (res as any)?.cases ? res : ((res as any)?.data || res)
    holidayMap.value = Object.fromEntries((holidayResponse.data || []).map((item) => [item.holidayDate, item]))
  } finally {
    loading.value = false
  }
}

function isHoliday(day: number) {
  return Boolean(holidayMap.value[`${selectedDate.value}-${String(day).padStart(2, '0')}`])
}

function getCell(row: any, day: number) {
  const dayKey = `${selectedDate.value}-${String(day).padStart(2, '0')}`
  return row.days?.[dayKey]
}

// 取得個案月曆趟數顯示文字：當月應搭日趟數一致時顯示 N 趟，不一致時顯示自訂
function getTripPatternDisplay(row: any): string {
  if (row.tripPattern === 'custom' || row.tripPatternText === '自訂') {
    return '自訂'
  }

  if (row.days) {
    const scheduledTripCounts = new Set<number>()
    for (const dateKey in row.days) {
      const cell = row.days[dateKey]
      if (cell && cell.isExpected) {
        const count = cell.expectedTripCount ?? cell.records?.length ?? 0
        if (count > 0) {
          scheduledTripCounts.add(count)
        }
      }
    }
    if (scheduledTripCounts.size > 1) {
      return '自訂'
    } else if (scheduledTripCounts.size === 1) {
      const count = Array.from(scheduledTripCounts)[0]
      return `${count} 趟`
    }
  }

  if (typeof row.tripPattern === 'number') {
    return `${row.tripPattern} 趟`
  }
  return '2 趟'
}

// 計算該個案在指定日期的搭乘槽位列表（依該日預期趟數與實際紀錄動態展開）
function getDaySlots(row: any, day: number) {
  const cell = getCell(row, day)
  const records = cell?.records || []
  const isExpected = cell ? cell.isExpected : false
  const expectedTripCount = cell?.expectedTripCount ?? (isExpected ? (typeof row.tripPattern === 'number' ? row.tripPattern : 2) : 0)

  let maxLegSeq = 0
  for (const r of records) {
    if (r.legSeq > maxLegSeq) {
      maxLegSeq = r.legSeq
    }
  }

  // 應搭日至少滿足預期趟數；非應搭日依實際紀錄數（無紀錄時為單一非應搭槽位）
  let totalSlots = 0
  if (isExpected) {
    totalSlots = Math.max(expectedTripCount, maxLegSeq, 1)
  } else {
    totalSlots = Math.max(maxLegSeq, records.length, 1)
  }

  const slots = []
  for (let legSeq = 1; legSeq <= totalSlots; legSeq++) {
    const record = records.find((r: any) => r.legSeq === legSeq)
    const direction = legSeq % 2 === 1 ? '去程' : '回程'
    slots.push({
      legSeq,
      direction,
      record: record || null,
      isExpected: isExpected && legSeq <= expectedTripCount
    })
  }
  return slots
}

function openCorrection(record: RideRecordDTO) {
  drawerRef.value?.open(record)
}

function openManualEntry(row: any, day: number, targetLegSeq?: number) {
  const dayKey = `${selectedDate.value}-${String(day).padStart(2, '0')}`
  const cell = getCell(row, day)
  const existingLegs = (cell?.records || []).map((r: any) => r.legSeq)
  const dayTripCount = cell?.expectedTripCount || (typeof row.tripPattern === 'number' ? row.tripPattern : 2)
  manualEntryDialogRef.value?.open({
    caseId: row.caseId,
    caseName: row.caseName,
    caseCode: row.caseCode,
    serviceDate: dayKey,
    tripPattern: dayTripCount || 2,
    targetLegSeq,
    existingLegs
  })
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

.inline-value {
  color: var(--el-text-color-regular);
}

.custom-trip-tag {
  font-weight: 500;
  border-radius: 4px;
}

.trip-count-text {
  color: var(--el-text-color-regular);
  font-size: 13px;
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
    gap: 6px;

    .label {
      font-size: 14px;
      font-weight: 500;
      white-space: nowrap;
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

        .el-icon {
          font-size: 13px;
        }
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

.calendar-cell {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  margin: 0 auto;
  transition: all 0.2s;

  &.status-boarded {
    background-color: var(--el-color-success-light-8);
    color: var(--el-color-success);
    border: 1px solid var(--el-color-success-light-5);
  }

  &.status-absent {
    background-color: var(--el-color-info-light-8);
    color: var(--el-text-color-secondary);
    border: 1px solid var(--el-border-color);
  }

  &.status-unreported {
    background-color: var(--el-color-warning-light-8);
    color: var(--el-color-warning);
    border: 1px solid var(--el-color-warning-light-5);
  }

  &.status-conflict {
    background-color: var(--el-color-danger-light-8);
    color: var(--el-color-danger);
    border: 1px solid var(--el-color-danger-light-5);
  }

  &.is-corrected {
    position: relative;
    &::after {
      content: '';
      position: absolute;
      top: 2px;
      right: 2px;
      width: 5px;
      height: 5px;
      border-radius: 50%;
      background-color: var(--el-color-primary);
    }
  }

  &.status-non-scheduled {
    background-color: var(--el-fill-color-light);
    border: 1px dashed var(--el-border-color-lighter);
    color: transparent;

    .add-icon {
      font-size: 12px;
      opacity: 0;
      transition: opacity 0.2s;
      color: var(--el-color-primary);
    }

    &:hover {
      background-color: var(--el-color-primary-light-9);
      border-color: var(--el-color-primary-light-5);
      .add-icon {
        opacity: 1;
      }
    }
  }
}

.cell-tooltip {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
}
</style>
