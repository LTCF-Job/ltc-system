<template>
  <div class="maintenance-view">
    <DataTablePage
      title="車輛維修保養管理"
      description="車隊定期保養、臨時維修紀錄登錄與空白檢查表下載"
      :max-width="1370"
      :loading="loading"
      v-model:page="page"
      v-model:pageSize="pageSize"
      :total="total"
      @page-change="fetchList"
      @size-change="fetchList"
    >
      <template #filter>
        <el-input
          v-model="searchQuery"
          placeholder="搜尋車牌／車名／保養廠／項目／備註"
          clearable
          style="width: 240px"
          @keyup.enter="fetchList"
        />

        <el-select
          v-model="queryVehicleId"
          placeholder="全部車輛"
          clearable
          style="width: 150px"
          @change="fetchList"
        >
          <el-option
            v-for="veh in vehicles"
            :key="veh.id"
            :label="veh.displayName"
            :value="veh.id"
          />
        </el-select>

        <el-date-picker
          v-model="dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="開始日期"
          end-placeholder="結束日期"
          value-format="YYYY-MM-DD"
          style="width: 240px"
          @change="fetchList"
        />

        <el-button type="primary" @click="fetchList">
          查詢
        </el-button>
        <el-button @click="handleReset">
          重設
        </el-button>
      </template>

      <template #actions>
        <el-button plain @click="handleDownloadBlank" :loading="downloadingBlank">
          下載空白保養表 (.xlsx)
        </el-button>

        <el-button v-if="authStore.can('staff')" type="primary" :icon="Plus" @click="openCreateDialog">
          新增保養紀錄
        </el-button>
      </template>

      <template #table>
      <el-table :data="records" border stripe size="small" table-layout="auto" style="width: 100%">
        <el-table-column prop="serviceDate" label="保養日期" width="110" align="center">
          <template #default="{ row }">
            <span>{{ row.serviceDate?.slice(0, 10) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="vehicleName" label="車輛名稱" width="120" />
        <el-table-column prop="plateNo" label="車牌號碼" width="110" align="center" />
        <el-table-column prop="mileage" label="里程數 (km)" width="120" align="right">
          <template #default="{ row }">
            {{ Number(row.mileage).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column prop="items" label="保養／維修項目" min-width="200" show-overflow-tooltip />
        <el-table-column prop="vendor" label="廠商／車廠" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.vendor || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="cost" label="花費金額" width="110" align="right">
          <template #default="{ row }">
            <span class="font-bold">${{ Number(row.cost).toLocaleString() }}</span>
          </template>
        </el-table-column>
        <el-table-column label="收據憑證" width="100" align="center">
          <template #default="{ row }">
            <el-link
              v-if="row.receiptUrl"
              type="primary"
              :href="row.receiptUrl"
              target="_blank"
            >
              檢視憑證
            </el-link>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="note" label="備註" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.note || '-' }}
          </template>
        </el-table-column>
        <el-table-column v-if="authStore.can('staff')" label="操作" width="140" align="center" fixed="right">
          <template #default="{ row }">
            <TableRowActions>
              <el-button link type="primary" size="small" @click="openEditDialog(row)">
                編輯
              </el-button>
              <el-button link type="danger" size="small" @click="handleDelete(row)">
                刪除
              </el-button>
            </TableRowActions>
          </template>
        </el-table-column>
      </el-table>
      </template>
    </DataTablePage>

    <!-- 新增 / 編輯保養紀錄 Dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '編輯保養紀錄' : '新增保養紀錄'"
      width="min(600px, calc(100vw - 32px))"
      destroy-on-close
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="110px"
        label-position="right"
      >
        <el-form-item label="保養車輛" prop="vehicleId">
          <el-select v-model="form.vehicleId" placeholder="請選擇車輛" style="width: 100%">
            <el-option
              v-for="veh in vehicles"
              :key="veh.id"
              :label="`${veh.displayName} (${veh.plateNo})`"
              :value="veh.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="保養日期" prop="serviceDate">
          <el-date-picker
            v-model="form.serviceDate"
            type="date"
            placeholder="選擇保養日期"
            value-format="YYYY-MM-DD"
            style="width: 100%"
          />
        </el-form-item>

        <el-form-item label="保養里程" prop="mileage">
          <el-input-number
            v-model="form.mileage"
            :min="0"
            :step="100"
            style="width: 100%"
            placeholder="當前公里數"
          />
        </el-form-item>

        <el-form-item label="保養項目" prop="items">
          <el-input
            v-model="form.items"
            type="textarea"
            :rows="3"
            placeholder="例如：更換機油、煞車皮、檢查五油三水"
          />
        </el-form-item>

        <el-form-item label="保養廠商" prop="vendor">
          <el-input v-model="form.vendor" placeholder="例如：順益汽車、原廠保修站" />
        </el-form-item>

        <el-form-item label="花費金額" prop="cost">
          <el-input-number
            v-model="form.cost"
            :min="0"
            :step="500"
            style="width: 100%"
            placeholder="維修保養總花費"
          />
        </el-form-item>

        <el-form-item label="收據連結" prop="receiptUrl">
          <el-input v-model="form.receiptUrl" placeholder="https://..." />
        </el-form-item>

        <el-form-item label="備註說明" prop="note">
          <el-input v-model="form.note" placeholder="選填補充說明" />
        </el-form-item>
      </el-form>

      <template #footer>
        <DialogFooter :loading="saving" @confirm="handleSave" @cancel="dialogVisible = false" />
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import DataTablePage from '@/components/DataTablePage.vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import DialogFooter from '@/components/DialogFooter.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import { resolveErrorMessage } from '@/api/errorCodes'
import {
  listMaintenance,
  createMaintenance,
  updateMaintenance,
  deleteMaintenance,
  downloadBlankMaintenanceExcel
} from '@/api/maintenance'
import { listVehicles } from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import { downloadBlob } from '@/utils/download'
import type { MaintenanceLogDTO, VehicleDTO } from '@/types/api'

const authStore = useAuthStore()

const loading = ref(false)
const saving = ref(false)
const downloadingBlank = ref(false)

const vehicles = ref<VehicleDTO[]>([])
const searchQuery = ref('')
const queryVehicleId = ref<string>()
const dateRange = ref<[string, string]>()

const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const records = ref<MaintenanceLogDTO[]>([])

const dialogVisible = ref(false)
const editingId = ref<string | null>(null)
const formRef = ref<FormInstance>()

const form = reactive({
  vehicleId: '',
  serviceDate: '',
  mileage: 0,
  items: '',
  vendor: '',
  cost: 0,
  receiptUrl: '',
  note: ''
})

const rules = {
  vehicleId: [{ required: true, message: '請選擇車輛', trigger: 'change' }],
  serviceDate: [{ required: true, message: '請選擇保養日期', trigger: 'change' }],
  mileage: [{ required: true, message: '請輸入當前里程數', trigger: 'blur' }],
  items: [{ required: true, message: '請輸入保養項目', trigger: 'blur' }],
  cost: [{ required: true, message: '請輸入保養金額', trigger: 'blur' }]
}

async function fetchFilterOptions() {
  try {
    const res = await listVehicles({ pageSize: 100 })
    vehicles.value = res.data
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '載入車輛清單失敗'))
  }
}

