<template>
  <div class="driver-list-view">
    <DataTablePage
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
          v-model="filters.active"
          placeholder="狀態"
          clearable
          style="width: 130px"
          @change="handleSearch"
        >
          <el-option label="全部狀態" value="" />
          <el-option label="在職中" :value="true" />
          <el-option label="已離職" :value="false" />
        </el-select>

        <el-button type="primary" @click="handleSearch">查詢</el-button>
        <el-button @click="handleReset">重設</el-button>
      </template>

      <!-- 操作按鈕 -->
      <template #actions>
        <el-button
          v-if="authStore.can('staff')"
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
          <el-table-column prop="name" label="司機姓名" width="130">
            <template #default="{ row }"><span class="driver-name">{{ row.name }}</span></template>
          </el-table-column>
          <el-table-column prop="nationalId" label="身分證字號" width="140" align="center">
            <template #default="{ row }">
              <span class="driver-data font-mono">{{ row.nationalId || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="phone" label="聯絡電話" width="130" align="center">
            <template #default="{ row }">
              <span class="driver-data">{{ row.phone || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="email" label="電子信箱" min-width="200">
            <template #default="{ row }"><span class="driver-data">{{ row.email || '-' }}</span></template>
          </el-table-column>
          <el-table-column label="目前指派車輛" min-width="180">
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
          <el-table-column prop="active" label="狀態" width="130" align="center">
            <template #default="{ row }">
              <el-tooltip
                v-if="authStore.can('staff')"
                :content="row.active ? '目前為在職中，點選切換為已離職' : '目前為已離職，點選切換為在職中'"
                placement="top"
                :show-after="300"
              >
                <button
                  type="button"
                  class="status-toggle-pill"
                  :class="row.active ? 'is-active' : 'is-inactive'"
                  @click="handleQuickToggleActive(row as any, !row.active)"
                >
                  <span class="status-indicator-dot"></span>
                  <span class="status-label-text">{{ row.active ? '在職中' : '已離職' }}</span>
                </button>
              </el-tooltip>
              <div
                v-else
                class="status-toggle-pill is-readonly"
                :class="row.active ? 'is-active' : 'is-inactive'"
              >
                <span class="status-indicator-dot"></span>
                <span class="status-label-text">{{ row.active ? '在職中' : '已離職' }}</span>
              </div>
            </template>
          </el-table-column>

          <el-table-column
            v-if="authStore.can('staff')"
            label="操作"
            width="200"
            fixed="right"
            align="center"
          >
            <template #default="{ row }">
              <el-button link type="success" size="small" :icon="Edit" @click="openEditDialog(row)">
                編輯
              </el-button>
              <el-button link type="primary" size="small" :icon="Van" @click="openAssignDialog(row)">
                指派車輛
              </el-button>
              <el-button link type="danger" size="small" :icon="Delete" @click="handleDeleteDriver(row as any)">
                刪除
              </el-button>
            </template>
          </el-table-column>

        </el-table>
      </template>
    </DataTablePage>

    <!-- 新增/編輯司機彈窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '編輯司機資料' : '新增司機'"
      width="500px"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="司機姓名" prop="name">
          <el-input v-model="form.name" placeholder="請輸入姓名" />
        </el-form-item>
        <el-form-item label="身分證字號" prop="nationalId">
          <el-input v-model="form.nationalId" placeholder="1 碼英文 + 9 碼數字" />
        </el-form-item>
        <el-form-item label="聯絡電話" prop="phone">
          <el-input v-model="form.phone" placeholder="如：0912345678" />
        </el-form-item>
        <el-form-item label="電子信箱" prop="email">
          <el-input v-model="form.email" placeholder="通知寄送用信箱" />
        </el-form-item>
        <el-form-item label="狀態" prop="active">
          <el-radio-group v-model="form.active" class="status-radio-group">
            <el-radio-button :value="true">
              <div class="radio-pill active-pill">
                <span class="radio-dot"></span>
                <span>在職中</span>
              </div>
            </el-radio-button>
            <el-radio-button :value="false">
              <div class="radio-pill inactive-pill">
                <span class="radio-dot"></span>
                <span>已離職</span>
              </div>
            </el-radio-button>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          確認儲存
        </el-button>
      </template>
    </el-dialog>

    <!-- 車輛期間指派彈窗 -->
    <el-dialog v-model="assignDialogVisible" title="指派駕駛車輛" width="500px">
      <el-form ref="assignFormRef" :model="assignForm" :rules="assignRules" label-width="120px">
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
        <el-button @click="assignDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleAssignSubmit">
          確認指派
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Plus, Edit, Van, Delete } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { resolveErrorMessage } from '@/api/errorCodes'
import DataTablePage from '@/components/DataTablePage.vue'
import {
  listDrivers,
  createDriver,
  updateDriver,
  deleteDriver,
  assignDriverVehicle,
  listVehicles
} from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import type { DriverDTO, CreateDriverRequest, VehicleDTO } from '@/types/api'

const authStore = useAuthStore()
const drivers = ref<DriverDTO[]>([])
const allVehicles = ref<VehicleDTO[]>([])

const dialogVisible = ref(false)
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
  phone: '',
  email: '',
  active: true
})

const rules = {
  name: [{ required: true, message: '請輸入司機姓名', trigger: 'blur' }],
  nationalId: [{ required: true, message: '請輸入身分證字號', trigger: 'blur' }]
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
    active: ''
  },
  onFetch: async () => {
    const res = await listDrivers({
      page: page.value,
      pageSize: pageSize.value,
      q: filters.q,
      active: filters.active === '' ? undefined : Boolean(filters.active)
    })
    drivers.value = res.data
    total.value = res.meta.total
  }
})

