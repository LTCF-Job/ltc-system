<template>
  <div class="attendance-fuel-view">
    <PageHeader title="出勤與油資登錄" />
    <el-tabs v-model="activeTab" type="border-card" class="main-tabs">
      <!-- 分頁 1: 司機出勤與請假月曆 -->
      <el-tab-pane label="司機出勤與請假" name="attendance">
        <div class="tab-content" v-loading="attendanceLoading">
          <!-- 出勤過濾與統計列 -->
          <div class="filter-header">
            <div class="left-controls" style="display: flex; gap: 8px; align-items: center;">
              <el-date-picker
                v-model="attendanceMonth"
                type="month"
                placeholder="選擇月份"
                value-format="YYYY-MM"
                :clearable="false"
                style="width: 140px"
                @change="fetchAttendance"
              />

              <el-input
                v-model="attendanceQuery"
                placeholder="搜尋司機姓名／代碼"
                clearable
                style="width: 240px"
                @keyup.enter="fetchAttendance"
              />

              <el-select
                v-model="selectedDriverId"
                placeholder="全部司機"
                clearable
                style="width: 140px"
                @change="fetchAttendance"
              >
                <el-option
                  v-for="d in drivers"
                  :key="d.id"
                  :label="d.name"
                  :value="d.id"
                />
              </el-select>

              <el-button type="primary" @click="fetchAttendance">
                查詢
              </el-button>
              <el-button @click="handleResetAttendance">
                重設
              </el-button>
            </div>

            <!-- 出勤彙總指標 -->
            <div class="attendance-summary-pills">
              <div class="summary-pill">
                <span class="dot dot-work"></span>
                <span class="label">出勤 (O)</span>
                <span class="val">{{ totalAttendanceStats.work }}</span>
                <span class="unit">人天</span>
              </div>
              <div class="summary-pill">
                <span class="dot dot-leave"></span>
                <span class="label">事假 (事)</span>
                <span class="val">{{ totalAttendanceStats.leave }}</span>
                <span class="unit">人天</span>
              </div>
              <div class="summary-pill">
                <span class="dot dot-sick"></span>
                <span class="label">病假 (病)</span>
                <span class="val">{{ totalAttendanceStats.sick }}</span>
                <span class="unit">人天</span>
              </div>
              <div class="summary-pill">
                <span class="dot dot-off"></span>
                <span class="label">休假 (休)</span>
                <span class="val">{{ totalAttendanceStats.off }}</span>
                <span class="unit">人天</span>
              </div>
              <div class="summary-pill">
                <span class="dot dot-absent"></span>
                <span class="label">漏報缺勤 (缺)</span>
                <span class="val">{{ totalAttendanceStats.absent }}</span>
                <span class="unit">人天</span>
              </div>
              <div class="summary-pill holiday-pill">
                <span class="dot dot-holiday"></span>
                <span class="label">國定假日</span>
                <span class="unit">灰底</span>
              </div>
            </div>
          </div>

          <!-- 出勤月曆矩陣表格 -->
          <div class="matrix-table-box mt-3">
            <el-table
              :data="attendanceReport?.drivers || []"
              border
              size="small"
              height="580"
              class="attendance-matrix"
              style="width: 100%;"
            >
              <el-table-column
                prop="driverName"
                label="司機姓名"
                width="160"
                fixed="left"
                align="center"
                show-overflow-tooltip
              >
                <template #default="{ row }">
                  <span class="driver-name-text">
                    {{ row.driverName }}({{ row.driverCode }})
                  </span>
                </template>
              </el-table-column>

              <el-table-column label="出勤統計" width="130" fixed="left" align="center">
                <template #default="{ row }">
                  <div class="driver-summary">
                    <span>出 {{ row.workDays }}</span>
                    <span class="summary-divider">/</span>
                    <span>假 {{ row.leaveDays + row.sickDays }}</span>
                    <template v-if="row.absentDays > 0">
                      <span class="summary-divider">/</span>
                      <span class="summary-absent">缺 {{ row.absentDays }}</span>
                    </template>
                  </div>
                </template>
              </el-table-column>

              <!-- 動態產生當月 1 ~ N 天之欄位 -->
              <el-table-column
                v-for="day in (attendanceReport?.daysInMonth || 31)"
                :key="day"
                min-width="48"
                align="center"
                :class-name="isHoliday(day) ? 'col-holiday' : ''"
              >
                <template #header>
                  <div class="day-header" :class="{ 'is-holiday-header': isHoliday(day) }">
                    <el-tooltip
                      v-if="isHoliday(day)"
                      :content="`${getDayKey(day)} 國定假日：${getHolidayName(day) || '放假'}`"
                      placement="top"
                    >
                      <span class="day-num holiday-text">{{ day }}</span>
                    </el-tooltip>
                    <span v-else class="day-num">{{ day }}</span>
                  </div>
                </template>
                <template #default="{ row }">
                  <div
                    class="day-cell"
                    :class="getCellBgClass(row, day)"
                    @click="handleCellClick(row, day)"
                  >
                    <el-tooltip
                      v-if="getDayRecord(row, day) || isHoliday(day)"
                      :content="getTooltipContent(row, day)"
                      placement="top"
                    >
                      <div class="cell-content">
                        <template v-if="getDayRecord(row, day)">
                          <span
                            class="cell-symbol"
                            :class="`symbol-${getDayRecord(row, day)?.status}`"
                          >
                            {{ getStatusDisplay(getDayRecord(row, day)?.status).symbol }}
                          </span>
                        </template>
                        <span v-else class="cell-symbol cell-empty">-</span>
                      </div>
                    </el-tooltip>
                    <span v-else class="cell-symbol cell-empty">-</span>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </div>
      </el-tab-pane>

      <!-- 分頁 2: 車輛油資登錄 -->
      <el-tab-pane label="車輛油資登錄" name="fuel">
        <DataTablePage
          :max-width="1100"
          :loading="fuelLoading"
          v-model:page="fuelPage"
          v-model:pageSize="fuelPageSize"
          :total="fuelTotal"
          :page-sizes="[10, 20, 50]"
          @page-change="fetchFuelLogs"
          @size-change="fetchFuelLogs"
        >
          <template #filter>
            <el-input
              v-model="fuelQuery"
              placeholder="搜尋車牌／車名／司機"
              clearable
              style="width: 240px"
              @keyup.enter="fetchFuelLogs"
            />

            <el-select
              v-model="fuelVehicleId"
              placeholder="全部車輛"
              clearable
              style="width: 140px"
              @change="fetchFuelLogs"
            >
              <el-option
                v-for="veh in vehicles"
                :key="veh.id"
                :label="veh.displayName"
                :value="veh.id"
              />
            </el-select>

            <el-select
              v-model="fuelDriverId"
              placeholder="全部司機"
              clearable
              style="width: 140px"
              @change="fetchFuelLogs"
            >
              <el-option
                v-for="d in drivers"
                :key="d.id"
                :label="d.name"
                :value="d.id"
              />
            </el-select>

            <el-date-picker
              v-model="fuelDateRange"
              type="daterange"
              range-separator="至"
              start-placeholder="開始日期"
              end-placeholder="結束日期"
              value-format="YYYY-MM-DD"
              style="width: 230px"
              @change="fetchFuelLogs"
            />

            <el-button type="primary" @click="fetchFuelLogs">
              查詢
            </el-button>
            <el-button @click="handleResetFuel">
              重設
            </el-button>
          </template>

          <template #actions>
            <el-button v-if="authStore.can('staff')" type="primary" :icon="Plus" @click="openFuelDialog()">
              新增加油紀錄
            </el-button>
          </template>

          <template #table>
          <!-- 油資統計指標 -->
          <el-row :gutter="16" class="stat-row mb-3">
            <el-col :xs="24" :sm="8">
              <el-card shadow="never" class="stat-card">
                <div class="stat-inner">
                  <span class="stat-label">加油總筆數</span>
                  <span class="stat-val">{{ fuelTotal }} <span class="unit">筆</span></span>
                </div>
              </el-card>
            </el-col>
            <el-col :xs="24" :sm="8">
              <el-card shadow="never" class="stat-card">
                <div class="stat-inner">
                  <span class="stat-label">總加油公升數</span>
                  <span class="stat-val">{{ fuelTotalLiters.toFixed(1) }} <span class="unit">L</span></span>
                </div>
              </el-card>
            </el-col>
            <el-col :xs="24" :sm="8">
              <el-card shadow="never" class="stat-card">
                <div class="stat-inner">
                  <span class="stat-label">總花費金額</span>
                  <span class="stat-val">${{ fuelTotalCost.toLocaleString() }}</span>
                </div>
              </el-card>
            </el-col>
          </el-row>

          <!-- 油資表格 -->
          <el-table :data="fuelLogs" border stripe size="small" table-layout="auto" style="width: 100%;">
            <el-table-column prop="fuelDate" label="加油日期" min-width="120" align="center">
              <template #default="{ row }">
                <span>{{ row.fuelDate?.slice(0, 10) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="vehicleName" label="車輛名稱" min-width="120" class-name="fuel-vehicle-col" />
            <el-table-column prop="plateNo" label="車牌號碼" min-width="120" align="center" />
            <el-table-column prop="driverName" label="加油司機" min-width="100" align="center" class-name="fuel-driver-col">
              <template #default="{ row }">
                {{ row.driverName || '-' }}
              </template>
            </el-table-column>
            <el-table-column prop="liters" label="加油公升數" min-width="130" align="right">
              <template #default="{ row }">
                {{ Number(row.liters).toFixed(2) }} L
              </template>
            </el-table-column>
            <el-table-column prop="cost" label="花費金額" min-width="130" align="right">
              <template #default="{ row }">
                <span class="font-bold">${{ Number(row.cost).toLocaleString() }}</span>
              </template>
            </el-table-column>
            <el-table-column label="發票/收據憑證" min-width="130" align="center">
              <template #default="{ row }">
                <el-link
                  v-if="row.receiptUrl"
                  type="primary"
                  :href="row.receiptUrl"
                  target="_blank"
                >
                  檢視發票
                </el-link>
                <span v-else class="text-muted">-</span>
              </template>
            </el-table-column>
            <el-table-column v-if="authStore.can('staff')" label="操作" width="140" align="center" fixed="right">
              <template #default="{ row }">
                <TableRowActions>
                  <el-button link type="primary" size="small" @click="openFuelDialog(row)">
                    編輯
                  </el-button>
                  <el-button link type="danger" size="small" @click="handleDeleteFuel(row)">
                    刪除
                  </el-button>
                </TableRowActions>
              </template>
            </el-table-column>
          </el-table>
          </template>
        </DataTablePage>
      </el-tab-pane>
    </el-tabs>

    <!-- 司機單日出勤登記 Dialog -->
    <el-dialog
      v-model="attendanceDialogVisible"
      title="登記司機出勤狀態"
      width="min(480px, calc(100vw - 32px))"
      destroy-on-close
    >
      <div v-if="selectedCell" class="dialog-content">
        <el-descriptions :column="1" border size="small" class="mb-3">
          <el-descriptions-item label="司機姓名">
            {{ selectedCell.driverName }}
          </el-descriptions-item>
          <el-descriptions-item label="日期">
            <div style="display: flex; align-items: center; gap: 8px;">
              <span>{{ selectedCell.date }}</span>
              <el-tag
                v-if="holidayMap[selectedCell.date]"
                type="info"
                size="small"
                effect="plain"
              >
                國定假日：{{ holidayMap[selectedCell.date]?.name || '放假' }}
              </el-tag>
            </div>
          </el-descriptions-item>
        </el-descriptions>

        <el-form label-width="90px" label-position="right">
          <el-form-item label="出勤狀態">
            <el-radio-group v-model="editStatus">
              <el-radio-button label="work">出勤 (O)</el-radio-button>
              <el-radio-button label="leave">事假 (事)</el-radio-button>
              <el-radio-button label="sick">病假 (病)</el-radio-button>
              <el-radio-button label="off">休假 (休)</el-radio-button>
            </el-radio-group>
          </el-form-item>

          <el-form-item label="請假原因">
            <el-input
              v-model="editNote"
              placeholder="選填事由或備註說明"
              type="textarea"
              :rows="2"
            />
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <DialogFooter :loading="attendanceSaving" @confirm="handleSaveAttendance" @cancel="attendanceDialogVisible = false" />
      </template>
    </el-dialog>

    <!-- 加油紀錄新增/編輯 Dialog -->
    <el-dialog
      v-model="fuelDialogVisible"
      :title="editingFuelId ? '編輯加油紀錄' : '新增加油紀錄'"
      width="min(480px, calc(100vw - 32px))"
      destroy-on-close
    >
      <el-form
        ref="fuelFormRef"
        :model="fuelForm"
        :rules="fuelRules"
        label-width="110px"
        label-position="right"
      >
        <el-form-item label="加油車輛" prop="vehicleId">
          <el-select v-model="fuelForm.vehicleId" placeholder="選擇車輛" style="width: 100%">
            <el-option
              v-for="veh in vehicles"
              :key="veh.id"
              :label="`${veh.displayName} (${veh.plateNo})`"
              :value="veh.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="加油司機" prop="driverId">
          <el-select v-model="fuelForm.driverId" placeholder="選填司機" clearable style="width: 100%">
            <el-option
              v-for="d in drivers"
              :key="d.id"
              :label="d.name"
              :value="d.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="加油日期" prop="fuelDate">
          <el-date-picker
            v-model="fuelForm.fuelDate"
            type="date"
            placeholder="選擇加油日期"
            value-format="YYYY-MM-DD"
            style="width: 100%"
          />
        </el-form-item>

        <el-form-item label="加油公升" prop="liters">
          <el-input-number
            v-model="fuelForm.liters"
            :min="1"
            :step="5"
            :precision="2"
            style="width: 100%"
            placeholder="公升數"
          />
        </el-form-item>

        <el-form-item label="花費金額" prop="cost">
          <el-input-number
            v-model="fuelForm.cost"
            :min="1"
            :step="100"
            style="width: 100%"
            placeholder="加油總金額"
          />
        </el-form-item>

        <el-form-item label="發票/收據" prop="receiptUrl">
          <el-input v-model="fuelForm.receiptUrl" placeholder="發票圖片 URL" />
        </el-form-item>
      </el-form>

      <template #footer>
        <DialogFooter :loading="fuelSaving" @confirm="handleSaveFuel" @cancel="fuelDialogVisible = false" />
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import PageHeader from '@/components/PageHeader.vue'
import DialogFooter from '@/components/DialogFooter.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import DataTablePage from '@/components/DataTablePage.vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { resolveErrorMessage } from '@/api/errorCodes'
import {
  getMonthAttendance,
  upsertAttendance,
  listFuelLogs,
  createFuelLog,
  updateFuelLog,
  deleteFuelLog
} from '@/api/attendance'
import { listHolidays, type HolidayItem } from '@/api/holidays'
import { listDrivers, listVehicles } from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import type {
  MonthAttendanceReportDTO,
  DriverMonthAttendanceDTO,
  DriverDayAttendanceDTO,
  FuelLogDTO,
  DriverDTO,
  VehicleDTO
} from '@/types/api'

const authStore = useAuthStore()
const activeTab = ref('attendance')

// --- 出勤相關狀態 ---
const attendanceLoading = ref(false)
const attendanceSaving = ref(false)
const attendanceMonth = ref('2026-07')
const attendanceQuery = ref('')
const selectedDriverId = ref<string>()
const attendanceReport = ref<MonthAttendanceReportDTO | null>(null)
const holidayMap = ref<Record<string, HolidayItem>>({})

const drivers = ref<DriverDTO[]>([])
const vehicles = ref<VehicleDTO[]>([])

const attendanceDialogVisible = ref(false)
const selectedCell = ref<{ driverId: string; driverName: string; date: string } | null>(null)
const editStatus = ref<'work' | 'leave' | 'sick' | 'off'>('work')
const editNote = ref('')

const totalAttendanceStats = computed(() => {
  const stats = { work: 0, leave: 0, sick: 0, off: 0, absent: 0 }
  if (!attendanceReport.value) return stats
  for (const d of attendanceReport.value.drivers) {
    stats.work += d.workDays
    stats.leave += d.leaveDays
    stats.sick += d.sickDays
    stats.off += d.offDays
    stats.absent += d.absentDays
  }
  return stats
})

// --- 油資相關狀態 ---
const fuelLoading = ref(false)
const fuelSaving = ref(false)
const fuelQuery = ref('')
const fuelVehicleId = ref<string>()
const fuelDriverId = ref<string>()
const fuelDateRange = ref<[string, string]>()
const fuelPage = ref(1)
const fuelPageSize = ref(20)
const fuelTotal = ref(0)
const fuelLogs = ref<FuelLogDTO[]>([])

const fuelTotalLiters = computed(() => {
  return fuelLogs.value.reduce((sum, item) => sum + Number(item.liters || 0), 0)
})

const fuelTotalCost = computed(() => {
  return fuelLogs.value.reduce((sum, item) => sum + Number(item.cost || 0), 0)
})

const fuelDialogVisible = ref(false)
const editingFuelId = ref<string | null>(null)
const fuelFormRef = ref<FormInstance>()

const fuelForm = reactive({
  vehicleId: '',
  driverId: undefined as string | undefined,
  fuelDate: '',
  liters: 0,
  cost: 0,
  receiptUrl: ''
})

const fuelRules = {
  vehicleId: [{ required: true, message: '請選擇車輛', trigger: 'change' }],
  fuelDate: [{ required: true, message: '請選擇加油日期', trigger: 'change' }],
  liters: [{ required: true, message: '請輸入公升數', trigger: 'blur' }],
  cost: [{ required: true, message: '請輸入金額', trigger: 'blur' }]
}

async function fetchOptions() {
  try {
    const [dRes, vRes] = await Promise.all([
      listDrivers({ pageSize: 100 }),
      listVehicles({ pageSize: 100 })
    ])
    drivers.value = dRes.data
    vehicles.value = vRes.data
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '載入主檔選項失敗'))
  }
}

