<template>
  <div class="region-list-view">
    <DataTablePage
      title="地區管理"
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
          placeholder="搜尋區域名稱／說明"
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
          <el-option label="啟用中" value="active" />
          <el-option label="已停用" value="inactive" />
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
          新增地區
        </el-button>
      </template>

      <!-- 表格 -->
      <template #table>
        <el-table
          :data="regions"
          border
          stripe
          style="width: 100%"
          :row-class-name="tableRowClassName"
          v-loading="loading || isSavingSort"
        >
          <el-table-column
            label="排序"
            width="85"
            align="center"
          >
            <template #header>
              <el-tooltip content="按住圖示上下拖曳可即時調整排序" placement="top">
                <span class="sort-col-header">
                  排序
                  <el-icon class="help-icon"><InfoFilled /></el-icon>
                </span>
              </el-tooltip>
            </template>
            <template #default="{ row, $index }">
              <div
                class="drag-handle-pill"
                :class="{
                  'is-draggable': !isSearching,
                  'is-disabled': isSearching,
                  'is-dragging': draggingIndex === $index
                }"
                :draggable="!isSearching"
                :title="isSearching ? '搜尋篩選狀態下暫停拖曳排序' : '按住上下拖曳調整順序'"
                @dragstart="onDragStart($event, $index)"
                @dragover.prevent="onDragOver($event, $index)"
                @dragenter.prevent="onDragEnter($event, $index)"
                @dragleave="onDragLeave($event, $index)"
                @drop="onDrop($event, $index)"
                @dragend="onDragEnd"
              >
                <el-icon class="drag-grip-icon"><Rank /></el-icon>
                <span class="sort-order-badge">{{ row.sortOrder }}</span>
              </div>
            </template>
          </el-table-column>

          <el-table-column prop="name" label="地區名稱" min-width="150">
            <template #default="{ row }">
              <span class="font-bold text-gray-800">{{ row.name }}</span>
            </template>
          </el-table-column>

          <el-table-column prop="description" label="說明與備註" min-width="240" show-overflow-tooltip>
            <template #default="{ row }">
              <span>{{ row.description || '-' }}</span>
            </template>
          </el-table-column>

          <el-table-column prop="status" label="狀態" width="130" align="center">
            <template #default="{ row }">
              <el-tooltip
                v-if="authStore.can('staff')"
                :content="row.status === 'active' ? '目前為啟用中，點選切換為停用' : '目前為已停用，點選切換為啟用'"
                placement="top"
                :show-after="300"
              >
                <button
                  type="button"
                  class="status-toggle-pill"
                  :class="row.status === 'active' ? 'is-active' : 'is-inactive'"
                  @click="handleToggleStatus(row as any, row.status !== 'active')"
                >
                  <span class="status-indicator-dot"></span>
                  <span class="status-label-text">{{ row.status === 'active' ? '啟用中' : '已停用' }}</span>
                </button>
              </el-tooltip>
              <div
                v-else
                class="status-toggle-pill is-readonly"
                :class="row.status === 'active' ? 'is-active' : 'is-inactive'"
              >
                <span class="status-indicator-dot"></span>
                <span class="status-label-text">{{ row.status === 'active' ? '啟用中' : '已停用' }}</span>
              </div>
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
            width="140"
            fixed="right"
            align="center"
          >
            <template #default="{ row }">
              <TableRowActions>
                <el-button link type="primary" size="small" @click="openEditDialog(row as any)">
                  編輯
                </el-button>
                <el-button
                  v-if="authStore.can('admin')"
                  link
                  type="danger"
                  size="small"
                  @click="handleDelete(row as any)"
                >
                  刪除
                </el-button>
              </TableRowActions>
            </template>
          </el-table-column>
        </el-table>
      </template>
    </DataTablePage>

    <!-- 新增/編輯彈窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '編輯地區設定' : '新增營運地區'"
      width="min(600px, calc(100vw - 32px))"
      destroy-on-close
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
        <el-form-item label="地區名稱" prop="name">
          <el-input v-model="form.name" placeholder="如：臺北市、臺中市" />
        </el-form-item>

        <el-form-item label="排序權重" prop="sortOrder">
          <el-input-number v-model="form.sortOrder" :min="1" :max="999" style="width: 160px" />
          <span class="form-tip-inline">數字愈小排在愈前面</span>
        </el-form-item>

        <el-form-item label="狀態" prop="status">
          <el-radio-group v-model="form.status" class="status-radio-group">
            <el-radio-button value="active">
              <div class="radio-pill active-pill">
                <span class="radio-dot"></span>
                <span>啟用中</span>
              </div>
            </el-radio-button>
            <el-radio-button value="inactive">
              <div class="radio-pill inactive-pill">
                <span class="radio-dot"></span>
                <span>已停用</span>
              </div>
            </el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="備註說明" prop="description">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="3"
            placeholder="請輸入地區備註或涵蓋範圍說明"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <DialogFooter :loading="submitting" @confirm="handleSubmit" @cancel="dialogVisible = false" />
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Plus, Rank, InfoFilled, Edit, Delete } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { resolveErrorMessage } from '@/api/errorCodes'
import DataTablePage from '@/components/DataTablePage.vue'
import DialogFooter from '@/components/DialogFooter.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import { listRegions, createRegion, updateRegion, deleteRegion } from '@/api/masters'
import { formatDateTime } from '@/utils/formatters'
import { useAuthStore } from '@/stores/auth'
import type { RegionDTO, CreateRegionRequest, UpdateRegionRequest } from '@/types/api'

