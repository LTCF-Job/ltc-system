<template>
  <div class="vehicle-list-view">
    <DataTablePage
      title="車輛管理"
      :max-width="1250"
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
          filterable
          style="width: 140px"
          @change="handleSearch"
        >
          <el-option label="全部區域" value="" />
          <el-option
            v-for="(label, key) in REGION_LABELS"
            :key="key"
            :label="label"
            :value="key"
          />
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
          style="width: 100%"
          :row-class-name="getRowClassName"
          @row-dblclick="(row: any) => handleRowDblClick(row)"
        >
          <el-table-column prop="displayName" label="顯示車名 (表單比對名)" min-width="180">
            <template #default="{ row }">
              <el-input
                v-if="editingRowId === row.id"
                v-model="editRowForm.displayName"
                size="small"
                placeholder="如：竹北一車"
                @keyup.enter="saveInlineEdit(row as any)"
                @keyup.esc="cancelInlineEdit"
              />
              <span v-else>{{ row.displayName }}</span>
            </template>
          </el-table-column>

          <el-table-column prop="plateNo" label="車牌號碼" width="160">
            <template #default="{ row }">
              <el-input
                v-if="editingRowId === row.id"
                v-model="editRowForm.plateNo"
                size="small"
                placeholder="如：BZG-7915"
                @keyup.enter="saveInlineEdit(row as any)"
                @keyup.esc="cancelInlineEdit"
              />
              <span v-else class="font-mono text-id">{{ row.plateNo }}</span>
            </template>
          </el-table-column>

          <el-table-column prop="region" label="所屬區域" width="130" align="center">
            <template #default="{ row }">
              <el-select
                v-if="editingRowId === row.id"
                v-model="editRowForm.region"
                size="small"
                filterable
                style="width: 100%"
              >
                <el-option
                  v-for="(label, key) in REGION_LABELS"
                  :key="key"
                  :label="label"
                  :value="key"
                />
              </el-select>
              <span v-else>
                {{ REGION_LABELS[row.region as Region] || row.region }}
              </span>
            </template>
          </el-table-column>

          <el-table-column prop="active" label="狀態" width="130" align="center">
            <template #default="{ row }">
              <template v-if="editingRowId === row.id">
                <button
                  type="button"
                  class="status-toggle-pill"
                  :class="editRowForm.active ? 'is-active' : 'is-inactive'"
                  @click="editRowForm.active = !editRowForm.active"
                >
                  <span class="status-indicator-dot"></span>
                  <span class="status-label-text">{{ editRowForm.active ? '服役中' : '已停用' }}</span>
                </button>
              </template>
              <template v-else>
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
            </template>
          </el-table-column>

          <el-table-column label="目前司機" min-width="200">
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
              <span v-else class="vehicle-driver-empty">尚未指派</span>
            </template>
          </el-table-column>

          <el-table-column prop="createdAt" label="建立時間" min-width="170" align="center" show-overflow-tooltip>
            <template #default="{ row }">
              <span>{{ formatDateTime(row.createdAt) }}</span>
            </template>
          </el-table-column>

          <el-table-column
            v-if="authStore.can('staff')"
            label="操作"
            width="220"
            fixed="right"
            align="center"
          >
            <template #default="{ row }">
              <TableRowActions v-if="editingRowId === row.id">
                <el-button type="primary" size="small" :loading="savingRow" @click="saveInlineEdit(row as any)">
                  儲存
                </el-button>
                <el-button size="small" :disabled="savingRow" @click="cancelInlineEdit">
                  取消
                </el-button>
              </TableRowActions>
              <TableRowActions v-else>
                <el-button link type="primary" size="small" @click="startInlineEdit(row as any)">
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

    <!-- 新增車輛彈窗 -->
    <el-dialog
      v-model="dialogVisible"
      title="新增車輛"
      width="min(480px, calc(100vw - 32px))"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
        <el-form-item label="顯示車名" prop="displayName">
          <el-input v-model="form.displayName" placeholder="如：竹北一車、竹南2車" />
        </el-form-item>
        <el-form-item label="車牌號碼" prop="plateNo">
          <el-input v-model="form.plateNo" placeholder="如：BZG-7915" />
        </el-form-item>
        <el-form-item label="所屬區域" prop="region">
          <el-select
            v-model="form.region"
            placeholder="請選擇區域"
            filterable
            style="width: 100%"
          >
            <el-option
              v-for="(label, key) in REGION_LABELS"
              :key="key"
              :label="label"
              :value="key"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="狀態" prop="active">
          <el-radio-group v-model="form.active" class="status-radio-group">
            <el-radio-button :value="true">
              <div class="radio-pill active-pill">
                <span class="radio-dot"></span>
                <span>服役中</span>
              </div>
            </el-radio-button>
            <el-radio-button :value="false">
              <div class="radio-pill inactive-pill">
                <span class="radio-dot"></span>
                <span>已停用</span>
              </div>
            </el-radio-button>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <DialogFooter :loading="submitting" @confirm="handleSubmit" @cancel="dialogVisible = false" />
      </template>
    </el-dialog>

    <!-- 車輛司機維護彈窗 -->
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
import { listVehicles, createVehicle, updateVehicle, deleteVehicle, listDrivers, setVehicleDrivers } from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import { formatDateTime } from '@/utils/formatters'
import { REGION_LABELS, type Region } from '@/types/domain'
import type { VehicleDTO, CreateVehicleRequest, DriverDTO } from '@/types/api'

