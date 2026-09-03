<template>
  <div class="driver-list-view">
    <DataTablePage
      title="司機管理"
      :max-width="1520"
      v-model:page="page"
      v-model:pageSize="pageSize"
      :total="total"
      :loading="loading"
      @page-change="handlePageChange"
      @size-change="handleSizeChange"
    >
      <!-- 篩選列 -->
      <template #filter>
        <el-input
          v-model="filters.q"
          placeholder="搜尋司機姓名／身分證"
          clearable
          style="width: 240px"
          @keyup.enter="handleSearch"
        />

        <el-select
          v-model="filters.status"
          placeholder="狀態"
          clearable
          style="width: 130px"
          @change="handleSearch"
        >
          <el-option label="全部狀態" value="" />
          <el-option label="啟用" value="active" />
          <el-option label="停用" value="inactive" />
        </el-select>

        <el-button type="primary" @click="handleSearch">查詢</el-button>
        <el-button @click="handleReset">重設</el-button>
      </template>

      <template #actions>
        <el-button
          v-if="authStore.hasPermission('masters_drivers', 'edit')"
          type="primary"
          @click="openCreateDialog"
        >
          <el-icon><Plus /></el-icon>
          新增司機
        </el-button>
      </template>

      <!-- 表格 -->
      <template #table>
        <el-table :data="drivers" border stripe table-layout="auto" style="width: 100%">
          <el-table-column prop="name" label="司機姓名" min-width="110" align="center" class-name="driver-name-col">
            <template #default="{ row }"><span class="driver-name">{{ row.name }}</span></template>
          </el-table-column>
          <el-table-column prop="nationalId" label="身分證字號" width="140" align="center">
            <template #default="{ row }">
              <span class="driver-data font-mono">{{ row.nationalId || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="licenseClass" label="駕照類別" min-width="120" align="center" class-name="license-class-col">
            <template #default="{ row }">
              <span class="driver-data license-value">
                {{ licenseClassLabel(row.licenseClass) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="licenseExpiryDate" label="駕照有效日期" min-width="130" align="center" class-name="license-expiry-col">
            <template #default="{ row }">
              <span class="driver-data license-value">{{ formatDate(row.licenseExpiryDate) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="phone" label="聯絡電話" width="130" align="center">
            <template #default="{ row }">
              <span class="driver-data">{{ row.phone || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="email" label="電子信箱" min-width="200" show-overflow-tooltip class-name="email-col">
            <template #default="{ row }"><span class="driver-data">{{ row.email || '-' }}</span></template>
          </el-table-column>
          <el-table-column label="目前指派車輛" min-width="180" class-name="assigned-vehicle-col">
            <template #default="{ row }">
              <div v-if="getAssignedVehicleDisplay(row)" class="assigned-vehicle-info">
                <span class="vehicle-name">{{ getAssignedVehicleDisplay(row)?.name }}</span>
                <span v-if="getAssignedVehicleDisplay(row)?.plateNo" class="vehicle-plate font-mono">
                  ({{ getAssignedVehicleDisplay(row)?.plateNo }})
                </span>
              </div>
              <span v-else class="assignment-empty">尚未指派</span>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="狀態" width="130" align="center">
            <template #default="{ row }">
              <el-tooltip
                v-if="authStore.hasPermission('masters_drivers', 'edit')"
                :content="row.status === 'active' ? '目前為啟用，點選切換為停用' : '目前為停用，點選切換為啟用'"
                placement="top"
                :show-after="300"
              >
                <button
                  type="button"
                  class="status-toggle-pill"
                  :class="row.status === 'active' ? 'is-active' : 'is-inactive'"
                  @click="handleQuickToggleActive(row as any, row.status !== 'active')"
                >
                  <span class="status-indicator-dot"></span>
                  <span class="status-label-text">{{ row.status === 'active' ? '啟用' : '停用' }}</span>
                </button>
              </el-tooltip>
              <div
                v-else
                class="status-toggle-pill is-readonly"
                :class="row.status === 'active' ? 'is-active' : 'is-inactive'"
              >
                <span class="status-indicator-dot"></span>
                <span class="status-label-text">{{ row.status === 'active' ? '啟用' : '停用' }}</span>
              </div>
            </template>
          </el-table-column>

          <el-table-column
            v-if="authStore.hasPermission('masters_drivers', 'edit') || authStore.hasPermission('masters_drivers', 'delete')"
            label="操作"
            width="240"
            fixed="right"
            align="center"
          >
            <template #default="{ row }">
              <TableRowActions>
                <template v-if="authStore.hasPermission('masters_drivers', 'edit')">
                  <el-button link type="primary" size="small" @click="openEditDialog(row)">
                    編輯
                  </el-button>
                  <el-button link type="primary" size="small" @click="openAssignDialog(row)">
                    指派車輛
                  </el-button>
                </template>
                <el-button
                  v-if="authStore.hasPermission('masters_drivers', 'delete')"
                  link
                  type="danger"
                  size="small"
                  @click="handleDeleteDriver(row as any)"
                >
                  刪除
                </el-button>
              </TableRowActions>
            </template>
          </el-table-column>

        </el-table>
      </template>
    </DataTablePage>

    <!-- 新增司機對話框：跟待維護資料頁籤的「新增司機並綁定」共用同一個元件與 API -->
    <DriverCreateDialog v-model="createDialogVisible" @created="handleDriverCreated" />

    <!-- 編輯司機對話框 -->
    <el-dialog v-model="editDialogVisible" title="編輯司機資料" width="min(480px, calc(100vw - 32px))">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
        <el-form-item label="司機姓名" prop="name">
          <el-input v-model="form.name" placeholder="請輸入姓名" />
        </el-form-item>
        <el-form-item label="身分證字號" prop="nationalId">
          <el-input v-model="form.nationalId" placeholder="1 碼英文 + 9 碼數字" />
        </el-form-item>
        <el-form-item label="所屬區域" prop="region">
          <el-select v-model="form.region" placeholder="請選擇區域" filterable style="width: 100%">
            <el-option
              v-for="(label, key) in REGION_LABELS"
              :key="key"
              :label="label"
              :value="key"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="聯絡電話" prop="phone">
          <el-input v-model="form.phone" placeholder="如：0912345678" />
        </el-form-item>
        <el-form-item label="電子信箱" prop="email">
          <el-input v-model="form.email" placeholder="通知寄送用信箱" />
        </el-form-item>
        <el-form-item label="駕照類別" prop="licenseClass">
          <el-select
            v-model="form.licenseClass"
            placeholder="請選擇駕照類別"
            clearable
            style="width: 100%"
          >
            <el-option
              v-for="(label, value) in DRIVER_LICENSE_CLASS_LABELS"
              :key="value"
              :label="label"
              :value="value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="駕照有效日期" prop="licenseExpiryDate">
          <el-date-picker
            v-model="form.licenseExpiryDate"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="請選擇駕照有效日期"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="狀態" prop="status">
          <el-radio-group v-model="form.status" class="status-radio-group">
            <el-radio-button value="active">
              <div class="radio-pill active-pill">
                <span class="radio-dot"></span>
                <span>啟用</span>
              </div>
            </el-radio-button>
            <el-radio-button value="inactive">
              <div class="radio-pill inactive-pill">
                <span class="radio-dot"></span>
                <span>停用</span>
              </div>
            </el-radio-button>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <DialogFooter :loading="submitting" @confirm="handleSubmit" @cancel="editDialogVisible = false" />
      </template>
    </el-dialog>

    <!-- 車輛期間指派對話框 -->
    <el-dialog v-model="assignDialogVisible" title="指派駕駛車輛" width="min(480px, calc(100vw - 32px))">
      <el-form ref="assignFormRef" :model="assignForm" :rules="assignRules" label-width="110px">
        <el-form-item label="選擇車輛" prop="vehicleId">
          <el-select v-model="assignForm.vehicleId" placeholder="請選擇車輛" style="width: 100%">
            <el-option
              v-for="v in allVehicles"
              :key="v.id"
              :label="`${v.displayName} (${v.plateNo})`"
              :value="v.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="起始日期" prop="startDate">
          <el-date-picker
            v-model="assignForm.startDate"
            type="date"
            value-format="YYYY-MM-DD"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="結束日期">
          <el-date-picker
            v-model="assignForm.endDate"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="留空代表持續有效"
            style="width: 100%"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <DialogFooter
          confirm-text="確認指派"
          :loading="submitting"
          @confirm="handleAssignSubmit"
          @cancel="assignDialogVisible = false"
        />
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { resolveErrorMessage } from '@/api/errorCodes'
import DataTablePage from '@/components/DataTablePage.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import DialogFooter from '@/components/DialogFooter.vue'
import DriverCreateDialog from '@/components/masters/DriverCreateDialog.vue'
import {
  listDrivers,
  updateDriver,
  deleteDriver,
  assignDriverVehicle,
  listVehicles
} from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import { formatDate } from '@/utils/formatters'
import { DRIVER_LICENSE_CLASS_LABELS, type DriverLicenseClass, REGION_LABELS } from '@/types/domain'
import type { DriverDTO, CreateDriverRequest, VehicleDTO } from '@/types/api'

const authStore = useAuthStore()
const drivers = ref<DriverDTO[]>([])
const allVehicles = ref<VehicleDTO[]>([])

const createDialogVisible = ref(false)
const editDialogVisible = ref(false)
const editingId = ref<string | null>(null)
const submitting = ref(false)
const formRef = ref<FormInstance>()

const assignDialogVisible = ref(false)
const selectedDriverId = ref<string | null>(null)
const assignFormRef = ref<FormInstance>()
const assignForm = reactive({
  vehicleId: '',
  startDate: new Date().toISOString().split('T')[0],
  endDate: ''
})

const assignRules = {
  vehicleId: [{ required: true, message: '請選擇車輛', trigger: 'change' }],
  startDate: [{ required: true, message: '請選擇起始日期', trigger: 'change' }]
}

const form = reactive<CreateDriverRequest>({
  name: '',
  nationalId: '',
  region: 'miaoli',
  phone: '',
  email: '',
  status: 'active',
  licenseClass: null,
  licenseExpiryDate: null
})

const rules = {
  name: [{ required: true, message: '請輸入司機姓名', trigger: 'blur' }],
  nationalId: [{ required: true, message: '請輸入身分證字號', trigger: 'blur' }],
  region: [{ required: true, message: '請選擇所屬區域', trigger: 'change' }]
}

const {
  page,
  pageSize,
  total,
  loading,
  filters,
  handlePageChange,
  handleSizeChange,
  handleSearch,
  handleReset,
  executeFetch
} = useListQuery({
  defaultFilters: {
    q: '',
    status: ''
  },
  onFetch: async () => {
    const res = await listDrivers({
      page: page.value,
      pageSize: pageSize.value,
      q: filters.q,
      status: filters.status || undefined
    })
    drivers.value = res.data
    total.value = res.meta.total
  }
})

function licenseClassLabel(value?: DriverLicenseClass | null): string {
  return value ? DRIVER_LICENSE_CLASS_LABELS[value] : '未補登'
}

async function handleQuickToggleActive(row: DriverDTO, newActive: boolean) {
  const newStatus = newActive ? 'active' : 'inactive'
  try {
    await updateDriver(row.id, { status: newStatus })
    row.status = newStatus
    ElMessage.success(`已將司機「${row.name}」狀態更新為 ${newActive ? '啟用' : '停用'}`)
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '更新司機狀態失敗'))
  }
}

function openCreateDialog() {
  createDialogVisible.value = true
}

function handleDriverCreated() {
  executeFetch()
}

function openEditDialog(row: any) {
  editingId.value = row.id
  form.name = row.name
  form.nationalId = row.nationalId
  form.region = row.region
  form.phone = row.phone || ''
  form.email = row.email || ''
  form.status = row.status
  form.licenseClass = row.licenseClass ?? null
  form.licenseExpiryDate = row.licenseExpiryDate ?? null
  editDialogVisible.value = true
}

// 一位司機同一期間只會有一台車，取目前生效的那筆指派即可
function getAssignedVehicleDisplay(row: any): { name: string; plateNo: string } | null {
  if (!row.assignments || row.assignments.length === 0) return null
  const assignment = row.assignments[row.assignments.length - 1]
  const veh = allVehicles.value.find((v) => v.id === assignment.vehicleId)

  const name = veh?.displayName || assignment.vehicleName || ''
  const plateNo = veh?.plateNo || assignment.vehiclePlateNo || assignment.plateNo || ''

  if (name && plateNo) {
    return { name, plateNo }
  }
  if (name) {
    return { name, plateNo: '' }
  }
  if (plateNo) {
    return { name: plateNo, plateNo: '' }
  }
  return { name: '已指派車輛', plateNo: '' }
}

function openAssignDialog(row: any) {
  selectedDriverId.value = row.id
  const assignment = row.assignments?.[row.assignments.length - 1]
  assignForm.vehicleId = assignment?.vehicleId || ''
  assignForm.startDate = new Date().toISOString().split('T')[0]
  assignForm.endDate = ''
  assignDialogVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value || !editingId.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      await updateDriver(editingId.value!, form)
      ElMessage.success('司機資料已更新')
      editDialogVisible.value = false
      executeFetch()
    } finally {
      submitting.value = false
    }
  })
}

async function handleAssignSubmit() {
  if (!assignFormRef.value || !selectedDriverId.value) return
  await assignFormRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      await assignDriverVehicle(selectedDriverId.value!, assignForm)
      ElMessage.success('車輛指派已更新')
      assignDialogVisible.value = false
      executeFetch()
    } finally {
      submitting.value = false
    }
  })
}

