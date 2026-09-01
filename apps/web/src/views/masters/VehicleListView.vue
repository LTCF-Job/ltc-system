<template>
  <div class="vehicle-list-view">
    <DataTablePage
      title="車輛管理"
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
          v-model="filters.siteId"
          placeholder="全部單位"
          clearable
          filterable
          style="width: 200px"
          @change="handleSearch"
        >
          <el-option label="全部單位" value="" />
          <el-option v-for="s in allSites" :key="s.id" :label="s.name" :value="s.id" />
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
        <el-table
          :data="vehicles"
          border
          stripe
          table-layout="auto"
          style="width: 100%"
          @row-dblclick="(row: any) => handleRowDblClick(row)"
        >
          <el-table-column type="index" label="編號" width="70" align="center" :index="rowIndex" />

          <el-table-column prop="plateNo" label="車號" min-width="120" class-name="vehicle-nowrap-col">
            <template #default="{ row }">
              <span class="font-mono text-id">{{ row.plateNo }}</span>
            </template>
          </el-table-column>

          <el-table-column prop="displayName" label="顯示車名 (表單比對名)" min-width="180" class-name="vehicle-nowrap-col" />

          <el-table-column prop="siteName" label="所屬單位" min-width="160" class-name="vehicle-nowrap-col">
            <template #default="{ row }">
              <span v-if="row.siteName">{{ row.siteName }}</span>
              <span v-else class="vehicle-empty-text">未指定</span>
            </template>
          </el-table-column>

          <el-table-column prop="ownerName" label="車主名稱" min-width="180" class-name="vehicle-nowrap-col" />

          <el-table-column prop="brand" label="廠牌" min-width="90" class-name="vehicle-nowrap-col" />

          <el-table-column prop="model" label="車型" min-width="120" class-name="vehicle-nowrap-col" />

          <el-table-column label="出廠年月" min-width="120" align="center" class-name="vehicle-nowrap-col">
            <template #default="{ row }">{{ formatYearMonth(row.manufactureYm) }}</template>
          </el-table-column>

          <el-table-column label="強制責任險 (年/月/日)" min-width="150" align="center" class-name="vehicle-nowrap-col">
            <template #default="{ row }">{{ formatRocDate(row.compulsoryInsuranceExpiry) }}</template>
          </el-table-column>

          <el-table-column label="乘客責任險 (年/月/日)" min-width="150" align="center" class-name="vehicle-nowrap-col">
            <template #default="{ row }">{{ formatRocDate(row.passengerInsuranceExpiry) }}</template>
          </el-table-column>

          <el-table-column label="第三人責任險 (年/月/日)" min-width="160" align="center" class-name="vehicle-nowrap-col">
            <template #default="{ row }">{{ formatRocDate(row.thirdPartyInsuranceExpiry) }}</template>
          </el-table-column>

          <el-table-column label="前次檢驗日期 (年/月/日)" min-width="160" align="center" class-name="vehicle-nowrap-col">
            <template #default="{ row }">{{ formatRocDate(row.lastInspectionDate) }}</template>
          </el-table-column>

          <el-table-column label="符合輪椅載運規定" min-width="140" align="center" class-name="vehicle-nowrap-col">
            <template #default="{ row }">
              <el-tag
                v-if="row.wheelchairAccessible !== null"
                size="small"
                :type="row.wheelchairAccessible ? 'success' : 'info'"
                effect="plain"
              >
                {{ row.wheelchairAccessible ? '是' : '否' }}
              </el-tag>
              <span v-else class="vehicle-empty-text">未填</span>
            </template>
          </el-table-column>

          <el-table-column prop="active" label="狀態" width="130" align="center">
            <template #default="{ row }">
              <el-tooltip
                v-if="authStore.can('staff')"
                :content="row.active ? '目前為服役中，點選切換為已停用' : '目前為已停用，點選切換為服役中'"
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
                  <span class="status-label-text">{{ row.active ? '服役中' : '已停用' }}</span>
                </button>
              </el-tooltip>
              <div
                v-else
                class="status-toggle-pill is-readonly"
                :class="row.active ? 'is-active' : 'is-inactive'"
              >
                <span class="status-indicator-dot"></span>
                <span class="status-label-text">{{ row.active ? '服役中' : '已停用' }}</span>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="目前司機" min-width="160">
            <template #default="{ row }">
              <div v-if="row.drivers && row.drivers.length" class="vehicle-driver-tags">
                <el-tag
                  v-for="d in row.drivers"
                  :key="d.id"
                  size="small"
                  type="info"
                  effect="plain"
                >
                  {{ d.name }}
                </el-tag>
              </div>
              <span v-else class="vehicle-empty-text">尚未指派</span>
            </template>
          </el-table-column>

          <el-table-column prop="createdAt" label="建立時間" min-width="170" align="center" class-name="vehicle-nowrap-col">
            <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
          </el-table-column>

          <el-table-column
            v-if="authStore.can('staff')"
            label="操作"
            width="200"
            fixed="right"
            align="center"
          >
            <template #default="{ row }">
              <TableRowActions>
                <el-button link type="primary" size="small" @click="openEditDialog(row as any)">
                  編輯
                </el-button>
                <el-button link type="primary" size="small" @click="openDriverDialog(row as any)">
                  司機
                </el-button>
                <el-button link type="danger" size="small" @click="handleDeleteVehicle(row as any)">
                  刪除
                </el-button>
              </TableRowActions>
            </template>
          </el-table-column>
        </el-table>
      </template>
    </DataTablePage>

    <!-- 新增／編輯車輛對話框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '編輯車輛' : '新增車輛'"
      width="min(620px, calc(100vw - 32px))"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="150px">
        <VehicleFormFields :form="form" :sites="allSites" show-status />
      </el-form>
      <template #footer>
        <DialogFooter :loading="submitting" @confirm="handleSubmit" @cancel="dialogVisible = false" />
      </template>
    </el-dialog>

    <!-- 車輛司機維護對話框 -->
    <el-dialog
      v-model="driverDialogVisible"
      :title="`維護司機 - ${driverDialogVehicle?.displayName || ''}`"
      width="min(480px, calc(100vw - 32px))"
    >
      <el-form label-width="110px">
        <el-form-item label="本車司機">
          <el-select
            v-model="driverDialogForm.driverIds"
            multiple
            filterable
            placeholder="可選擇多位司機"
            style="width: 100%"
          >
            <el-option
              v-for="d in allDrivers"
              :key="d.id"
              :label="d.code ? `${d.name} (${d.code})` : d.name"
              :value="d.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="生效日期">
          <el-date-picker
            v-model="driverDialogForm.effectiveFrom"
            type="date"
            value-format="YYYY-MM-DD"
            style="width: 100%"
          />
        </el-form-item>
      </el-form>
      <div class="driver-dialog-hint">
        一位司機同一期間只會有一台車：被加入本車的司機，其他車上尚未結束的指派會從生效日起收掉。
      </div>
      <template #footer>
        <DialogFooter :loading="savingDrivers" @confirm="handleSaveDrivers" @cancel="driverDialogVisible = false" />
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
import VehicleFormFields from '@/components/VehicleFormFields.vue'
import {
  listVehicles,
  createVehicle,
  updateVehicle,
  deleteVehicle,
  listDrivers,
  listSites,
  setVehicleDrivers
} from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import { formatDateTime, formatRocDate, formatYearMonth } from '@/utils/formatters'
import { emptyVehicleForm, vehicleFormRules } from '@/utils/vehicleForm'
import type { VehicleDTO, CreateVehicleRequest, DriverDTO, SiteDTO } from '@/types/api'

