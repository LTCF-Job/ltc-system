<template>
  <div class="schedule-editor">
    <el-form
      ref="formRef"
      :model="formData"
      :rules="rules"
      label-width="140px"
      :disabled="!authStore.can('staff')"
    >
      <!-- 排班優先順序引導說明 -->
      <div class="priority-guide-card">
        <div class="guide-header">
          <div class="guide-badge">排班優先順序</div>
          <div class="guide-steps">
            <span class="step" :class="{ active: scheduleMode === 'monthly' }">1. 當月排班 (月曆自訂，最高優先)</span>
            <span class="step-arrow">➔</span>
            <span class="step" :class="{ active: scheduleMode === 'by_weekday' }">2. 當周排班 (依星期個別設定)</span>
            <span class="step-arrow">➔</span>
            <span class="step" :class="{ active: scheduleMode === 'unified' }">3. 固定排班 (常態基準條件)</span>
          </div>
        </div>
        <div class="guide-desc">
          系統生成月曆與計算搭乘趟次時，優先採用【當月排班】；若當月未特別自訂，則依【當周排班】星期規則生效；若當周未特別設定，則回落至【固定排班】基準條件。
        </div>
      </div>

      <!-- 排班基本條件 -->
      <el-card shadow="never" class="section-card">
        <template #header>
          <div class="card-header-flex">
            <span class="card-title">排班條件與模式設定</span>
            <el-radio-group v-model="scheduleMode" size="small" class="mode-selector">
              <el-radio-button value="monthly">當月排班 (月曆自訂)</el-radio-button>
              <el-radio-button value="by_weekday">當周排班 (依星期設定)</el-radio-button>
              <el-radio-button value="unified">固定排班 (常態統一)</el-radio-button>
            </el-radio-group>
          </div>
        </template>

        <el-row :gutter="16">
          <el-col :xs="24" :lg="12">
            <el-form-item label="所屬據點" prop="siteId">
              <el-select
                v-model="formData.siteId"
                placeholder="請選擇據點"
                style="width: 100%"
                filterable
              >
                <el-option
                  v-for="site in availableSites"
                  :key="site.id"
                  :label="site.name"
                  :value="site.id"
                />
              </el-select>
            </el-form-item>
          </el-col>

          <el-col :xs="24" :lg="12">
            <el-form-item label="有效起始日" prop="effectiveFrom">
              <el-date-picker
                v-model="formData.effectiveFrom"
                type="date"
                placeholder="選擇生效日期"
                value-format="YYYY-MM-DD"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <!-- 模式 1：當月排班 (最高優先) -->
        <template v-if="scheduleMode === 'monthly'">
          <div class="monthly-schedule-container">
            <div class="monthly-toolbar">
              <div class="month-select-group">
                <span class="toolbar-label">設定月份：</span>
                <el-date-picker
                  v-model="selectedMonth"
                  type="month"
                  format="YYYY-MM"
                  value-format="YYYY-MM"
                  placeholder="選擇月份"
                  :clearable="false"
                  style="width: 140px"
                  @change="handleMonthChange"
                />
                <div class="month-stat-chips">
                  <span class="stat-chip">當月共 <strong>{{ daysInSelectedMonth }}</strong> 天</span>
                  <span class="stat-chip highlight">有排班 <strong>{{ activeDaysCountInMonth }}</strong> 天</span>
                  <span v-if="overriddenDaysCountInMonth > 0" class="stat-chip custom">已自訂 <strong>{{ overriddenDaysCountInMonth }}</strong> 天</span>
                  <span v-if="holidayDaysCountInMonth > 0" class="stat-chip holiday">國定假日 <strong>{{ holidayDaysCountInMonth }}</strong> 天</span>
                </div>
              </div>

              <div class="monthly-quick-actions">
                <el-button
                  size="small"
                  class="action-btn action-apply-weekly"
                  :icon="Calendar"
                  @click="applyWeeklyToMonth"
                >
                  套用當周排班
                </el-button>
                <el-button
                  size="small"
                  class="action-btn action-apply-fixed"
                  :icon="SetUp"
                  @click="applyFixedToMonth"
                >
                  套用固定排班
                </el-button>
                <el-button
                  size="small"
                  class="action-btn action-clear"
                  :icon="RefreshRight"
                  @click="clearMonthOverrides"
                >
                  清空當月自訂
                </el-button>
                <el-button
                  size="small"
                  class="action-btn action-absent"
                  :icon="CircleClose"
                  @click="setAllMonthAbsent"
                >
                  整月設為不搭乘
                </el-button>
              </div>
            </div>

            <div class="table-clip-container">
              <el-table
                :data="monthDaysList"
                border
                size="small"
                style="width: 100%"
                max-height="460"
                class="monthly-table"
              >
                <el-table-column label="日期" width="130" align="center">
                  <template #default="{ row }">
                    <div :class="['date-cell-label', { 'is-weekend': row.isWeekend }]">
                      <div class="date-main">
                        <span class="date-num">{{ row.date.slice(5) }}</span>
                        <span class="weekday-text">({{ row.weekdayLabel }})</span>
                      </div>
                      <el-tag
                        v-if="row.isHoliday"
                        size="small"
                        type="danger"
                        effect="light"
                        class="holiday-badge"
                      >
                        {{ row.holidayName || '國定假日' }}
                      </el-tag>
                    </div>
                  </template>
                </el-table-column>

                <el-table-column label="生效來源" width="120" align="center">
                  <template #default="{ row }">
                    <el-tag
                      v-if="row.isOverridden"
                      size="small"
                      type="primary"
                      effect="plain"
                      class="source-tag custom"
                    >
                      當月自訂
                    </el-tag>
                    <el-tag
                      v-else-if="row.source === 'weekly'"
                      size="small"
                      type="success"
                      effect="plain"
                      class="source-tag weekly"
                    >
                      依每週設定
                    </el-tag>
                    <el-tag
                      v-else
                      size="small"
                      type="info"
                      effect="plain"
                      class="source-tag fixed"
                    >
                      依固定基準
                    </el-tag>
                  </template>
                </el-table-column>

                <el-table-column label="當日趟數設定" width="170">
                  <template #default="{ row }">
                    <el-select
                      v-model="row.tripCount"
                      placeholder="選擇趟數"
                      style="width: 100%"
                      @change="markDayOverridden(row)"
                    >
                      <el-option :value="0" label="0: 不搭乘 (請假/停駛)" />
                      <el-option :value="1" label="1: 單趟去程" />
                      <el-option :value="2" label="2: 一般來回" />
                      <el-option :value="4" label="4: 四趟接送" />
                    </el-select>
                  </template>
                </el-table-column>

                <el-table-column label="去程出發時間" width="150">
                  <template #default="{ row }">
                    <el-time-picker
                      v-if="row.tripCount > 0"
                      v-model="row.departTime"
                      format="HH:mm"
                      value-format="HH:mm"
                      placeholder="去程 HH:mm"
                      style="width: 100%"
                      @change="markDayOverridden(row)"
                    />
                    <span v-else class="text-muted">-</span>
                  </template>
                </el-table-column>

                <el-table-column label="回程出發時間" width="150">
                  <template #default="{ row }">
                    <el-time-picker
                      v-if="row.tripCount >= 2"
                      v-model="row.returnTime"
                      format="HH:mm"
                      value-format="HH:mm"
                      placeholder="回程 HH:mm"
                      style="width: 100%"
                      @change="markDayOverridden(row)"
                    />
                    <span v-else class="text-muted">-</span>
                  </template>
                </el-table-column>

                <el-table-column label="指定車輛" min-width="160">
                  <template #default="{ row }">
                    <el-select
                      v-if="row.tripCount > 0"
                      v-model="row.vehicleId"
                      placeholder="選擇車輛"
                      clearable
                      style="width: 100%"
                      @change="markDayOverridden(row)"
                    >
                      <el-option
                        v-for="v in availableVehicles"
                        :key="v.id"
                        :label="`${v.displayName} (${v.plateNo})`"
                        :value="v.id"
                      />
                    </el-select>
                    <span v-else class="text-muted">-</span>
                  </template>
                </el-table-column>

                <el-table-column label="備註 (選填)" min-width="130">
                  <template #default="{ row }">
                    <el-input
                      v-model="row.note"
                      placeholder="如請假、回診"
                      clearable
                      @change="markDayOverridden(row)"
                    />
                  </template>
                </el-table-column>

                <el-table-column label="操作" width="95" align="center">
                  <template #default="{ row }">
                    <el-button
                      v-if="row.isOverridden"
                      size="small"
                      link
                      type="primary"
                      @click="resetDayToInherited(row)"
                    >
                      恢復預設
                    </el-button>
                    <span v-else class="text-muted text-xs">已繼承</span>
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </div>
        </template>

        <!-- 模式 2：當周各別排班 (依星期設定) -->
        <template v-else-if="scheduleMode === 'by_weekday'">
          <div class="weekday-table-container">
            <div class="weekday-tip">
              <el-alert
                type="info"
                :closable="false"
                show-icon
                title="可個別設定週一至週日每天的接送趟數與時段。若當月未設定該日自訂排班，將依此星期規則生效。"
                style="margin-bottom: 12px;"
              />
            </div>
            <el-table :data="weekdayConfigs" border size="small" style="width: 100%">
              <el-table-column prop="label" label="星期" width="80" align="center">
                <template #default="{ row }">
                  <strong>{{ row.label }}</strong>
                </template>
              </el-table-column>
              <el-table-column label="當日趟數設定" width="180">
                <template #default="{ row }">
                  <el-select v-model="row.tripCount" placeholder="選擇趟數" style="width: 100%">
                    <el-option :value="0" label="0: 不搭乘" />
                    <el-option :value="1" label="1: 單趟去程" />
                    <el-option :value="2" label="2: 一般來回" />
                    <el-option :value="4" label="4: 四趟接送" />
                  </el-select>
                </template>
              </el-table-column>
              <el-table-column label="去程出發時間" width="160">
                <template #default="{ row }">
                  <el-time-picker
                    v-if="row.tripCount > 0"
                    v-model="row.departTime"
                    format="HH:mm"
                    value-format="HH:mm"
                    placeholder="去程 HH:mm"
                    style="width: 100%"
                  />
                  <span v-else class="text-muted">-</span>
                </template>
              </el-table-column>
              <el-table-column label="回程出發時間" width="160">
                <template #default="{ row }">
                  <el-time-picker
                    v-if="row.tripCount >= 2"
                    v-model="row.returnTime"
                    format="HH:mm"
                    value-format="HH:mm"
                    placeholder="回程 HH:mm"
                    style="width: 100%"
                  />
                  <span v-else class="text-muted">-</span>
                </template>
              </el-table-column>
              <el-table-column label="預設指派車輛" min-width="160">
                <template #default="{ row }">
                  <el-select
                    v-if="row.tripCount > 0"
                    v-model="row.vehicleId"
                    placeholder="選擇車輛"
                    clearable
                    style="width: 100%"
                  >
                    <el-option
                      v-for="v in availableVehicles"
                      :key="v.id"
                      :label="`${v.displayName} (${v.plateNo})`"
                      :value="v.id"
                    />
                  </el-select>
                  <span v-else class="text-muted">-</span>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </template>

        <!-- 模式 3：固定統一排班 (常態基準) -->
        <template v-else>
          <el-row :gutter="16">
            <el-col :xs="24" :lg="12">
              <el-form-item label="趟數型態" prop="tripPattern">
                <el-radio-group v-model="formData.tripPattern" @change="(val: any) => handlePatternChange(val)">
                  <el-radio-button :value="1">單向 1 趟</el-radio-button>
                  <el-radio-button :value="2">一般 2 趟</el-radio-button>
                  <el-radio-button :value="4">四趟 (早/午去回)</el-radio-button>
                </el-radio-group>
              </el-form-item>
            </el-col>

            <el-col :xs="24" :lg="12">
              <el-form-item label="每週搭乘日" prop="weekdays">
                <el-checkbox-group v-model="formData.weekdays">
                  <el-checkbox :value="1">週一</el-checkbox>
                  <el-checkbox :value="2">週二</el-checkbox>
                  <el-checkbox :value="3">週三</el-checkbox>
                  <el-checkbox :value="4">週四</el-checkbox>
                  <el-checkbox :value="5">週五</el-checkbox>
                  <el-checkbox :value="6">週六</el-checkbox>
                  <el-checkbox :value="7">週日</el-checkbox>
                </el-checkbox-group>
              </el-form-item>
            </el-col>
          </el-row>
        </template>

        <!-- 費用與時長欄位 (共用) -->
        <el-row :gutter="16" style="margin-top: 16px;">
          <el-col :xs="24" :sm="8">
            <el-form-item label="申報單價 (元)" prop="unitPrice">
              <el-input-number
                v-model="formData.unitPrice"
                :min="1"
                :max="9999"
                :precision="2"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>

          <el-col :xs="24" :sm="8">
            <el-form-item label="單趟里程 (公里)" prop="distanceKm">
              <el-input-number
                v-model="formData.distanceKm"
                :min="0.1"
                :max="999"
                :precision="2"
                placeholder="無預設值，必填"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>

          <el-col :xs="24" :sm="8">
            <el-form-item label="服務時長 (分鐘)" prop="serviceDurationMin">
              <el-input-number
                v-model="formData.serviceDurationMin"
                :min="1"
                :max="240"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- 固定排班之時段 (Legs) 車輛與時間設定 (統一模式下顯示) -->
      <el-card v-if="scheduleMode === 'unified'" shadow="never" class="section-card">
        <template #header>
          <span class="card-title">固定排班時段與車輛設定 (共 {{ formData.legs.length }} 趟)</span>
        </template>

        <div v-for="(leg, idx) in formData.legs" :key="idx" class="leg-row-box">
          <div class="leg-header">
            <span class="leg-label">第 {{ leg.legSeq }} 趟</span>
            <span class="leg-direction">
              {{ leg.direction === 'outbound' ? '去程 (住家 -> 據點)' : '回程 (據點 -> 住家)' }}
            </span>
          </div>

          <el-row :gutter="16" class="leg-inputs">
            <el-col :xs="24" :sm="12" :lg="6">
              <el-form-item
                label="出發時間"
                :prop="`legs.${idx}.departTime`"
                :rules="[{ required: true, message: '請選擇出發時間', trigger: 'change' }]"
              >
                <el-time-picker
                  v-model="leg.departTime"
                  format="HH:mm"
                  value-format="HH:mm"
                  placeholder="出發 HH:mm"
                  style="width: 100%"
                />
              </el-form-item>
            </el-col>

            <el-col :xs="24" :sm="12" :lg="6">
              <el-form-item label="抵達時間 (選填)">
                <el-time-picker
                  v-model="leg.arriveTime"
                  format="HH:mm"
                  value-format="HH:mm"
                  placeholder="抵達 HH:mm"
                  style="width: 100%"
                />
              </el-form-item>
            </el-col>

            <el-col :xs="24" :sm="12" :lg="6">
              <el-form-item
                label="預設指派車輛"
                :prop="`legs.${idx}.vehicleId`"
                :rules="[{ required: true, message: '請選擇車輛', trigger: 'change' }]"
              >
                <el-select
                  v-model="leg.vehicleId"
                  placeholder="選擇車輛"
                  style="width: 100%"
                  filterable
                >
                  <el-option
                    v-for="v in availableVehicles"
                    :key="v.id"
                    :label="`${v.displayName} (${v.plateNo})`"
                    :value="v.id"
                  />
                </el-select>
              </el-form-item>
            </el-col>

            <el-col :xs="24" :sm="12" :lg="6">
              <el-form-item label="車次序號 (RunNo)">
                <el-input-number v-model="leg.runNo" :min="1" :max="20" style="width: 100%" />
              </el-form-item>
            </el-col>
          </el-row>
        </div>
      </el-card>

      <!-- 儲存按鈕 -->
      <div v-if="authStore.can('staff')" class="form-actions">
        <el-button type="primary" size="large" :loading="saving" @click="handleSave">
          儲存排班設定
        </el-button>
      </div>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { Calendar, SetUp, RefreshRight, CircleClose } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import { listSites, listVehicles } from '@/api/masters'