async function fetchAttendance() {
  attendanceLoading.value = true
  try {
    const [yearStr, monthStr] = attendanceMonth.value.split('-')
    const year = parseInt(yearStr, 10)
    const month = parseInt(monthStr, 10)
    const daysInMonth = new Date(year, month, 0).getDate()
    const startDate = `${attendanceMonth.value}-01`
    const endDate = `${attendanceMonth.value}-${String(daysInMonth).padStart(2, '0')}`

    const [attRes, holidayRes] = await Promise.all([
      getMonthAttendance(
        attendanceMonth.value,
        selectedDriverId.value,
        attendanceQuery.value || undefined
      ),
      listHolidays({ startDate, endDate })
    ])
    attendanceReport.value = attRes
    holidayMap.value = Object.fromEntries((holidayRes.data || []).map((item) => [item.holidayDate, item]))
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '查詢出勤紀錄失敗'))
  } finally {
    attendanceLoading.value = false
  }
}

function handleResetAttendance() {
  attendanceQuery.value = ''
  selectedDriverId.value = undefined
  fetchAttendance()
}

function getDayKey(day: number): string {
  const month = attendanceMonth.value
  const d = String(day).padStart(2, '0')
  return `${month}-${d}`
}

function getDayRecord(driver: any, day: number): DriverDayAttendanceDTO | undefined {
  if (!driver || !driver.days) return undefined
  const dateKey = getDayKey(day)
  return driver.days[dateKey]
}

