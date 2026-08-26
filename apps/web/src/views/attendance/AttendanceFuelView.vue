<template>
  <div class="attendance-fuel-view">
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
                style="width: 180px"
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
                  :label="`${d.name} (${d.code})`"
                  :value="d.id"
                />
              </el-select>

              <el-button type="primary" icon="Search" @click="fetchAttendance">
                查詢
              </el-button>
              <el-button @click="handleResetAttendance">
                重設
              </el-button>
            </div>

            <!-- 出勤彙總指標 -->
            <div class="stats-badges">
              <el-tag effect="dark" type="success" size="large">
                出勤：{{ totalAttendanceStats.work }} 人天
              </el-tag>
              <el-tag effect="dark" type="warning" size="large">
                事假：{{ totalAttendanceStats.leave }} 人天
              </el-tag>
              <el-tag effect="dark" type="danger" size="large">
                病假：{{ totalAttendanceStats.sick }} 人天
              </el-tag>
              <el-tag effect="dark" type="info" size="large">
                休假：{{ totalAttendanceStats.off }} 人天
              </el-tag>
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
            >
              <el-table-column
                prop="driverName"
                label="司機姓名"
                width="140"
                fixed="left"
              >
                <template #default="{ row }">
                  <div class="driver-cell">
                    <span class="driver-name">{{ row.driverName }}</span>
                    <el-tag size="small" type="info" effect="plain">{{ row.driverCode }}</el-tag>
                  </div>
                </template>
              </el-table-column>

              <el-table-column label="出勤統計" width="130" fixed="left" align="center">
                <template #default="{ row }">
                  <div class="driver-summary">
                    <span class="text-success">出:{{ row.workDays }}</span>
                    <span class="text-warning">假:{{ row.leaveDays + row.sickDays }}</span>
                  </div>
                </template>
              </el-table-column>

              <!-- 動態產生當月 1 ~ N 天之欄位 -->
              <el-table-column
                v-for="day in (attendanceReport?.daysInMonth || 31)"
                :key="day"
                :label="String(day)"
                width="42"
                align="center"
              >
                <template #default="{ row }">
                  <div
                    class="day-cell"
                    :class="getCellBgClass(row, day)"
                    @click="handleCellClick(row, day)"
                  >
                    <el-tooltip
                      v-if="getDayRecord(row, day)"
                      :content="getTooltipContent(row, day)"
                      placement="top"
                    >
                      <span class="cell-symbol">
                        {{ getStatusShort(getDayRecord(row, day)?.status) }}
                      </span>
                    </el-tooltip>
                    <span v-else class="cell-symbol">-</span>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </div>
      </el-tab-pane>

      <!-- 分頁 2: 車輛油資登錄 -->
      <el-tab-pane label="車輛油資登錄" name="fuel">
        <div class="tab-content" v-loading="fuelLoading">
          <!-- 油資過濾與統計卡片 -->
          <div class="fuel-top-section">
            <div class="filter-header">
              <div class="left-controls" style="display: flex; gap: 8px; align-items: center;">
                <el-input
                  v-model="fuelQuery"
                  placeholder="搜尋車牌／車名／司機"
                  clearable
                  style="width: 180px"
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

                <el-button type="primary" icon="Search" @click="fetchFuelLogs">
                  查詢
                </el-button>
                <el-button @click="handleResetFuel">
                  重設
                </el-button>
              </div>

              <el-button v-if="authStore.can('staff')" type="primary" @click="openFuelDialog()">
                <el-icon><Plus /></el-icon>
                新增加油紀錄
              </el-button>
            </div>

            <!-- 油資統計指標 -->
            <el-row :gutter="16" class="mt-3">
              <el-col :span="8">
                <el-card shadow="hover" class="stat-card">
                  <div class="stat-inner">
                    <span class="stat-label">加油總筆數</span>
                    <span class="stat-val">{{ fuelTotal }} 筆</span>
                  </div>
                </el-card>
              </el-col>
              <el-col :span="8">
                <el-card shadow="hover" class="stat-card">
                  <div class="stat-inner">
                    <span class="stat-label">總加油公升數</span>
                    <span class="stat-val text-primary">{{ fuelTotalLiters.toFixed(1) }} L</span>
                  </div>
                </el-card>
              </el-col>
              <el-col :span="8">
                <el-card shadow="hover" class="stat-card">
                  <div class="stat-inner">
                    <span class="stat-label">總花費金額</span>
                    <span class="stat-val text-danger">${{ fuelTotalCost.toLocaleString() }}</span>
                  </div>
                </el-card>
              </el-col>
            </el-row>
          </div>

          <!-- 油資表格 -->
          <el-table :data="fuelLogs" border stripe size="small" class="mt-3">
            <el-table-column prop="fuelDate" label="加油日期" width="110" align="center">
              <template #default="{ row }">
                <el-tag size="small" type="info">{{ row.fuelDate?.slice(0, 10) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="vehicleName" label="車輛名稱" width="120" />
            <el-table-column prop="plateNo" label="車牌號碼" width="110" align="center" />
            <el-table-column prop="driverName" label="加油司機" width="110">
              <template #default="{ row }">
                {{ row.driverName || '-' }}
              </template>
            </el-table-column>
            <el-table-column prop="liters" label="加油公升數" width="120" align="right">
              <template #default="{ row }">
                {{ Number(row.liters).toFixed(2) }} L
              </template>
            </el-table-column>
            <el-table-column prop="cost" label="花費金額" width="120" align="right">
              <template #default="{ row }">
                <span class="font-bold">${{ Number(row.cost).toLocaleString() }}</span>
              </template>
            </el-table-column>
            <el-table-column label="發票/收據憑證" width="120" align="center">
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
            <el-table-column v-if="authStore.can('staff')" label="操作" width="120" align="center" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="openFuelDialog(row)">
                  編輯
                </el-button>
                <el-button link type="danger" size="small" @click="handleDeleteFuel(row)">
                  刪除
                </el-button>
              </template>
            </el-table-column>
          </el-table>

          <!-- 分頁 -->
          <div class="pagination-box">
            <el-pagination
              v-model:current-page="fuelPage"
              v-model:page-size="fuelPageSize"
              :page-sizes="[10, 20, 50]"
              :total="fuelTotal"
              layout="total, sizes, prev, pager, next, jumper"
              @size-change="fetchFuelLogs"
              @current-change="fetchFuelLogs"
            />
          </div>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 司機單日出勤登記 Dialog -->
    <el-dialog
      v-model="attendanceDialogVisible"
      title="登記司機出勤狀態"
      width="400px"
      destroy-on-close
    >
      <div v-if="selectedCell" class="dialog-content">
        <el-descriptions :column="1" border size="small" class="mb-3">
          <el-descriptions-item label="司機姓名">
            {{ selectedCell.driverName }}
          </el-descriptions-item>
          <el-descriptions-item label="日期">
            {{ selectedCell.date }}
          </el-descriptions-item>
        </el-descriptions>

        <el-form label-width="80px" label-position="right">
          <el-form-item label="出勤狀態">
            <el-radio-group v-model="editStatus">
              <el-radio-button label="work">出勤</el-radio-button>
              <el-radio-button label="leave">事假</el-radio-button>
              <el-radio-button label="sick">病假</el-radio-button>
              <el-radio-button label="off">休假</el-radio-button>
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
        <span class="dialog-footer">
          <el-button @click="attendanceDialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="attendanceSaving" @click="handleSaveAttendance">
            確定變更
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 加油紀錄新增/編輯 Dialog -->
    <el-dialog
      v-model="fuelDialogVisible"
      :title="editingFuelId ? '編輯加油紀錄' : '新增加油紀錄'"
      width="480px"
      destroy-on-close
    >
      <el-form
        ref="fuelFormRef"
        :model="fuelForm"
        :rules="fuelRules"
        label-width="100px"
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
        <span class="dialog-footer">
          <el-button @click="fuelDialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="fuelSaving" @click="handleSaveFuel">
            確定儲存
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Refresh, Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import {
  getMonthAttendance,
  upsertAttendance,
  listFuelLogs,
  createFuelLog,
  updateFuelLog,
  deleteFuelLog
} from '@/api/attendance'
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
const attendanceMonth = ref(new Date().toISOString().slice(0, 7))
const attendanceQuery = ref('')
const selectedDriverId = ref<string>()
const attendanceReport = ref<MonthAttendanceReportDTO | null>(null)

const drivers = ref<DriverDTO[]>([])
const vehicles = ref<VehicleDTO[]>([])

const attendanceDialogVisible = ref(false)
const selectedCell = ref<{ driverId: string; driverName: string; date: string } | null>(null)
const editStatus = ref<'work' | 'leave' | 'sick' | 'off'>('work')
const editNote = ref('')

const totalAttendanceStats = computed(() => {
  const stats = { work: 0, leave: 0, sick: 0, off: 0 }
  if (!attendanceReport.value) return stats
  for (const d of attendanceReport.value.drivers) {
    stats.work += d.workDays
    stats.leave += d.leaveDays
    stats.sick += d.sickDays
    stats.off += d.offDays
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
    ElMessage.error(err.message || '載入主檔選項失敗')
  }
}

async function fetchAttendance() {
  attendanceLoading.value = true
  try {
    attendanceReport.value = await getMonthAttendance(
      attendanceMonth.value,
      selectedDriverId.value,
      attendanceQuery.value || undefined
    )
  } catch (err: any) {
    ElMessage.error(err.message || '查詢出勤紀錄失敗')
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

function getCellBgClass(driver: any, day: number): string {
  const rec = getDayRecord(driver, day)
  if (!rec) return ''
  switch (rec.status) {
    case 'work': return 'cell-work'
    case 'leave': return 'cell-leave'
    case 'sick': return 'cell-sick'
    case 'off': return 'cell-off'
    default: return ''
  }
}

function getStatusShort(status?: string): string {
  switch (status) {
    case 'work': return '出'
    case 'leave': return '事'
    case 'sick': return '病'
    case 'off': return '休'
    default: return '-'
  }
}

function getTooltipContent(driver: any, day: number): string {
  const rec = getDayRecord(driver, day)
  if (!rec) return ''
  const date = getDayKey(day)
  const map: Record<string, string> = { work: '正常出勤', leave: '事假', sick: '病假', off: '休假' }
  let str = `${date}：${map[rec.status] || rec.status}`
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
  editStatus.value = rec?.status || 'work'
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
    ElMessage.error(err.message || '更新出勤狀態失敗')
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
    ElMessage.error(err.message || '查詢油資紀錄失敗')
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
      ElMessage.error(err.message || '儲存油資紀錄失敗')
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
      ElMessage.error(err.message || '刪除失敗')
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

  .stats-badges {
    display: flex;
    gap: 8px;
  }
}

.matrix-table-box {
  border-radius: 6px;
  overflow: hidden;
}

.driver-cell {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;

  .driver-name {
    font-weight: 600;
  }
}

.driver-summary {
  display: flex;
  justify-content: space-around;
  font-size: 12px;
  font-weight: 600;
}

.day-cell {
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 600;
  font-size: 12px;
  transition: all 0.2s;

  &:hover {
    filter: brightness(0.9);
  }

  &.cell-work {
    background-color: var(--el-color-success-light-9);
    color: var(--el-color-success);
  }

  &.cell-leave {
    background-color: var(--el-color-warning-light-9);
    color: var(--el-color-warning);
  }

  &.cell-sick {
    background-color: var(--el-color-danger-light-9);
    color: var(--el-color-danger);
  }

  &.cell-off {
    background-color: var(--el-fill-color-light);
    color: var(--el-text-color-secondary);
  }
}

.stat-card {
  border-radius: 6px;

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
      font-weight: bold;
    }
  }
}

.font-bold {
  font-weight: 600;
}

.text-success {
  color: var(--el-color-success);
}

.text-warning {
  color: var(--el-color-warning);
}

.text-danger {
  color: var(--el-color-danger);
}

.text-muted {
  color: var(--el-text-color-placeholder);
}

.pagination-box {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.mt-3 {
  margin-top: 12px;
}

.mb-3 {
  margin-bottom: 12px;
}
</style>