import { saveCaseSchedule } from '@/api/cases'
import { listHolidays } from '@/api/holidays'
import type {
  CaseScheduleDTO,
  CreateScheduleRequest,
  SiteDTO,
  VehicleDTO,
  ScheduleMode,
  DayScheduleConfig
} from '@/types/api'
import type { Region } from '@/types/domain'

const props = defineProps<{
  caseId: string
  region: Region
  schedule?: CaseScheduleDTO | null
}>()

const emit = defineEmits<{
  (e: 'saved'): void
}>()

const authStore = useAuthStore()
const formRef = ref<FormInstance>()
const saving = ref(false)
const availableSites = ref<SiteDTO[]>([])
const availableVehicles = ref<VehicleDTO[]>([])
const scheduleMode = ref<ScheduleMode>('monthly')
const selectedMonth = ref<string>('2026-07')
const holidayMap = ref<Record<string, { name: string; isDayOff?: boolean }>>({})

const weekdayLabels = ['週一', '週二', '週三', '週四', '週五', '週六', '週日']

const weekdayConfigs = reactive([
  { weekday: 1, label: '週一', tripCount: 2, departTime: '09:00', returnTime: '16:00', vehicleId: '' },
  { weekday: 2, label: '週二', tripCount: 2, departTime: '09:00', returnTime: '16:00', vehicleId: '' },
  { weekday: 3, label: '週三', tripCount: 2, departTime: '09:00', returnTime: '16:00', vehicleId: '' },
  { weekday: 4, label: '週四', tripCount: 2, departTime: '09:00', returnTime: '16:00', vehicleId: '' },
  { weekday: 5, label: '週五', tripCount: 2, departTime: '09:00', returnTime: '16:00', vehicleId: '' },
  { weekday: 6, label: '週六', tripCount: 0, departTime: '09:00', returnTime: '16:00', vehicleId: '' },
  { weekday: 7, label: '週日', tripCount: 0, departTime: '09:00', returnTime: '16:00', vehicleId: '' },
])