const authStore = useAuthStore()
const vehicles = ref<VehicleDTO[]>([])
const allDrivers = ref<DriverDTO[]>([])
const allSites = ref<SiteDTO[]>([])
const dialogVisible = ref(false)
const editingId = ref<string | null>(null)
const driverDialogVisible = ref(false)
const driverDialogVehicle = ref<VehicleDTO | null>(null)
const savingDrivers = ref(false)
const driverDialogForm = reactive<{ driverIds: string[]; effectiveFrom: string }>({
  driverIds: [],
  effectiveFrom: new Date().toISOString().split('T')[0]
})
const submitting = ref(false)
const formRef = ref<FormInstance>()

const form = reactive<CreateVehicleRequest>(emptyVehicleForm())

const rules = vehicleFormRules

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
    siteId: ''
  },
  onFetch: async () => {
    const res = await listVehicles({
      page: page.value,
      pageSize: pageSize.value,
      q: filters.q,
      siteId: filters.siteId
    })
    vehicles.value = res.data
    total.value = res.meta.total
  }
})

// 表格「編號」是跨頁連續的清冊序號，不是每頁重新從 1 起算
function rowIndex(index: number) {
  return (page.value - 1) * pageSize.value + index + 1
}

async function loadDrivers() {
  try {
    const res = await listDrivers({ active: true, pageSize: 200 })
    allDrivers.value = res.data
  } catch {
    allDrivers.value = []
  }
}

async function loadSites() {
  try {
    const res = await listSites({ pageSize: 200 })
    allSites.value = res.data
  } catch {
    allSites.value = []
  }
}