const authStore = useAuthStore()

const page = ref(1)
const pageSize = ref(50)
const total = ref(0)
const loading = ref(false)
const submitting = ref(false)

const regions = ref<RegionDTO[]>([])

// 拖曳排序狀態
const draggingIndex = ref<number | null>(null)
const dropTargetIndex = ref<number | null>(null)
const isSavingSort = ref(false)

const filters = reactive({
  q: '',
  status: ''
})

const isSearching = computed(() => Boolean(filters.q.trim() || filters.status))

const dialogVisible = ref(false)
const editingId = ref<string | null>(null)
const formRef = ref<FormInstance>()

const form = reactive<{
  name: string
  description: string
  status: 'active' | 'inactive'
  sortOrder: number
}>({
  name: '',
  description: '',
  status: 'active',
  sortOrder: 1
})

function tableRowClassName({ rowIndex }: { rowIndex: number }) {
  const classes: string[] = []
  if (draggingIndex.value === rowIndex) {
    classes.push('row-is-dragging')
  }
  if (
    dropTargetIndex.value === rowIndex &&
    draggingIndex.value !== null &&
    draggingIndex.value !== rowIndex
  ) {
    classes.push(draggingIndex.value > rowIndex ? 'row-drop-target-above' : 'row-drop-target-below')
  }
  return classes.join(' ')
}

function onDragStart(event: DragEvent, index: number) {
  if (isSearching.value) return
  draggingIndex.value = index
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', String(index))
  }
}

function onDragOver(event: DragEvent, index: number) {
  if (isSearching.value || draggingIndex.value === null) return
  event.preventDefault()
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'move'
  }
  if (dropTargetIndex.value !== index) {
    dropTargetIndex.value = index
  }
}

function onDragEnter(event: DragEvent, index: number) {
  if (isSearching.value || draggingIndex.value === null) return
  event.preventDefault()
  dropTargetIndex.value = index
}

function onDragLeave(_event: DragEvent, _index: number) {
  // 由 onDragOver 動態更新目標行
}

function onDragEnd() {
  draggingIndex.value = null
  dropTargetIndex.value = null
}

async function onDrop(event: DragEvent, targetIndex: number) {
  event.preventDefault()
  if (isSearching.value || draggingIndex.value === null || draggingIndex.value === targetIndex) {
    onDragEnd()
    return
  }

  const srcIdx = draggingIndex.value
  const dstIdx = targetIndex
  onDragEnd()

  // 本地陣列移動
  const movedItem = regions.value.splice(srcIdx, 1)[0]
  regions.value.splice(dstIdx, 0, movedItem)

  // 重新指派連續 sortOrder
  const changedUpdates: { id: string; sortOrder: number }[] = []
  regions.value.forEach((reg, idx) => {
    const newSort = idx + 1
    if (reg.sortOrder !== newSort) {
      reg.sortOrder = newSort
      changedUpdates.push({ id: reg.id, sortOrder: newSort })
    }
  })

  if (changedUpdates.length > 0) {
    isSavingSort.value = true
    try {
      await Promise.all(
        changedUpdates.map((u) => updateRegion(u.id, { sortOrder: u.sortOrder }))
      )
      ElMessage.success(`已將「${movedItem.name}」排序更新`)
    } catch (err: any) {
      ElMessage.error('更新排序順序失敗，正在重新整理清單')
      fetchRegions()
    } finally {
      isSavingSort.value = false
    }
  }
}

const rules: FormRules = {
  name: [{ required: true, message: '請輸入區域名稱', trigger: 'blur' }],
  sortOrder: [{ required: true, message: '請設定排序權重', trigger: 'change' }]
}

async function fetchRegions() {
  loading.value = true
  try {
    const res = await listRegions({
      page: page.value,
      pageSize: pageSize.value,
      q: filters.q || undefined,
      status: filters.status || undefined
    })
    regions.value = res.data || []
    total.value = res.meta?.total ?? regions.value.length
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '查詢區域清單失敗'))
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  fetchRegions()
}

function handleReset() {
  filters.q = ''
  filters.status = ''
  page.value = 1
  fetchRegions()
}

