<template>
  <div class="field-mapping-view">
    <el-card shadow="never" class="filter-card">
      <el-row :gutter="16" justify="space-between" align="middle">
        <el-col :span="18" class="filter-inputs">
          <el-select
            v-model="selectedFormId"
            placeholder="選擇表單"
            style="width: 260px"
            @change="fetchColumns"
          >
            <el-option
              v-for="f in forms"
              :key="f.id"
              :label="`${f.title} (${f.pendingColumns} 待對應)`"
              :value="f.id"
            />
          </el-select>

          <el-radio-group v-model="statusFilter" @change="fetchColumns">
            <el-radio-button value="pending">待對應欄位</el-radio-button>
            <el-radio-button value="mapped">已對應</el-radio-button>
            <el-radio-button value="ignored">已略過</el-radio-button>
            <el-radio-button value="">全部欄位</el-radio-button>
          </el-radio-group>
        </el-col>

        <el-col :span="6" class="actions-col">
          <el-button
            v-if="authStore.can('staff') && statusFilter === 'pending'"
            type="success"
            :disabled="pendingHighConfidenceCount === 0"
            @click="handleBatchConfirmHighConfidence"
          >
            一鍵套用高信心度推薦 ({{ pendingHighConfidenceCount }})
          </el-button>
        </el-col>
      </el-row>
    </el-card>

    <!-- 欄位對應雙欄對照表 -->
    <el-card shadow="never" class="table-card">
      <el-table :data="columns" border stripe v-loading="loading">
        <el-table-column prop="columnSeq" label="#" width="60" align="center" />
        
        <!-- 左側：原始欄位與推薦 -->
        <el-table-column label="原始表單欄位名稱 (左側)" min-width="260">
          <template #default="{ row }">
            <div class="raw-column-box">
              <span class="raw-name">{{ row.columnName }}</span>
              <div class="tags-row">
                <el-tag size="small" :type="getKindTagType(row.kind)">
                  {{ COLUMN_KIND_LABELS[row.kind as ColumnKind] }}
                </el-tag>
                <el-tag
                  v-if="row.suggestionScore !== undefined"
                  size="small"
                  :type="row.suggestionScore >= 0.8 ? 'success' : 'warning'"
                >
                  系統推薦：{{ row.suggestedCaseName || '無相符個案' }}
                  (信心度: {{ (row.suggestionScore * 100).toFixed(0) }}%)
                </el-tag>
              </div>
            </div>
          </template>
        </el-table-column>

        <!-- 右側：目標綁定個案與時段 -->
        <el-table-column label="目標個案與排班時段 (右側)" min-width="320">
          <template #default="{ row }">
            <div class="target-binding-box">
              <el-select
                v-model="row.editCaseId"
                placeholder="搜尋綁定個案"
                filterable
                clearable
                style="width: 180px"
                :disabled="!authStore.can('staff')"
              >
                <el-option
                  v-for="c in cases"
                  :key="c.id"
                  :label="`${c.name} (${c.code})`"
                  :value="c.id"
                />
              </el-select>

              <el-select
                v-model="row.editLegSeq"
                placeholder="選擇時段"
                style="width: 130px"
                :disabled="!authStore.can('staff')"
              >
                <el-option :value="1" label="第 1 趟 (去程)" />
                <el-option :value="2" label="第 2 趟 (回程)" />
                <el-option :value="3" label="第 3 趟 (去程)" />
                <el-option :value="4" label="第 4 趟 (回程)" />
              </el-select>
            </div>
          </template>
        </el-table-column>

        <!-- 對應狀態 -->
        <el-table-column label="狀態" width="100" align="center">
          <template #default="{ row }">
            <el-tag
              size="small"
              :type="row.mappingStatus === 'mapped' ? 'success' : (row.mappingStatus === 'pending' ? 'warning' : 'info')"
            >
              {{ MAPPING_STATUS_LABELS[row.mappingStatus as MappingStatus] }}
            </el-tag>
          </template>
        </el-table-column>

        <!-- 操作 -->
        <el-table-column
          v-if="authStore.can('staff')"
          label="操作"
          width="180"
          fixed="right"
          align="center"
        >
          <template #default="{ row }">
            <el-button
              type="success"
              link
              size="small"
              :icon="Check"
              :disabled="!row.editCaseId"
              @click="handleSaveMapping(row)"
            >
              確認綁定
            </el-button>
            <el-button
              v-if="row.mappingStatus !== 'ignored'"
              type="info"
              link
              size="small"
              :icon="Close"
              @click="handleIgnoreMapping(row)"
            >
              略過此欄
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Check, Close } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import {
  listForms,
  listFormColumns,
  updateColumnMapping,
  batchUpdateColumnMappings
} from '@/api/forms'
import { listCases } from '@/api/cases'
import { useAuthStore } from '@/stores/auth'
import {
  COLUMN_KIND_LABELS,
  MAPPING_STATUS_LABELS,
  type ColumnKind,
  type MappingStatus
} from '@/types/domain'
import type { FormDTO, FormColumnDTO, CaseDTO } from '@/types/api'