function isHoliday(day: number): boolean {
  const key = getDayKey(day)
  const h = holidayMap.value[key]
  return Boolean(h && h.isDayOff !== false)
}

function getHolidayName(day: number): string | undefined {
  const key = getDayKey(day)
  return holidayMap.value[key]?.name
}

function getStatusDisplay(status?: string): { symbol: string; text: string } {
  switch (status) {
    case 'work':
      return { symbol: 'O', text: 'O' }
    case 'leave':
      return { symbol: '事', text: '事' }
    case 'sick':
      return { symbol: '病', text: '病' }
    case 'off':
      return { symbol: '休', text: '休' }
    case 'absent':
      return { symbol: '缺', text: '缺' }
    default:
      return { symbol: '-', text: '-' }
  }
}

function getCellBgClass(driver: any, day: number): string {
  const rec = getDayRecord(driver, day)
  const holidayClass = isHoliday(day) ? 'is-holiday' : ''
  if (!rec) {
    return holidayClass ? 'cell-holiday' : ''
  }
  switch (rec.status) {
    case 'work': return `cell-work ${holidayClass}`.trim()
    case 'leave': return `cell-leave ${holidayClass}`.trim()
    case 'sick': return `cell-sick ${holidayClass}`.trim()
    case 'off': return `cell-off ${holidayClass}`.trim()
    case 'absent': return `cell-absent ${holidayClass}`.trim()
    default: return holidayClass ? 'cell-holiday' : ''
  }
}