const monthlyConfigs = reactive<Record<string, DayScheduleConfig>>({})

const formData = reactive<CreateScheduleRequest>({
  siteId: '',
  effectiveFrom: new Date().toISOString().split('T')[0],
  tripPattern: 2,
  weekdays: [1, 2, 3, 4, 5],
  unitPrice: 115,
  distanceKm: 5,
  serviceDurationMin: 10,
  scheduleMode: 'monthly',
  legs: [
    { legSeq: 1, direction: 'outbound', departTime: '09:00', runNo: 1, vehicleId: '' },
    { legSeq: 2, direction: 'inbound', departTime: '16:00', runNo: 1, vehicleId: '' }
  ]
})

const rules = {
  siteId: [{ required: true, message: '請選擇據點', trigger: 'change' }],
  effectiveFrom: [{ required: true, message: '請選擇生效日期', trigger: 'change' }],
  weekdays: [{ type: 'array', required: true, min: 1, message: '請至少選擇一個每週搭乘日', trigger: 'change' }],
  unitPrice: [{ required: true, message: '請輸入單價', trigger: 'blur' }],
  distanceKm: [{ required: true, message: '請輸入單趟里程（公里）', trigger: 'blur' }],
  serviceDurationMin: [{ required: true, message: '請輸入服務時長', trigger: 'blur' }]
}