const route = useRoute()
const authStore = useAuthStore()

const forms = ref<FormDTO[]>([])
const cases = ref<CaseDTO[]>([])
const selectedFormId = ref<string>('')
const statusFilter = ref<string>('pending')
const columns = ref<(FormColumnDTO & { editCaseId?: string; editLegSeq?: number })[]>([])
const loading = ref(false)

const pendingHighConfidenceCount = computed(() => {
  return columns.value.filter(
    (c) => c.mappingStatus === 'pending' && (c.suggestionScore || 0) >= 0.8 && c.suggestedCaseId
  ).length
})

function getKindTagType(kind: string) {
  switch (kind) {
    case 'ride':
      return 'success'
    case 'issue':
      return 'warning'
    case 'meta':
      return 'info'
    default:
      return 'danger'
  }
}

async function fetchColumns() {
  if (!selectedFormId.value) return
  loading.value = true
  try {
    const res = await listFormColumns({
      formId: selectedFormId.value,
      mappingStatus: statusFilter.value || undefined
    })
    columns.value = res.map((c) => ({
      ...c,
      editCaseId: c.mappedCaseId || c.suggestedCaseId || '',
      editLegSeq: c.mappedLegSeq || c.suggestedLegSeq || 1
    }))
  } finally {
    loading.value = false
  }
}

async function handleSaveMapping(row: any) {
  await updateColumnMapping(row.id, {
    caseId: row.editCaseId,
    legSeq: row.editLegSeq,
    mappingStatus: 'mapped'
  })
  ElMessage.success(`已將「${row.columnName}」成功綁定`)
  fetchColumns()
}

async function handleIgnoreMapping(row: any) {
  await updateColumnMapping(row.id, {
    mappingStatus: 'ignored'
  })
  ElMessage.info(`已略過「${row.columnName}」`)
  fetchColumns()
}

async function handleBatchConfirmHighConfidence() {
  const highConfidenceList = columns.value.filter(
    (c) => c.mappingStatus === 'pending' && (c.suggestionScore || 0) >= 0.8 && c.suggestedCaseId
  )

  await batchUpdateColumnMappings({
    mappings: highConfidenceList.map((c) => ({
      columnId: c.id,
      caseId: c.suggestedCaseId,
      legSeq: c.suggestedLegSeq || 1,
      mappingStatus: 'mapped'
    }))
  })

  ElMessage.success(`已批次確認 ${highConfidenceList.length} 個高信心度推薦欄位`)
  fetchColumns()
}

onMounted(async () => {
  const [fRes, cRes] = await Promise.all([
    listForms(),
    listCases({ pageSize: 100 })
  ])
  forms.value = fRes
  cases.value = cRes.data

  const queryFormId = route.query.formId as string
  if (queryFormId && forms.value.some((f) => f.id === queryFormId)) {
    selectedFormId.value = queryFormId
  } else if (forms.value.length > 0) {
    selectedFormId.value = forms.value[0].id
  }

  fetchColumns()
})
</script>

<style scoped>
.field-mapping-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.filter-card {
  border-radius: 8px;
}

.filter-inputs {
  display: flex;
  align-items: center;
  gap: 16px;
}

.actions-col {
  display: flex;
  justify-content: flex-end;
}

.table-card {
  border-radius: 8px;
}

.raw-column-box {
  display: flex;
  flex-direction: column;
  gap: 6px;

  .raw-name {
    font-weight: 500;
    font-size: 14px;
  }

  .tags-row {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
  }
}

.target-binding-box {
  display: flex;
  gap: 8px;
  align-items: center;
}
</style>