function getTooltipContent(driver: any, day: number): string {
  const rec = getDayRecord(driver, day)
  const date = getDayKey(day)
  const holidayName = getHolidayName(day)
  const holidayInfo = holidayName ? `【國定假日：${holidayName}】` : (isHoliday(day) ? '【國定假日】' : '')

  if (!rec) {
    return holidayInfo ? `${date} ${holidayInfo}` : date
  }

  const map: Record<string, string> = {
    work: '正常出勤 (O)',
    leave: '事假 (事)',
    sick: '病假 (病)',
    off: '休假 (休)',
    absent: '應出勤未回報 (缺)'
  }
  let str = `${date} ${holidayInfo}：${map[rec.status] || rec.status}`.trim()
  if (rec.note) {
    str += ` (${rec.note})`
  }
  return str
}

function handleCellClick(driver: any, day: number) {
  if (!authStore.can('staff')) return
  const dateKey = getDayKey(day)
  const rec = driver.days ? driver.days[dateKey] : undefined
  selectedCell.value = {
    driverId: driver.driverId,
    driverName: driver.driverName,
    date: dateKey
  }
  editStatus.value = rec?.status && rec.status !== 'absent' ? rec.status : 'work'
  editNote.value = rec?.note || ''
  attendanceDialogVisible.value = true
}

