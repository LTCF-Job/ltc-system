<template>
  <div class="driver-report-status-view">
    <DataTablePage
      title="接送匯報總覽"
      :description="`共 ${forms.length} 台車，顯示各車已有對應資料的月份`"
      :max-width="700"
      :loading="loading"
    >
      <template #filter>
        <el-input
          v-model="searchQuery"
          placeholder="搜尋車輛"
          clearable
          style="width: 240px"
          @keyup.enter="fetchForms"
        />
        <el-button type="primary" @click="fetchForms">查詢</el-button>
        <el-button @click="handleReset">重設</el-button>
      </template>

      <template #table>
      <el-table
        :data="forms"
        border
        stripe
        row-key="id"
        table-layout="auto"
        style="width: 100%"
        :expand-row-keys="expandedIds"
        @expand-change="onExpandChange"
      >
        <el-table-column type="expand">
          <template #default="{ row }">
            <div class="months-detail">
              <template v-if="monthsByForm.get(row.id)?.length">
                <div v-for="m in monthsByForm.get(row.id)" :key="m.yearMonth" class="month-item">
                  <span class="month-label">{{ m.yearMonth }}</span>
                  <span class="month-count">{{ m.submissionCount }} 天</span>
                  <span class="text-secondary">最後匯入 {{ formatDateTime(m.lastImportedAt, '—') }}</span>
                </div>
              </template>
              <p v-else class="text-secondary">這台車尚未有任何月份的匯入紀錄。</p>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="車輛" min-width="140" class-name="vehicle-col">
          <template #default="{ row }">
            {{ row.vehicleName }}
          </template>
        </el-table-column>

        <el-table-column label="已有資料月份" min-width="260" class-name="months-col">
          <template #default="{ row }">
            <div v-if="monthsByForm.get(row.id)?.length" class="month-tags">
              <el-tag v-for="m in monthsByForm.get(row.id)" :key="m.yearMonth" size="small">
                {{ m.yearMonth }}（{{ m.submissionCount }}天）
              </el-tag>
            </div>
            <span v-else class="text-muted">尚未匯入</span>
          </template>
        </el-table-column>

        <el-table-column label="最後匯入時間" width="170" align="center">
          <template #default="{ row }">
            {{ formatDateTime(row.lastImportedAt, '尚未匯入') }}
          </template>
        </el-table-column>

        <template #empty>
          <div class="empty-state">
            <p>尚未建立任何車輛的接送匯報表。</p>
            <p class="text-secondary">請到「批次上傳」頁面上傳司機填寫的 .xlsx 匯報檔，系統會自動建立對應車輛的匯報表。</p>
          </div>
        </template>
      </el-table>
      </template>
    </DataTablePage>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import DataTablePage from '@/components/DataTablePage.vue'
import { listDriverReportForms, listDriverReportImportedMonths } from '@/api/driverReports'
import { formatDateTime } from '@/utils/formatters'
import type { DriverReportFormDTO, DriverReportImportedMonthDTO } from '@/types/api'

const forms = ref<DriverReportFormDTO[]>([])
const importedMonths = ref<DriverReportImportedMonthDTO[]>([])
const searchQuery = ref('')
const loading = ref(false)
const expandedIds = ref<string[]>([])

// 依 formId 分組並依月份新到舊排序，供展開列與月份標籤欄共用
const monthsByForm = computed(() => {
  const grouped = new Map<string, DriverReportImportedMonthDTO[]>()
  for (const m of importedMonths.value) {
    const list = grouped.get(m.formId) ?? []
    list.push(m)
    grouped.set(m.formId, list)
  }
  for (const list of grouped.values()) list.sort((a, b) => b.yearMonth.localeCompare(a.yearMonth))
  return grouped
})

async function fetchForms() {
  loading.value = true
  try {
    const [formList, months] = await Promise.all([
      listDriverReportForms({ q: searchQuery.value || undefined }),
      listDriverReportImportedMonths()
    ])
    forms.value = formList
    importedMonths.value = months
  } finally {
    loading.value = false
  }
}

function handleReset() {
  searchQuery.value = ''
  fetchForms()
}

function onExpandChange(_row: unknown, expanded: unknown) {
  if (!Array.isArray(expanded)) return
  expandedIds.value = (expanded as DriverReportFormDTO[]).map((f) => f.id)
}

onMounted(fetchForms)
</script>

<style scoped>
.driver-report-status-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.text-secondary {
  color: var(--app-text-secondary);
  font-size: 13px;
}

:deep(.vehicle-col .cell) {
  white-space: nowrap;
  min-width: 140px;
}

:deep(.months-col .cell) {
  min-width: 260px;
}

.month-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.months-detail {
  padding: 8px 24px 12px;
}

.month-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 0;
  font-size: 13px;
}

.month-label {
  font-weight: 500;
  min-width: 70px;
}

.month-count {
  color: var(--app-status-success-fg);
}

.empty-state {
  padding: 24px 0;
  line-height: 1.8;
}
</style>