// 計算選取月份的天數
const daysInSelectedMonth = computed(() => {
  if (!selectedMonth.value) return 31
  const [y, m] = selectedMonth.value.split('-').map(Number)
  return new Date(y, m, 0).getDate()
})

// 當月排班表格資料來源（依優先級組裝當前各日之狀態）
interface MonthDayRow {
  date: string
  dayNum: number
  weekday: number
  weekdayLabel: string
  isWeekend: boolean
  holidayName?: string
  isHoliday: boolean
  isOverridden: boolean
  source: 'monthly' | 'weekly' | 'fixed'
  tripCount: number
  departTime: string
  returnTime: string
  vehicleId: string
  note: string
}

const monthDaysList = ref<MonthDayRow[]>([])

function buildMonthDaysList() {
  if (!selectedMonth.value) return
  const [yearStr, monthStr] = selectedMonth.value.split('-')
  const year = Number(yearStr)
  const month = Number(monthStr)
  const daysCount = new Date(year, month, 0).getDate()

  const list: MonthDayRow[] = []
  const defaultVehicle = formData.legs[0]?.vehicleId || availableVehicles.value[0]?.id || ''
  const defaultDepart = formData.legs[0]?.departTime || '09:00'
  const defaultReturn = formData.legs[1]?.departTime || '16:00'

  for (let day = 1; day <= daysCount; day++) {
    const dayPad = String(day).padStart(2, '0')
    const dateStr = `${selectedMonth.value}-${dayPad}`
    const d = new Date(year, month - 1, day)
    let goWeekday = d.getDay()
    if (goWeekday === 0) goWeekday = 7 // 1..7 (週一..週日)
    const isWeekend = goWeekday >= 6
    const holiday = holidayMap.value[dateStr]
    // 只有在 holiday 存在且 holiday.isDayOff !== false (非補班日) 才是國定休假日
    const isHoliday = Boolean(holiday && holiday.isDayOff !== false)
    const holidayName = holiday?.name || ''

    const mOverride = monthlyConfigs[dateStr]
    const wConfig = weekdayConfigs.find((c) => c.weekday === goWeekday)
    const isFixedActive = formData.weekdays.includes(goWeekday)

    let isOverridden = false
    let source: 'monthly' | 'weekly' | 'fixed' = 'fixed'
    let tripCount = isFixedActive ? (formData.tripPattern || 2) : 0
    let departTime = defaultDepart
    let returnTime = defaultReturn
    let vehicleId = defaultVehicle
    let note = ''

    // 層級 2：當周設定 (非國定休假日生效)
    if (wConfig && !isHoliday) {
      source = 'weekly'
      tripCount = wConfig.tripCount
      departTime = wConfig.departTime || defaultDepart
      returnTime = wConfig.returnTime || defaultReturn
      vehicleId = wConfig.vehicleId || defaultVehicle
    }

    // 層級 1：當月自訂 (最高優先)
    if (mOverride !== undefined) {
      isOverridden = true
      source = 'monthly'
      tripCount = mOverride.tripCount
      departTime = mOverride.departTime || defaultDepart
      returnTime = mOverride.returnTime || defaultReturn
      vehicleId = mOverride.vehicleId || defaultVehicle
      note = mOverride.note || ''
    }

    list.push({
      date: dateStr,
      dayNum: day,
      weekday: goWeekday,
      weekdayLabel: weekdayLabels[goWeekday - 1],
      isWeekend,
      holidayName,
      isHoliday,
      isOverridden,
      source,
      tripCount,
      departTime,
      returnTime,
      vehicleId,
      note
    })
  }

  monthDaysList.value = list
}