async function handleSaveAttendance() {
  if (!selectedCell.value) return
  attendanceSaving.value = true
  try {
    await upsertAttendance({
      driverId: selectedCell.value.driverId,
      recordDate: selectedCell.value.date,
      status: editStatus.value,
      note: editNote.value || undefined
    })
    ElMessage.success('出勤狀態更新成功')
    attendanceDialogVisible.value = false
    fetchAttendance()
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '更新出勤狀態失敗'))
  } finally {
    attendanceSaving.value = false
  }
}

// --- 油資函式 ---
async function fetchFuelLogs() {
  fuelLoading.value = true
  try {
    const res = await listFuelLogs({
      page: fuelPage.value,
      pageSize: fuelPageSize.value,
      vehicleId: fuelVehicleId.value,
      driverId: fuelDriverId.value,
      startDate: fuelDateRange.value?.[0],
      endDate: fuelDateRange.value?.[1],
      q: fuelQuery.value || undefined
    })
    fuelLogs.value = res.data
    fuelTotal.value = res.meta.total
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '查詢油資紀錄失敗'))
  } finally {
    fuelLoading.value = false
  }
}

function handleResetFuel() {
  fuelQuery.value = ''
  fuelVehicleId.value = undefined
  fuelDriverId.value = undefined
  fuelDateRange.value = undefined
  fuelPage.value = 1
  fetchFuelLogs()
}