const authStore = useAuthStore()
const vehicles = ref<VehicleDTO[]>([])
const allDrivers = ref<DriverDTO[]>([])
const dialogVisible = ref(false)
const driverDialogVisible = ref(false)
const driverDialogVehicle = ref<VehicleDTO | null>(null)
const savingDrivers = ref(false)
const driverDialogForm = reactive<{ driverIds: string[]; effectiveFrom: string }>({
  driverIds: [],
  effectiveFrom: new Date().toISOString().split('T')[0]
})
const submitting = ref(false)
const formRef = ref<FormInstance>()

// 行內編輯狀態
const editingRowId = ref<string | null>(null)
const savingRow = ref(false)
const editRowForm = reactive<CreateVehicleRequest>({
  displayName: '',
  plateNo: '',
  region: 'miaoli',
  active: true
})

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

async function loadDrivers() {
  try {
    const res = await listDrivers({ active: true, pageSize: 200 })
    allDrivers.value = res.data
  } catch {
    allDrivers.value = []
  }
}

onMounted(loadDrivers)

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

function getRowClassName({ row }: { row: any }) {
  return editingRowId.value === row.id ? 'editing-row' : ''
}

function handleRowDblClick(row: VehicleDTO) {
  if (authStore.can('staff')) {
    startInlineEdit(row)
  }
}

function startInlineEdit(row: VehicleDTO) {
  editingRowId.value = row.id
  editRowForm.displayName = row.displayName
  editRowForm.plateNo = row.plateNo
  editRowForm.region = row.region
  editRowForm.active = row.active
}

function cancelInlineEdit() {
  editingRowId.value = null
}

async function saveInlineEdit(row: VehicleDTO) {
  if (!editRowForm.displayName.trim()) {
    ElMessage.warning('請輸入顯示車名')
    return
  }
  if (!editRowForm.plateNo.trim()) {
    ElMessage.warning('請輸入車牌號碼')
    return
  }
  if (!/^[A-Z0-9]{2,4}-[A-Z0-9]{2,4}$/.test(editRowForm.plateNo.trim())) {
    ElMessage.warning('車牌格式錯誤 (例如 BZG-7915)')
    return
  }

  savingRow.value = true
  try {
    await updateVehicle(row.id, {
      displayName: editRowForm.displayName.trim(),
      plateNo: editRowForm.plateNo.trim(),
      region: editRowForm.region,
      active: editRowForm.active
    })
    row.displayName = editRowForm.displayName.trim()
    row.plateNo = editRowForm.plateNo.trim()
    row.region = editRowForm.region
    row.active = Boolean(editRowForm.active)
    editingRowId.value = null
    ElMessage.success(`車輛「${row.displayName}」資料已更新`)
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '更新車輛資料失敗'))
  } finally {
    savingRow.value = false
  }
}

async function handleQuickToggleActive(row: VehicleDTO, newActive: boolean) {
  try {
    await updateVehicle(row.id, { active: newActive })
    row.active = newActive
    ElMessage.success(`已將車輛「${row.displayName}」狀態切換為 ${newActive ? '服役中' : '已停用'}`)
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '切換狀態失敗'))
  }
}

function openCreateDialog() {
  form.displayName = ''
  form.plateNo = ''
  form.region = 'miaoli'
  form.active = true
  dialogVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      await createVehicle(form)
      ElMessage.success('車輛新增成功')
      dialogVisible.value = false
      executeFetch()
    } finally {
      submitting.value = false
    }
  })
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

.vehicle-driver-empty {
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

.driver-dialog-hint {
  margin-top: -4px;
  padding-left: 120px;
  color: var(--el-text-color-secondary);
  font-size: var(--app-font-xs);
  line-height: 1.6;
}

:deep(.el-table .editing-row) {
  background-color: var(--el-color-primary-light-9) !important;
}


/* 彈窗內狀態單選群組 */
.region-label {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}

.region-label::before {
  content: '';
  width: 7px;
  height: 7px;
  border: 2px solid var(--el-border-color);
  border-radius: 50%;
}

.region-label.region-miaoli::before { border-color: var(--el-color-warning); }
.region-label.region-hsinchu::before { border-color: var(--el-color-primary); }
</style>