const activeDaysCountInMonth = computed(() => {
  return monthDaysList.value.filter((d) => d.tripCount > 0).length
})

const holidayDaysCountInMonth = computed(() => {
  return monthDaysList.value.filter((d) => d.isHoliday).length
})

const overriddenDaysCountInMonth = computed(() => {
  return monthDaysList.value.filter((d) => d.isOverridden).length
})

function handleMonthChange() {
  void loadHolidays()
  buildMonthDaysList()
}

async function loadHolidays() {
  const response = await listHolidays({
    startDate: `${selectedMonth.value}-01`,
    endDate: `${selectedMonth.value}-${String(daysInSelectedMonth.value).padStart(2, '0')}`,
    region: props.region
  })
  holidayMap.value = Object.fromEntries((response.data || []).map((item) => [item.holidayDate, item]))
  buildMonthDaysList()
}

function markDayOverridden(row: any) {
  row.isOverridden = true
  row.source = 'monthly'
  monthlyConfigs[row.date] = {
    date: row.date,
    tripCount: row.tripCount,
    departTime: row.departTime,
    returnTime: row.returnTime,
    vehicleId: row.vehicleId,
    note: row.note
  }
}

function resetDayToInherited(row: any) {
  delete monthlyConfigs[row.date]
  buildMonthDaysList()
  ElMessage.info(`已重設 ${row.date} 為自動繼承預設排班`)
}