function openFuelDialog(row?: any) {
  if (row) {
    editingFuelId.value = row.id
    fuelForm.vehicleId = row.vehicleId
    fuelForm.driverId = row.driverId
    fuelForm.fuelDate = row.fuelDate?.slice(0, 10)
    fuelForm.liters = row.liters
    fuelForm.cost = row.cost
    fuelForm.receiptUrl = row.receiptUrl || ''
  } else {
    editingFuelId.value = null
    fuelForm.vehicleId = fuelVehicleId.value || ''
    fuelForm.driverId = undefined
    fuelForm.fuelDate = new Date().toISOString().slice(0, 10)
    fuelForm.liters = 0
    fuelForm.cost = 0
    fuelForm.receiptUrl = ''
  }
  fuelDialogVisible.value = true
}

async function handleSaveFuel() {
  if (!fuelFormRef.value) return
  await fuelFormRef.value.validate(async (valid) => {
    if (!valid) return
    fuelSaving.value = true
    try {
      if (editingFuelId.value) {
        await updateFuelLog(editingFuelId.value, fuelForm)
        ElMessage.success('油資紀錄修改成功')
      } else {
        await createFuelLog(fuelForm)
        ElMessage.success('油資紀錄新增成功')
      }
      fuelDialogVisible.value = false
      fetchFuelLogs()
    } catch (err: any) {
      ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '儲存油資紀錄失敗'))
    } finally {
      fuelSaving.value = false
    }
  })
}