onMounted(() => {
  loadDrivers()
  loadSites()
})

function openDriverDialog(row: VehicleDTO) {
  driverDialogVehicle.value = row
  driverDialogForm.driverIds = (row.drivers || []).map((d) => d.id)
  driverDialogForm.effectiveFrom = new Date().toISOString().split('T')[0]
  driverDialogVisible.value = true
}

async function handleSaveDrivers() {
  if (!driverDialogVehicle.value) return
  savingDrivers.value = true
  try {
    await setVehicleDrivers(driverDialogVehicle.value.id, {
      driverIds: driverDialogForm.driverIds,
      effectiveFrom: driverDialogForm.effectiveFrom
    })
    ElMessage.success(`車輛「${driverDialogVehicle.value.displayName}」司機已更新`)
    driverDialogVisible.value = false
    await Promise.all([executeFetch(), loadDrivers()])
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '更新車輛司機失敗'))
  } finally {
    savingDrivers.value = false
  }
}

function handleRowDblClick(row: VehicleDTO) {
  if (authStore.can('staff')) {
    openEditDialog(row)
  }
}

function openCreateDialog() {
  editingId.value = null
  Object.assign(form, emptyVehicleForm())
  formRef.value?.clearValidate()
  dialogVisible.value = true
}

function openEditDialog(row: VehicleDTO) {
  editingId.value = row.id
  Object.assign(form, emptyVehicleForm(), {
    plateNo: row.plateNo,
    displayName: row.displayName,
    siteId: row.siteId || '',
    ownerName: row.ownerName,
    brand: row.brand,
    model: row.model,
    manufactureYm: row.manufactureYm,
    compulsoryInsuranceExpiry: row.compulsoryInsuranceExpiry || '',
    passengerInsuranceExpiry: row.passengerInsuranceExpiry || '',
    thirdPartyInsuranceExpiry: row.thirdPartyInsuranceExpiry || '',
    lastInspectionDate: row.lastInspectionDate || '',
    wheelchairAccessible: row.wheelchairAccessible ?? true,
    active: row.active
  })
  formRef.value?.clearValidate()
  dialogVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (editingId.value) {
        await updateVehicle(editingId.value, { ...form })
        ElMessage.success(`車輛「${form.displayName}」資料已更新`)
      } else {
        await createVehicle({ ...form })
        ElMessage.success('車輛新增成功')
      }
      dialogVisible.value = false
      executeFetch()
    } catch (err: any) {
      ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '儲存車輛資料失敗'))
    } finally {
      submitting.value = false
    }
  })
}

// 快速切換狀態仍送出完整車輛內容：更新 API 是整筆覆寫，只送 active 會清掉其餘欄位
async function handleQuickToggleActive(row: VehicleDTO, newActive: boolean) {
  try {
    await updateVehicle(row.id, {
      plateNo: row.plateNo,
      displayName: row.displayName,
      siteId: row.siteId || '',
      ownerName: row.ownerName,
      brand: row.brand,
      model: row.model,
      manufactureYm: row.manufactureYm,
      compulsoryInsuranceExpiry: row.compulsoryInsuranceExpiry || '',
      passengerInsuranceExpiry: row.passengerInsuranceExpiry || '',
      thirdPartyInsuranceExpiry: row.thirdPartyInsuranceExpiry || '',
      lastInspectionDate: row.lastInspectionDate || '',
      wheelchairAccessible: row.wheelchairAccessible ?? true,
      active: newActive
    })
    row.active = newActive
    ElMessage.success(`已將車輛「${row.displayName}」狀態切換為 ${newActive ? '服役中' : '已停用'}`)
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '切換狀態失敗'))
  }
}

async function handleDeleteVehicle(row: VehicleDTO) {
  try {
    await ElMessageBox.confirm(
      `確定要刪除車輛「${row.displayName} (${row.plateNo})」？`,
      '刪除確認',
      {
        confirmButtonText: '刪除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )
    await deleteVehicle(row.id)
    ElMessage.success(`車輛「${row.displayName}」已成功刪除`)
    executeFetch()
  } catch (err: any) {
    if (err !== 'cancel') {
      ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '刪除車輛失敗'))
    }
  }
}

executeFetch()
</script>

<style scoped>
.vehicle-driver-tags {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.vehicle-empty-text {
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

:deep(.vehicle-nowrap-col .cell) {
  white-space: nowrap;
}

.driver-dialog-hint {
  margin-top: -4px;
  padding-left: 120px;
  color: var(--el-text-color-secondary);
  font-size: var(--app-font-xs);
  line-height: 1.6;
}
</style>