function applyWeeklyToMonth() {
  monthDaysList.value.forEach((row) => {
    const wConfig = weekdayConfigs.find((c) => c.weekday === row.weekday)
    if (wConfig) {
      row.tripCount = wConfig.tripCount
      row.departTime = wConfig.departTime || '09:00'
      row.returnTime = wConfig.returnTime || '16:00'
      row.vehicleId = wConfig.vehicleId || availableVehicles.value[0]?.id || ''
      markDayOverridden(row)
    }
  })
  ElMessage.success(`已將當周排班設定套用至 ${selectedMonth.value} 全部日期`)
}

function applyFixedToMonth() {
  const activeSet = new Set(formData.weekdays)
  const defaultVehicle = formData.legs[0]?.vehicleId || availableVehicles.value[0]?.id || ''
  monthDaysList.value.forEach((row) => {
    row.tripCount = activeSet.has(row.weekday) ? (formData.tripPattern || 2) : 0
    row.departTime = formData.legs[0]?.departTime || '09:00'
    row.returnTime = formData.legs[1]?.departTime || '16:00'
    row.vehicleId = defaultVehicle
    markDayOverridden(row)
  })
  ElMessage.success(`已將固定排班設定套用至 ${selectedMonth.value} 全部日期`)
}

function clearMonthOverrides() {
  Object.keys(monthlyConfigs).forEach((k) => {
    if (k.startsWith(selectedMonth.value)) {
      delete monthlyConfigs[k]
    }
  })
  buildMonthDaysList()
  ElMessage.success(`已清空 ${selectedMonth.value} 全部當月自訂設定，恢復自動繼承`)
}

function setAllMonthAbsent() {
  monthDaysList.value.forEach((row) => {
    row.tripCount = 0
    markDayOverridden(row)
  })
  ElMessage.warning(`已將 ${selectedMonth.value} 全月設為不搭乘 (請假)`)
}

// 趟數切換時調整 legs 陣列
function handlePatternChange(pattern: any) {
  const currentVehicle = formData.legs[0]?.vehicleId || availableVehicles.value[0]?.id || ''

  if (pattern === 1) {
    formData.legs = [
      { legSeq: 1, direction: 'outbound', departTime: '09:00', runNo: 1, vehicleId: currentVehicle }
    ]
  } else if (pattern === 2) {
    formData.legs = [
      { legSeq: 1, direction: 'outbound', departTime: '09:00', runNo: 1, vehicleId: currentVehicle },
      { legSeq: 2, direction: 'inbound', departTime: '16:00', runNo: 1, vehicleId: currentVehicle }
    ]
  } else if (pattern === 4) {
    formData.legs = [
      { legSeq: 1, direction: 'outbound', departTime: '08:30', runNo: 1, vehicleId: currentVehicle },
      { legSeq: 2, direction: 'inbound', departTime: '11:30', runNo: 1, vehicleId: currentVehicle },
      { legSeq: 3, direction: 'outbound', departTime: '13:30', runNo: 1, vehicleId: currentVehicle },
      { legSeq: 4, direction: 'inbound', departTime: '16:30', runNo: 1, vehicleId: currentVehicle }
    ]
  }
}

watch(
  () => props.schedule,
  (s) => {
    if (s) {
      scheduleMode.value = s.scheduleMode || 'monthly'
      formData.siteId = s.siteId
      formData.effectiveFrom = s.effectiveFrom || new Date().toISOString().split('T')[0]
      formData.tripPattern = s.tripPattern || 2
      // 既有排班從 API 載入，欄位缺漏時保持未填，不得用猜測值頂替申報單價、里程與時長
      formData.weekdays = s.weekdays ? [...s.weekdays] : []
      formData.unitPrice = s.unitPrice
      formData.distanceKm = s.distanceKm
      formData.serviceDurationMin = s.serviceDurationMin
      if (s.legs && s.legs.length > 0) {
        formData.legs = s.legs.map((leg) => ({
          legSeq: leg.legSeq,
          direction: leg.direction,
          departTime: leg.departTime || '',
          arriveTime: leg.arriveTime || '',
          runNo: leg.runNo || 1,
          vehicleId: leg.vehicleId
        }))
      }

      // 同步 weekdayConfigs
      if (s.weeklyConfigs && s.weeklyConfigs.length > 0) {
        s.weeklyConfigs.forEach((wc) => {
          const match = weekdayConfigs.find((c) => c.weekday === wc.weekday)
          if (match) {
            match.tripCount = wc.tripCount
            if (wc.departTime) match.departTime = wc.departTime
            if (wc.returnTime) match.returnTime = wc.returnTime
            if (wc.vehicleId) match.vehicleId = wc.vehicleId
          }
        })
      } else {
        const activeSet = new Set(formData.weekdays)
        const defaultVehicle = formData.legs[0]?.vehicleId || ''
        weekdayConfigs.forEach((cfg) => {
          if (activeSet.has(cfg.weekday)) {
            cfg.tripCount = formData.tripPattern || 2
            cfg.departTime = formData.legs[0]?.departTime || ''
            cfg.returnTime = formData.legs[1]?.departTime || ''
            cfg.vehicleId = defaultVehicle
          } else {
            cfg.tripCount = 0
          }
        })
      }

      // 同步 monthlyConfigs
      if (s.monthlyConfigs) {
        Object.assign(monthlyConfigs, s.monthlyConfigs)
      }

      buildMonthDaysList()
    }
  },
  { immediate: true, deep: true }
)