async function fetchList() {
  loading.value = true
  try {
    const res = await listMaintenance({
      page: page.value,
      pageSize: pageSize.value,
      vehicleId: queryVehicleId.value,
      startDate: dateRange.value?.[0],
      endDate: dateRange.value?.[1],
      q: searchQuery.value || undefined
    })
    records.value = res.data
    total.value = res.meta.total
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '查詢維修保養紀錄失敗'))
  } finally {
    loading.value = false
  }
}

function handleReset() {
  searchQuery.value = ''
  queryVehicleId.value = undefined
  dateRange.value = undefined
  page.value = 1
  fetchList()
}

function openCreateDialog() {
  editingId.value = null
  form.vehicleId = queryVehicleId.value || ''
  form.serviceDate = new Date().toISOString().slice(0, 10)
  form.mileage = 0
  form.items = ''
  form.vendor = ''
  form.cost = 0
  form.receiptUrl = ''
  form.note = ''
  dialogVisible.value = true
}

function openEditDialog(row: any) {
  editingId.value = row.id
  form.vehicleId = row.vehicleId
  form.serviceDate = row.serviceDate?.slice(0, 10)
  form.mileage = row.mileage
  form.items = row.items
  form.vendor = row.vendor || ''
  form.cost = row.cost
  form.receiptUrl = row.receiptUrl || ''
  form.note = row.note || ''
  dialogVisible.value = true
}

async function handleSave() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      if (editingId.value) {
        await updateMaintenance(editingId.value, form)
        ElMessage.success('保養紀錄修改成功')
      } else {
        await createMaintenance(form)
        ElMessage.success('保養紀錄新增成功')
      }
      dialogVisible.value = false
      fetchList()
    } catch (err: any) {
      ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '儲存保養紀錄失敗'))
    } finally {
      saving.value = false
    }
  })
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(
      `確定刪除「${row.vehicleName || '車輛'}」於 ${row.serviceDate?.slice(0, 10)} 之保養紀錄？`,
      '刪除確認',
      { type: 'warning' }
    )
    await deleteMaintenance(row.id)
    ElMessage.success('保養紀錄已刪除')
    fetchList()
  } catch (err: any) {
    if (err !== 'cancel') {
      ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '刪除失敗'))
    }
  }
}

async function handleDownloadBlank() {
  downloadingBlank.value = true
  try {
    const blob = await downloadBlankMaintenanceExcel()
    downloadBlob(blob, '車輛定期保養檢查表_空白範本.xlsx')
    ElMessage.success('空白保養表下載成功')
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '下載空白保養表失敗'))
  } finally {
    downloadingBlank.value = false
  }
}

onMounted(async () => {
  await fetchFilterOptions()
  await fetchList()
})
</script>

<style scoped>
.maintenance-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.font-bold {
  font-weight: 600;
}

.text-muted {
  color: var(--el-text-color-placeholder);
}

</style>
