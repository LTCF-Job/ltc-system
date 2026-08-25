<template>
  <div class="vehicle-list-view">
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
          placeholder="搜尋車牌號碼／顯示車名"
          clearable
          style="width: 240px"
          @keyup.enter="handleSearch"
        />

        <el-select
          v-model="filters.region"
          placeholder="全部區域"
          clearable
          style="width: 130px"
          @change="handleSearch"
        >
          <el-option label="全部區域" value="" />
          <el-option label="苗栗" value="miaoli" />
          <el-option label="新竹" value="hsinchu" />
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
          新增車輛
        </el-button>
      </template>

      <!-- 表格 -->
      <template #table>
        <el-table :data="vehicles" border stripe style="width: 100%">
          <el-table-column prop="displayName" label="顯示車名 (表單比對名)" width="180" />
          <el-table-column prop="plateNo" label="車牌號碼" width="140" />
          <el-table-column prop="region" label="所屬區域" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="row.region === 'miaoli' ? 'warning' : 'primary'">
                {{ REGION_LABELS[row.region as Region] }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="active" label="營運狀態" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="row.active ? 'success' : 'info'">
                {{ row.active ? '服役中' : '已停用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="createdAt" label="建立時間" min-width="160" />

          <el-table-column
            v-if="authStore.can('staff')"
            label="操作"
            width="120"
            fixed="right"
            align="center"
          >
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="openEditDialog(row)">
                編輯
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>
    </DataTablePage>

    <!-- 新增/編輯彈窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '編輯車輛資料' : '新增車輛'"
      width="500px"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="120px">
        <el-form-item label="顯示車名" prop="displayName">
          <el-input v-model="form.displayName" placeholder="如：竹北一車、竹南2車" />
        </el-form-item>
        <el-form-item label="車牌號碼" prop="plateNo">
          <el-input v-model="form.plateNo" placeholder="如：BZG-7915" />
        </el-form-item>
        <el-form-item label="所屬區域" prop="region">
          <el-radio-group v-model="form.region">
            <el-radio value="miaoli">苗栗</el-radio>
            <el-radio value="hsinchu">新竹</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="啟用狀態" prop="active">
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
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance } from 'element-plus'
import DataTablePage from '@/components/DataTablePage.vue'
import { listVehicles, createVehicle, updateVehicle } from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import { REGION_LABELS, type Region } from '@/types/domain'
import type { VehicleDTO, CreateVehicleRequest } from '@/types/api'

const authStore = useAuthStore()
const vehicles = ref<VehicleDTO[]>([])
const dialogVisible = ref(false)
const editingId = ref<string | null>(null)
const submitting = ref(false)
const formRef = ref<FormInstance>()

const form = reactive<CreateVehicleRequest>({
  displayName: '',
  plateNo: '',
  region: 'miaoli',
  active: true
})

const rules = {
  displayName: [{ required: true, message: '請輸入顯示車名', trigger: 'blur' }],
  plateNo: [
    { required: true, message: '請輸入車牌號碼', trigger: 'blur' },
    { pattern: /^[A-Z0-9]{2,4}-[A-Z0-9]{2,4}$/, message: '車牌格式錯誤 (例如 BZG-7915)', trigger: 'blur' }
  ],
  region: [{ required: true, message: '請選擇區域', trigger: 'change' }]
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
    region: ''
  },
  onFetch: async () => {
    const res = await listVehicles({
      page: page.value,
      pageSize: pageSize.value,
      q: filters.q,
      region: filters.region
    })
    vehicles.value = res.data
    total.value = res.meta.total
  }
})

function openCreateDialog() {
  editingId.value = null
  form.displayName = ''
  form.plateNo = ''
  form.region = 'miaoli'
  form.active = true
  dialogVisible.value = true
}

function openEditDialog(row: VehicleDTO) {
  editingId.value = row.id
  form.displayName = row.displayName
  form.plateNo = row.plateNo
  form.region = row.region
  form.active = row.active
  dialogVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (editingId.value) {
        await updateVehicle(editingId.value, form)
        ElMessage.success('車輛資料已更新')
      } else {
        await createVehicle(form)
        ElMessage.success('車輛新增成功')
      }
      dialogVisible.value = false
      executeFetch()
    } finally {
      submitting.value = false
    }
  })
}

executeFetch()
</script>