async function loadSitesAndVehicles() {
  const [sitesRes, vehiclesRes] = await Promise.all([
    listSites({ region: props.region, pageSize: 100 }),
    listVehicles({ region: props.region, active: true, pageSize: 100 })
  ])
  availableSites.value = sitesRes.data
  availableVehicles.value = vehiclesRes.data

  if (!formData.siteId && availableSites.value.length > 0) {
    formData.siteId = availableSites.value[0].id
  }
  if (availableVehicles.value.length > 0) {
    formData.legs.forEach((leg) => {
      if (!leg.vehicleId) {
        leg.vehicleId = availableVehicles.value[0].id
      }
    })
    weekdayConfigs.forEach((cfg) => {
      if (!cfg.vehicleId) {
        cfg.vehicleId = availableVehicles.value[0].id
      }
    })
  }
  buildMonthDaysList()
}

watch(
  () => props.region,
  () => {
    loadSitesAndVehicles()
  }
)

onMounted(async () => {
  await loadSitesAndVehicles()
  await loadHolidays()
  buildMonthDaysList()
})

async function handleSave() {
  if (!formRef.value) return

  // 1. 組裝各模式的相容性 weekdays 與 legs
  if (scheduleMode.value === 'by_weekday') {
    const activeDays = weekdayConfigs.filter((cfg) => cfg.tripCount > 0)
    if (activeDays.length === 0) {
      ElMessage.warning('請至少設定一天的接送趟數大於 0')
      return
    }

    formData.weekdays = activeDays.map((cfg) => cfg.weekday)
    let maxTrips = 1
    activeDays.forEach((cfg) => {
      if (cfg.tripCount > maxTrips) maxTrips = cfg.tripCount
    })
    formData.tripPattern = (maxTrips === 4 ? 4 : (maxTrips === 1 ? 1 : 2)) as any
    handlePatternChange(formData.tripPattern)

    const firstActive = activeDays[0]
    if (formData.legs[0]) {
      formData.legs[0].departTime = firstActive.departTime || '09:00'
      if (firstActive.vehicleId) formData.legs[0].vehicleId = firstActive.vehicleId
    }
    if (formData.legs[1] && firstActive.returnTime) {
      formData.legs[1].departTime = firstActive.returnTime
      if (firstActive.vehicleId) formData.legs[1].vehicleId = firstActive.vehicleId
    }
  } else if (scheduleMode.value === 'monthly') {
    const activeDays = monthDaysList.value.filter((d) => d.tripCount > 0)
    if (activeDays.length > 0) {
      const activeWeekdays = Array.from(new Set(activeDays.map((d) => d.weekday)))
      formData.weekdays = activeWeekdays.length > 0 ? activeWeekdays : [1, 2, 3, 4, 5]
    }
  }

  // 2. 附加三層級完整設定
  formData.scheduleMode = scheduleMode.value
  formData.weeklyConfigs = weekdayConfigs.map((cfg) => ({
    weekday: cfg.weekday,
    label: cfg.label,
    tripCount: cfg.tripCount,
    departTime: cfg.departTime,
    returnTime: cfg.returnTime,
    vehicleId: cfg.vehicleId
  }))
  formData.monthlyConfigs = { ...monthlyConfigs }

  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      await saveCaseSchedule(props.caseId, formData)
      ElMessage.success('排班設定儲存成功')
      emit('saved')
    } finally {
      saving.value = false
    }
  })
}
</script>

<style scoped>
.schedule-editor {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.priority-guide-card {
  background: #ffffff;
  border: 1px solid var(--el-border-color-lighter);
  border-left: 4px solid #3b82f6;
  border-radius: 8px;
  padding: 12px 16px;
  margin-bottom: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.02);

  .guide-header {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
    margin-bottom: 6px;
  }

  .guide-badge {
    font-size: var(--app-font-xs);
    font-weight: 700;
    color: #1e293b;
    background: #f1f5f9;
    padding: 2px 8px;
    border-radius: 4px;
    border: 1px solid #e2e8f0;
  }

  .guide-steps {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    flex-wrap: wrap;

    .step {
      font-weight: 500;
      color: #64748b;

      &.active {
        color: #0f172a;
        font-weight: 700;
      }
    }

    .step-arrow {
      color: #94a3b8;
      font-size: var(--app-font-xs);
    }
  }

  .guide-desc {
    font-size: var(--app-font-xs);
    line-height: 1.6;
    color: #475569;
  }
}