function handlePageChange(p: number) {
  page.value = p
  fetchRegions()
}

function handleSizeChange(s: number) {
  pageSize.value = s
  page.value = 1
  fetchRegions()
}

function openCreateDialog() {
  editingId.value = null
  form.name = ''
  form.description = ''
  form.status = 'active'
  form.sortOrder = (regions.value.length > 0 ? Math.max(...regions.value.map(r => r.sortOrder || 0)) + 1 : 1)
  dialogVisible.value = true
}

function openEditDialog(row: RegionDTO) {
  editingId.value = row.id
  form.name = row.name
  form.description = row.description || ''
  form.status = row.status
  form.sortOrder = row.sortOrder ?? 1
  dialogVisible.value = true
}

async function handleToggleStatus(row: RegionDTO, newActive: boolean) {
  const newStatus = newActive ? 'active' : 'inactive'
  try {
    await updateRegion(row.id, { status: newStatus })
    row.status = newStatus
    ElMessage.success(`已將「${row.name}」切換為 ${newActive ? '啟用中' : '已停用'}`)
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '更新狀態失敗'))
  }
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (editingId.value) {
        const updateData: UpdateRegionRequest = {
          name: form.name.trim(),
          description: form.description.trim(),
          status: form.status,
          sortOrder: form.sortOrder
        }
        await updateRegion(editingId.value, updateData)
        ElMessage.success('區域資料更新成功')
      } else {
        const createData: CreateRegionRequest = {
          name: form.name.trim(),
          description: form.description.trim(),
          status: form.status,
          sortOrder: form.sortOrder
        }
        await createRegion(createData)
        ElMessage.success('區域新增成功')
      }
      dialogVisible.value = false
      fetchRegions()
    } catch (err: any) {
      ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '儲存失敗'))
    } finally {
      submitting.value = false
    }
  })
}

async function handleDelete(row: RegionDTO) {
  try {
    await ElMessageBox.confirm(
      `確定要刪除區域「${row.name}」嗎？若該區域已有綁定個案、車輛或據點，建議優先改為「停用」。`,
      '刪除確認',
      {
        confirmButtonText: '刪除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )
    await deleteRegion(row.id)
    ElMessage.success(`已成功刪除區域「${row.name}」`)
    fetchRegions()
  } catch (action) {
    // 使用者取消操作
  }
}

onMounted(() => {
  fetchRegions()
})
</script>

<style scoped>
.region-list-view {
  padding: 0;
}
.font-mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}
.font-bold {
  font-weight: 600;
}
.form-tip {
  font-size: 12px;
  color: var(--app-text-muted);
  margin-top: 4px;
  line-height: 1.4;
}
.form-tip-inline {
  font-size: 12px;
  color: var(--app-text-muted);
  margin-left: 10px;
}


/* 拖曳排序握把與狀態 */
.drag-hint-tag {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  padding: 0 10px;
  height: 32px;
  border-radius: 6px;
}

.sort-col-header {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  cursor: help;
}

.sort-col-header .help-icon {
  font-size: 13px;
  color: var(--app-text-muted);
}

.drag-handle-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 4px 8px;
  border-radius: 6px;
  background-color: var(--app-card-bg);
  border: 1px solid var(--app-border-color);
  transition: all 0.2s ease;
  user-select: none;
}

.drag-handle-pill.is-draggable {
  cursor: grab;
}

.drag-handle-pill.is-draggable:hover {
  background-color: var(--app-status-success-bg);
  border-color: var(--app-status-success-fg);
  color: var(--app-status-success-fg);
  box-shadow: 0 2px 5px rgba(5, 150, 105, 0.15);
}

.drag-handle-pill.is-draggable:active {
  cursor: grabbing;
}

.drag-handle-pill.is-disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.drag-grip-icon {
  font-size: 14px;
  color: var(--app-text-muted);
  transition: color 0.2s;
}

.drag-handle-pill:hover .drag-grip-icon {
  color: var(--app-status-success-fg);
}

.sort-order-badge {
  font-size: 12px;
  font-weight: 600;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  color: var(--app-text-primary);
  min-width: 16px;
}

/* 拖曳時整列的高亮與插入指示線 */
:deep(.el-table__row.row-is-dragging) {
  opacity: 0.45;
  background-color: var(--app-status-neutral-bg) !important;
}

:deep(.el-table__row.row-drop-target-above td) {
  border-top: 3px solid var(--app-status-success-fg) !important;
  background-color: var(--app-status-success-bg) !important;
  transition: background-color 0.15s ease;
}

:deep(.el-table__row.row-drop-target-below td) {
  border-bottom: 3px solid var(--app-status-success-fg) !important;
  background-color: var(--app-status-success-bg) !important;
  transition: background-color 0.15s ease;
}
</style>