async function handleQuickToggleActive(row: DriverDTO, newActive: boolean) {
  try {
    await updateDriver(row.id, { active: newActive })
    row.active = newActive
    ElMessage.success(`已將司機「${row.name}」狀態更新為 ${newActive ? '在職' : '離職'}`)
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '更新司機狀態失敗'))
  }
}

function openCreateDialog() {
  editingId.value = null
  form.name = ''
  form.nationalId = ''
  form.phone = ''
  form.email = ''
  form.active = true
  dialogVisible.value = true
}

function openEditDialog(row: any) {
  editingId.value = row.id
  form.name = row.name
  form.nationalId = row.nationalId
  form.phone = row.phone || ''
  form.email = row.email || ''
  form.active = row.active
  dialogVisible.value = true
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
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (editingId.value) {
        await updateDriver(editingId.value, form)
        ElMessage.success('司機資料已更新')
      } else {
        await createDriver(form)
        ElMessage.success('司機新增成功')
      }
      dialogVisible.value = false
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
      `確定要刪除司機「${row.name} (${row.code})」？`,
      '刪除確認',
      {
        confirmButtonText: '確定刪除',
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
  const vRes = await listVehicles({ active: true, pageSize: 100 })
  allVehicles.value = vRes.data
})

executeFetch()
</script>

<style scoped>
/* 狀態互動切換按鈕 / 膠囊標籤 */
.status-toggle-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  padding: 5px 14px;
  border-radius: 9999px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  border: 1px solid transparent;
  outline: none;
  background: transparent;
  user-select: none;
  line-height: 1;
}

.status-toggle-pill.is-active {
  color: #047857;
  border-color: transparent;
}

.status-toggle-pill.is-active:hover {
  border-color: #10b981;
  color: #065f46;
}

.status-toggle-pill.is-inactive {
  color: #4b5563;
  border-color: transparent;
}

.status-toggle-pill.is-inactive:hover {
  border-color: #9ca3af;
  color: #1f2937;
}

.status-toggle-pill.is-readonly {
  cursor: default;
  pointer-events: none;
}

.status-indicator-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  transition: all 0.2s ease;
}

.is-active .status-indicator-dot {
  background-color: #10b981;
  box-shadow: 0 0 0 2px rgba(16, 185, 129, 0.25);
}

.is-inactive .status-indicator-dot {
  background-color: #9ca3af;
}

.status-label-text {
  letter-spacing: 0.02em;
}

/* 彈窗內狀態單選群組 */
.status-radio-group :deep(.el-radio-button__inner) {
  padding: 8px 16px;
  border-radius: 6px !important;
  margin-right: 8px;
  border: 1px solid #dcdfe6 !important;
}

.status-radio-group :deep(.el-radio-button:first-child .el-radio-button__inner) {
  border-left: 1px solid #dcdfe6 !important;
}

.status-radio-group :deep(.el-radio-button.is-active .el-radio-button__inner) {
  background-color: #f0fdf4;
  border-color: #10b981 !important;
  color: #047857;
  box-shadow: none !important;
}

.radio-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
}

.radio-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background-color: #9ca3af;
}

.active-pill .radio-dot {
  background-color: #10b981;
}

.inactive-pill .radio-dot {
  background-color: #6b7280;
}

.assigned-vehicle-info {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.driver-name,
.vehicle-name {
  font-weight: 650;
  color: var(--el-text-color-primary);
}

.driver-data {
  color: var(--el-text-color-regular);
  letter-spacing: 0.01em;
}

.assignment-empty {
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

.vehicle-plate {
  color: #606266;
  font-size: 13px;
}
</style>