.card-header-flex {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.section-card {
  margin-bottom: 16px;
  border-radius: 8px;

  .card-title {
    font-size: 15px;
    font-weight: bold;
    color: #1e293b;
  }
}

.monthly-schedule-container {
  margin: 12px 0 16px 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.monthly-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  padding: 12px 16px;
  background-color: #f8fafc;
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);

  .month-select-group {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;

    .toolbar-label {
      font-size: 13px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }
  }

  .month-stat-chips {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;

    .stat-chip {
      font-size: var(--app-font-xs);
      color: var(--el-text-color-secondary);
      padding: 2px 8px;
      background: #ffffff;
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 12px;

      strong {
        color: var(--el-text-color-primary);
      }

      &.highlight {
        color: var(--app-status-success-fg);
        border-color: var(--app-status-success-bg);
        background: var(--app-status-success-bg);
        strong {
          color: var(--app-status-success-fg);
        }
      }

      &.custom {
        color: var(--app-status-info-fg);
        border-color: var(--app-status-info-bg);
        background: var(--app-status-info-bg);
        strong {
          color: var(--app-status-info-fg);
        }
      }

      &.holiday {
        color: var(--app-status-danger-fg);
        border-color: var(--app-status-danger-bg);
        background: var(--app-status-danger-bg);
        strong {
          color: var(--app-status-danger-fg);
        }
      }
    }
  }

  .monthly-quick-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;

    .action-btn {
      font-size: var(--app-font-xs);
      font-weight: 600;
      height: 32px;
      padding: 0 12px;
      border-radius: 6px;
      border: 1px solid #cbd5e1;
      background: #ffffff;
      color: #1e293b;
      display: inline-flex;
      align-items: center;
      gap: 5px;
      transition: all 0.15s ease;

      &:hover {
        border-color: #94a3b8;
        background: #f8fafc;
        color: #0f172a;
      }

      &.action-apply-weekly {
        border-color: #cbd5e1;
        background: #ffffff;
        color: #1e293b;

        &:hover {
          border-color: #86efac;
          background: #f0fdf4;
          color: #15803d;
        }
      }

      &.action-apply-fixed {
        border-color: #cbd5e1;
        background: #ffffff;
        color: #1e293b;

        &:hover {
          border-color: #93c5fd;
          background: #eff6ff;
          color: #1d4ed8;
        }
      }

      &.action-clear {
        border-color: #fed7aa;
        background: #fffaf5;
        color: #9a3412;

        &:hover {
          border-color: #fdba74;
          background: #ffedd5;
          color: #7c2d12;
        }
      }

      &.action-absent {
        border-color: #cbd5e1;
        background: #ffffff;
        color: #475569;

        &:hover {
          border-color: #fca5a5;
          background: #fef2f2;
          color: #b91c1c;
        }
      }
    }
  }
}

.table-clip-container {
  border-radius: 6px;
  overflow: hidden;
}

.date-cell-label {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;

  .date-main {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 13px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .weekday-text {
    font-size: var(--app-font-xs);
    font-weight: normal;
    color: var(--el-text-color-secondary);
  }

  &.is-weekend {
    .date-main {
      color: #94a3b8;
    }
    .weekday-text {
      color: #94a3b8;
    }
  }

  .holiday-badge {
    font-size: var(--app-font-xs);
    padding: 0 4px;
    height: 18px;
    line-height: 16px;
    border-radius: 4px;
    margin-top: 2px;
  }
}

.text-muted {
  color: var(--el-text-color-secondary);
}

.text-xs {
  font-size: var(--app-font-xs);
}

.month-info,
.leg-label,
.leg-direction {
  color: var(--el-text-color-secondary);
  font-size: var(--app-font-xs);
}

.source-tag {
  font-size: var(--app-font-xs);
  font-weight: 500;
  border-radius: 4px;
  background: transparent !important; /* 確保無背景底色 */

  &.custom {
    color: var(--el-color-primary) !important;
    border-color: var(--el-color-primary) !important;
  }

  &.weekly {
    color: var(--app-status-success-fg) !important;
    border-color: var(--app-status-success-fg) !important;
  }

  &.fixed {
    color: var(--el-text-color-regular) !important;
    border-color: var(--el-border-color) !important;
  }
}

.leg-label {
  color: var(--el-text-color-primary);
  font-weight: 600;
}

.weekday-table-container {
  margin: 8px 0 16px 0;
}

.leg-row-box {
  padding: 12px 16px;
  background-color: var(--el-fill-color-light);
  border-radius: 6px;
  margin-bottom: 12px;

  .leg-header {
    display: flex;
    gap: 8px;
    margin-bottom: 12px;
  }
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 10px;
}
</style>
