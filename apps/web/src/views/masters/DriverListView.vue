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
          placeholder="營運狀態"
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
          <el-table-column prop="name" label="司機姓名" width="120" />
          <el-table-column label="身分證字號" width="160">
            <template #default="{ row }">
              <MaskedId
                :masked-value="row.nationalId"
                :on-reveal="() => fetchPlainId(row.id)"
              />
            </template>
          </el-table-column>
          <el-table-column prop="phone" label="聯絡電話" width="130" />
          <el-table-column prop="email" label="電子信箱" min-width="180" />
          <el-table-column label="目前指派主要車輛" min-width="160">
            <template #default="{ row }">
              <span v-if="row.assignments && row.assignments.length > 0">
                {{ row.assignments[0].vehicleName || '已指派車輛' }}
              </span>
              <el-tag v-else size="small" type="info">未指派</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="active" label="狀態" width="90">
            <template #default="{ row }">
              <el-tag size="small" :type="row.active ? 'success' : 'info'">
                {{ row.active ? '在職' : '離職' }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column
            v-if="authStore.can('staff')"
            label="操作"
            width="160"
            fixed="right"
            align="center"
          >
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="openEditDialog(row)">
                編輯
              </el-button>
              <el-button link type="success" size="small" @click="openAssignDialog(row)">
                指派車輛
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
          <el-input v-model="form.phone" placeholder="如：0912-345678" />
        </el-form-item>
        <el-form-item label="電子信箱" prop="email">
          <el-input v-model="form.email" placeholder="通知寄送用信箱" />
        </el-form-item>
        <el-form-item label="在職狀態" prop="active">
          <el-switch v-model="form.active" />
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
    <el-dialog v-model="assignDialogVisible" title="指派主要駕駛車輛" width="500px">
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
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance } from 'element-plus'
import MaskedId from '@/components/MaskedId.vue'
import DataTablePage from '@/components/DataTablePage.vue'
import {
  listDrivers,
  createDriver,
  updateDriver,
  revealDriverId,
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
  endDate: '',
  isPrimary: true
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

async function fetchPlainId(driverId: string): Promise<string> {
  const res = await revealDriverId(driverId)
  return res.nationalId
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

function openEditDialog(row: DriverDTO) {
  editingId.value = row.id
  form.name = row.name
  form.nationalId = row.nationalId
  form.phone = row.phone || ''
  form.email = row.email || ''
  form.active = row.active
  dialogVisible.value = true
}

function openAssignDialog(row: DriverDTO) {
  selectedDriverId.value = row.id
  assignForm.vehicleId = row.assignments?.[0]?.vehicleId || ''
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

onMounted(async () => {
  const vRes = await listVehicles({ active: true, pageSize: 100 })
  allVehicles.value = vRes.data
})

executeFetch()
</script>
