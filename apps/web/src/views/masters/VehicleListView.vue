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
              <span v-else class="font-mono">{{ row.plateNo }}</span>
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
              <el-tag v-else size="small" :type="row.region === 'miaoli' ? 'warning' : 'primary'">
                {{ REGION_LABELS[row.region as Region] || row.region }}
              </el-tag>
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

          <el-table-column prop="createdAt" label="建立時間" min-width="170" align="center">
            <template #default="{ row }">
              <span>{{ formatDateTime(row.createdAt) }}</span>
            </template>
          </el-table-column>

          <el-table-column
            v-if="authStore.can('staff')"
            label="操作"
            width="150"
            fixed="right"
            align="center"
          >
            <template #default="{ row }">
              <template v-if="editingRowId === row.id">
                <el-button type="primary" size="small" :loading="savingRow" @click="saveInlineEdit(row as any)">
                  儲存
                </el-button>
                <el-button size="small" :disabled="savingRow" @click="cancelInlineEdit">
                  取消
                </el-button>
              </template>
              <template v-else>
                <el-button link type="primary" size="small" @click="startInlineEdit(row as any)">
                  編輯
                </el-button>
                <el-button link type="danger" size="small" @click="handleDeleteVehicle(row as any)">
                  刪除
                </el-button>
              </template>
            </template>
          </el-table-column>
        </el-table>
      </template>
    </DataTablePage>

    <!-- 新增車輛彈窗 -->
    <el-dialog
      v-model="dialogVisible"
      title="新增車輛"
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
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import DataTablePage from '@/components/DataTablePage.vue'
import { listVehicles, createVehicle, updateVehicle, deleteVehicle } from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import { formatDateTime } from '@/utils/formatters'
import { REGION_LABELS, type Region } from '@/types/domain'
import type { VehicleDTO, CreateVehicleRequest } from '@/types/api'

const authStore = useAuthStore()
const vehicles = ref<VehicleDTO[]>([])
const dialogVisible = ref(false)
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
    ElMessage.error(err.message || '更新車輛資料失敗')
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
    ElMessage.error(err.message || '切換狀態失敗')
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
        confirmButtonText: '確定刪除',
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
      ElMessage.error(err.message || '刪除車輛失敗')
    }
  }
}

executeFetch()
</script>

<style scoped>
:deep(.el-table .editing-row) {
  background-color: var(--el-color-primary-light-9) !important;
}

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
  background-color: #ecfdf5;
  color: #047857;
  border-color: #a7f3d0;
}

.status-toggle-pill.is-active:hover {
  background-color: #d1fae5;
  border-color: #6ee7b7;
  color: #065f46;
  transform: translateY(-1px);
  box-shadow: 0 3px 8px rgba(5, 150, 105, 0.18);
}

.status-toggle-pill.is-inactive {
  background-color: #f3f4f6;
  color: #4b5563;
  border-color: #e5e7eb;
}

.status-toggle-pill.is-inactive:hover {
  background-color: #e5e7eb;
  border-color: #d1d5db;
  color: #1f2937;
  transform: translateY(-1px);
  box-shadow: 0 3px 8px rgba(0, 0, 0, 0.08);
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
</style>