async function handleDeleteFuel(row: any) {
  try {
    await ElMessageBox.confirm(
      `確定刪除「${row.vehicleName || '車輛'}」於 ${row.fuelDate?.slice(0, 10)} 的加油紀錄？`,
      '刪除確認',
      { type: 'warning' }
    )
    await deleteFuelLog(row.id)
    ElMessage.success('油資紀錄已刪除')
    fetchFuelLogs()
  } catch (err: any) {
    if (err !== 'cancel') {
      ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '刪除失敗'))
    }
  }
}

onMounted(async () => {
  await fetchOptions()
  await fetchAttendance()
  await fetchFuelLogs()
})
</script>

<style scoped>
.attendance-fuel-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.main-tabs {
  border-radius: 8px;
  background-color: #ffffff;
}

.filter-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;

  .left-controls {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
  }

  .attendance-summary-pills {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;

    .summary-pill {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 4px 10px;
      border-radius: 6px;
      background-color: var(--el-fill-color-light);
      font-size: 13px;
      color: var(--el-text-color-regular);

      .dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;

        &.dot-work {
          background-color: var(--el-color-success);
        }
        &.dot-leave {
          background-color: var(--el-color-warning);
        }
        &.dot-sick {
          background-color: var(--el-color-danger);
        }
        &.dot-off {
          background-color: var(--el-text-color-placeholder);
        }
        &.dot-absent {
          background-color: #f97316;
        }
        &.dot-holiday {
          background-color: #94a3b8;
        }
      }

      .label {
        font-size: var(--app-font-xs);
        color: var(--el-text-color-secondary);
      }

      .val {
        font-weight: 600;
        color: var(--el-text-color-primary);
      }

      .unit {
        font-size: var(--app-font-xs);
        color: var(--el-text-color-placeholder);
      }

      &.holiday-pill {
        background-color: #f1f5f9;
      }
    }
  }
}