async function handleDeleteDriver(row: DriverDTO) {
  try {
    await ElMessageBox.confirm(
      `確定要刪除司機「${row.name}」？`,
      '刪除確認',
      {
        confirmButtonText: '刪除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )
    await deleteDriver(row.id)
    ElMessage.success(`司機「${row.name}」已成功刪除`)
    executeFetch()
  } catch (err: any) {
    if (err !== 'cancel') {
      ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '刪除司機失敗'))
    }
  }
}

onMounted(async () => {
  const vRes = await listVehicles({ status: 'active', pageSize: 100 })
  allVehicles.value = vRes.data
})

executeFetch()
</script>

<style scoped>
/* 狀態互動切換按鈕 / 膠囊標籤 */
/* 對話框內狀態單選群組 */
.assigned-vehicle-info {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.driver-name,
.vehicle-name {
  font-weight: 650;
  color: var(--app-text-primary);
  white-space: nowrap;
}

.driver-data {
  color: var(--app-text-regular);
  letter-spacing: 0.01em;
}

/* table-layout="auto" 下 min-width 欄位需自行鎖 nowrap，否則欄寬吃緊時會逐字換行 */
.license-value {
  white-space: nowrap;
}

:deep(.driver-name-col .cell) {
  min-width: 110px;
}

:deep(.license-class-col .cell) {
  min-width: 120px;
}

:deep(.license-expiry-col .cell) {
  min-width: 130px;
}

:deep(.email-col .cell) {
  min-width: 200px;
}

:deep(.assigned-vehicle-col .cell) {
  white-space: nowrap;
  min-width: 180px;
}

.assignment-empty {
  color: var(--app-text-muted);
  font-size: 13px;
}

.vehicle-plate {
  color: #606266;
  font-size: var(--app-font-md);
}
</style>
