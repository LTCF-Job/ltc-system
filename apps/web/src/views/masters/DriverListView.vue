<template>
  <div class="driver-list-view">
    <DataTablePage
      title="司機管理"
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
        <el-table :data="drivers" border stripe style="width: 100%">
          <el-table-column prop="name" label="司機姓名" min-width="110" align="center" class-name="driver-nowrap-col driver-name-col">
            <template #default="{ row }"><span class="driver-name">{{ row.name }}</span></template>
          </el-table-column>
          <el-table-column prop="nationalId" label="身分證字號" min-width="140" align="center" class-name="driver-nowrap-col driver-id-col">
            <template #default="{ row }">
              <span class="driver-data font-mono">{{ row.nationalId || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="gender" label="性別" min-width="70" align="center" class-name="driver-nowrap-col driver-gender-col">
            <template #default="{ row }">
              <span class="driver-data">{{ row.gender || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="birthDate" label="生日" min-width="120" align="center" class-name="driver-nowrap-col driver-birth-col">
            <template #default="{ row }">
              <span class="driver-data">{{ formatDate(row.birthDate) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="歲數" min-width="70" align="center" class-name="driver-nowrap-col driver-age-col">
            <template #default="{ row }">
              <span class="driver-data font-mono">{{ calcAge(row.birthDate) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="hasProfessionalLicense" label="職業駕照" min-width="90" align="center" class-name="driver-nowrap-col driver-pro-license-col">
            <template #default="{ row }">
              <span class="driver-data font-semibold text-emerald-600">
                {{ row.hasProfessionalLicense ? 'V' : '-' }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="hasTransferCert" label="異動登記書" min-width="105" align="center" class-name="driver-nowrap-col driver-transfer-cert-col">
            <template #default="{ row }">
              <span class="driver-data font-semibold text-emerald-600">
                {{ row.hasTransferCert ? 'V' : '-' }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="licenseClass" label="駕照類別" min-width="120" align="center" class-name="driver-nowrap-col license-class-col">
            <template #default="{ row }">
              <span class="driver-data license-value">
                {{ licenseClassLabel(row.licenseClass) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="licenseExpiryDate" label="駕照有效日期" min-width="130" align="center" class-name="driver-nowrap-col license-expiry-col">
            <template #default="{ row }">
              <span class="driver-data license-value">{{ formatDate(row.licenseExpiryDate) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="employmentDate" label="到職日" min-width="120" align="center" class-name="driver-nowrap-col driver-employment-col">
            <template #default="{ row }">
              <span class="driver-data">{{ formatDate(row.employmentDate) }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="phone" label="聯絡電話" min-width="130" align="center" class-name="driver-nowrap-col driver-phone-col">
            <template #default="{ row }">
              <span class="driver-data">{{ row.phone || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="email" label="電子信箱" min-width="180" show-overflow-tooltip class-name="driver-nowrap-col email-col">
            <template #default="{ row }"><span class="driver-data">{{ row.email || '-' }}</span></template>
          </el-table-column>
          <el-table-column label="目前指派車輛" min-width="220" class-name="driver-nowrap-col assigned-vehicle-col">
            <template #default="{ row }">
              <InlineOptionPicker
                v-if="authStore.hasPermission('masters_drivers', 'edit')"
                :model-value="getAssignedVehicleId(row)"
                :options="vehicleOptions"
                clearable
                placeholder="尚未指派"
                :loading="assigningDriverId === row.id"
                :disabled="assigningDriverId === row.id || row.status !== 'active'"
                @change="(val) => handleInlineAssignVehicle(row, val as string)"
              />
              <div v-else-if="getAssignedVehicleDisplay(row)" class="assigned-vehicle-info">
                <span class="vehicle-name">{{ getAssignedVehicleDisplay(row)?.name }}</span>
                <span v-if="getAssignedVehicleDisplay(row)?.plateNo" class="vehicle-plate font-mono">
                  ({{ getAssignedVehicleDisplay(row)?.plateNo }})
                </span>
              </div>
              <span v-else class="assignment-empty">尚未指派</span>
            </template>
          </el-table-column>
          <el-table-column prop="remarks" label="備註" min-width="140" show-overflow-tooltip class-name="driver-remarks-col">
            <template #default="{ row }">
              <span class="driver-data">{{ row.remarks || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="狀態" min-width="130" align="center" class-name="driver-nowrap-col driver-status-col">
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
            min-width="140"
            fixed="right"
            align="center"
            class-name="driver-nowrap-col driver-actions-col"
          >
            <template #default="{ row }">
              <TableRowActions>
                <el-button
                  v-if="authStore.hasPermission('masters_drivers', 'edit')"
                  link
                  type="primary"
                  size="small"
                  @click="openEditDialog(row)"
                >
                  編輯
                </el-button>
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
    <el-dialog v-model="editDialogVisible" title="編輯司機資料" width="min(560px, calc(100vw - 32px))">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
        <el-form-item label="司機姓名" prop="name">
          <el-input v-model="form.name" placeholder="請輸入姓名" />
        </el-form-item>
        <el-form-item label="身分證字號" prop="nationalId">
          <el-input
            v-model="form.nationalId"
            :placeholder="`目前為 ${editingCurrentId || '未設定'}，留空表示不變更`"
            clearable
          />
        </el-form-item>
        <el-form-item label="性別" prop="gender">
          <el-select v-model="form.gender" placeholder="請選擇性別" clearable style="width: 100%">
            <el-option value="男" label="男" />
            <el-option value="女" label="女" />
          </el-select>
        </el-form-item>
        <el-form-item label="生日" prop="birthDate">
          <el-date-picker
            v-model="form.birthDate"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="請選擇出生日期"
            clearable
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="電子信箱" prop="email">
          <el-input v-model="form.email" placeholder="通知寄送用信箱" clearable />
        </el-form-item>
        <el-form-item label="到職日" prop="employmentDate">
          <el-date-picker
            v-model="form.employmentDate"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="請選擇到職日"
            clearable
            style="width: 100%"
          />
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
            clearable
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="證照證明">
          <el-checkbox v-model="form.hasProfessionalLicense">職業駕照</el-checkbox>
          <el-checkbox v-model="form.hasTransferCert">異動登記書</el-checkbox>
        </el-form-item>
        <el-form-item label="備註" prop="remarks">
          <el-input v-model="form.remarks" type="textarea" :rows="2" placeholder="選填備註" clearable />
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
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import DataTablePage from '@/components/DataTablePage.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import DialogFooter from '@/components/DialogFooter.vue'
import DriverCreateDialog from '@/components/masters/DriverCreateDialog.vue'
import InlineOptionPicker from '@/components/InlineOptionPicker.vue'
import {
  listDrivers,
  updateDriver,
  deleteDriver,
  assignDriverVehicle,
  unassignDriverVehicle,
  listAllVehicles
} from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import { formatDate } from '@/utils/formatters'
import { DRIVER_LICENSE_CLASS_LABELS, type DriverLicenseClass } from '@/types/domain'
import type { DriverDTO, CreateDriverRequest, UpdateDriverRequest, VehicleDTO } from '@/types/api'

import { nationalIdRules } from '@/utils/driverForm'

const editingCurrentId = ref('')

const authStore = useAuthStore()
const drivers = ref<DriverDTO[]>([])
const allVehicles = ref<VehicleDTO[]>([])

const createDialogVisible = ref(false)
const editDialogVisible = ref(false)
const editingId = ref<string | null>(null)
const submitting = ref(false)
const formRef = ref<FormInstance>()

const assigningDriverId = ref<string | null>(null)

const form = reactive<CreateDriverRequest & UpdateDriverRequest>({
  name: '',
  nationalId: '',
  email: '',
  gender: '',
  birthDate: null,
  hasProfessionalLicense: false,
  employmentDate: null,
  hasTransferCert: false,
  remarks: '',
  status: 'active',
  licenseClass: null,
  licenseExpiryDate: null
})

const rules = {
  name: [{ required: true, message: '請輸入司機姓名', trigger: 'blur' }],
  nationalId: nationalIdRules(false)
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

function calcAge(birthDate?: string | null): number | string {
  if (!birthDate) return '-'
  const birth = new Date(birthDate)
  if (isNaN(birth.getTime())) return '-'
  const today = new Date()
  let age = today.getFullYear() - birth.getFullYear()
  const m = today.getMonth() - birth.getMonth()
  if (m < 0 || (m === 0 && today.getDate() < birth.getDate())) {
    age--
  }
  return age >= 0 ? age : '-'
}

async function handleQuickToggleActive(row: DriverDTO, newActive: boolean) {
  const newStatus = newActive ? 'active' : 'inactive'
  try {
    await updateDriver(row.id, { status: newStatus })
    row.status = newStatus
    ElMessage.success(`已將司機「${row.name}」狀態更新為 ${newActive ? '啟用' : '停用'}`)
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  }
}

function openCreateDialog() {
  createDialogVisible.value = true
}

const vehicleOptions = computed(() =>
  allVehicles.value.map((v) => ({ label: `${v.displayName} (${v.plateNo})`, value: v.id }))
)

async function reloadVehicles() {
  try {
    allVehicles.value = await listAllVehicles({ status: 'active' })
  } catch {
    // 忽略抓取車輛錯誤
  }
}

async function handleDriverCreated() {
  await reloadVehicles()
  executeFetch()
}

function openEditDialog(row: any) {
  editingId.value = row.id
  form.name = row.name
  form.nationalId = ''
  editingCurrentId.value = row.nationalId || ''
  form.email = row.email || ''
  form.gender = row.gender || ''
  form.birthDate = row.birthDate ? row.birthDate.substring(0, 10) : null
  form.hasProfessionalLicense = !!row.hasProfessionalLicense
  form.employmentDate = row.employmentDate ? row.employmentDate.substring(0, 10) : null
  form.hasTransferCert = !!row.hasTransferCert
  form.remarks = row.remarks || ''
  form.status = row.status
  form.licenseClass = row.licenseClass ?? null
  form.licenseExpiryDate = row.licenseExpiryDate ? row.licenseExpiryDate.substring(0, 10) : null
  editDialogVisible.value = true
}

// 一位司機同一期間只會有一台車，取目前生效的那筆指派即可
function getAssignedVehicleDisplay(row: any): { name: string; plateNo: string } | null {
  if (row.assignments && row.assignments.length > 0) {
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

  // 若 driver 物件無 assignments 欄位，回查車輛清單中掛載的司機
  const matchedVeh = allVehicles.value.find((v) => v.drivers?.some((d) => d.id === row.id))
  if (matchedVeh) {
    return { name: matchedVeh.displayName, plateNo: matchedVeh.plateNo }
  }

  return null
}

function getAssignedVehicleId(row: any): string {
  if (row.assignments && row.assignments.length > 0) {
    const assignment = row.assignments[row.assignments.length - 1]
    if (assignment?.vehicleId) {
      return assignment.vehicleId
    }
  }
  const matchedVeh = allVehicles.value.find((v) => v.drivers?.some((d) => d.id === row.id))
  return matchedVeh?.id || ''
}

async function handleInlineAssignVehicle(row: any, newVehicleId: string) {
  const currentVehicleId = getAssignedVehicleId(row)
  if (newVehicleId === currentVehicleId) return

  assigningDriverId.value = row.id
  try {
    if (newVehicleId) {
      await assignDriverVehicle(row.id, { vehicleId: newVehicleId })
      const targetVeh = allVehicles.value.find((v) => v.id === newVehicleId)
      const vehName = targetVeh ? `${targetVeh.displayName} (${targetVeh.plateNo})` : '車輛'
      ElMessage.success(`已將司機「${row.name}」指派至 ${vehName}`)
    } else {
      await unassignDriverVehicle(row.id)
      ElMessage.success(`已解除司機「${row.name}」的車輛指派`)
    }
    await reloadVehicles()
    executeFetch()
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  } finally {
    assigningDriverId.value = null
  }
}

async function handleSubmit() {
  if (!formRef.value || !editingId.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      await updateDriver(editingId.value!, {
        name: form.name,
        // 留空代表不變更身分證，避免把遮罩值或空字串送成新的身分證。
        nationalId: form.nationalId?.trim() || undefined,
        email: form.email,
        gender: form.gender,
        birthDate: form.birthDate,
        hasProfessionalLicense: form.hasProfessionalLicense,
        employmentDate: form.employmentDate,
        hasTransferCert: form.hasTransferCert,
        remarks: form.remarks,
        status: form.status,
        licenseClass: form.licenseClass,
        licenseExpiryDate: form.licenseExpiryDate
      })
      ElMessage.success('司機資料已更新')
      editDialogVisible.value = false
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
    await reloadVehicles()
    executeFetch()
  } catch {
    // 使用者取消或 API 錯誤皆不在此重複顯示。
  }
}

onMounted(async () => {
  await reloadVehicles()
})

executeFetch()
</script>

<style scoped>
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
  white-space: nowrap;
}

/* table-layout="auto" 下 min-width 欄位需自行鎖 nowrap，否則欄寬吃緊時會逐字換行 */
.license-value {
  white-space: nowrap;
}

:deep(.driver-nowrap-col .cell) {
  white-space: nowrap;
}

:deep(.driver-name-col .cell) {
  min-width: 110px;
}

:deep(.driver-id-col .cell) {
  min-width: 140px;
}

:deep(.driver-gender-col .cell) {
  min-width: 70px;
}

:deep(.driver-birth-col .cell) {
  min-width: 120px;
}

:deep(.driver-age-col .cell) {
  min-width: 70px;
}

:deep(.driver-pro-license-col .cell) {
  min-width: 90px;
}

:deep(.driver-transfer-cert-col .cell) {
  min-width: 105px;
}

:deep(.license-class-col .cell) {
  min-width: 120px;
}

:deep(.license-expiry-col .cell) {
  min-width: 130px;
}

:deep(.driver-employment-col .cell) {
  min-width: 120px;
}

:deep(.driver-phone-col .cell) {
  min-width: 130px;
}

:deep(.email-col .cell) {
  min-width: 180px;
}

:deep(.assigned-vehicle-col .cell) {
  min-width: 220px;
}

:deep(.driver-remarks-col .cell) {
  min-width: 140px;
}

:deep(.driver-status-col .cell) {
  min-width: 130px;
}

:deep(.driver-actions-col .cell) {
  min-width: 140px;
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