.matrix-table-box {
  border-radius: 6px;
  overflow: hidden;
}

.day-header {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  font-weight: 600;

  &.is-holiday-header {
    background-color: #f1f5f9;
    color: #475569;
    border-radius: 3px;
    padding: 1px 0;
  }

  .holiday-text {
    cursor: pointer;
    text-decoration: underline dotted #94a3b8;
  }
}

:deep(.col-holiday) {
  background-color: #f8fafc !important;
}

.driver-name-text {
  font-weight: 600;
  color: var(--el-text-color-primary);
  font-size: 13px;
  display: inline-block;
  width: 100%;
  text-align: center;
}

.driver-summary {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  font-size: var(--app-font-xs);
  color: var(--el-text-color-regular);

  .summary-divider {
    color: var(--el-text-color-placeholder);
  }

  .summary-absent {
    color: #c2410c;
    font-weight: 700;
  }
}

.day-cell {
  height: 28px;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 500;
  font-size: var(--app-font-xs);
  transition: all 0.15s ease;
  background-color: transparent;
  color: var(--el-text-color-regular);

  &:hover {
    background-color: var(--el-fill-color);
  }

  .cell-content {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 2px;
  }

  .cell-symbol {
    font-weight: 700;
    font-size: 13px;

    &.symbol-work {
      color: #16a34a;
    }

    &.symbol-leave {
      color: #b45309;
    }

    &.symbol-sick {
      color: #dc2626;
    }

    &.symbol-off {
      color: #64748b;
    }

    &.symbol-absent {
      color: #c2410c;
    }
  }

  &.cell-work {
    background-color: #ffffff;
    color: #16a34a;
  }

  &.cell-leave {
    background-color: #fef3c7;
    color: #92400e;
    font-weight: 600;
  }

  &.cell-sick {
    background-color: #fee2e2;
    color: #991b1b;
    font-weight: 600;
  }

  &.cell-off {
    background-color: var(--el-fill-color-light);
    color: var(--el-text-color-placeholder);
  }

  &.cell-absent {
    background-color: #fff7ed;
    color: #c2410c;
    font-weight: 700;
    border: 1px solid #fb923c;
  }

  &.cell-holiday {
    background-color: #f1f5f9;
    color: #64748b;
  }

  &.is-holiday {
    background-color: #f1f5f9;

    &.cell-work {
      background-color: #f0fdf4;
      border: 1px solid #bbf7d0;
    }

    &.cell-leave {
      background-color: #fef3c7;
    }

    &.cell-sick {
      background-color: #fee2e2;
    }

    &.cell-off {
      background-color: #e2e8f0;
    }
  }

  .cell-empty {
    color: var(--el-text-color-placeholder);
  }
}

.stat-row {
  margin-left: 0 !important;
  margin-right: 0 !important;
}

.stat-card {
  border-radius: 6px;
  border: 1px solid var(--el-border-color-light);

  .stat-inner {
    display: flex;
    justify-content: space-between;
    align-items: center;

    .stat-label {
      font-size: 13px;
      color: var(--el-text-color-secondary);
    }

    .stat-val {
      font-size: 18px;
      font-weight: 700;
      color: var(--el-text-color-primary);

      .unit {
        font-size: 13px;
        font-weight: 400;
        color: var(--el-text-color-secondary);
        margin-left: 2px;
      }
    }
  }
}

.font-bold {
  font-weight: 600;
}

.text-muted {
  color: var(--el-text-color-placeholder);
}

:deep(.fuel-vehicle-col .cell),
:deep(.fuel-driver-col .cell) {
  white-space: nowrap;
}

.mt-3 {
  margin-top: 12px;
}

.mb-3 {
  margin-bottom: 12px;
}
</style>
