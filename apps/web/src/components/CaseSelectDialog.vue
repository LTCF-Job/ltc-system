<template>
  <el-dialog
    :model-value="modelValue"
    :title="title"
    width="min(820px, calc(100vw - 32px))"
    @update:model-value="emit('update:modelValue', $event)"
    @open="loadCandidates"
  >
    <div class="case-select-toolbar">
      <el-input
        v-model="keyword"
        placeholder="搜尋姓名／編號"
        clearable
        style="width: 220px"
      />
      <div class="case-select-actions">
        <el-button size="small" @click="handleSelectAll">全選</el-button>
        <el-button size="small" @click="handleClearSelection">取消全選</el-button>
        <span class="case-select-count">已選擇 {{ selectedRows.length }} / {{ candidates.length }} 筆</span>
      </div>
    </div>

    <el-table
      ref="tableRef"
      v-loading="loading"
      :data="filteredCandidates"
      border
      stripe
      height="360px"
      row-key="id"
      @selection-change="handleSelectionChange"
    >
      <el-table-column type="selection" width="45" :reserve-selection="true" />
      <el-table-column prop="code" label="個案編號" width="95" align="center" />
      <el-table-column prop="name" label="姓名" width="110" />
      <el-table-column prop="region" label="區域" width="100" align="center">
        <template #default="{ row }">{{ REGION_LABELS[row.region as Region] || row.region }}</template>
      </el-table-column>
      <el-table-column prop="status" label="狀態" width="90" align="center">
        <template #default="{ row }">{{ CASE_STATUS_LABELS[row.status as CaseStatus] || row.status }}</template>
      </el-table-column>
      <el-table-column prop="homeAddress" label="住家地址" min-width="180" show-overflow-tooltip />
    </el-table>

    <template #footer>
      <DialogFooter
        :confirm-text="`${confirmText}（${selectedRows.length} 筆）`"
        :loading="confirmLoading"
        :confirm-disabled="selectedRows.length === 0"
        @confirm="handleConfirm"
        @cancel="emit('update:modelValue', false)"
      />
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import type { TableInstance } from 'element-plus'
import DialogFooter from '@/components/DialogFooter.vue'
import { listCases } from '@/api/cases'
import { resolveErrorMessage } from '@/api/errorCodes'
import { REGION_LABELS, CASE_STATUS_LABELS } from '@/types/domain'
import type { Region, CaseStatus } from '@/types/domain'
import type { CaseDTO } from '@/types/api'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title?: string
    confirmText?: string
    confirmLoading?: boolean
    region?: string
    initialSelectedIds?: string[]
  }>(),
  {
    title: '選擇個案',
    confirmText: '確認',
    confirmLoading: false,
    region: '',
    initialSelectedIds: () => []
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'confirm', cases: CaseDTO[]): void
}>()

const tableRef = ref<TableInstance>()
const loading = ref(false)
const keyword = ref('')
const candidates = ref<CaseDTO[]>([])
const selectedRows = ref<CaseDTO[]>([])

const filteredCandidates = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  if (!text) return candidates.value
  return candidates.value.filter(
    (c) => c.name.toLowerCase().includes(text) || c.code.toLowerCase().includes(text)
  )
})

async function loadCandidates() {
  keyword.value = ''
  selectedRows.value = []
  tableRef.value?.clearSelection()
  loading.value = true
  try {
    const res = await listCases({ pageSize: 1000, region: props.region || undefined })
    candidates.value = res.data
    await nextTick()
    restoreInitialSelection()
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '載入個案清單失敗'))
  } finally {
    loading.value = false
  }
}

// 重新開啟對話框時沿用上次的勾選，避免使用者為了改一筆而重勾整份清單
function restoreInitialSelection() {
  if (props.initialSelectedIds.length === 0) return
  const wanted = new Set(props.initialSelectedIds)
  candidates.value
    .filter((row) => wanted.has(row.id))
    .forEach((row) => tableRef.value?.toggleRowSelection(row, true))
}

function handleSelectionChange(rows: CaseDTO[]) {
  selectedRows.value = rows
}

function handleSelectAll() {
  filteredCandidates.value.forEach((row) => tableRef.value?.toggleRowSelection(row, true))
}

function handleClearSelection() {
  tableRef.value?.clearSelection()
}

function handleConfirm() {
  if (selectedRows.value.length === 0) return
  emit('confirm', [...selectedRows.value])
}
</script>

<style scoped>
.case-select-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--app-space-2);
  margin-bottom: var(--app-space-2);
  flex-wrap: wrap;
}

.case-select-actions {
  display: flex;
  align-items: center;
  gap: var(--app-space-2);
}

.case-select-count {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  white-space: nowrap;
}
</style>
