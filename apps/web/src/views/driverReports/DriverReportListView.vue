<template>
  <div class="driver-report-list-view">
    <el-card shadow="never" class="table-card">
      <template #header>
        <div class="card-header">
          <div class="header-filters">
            <span class="title">司機接送匯報（共 {{ forms.length }} 台車）</span>
            <el-input
              v-model="searchQuery"
              placeholder="搜尋匯報表名稱／車輛"
              clearable
              style="width: 240px"
              @keyup.enter="fetchForms"
            />
            <el-button type="primary" icon="Search" @click="fetchForms">查詢</el-button>
            <el-button @click="handleReset">重設</el-button>
          </div>
          <div class="header-actions">
            <el-button
              v-if="canEdit"
              type="primary"
              icon="Plus"
              @click="openCreateDialog"
            >
              新增車輛匯報表
            </el-button>
          </div>
        </div>
      </template>

      <el-table :data="forms" border stripe v-loading="loading">
        <el-table-column prop="title" label="匯報表名稱" min-width="200" />

        <el-table-column label="所屬車輛／地區" width="180">
          <template #default="{ row }">
            <span>{{ row.vehicleName }}</span>
            <span v-if="row.region" class="text-secondary">
              （{{ REGION_LABELS[row.region] || row.region }}）
            </span>
          </template>
        </el-table-column>

        <el-table-column label="欄位對應進度" width="180" align="center">
          <template #default="{ row }">
            <span class="mapping-status" :class="row.pendingColumns > 0 ? 'is-pending' : 'is-ready'">
              已對應 {{ row.mappedColumns }} / {{ row.totalColumns }}
            </span>
            <div v-if="row.pendingColumns > 0" class="pending-hint">
              {{ row.pendingColumns }} 欄待對應
            </div>
          </template>
        </el-table-column>

        <el-table-column label="最後匯入時間" width="190" align="center">
          <template #default="{ row }">
            {{ formatDateTime(row.lastImportedAt, '尚未匯入') }}
          </template>
        </el-table-column>

        <el-table-column label="累計匯報天數" width="130" align="center" prop="submissionCount" />

        <el-table-column label="操作" width="260" fixed="right" align="center">
          <template #default="{ row }">
            <el-button
              v-if="canEdit"
              link
              type="primary"
              size="small"
              :icon="Upload"
              @click="openImportDialog(row)"
            >
              匯入
            </el-button>
            <el-button
              link
              type="success"
              size="small"
              :icon="Edit"
              @click="$router.push(`/driver-reports/mappings?formId=${row.id}`)"
            >
              欄位對應
            </el-button>
            <el-button
              v-if="canEdit"
              link
              type="danger"
              size="small"
              :icon="Delete"
              @click="handleDelete(row)"
            >
              刪除
            </el-button>
          </template>
        </el-table-column>

        <template #empty>
          <div class="empty-state">
            <p>尚未建立任何車輛的接送匯報表。</p>
            <p class="text-secondary">先為一台車建立匯報表，再上傳司機填寫的 .xlsx 匯報檔。</p>
          </div>
        </template>
      </el-table>
    </el-card>

    <!-- 新增車輛匯報表 -->
    <el-dialog v-model="createDialogVisible" title="新增車輛匯報表" width="480px" destroy-on-close>
      <el-form :model="createForm" label-width="100px">
        <el-form-item label="所屬車輛" required>
          <el-select
            v-model="createForm.vehicleId"
            placeholder="選擇車輛"
            filterable
            style="width: 100%"
            @change="onVehicleChange"
          >
            <el-option
              v-for="v in vehicles"
              :key="v.id"
              :label="v.displayName"
              :value="v.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="匯報表名稱" required>
          <el-input v-model="createForm.title" placeholder="例如：竹南2車接送匯報" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="creating"
          :disabled="!createForm.vehicleId || !createForm.title"
          @click="handleCreate"
        >
          建立
        </el-button>
      </template>
    </el-dialog>

    <DriverReportImportDialog ref="importDialogRef" @success="fetchForms" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Delete, Edit, Upload } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listDriverReportForms,
  createDriverReportForm,
  deleteDriverReportForm
} from '@/api/driverReports'
import { listVehicles } from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/utils/formatters'
import { REGION_LABELS } from '@/types/domain'
import DriverReportImportDialog from './DriverReportImportDialog.vue'
import type { DriverReportFormDTO, VehicleDTO } from '@/types/api'

const authStore = useAuthStore()

const forms = ref<DriverReportFormDTO[]>([])
const vehicles = ref<VehicleDTO[]>([])
const searchQuery = ref('')
const loading = ref(false)
const creating = ref(false)
const createDialogVisible = ref(false)
const createForm = ref<{ vehicleId: string; title: string }>({ vehicleId: '', title: '' })
const importDialogRef = ref<InstanceType<typeof DriverReportImportDialog> | null>(null)

const canEdit = computed(() => authStore.hasPermission('driver_reports', 'edit'))

async function fetchForms() {
  loading.value = true
  try {
    forms.value = await listDriverReportForms({ q: searchQuery.value || undefined })
  } finally {
    loading.value = false
  }
}

function handleReset() {
  searchQuery.value = ''
  fetchForms()
}

async function openCreateDialog() {
  createForm.value = { vehicleId: '', title: '' }
  createDialogVisible.value = true
  if (vehicles.value.length === 0) {
    const res = await listVehicles({ pageSize: 100, active: true })
    vehicles.value = res.data
  }
}

// 車輛選定後預填名稱，讓多數情況不必手動輸入
function onVehicleChange(vehicleId: string) {
  const vehicle = vehicles.value.find((v) => v.id === vehicleId)
  if (vehicle && !createForm.value.title) {
    createForm.value.title = `${vehicle.displayName}接送匯報`
  }
}

async function handleCreate() {
  creating.value = true
  try {
    await createDriverReportForm({
      vehicleId: createForm.value.vehicleId,
      title: createForm.value.title
    })
    ElMessage.success('已建立車輛匯報表')
    createDialogVisible.value = false
    fetchForms()
  } finally {
    creating.value = false
  }
}

function openImportDialog(row: any) {
  importDialogRef.value?.open(row)
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(
    `刪除「${row.title}」會一併移除其欄位對應與匯報紀錄，已寫入的搭乘紀錄不受影響。確定刪除？`,
    '刪除匯報表',
    { type: 'warning', confirmButtonText: '確定刪除', cancelButtonText: '取消' }
  )
  await deleteDriverReportForm(row.id)
  ElMessage.success('已刪除匯報表')
  fetchForms()
}

onMounted(fetchForms)
</script>

<style scoped>
.driver-report-list-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.table-card {
  border-radius: 8px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.header-filters {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.header-filters .title {
  font-weight: 600;
  font-size: 16px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.text-secondary {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.mapping-status {
  font-weight: 500;
}

.mapping-status.is-pending {
  color: var(--el-color-warning);
}

.mapping-status.is-ready {
  color: var(--el-color-success);
}

.pending-hint {
  font-size: 12px;
  color: var(--el-color-warning);
}

.empty-state {
  padding: 24px 0;
  line-height: 1.8;
}
</style>
