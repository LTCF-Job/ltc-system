<template>
  <div class="form-list-view">
    <el-card shadow="never" class="table-card">
      <template #header>
        <div class="card-header">
          <div style="display: flex; align-items: center; gap: 16px;">
            <span class="title">Google 表單同步管理 (共 {{ forms.length }} 份表單)</span>
            <el-input
              v-model="searchQuery"
              placeholder="搜尋表單名稱／表單 ID"
              clearable
              style="width: 240px"
              @keyup.enter="fetchForms"
            />
            <el-button type="primary" icon="Search" @click="fetchForms">查詢</el-button>
            <el-button @click="handleReset">重設</el-button>
          </div>
          <el-button
            v-if="authStore.can('staff')"
            type="primary"
            :loading="syncingAll"
            @click="syncAllForms"
          >
            <el-icon><Refresh /></el-icon>
            全部手動同步
          </el-button>
        </div>
      </template>

      <el-alert
        v-if="hasOutdatedForms"
        type="error"
        show-icon
        :closable="false"
        title="系統偵測到部分表單已逾 48 小時未同步，請檢查 Webhook 推送或執行手動同步。"
        style="margin-bottom: 16px"
      />

      <el-table :data="forms" border stripe v-loading="loading">
        <el-table-column prop="title" label="表單名稱" min-width="180" />
        <el-table-column prop="formId" label="表單 ID" width="180" />
        <el-table-column label="最後同步時間" min-width="180" align="center">
          <template #default="{ row }">
            <span :class="{ 'text-danger font-bold': row.hasSyncAlert }">
              {{ formatDateTime(row.lastSyncedAt, '從未同步') }}
            </span>
            <el-tag v-if="row.hasSyncAlert" type="danger" size="small" style="margin-left: 6px">
              逾 48h
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="totalColumns" label="總欄位數" width="100" align="center" />
        <el-table-column label="待對應欄位" width="120" align="center">
          <template #default="{ row }">
            <el-tag :type="row.pendingColumns > 0 ? 'warning' : 'success'">
              {{ row.pendingColumns }} 欄待對應
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="200" fixed="right" align="center">
          <template #default="{ row }">
            <el-button
              link
              type="primary"
              size="small"
              @click="$router.push(`/forms/mappings?formId=${row.id}`)"
            >
              設定欄位對應
            </el-button>
            <el-button
              v-if="authStore.can('staff')"
              link
              type="success"
              size="small"
              :loading="syncingId === row.id"
              @click="handleSyncForm(row)"
            >
              立即同步
            </el-button>
          </template>
        </el-table-column>


      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { listForms, syncForm } from '@/api/forms'
import { formatDateTime } from '@/utils/formatters'
import { useAuthStore } from '@/stores/auth'
import type { FormDTO } from '@/types/api'

const authStore = useAuthStore()
const forms = ref<FormDTO[]>([])
const loading = ref(false)
const searchQuery = ref('')
const syncingId = ref<string | null>(null)
const syncingAll = ref(false)

const hasOutdatedForms = computed(() => {
  return forms.value.some((f) => f.hasSyncAlert)
})

async function fetchForms() {
  loading.value = true
  try {
    forms.value = await listForms({
      q: searchQuery.value || undefined
    })
  } finally {
    loading.value = false
  }
}

function handleReset() {
  searchQuery.value = ''
  fetchForms()
}

async function handleSyncForm(form: any) {
  syncingId.value = form.id
  try {
    const res = await syncForm(form.id)
    ElMessage.success(`表單同步完成：新增 ${res.syncedRows} 筆紀錄、${res.newColumns} 個新欄位`)
    fetchForms()
  } finally {
    syncingId.value = null
  }
}

async function syncAllForms() {
  syncingAll.value = true
  try {
    for (const f of forms.value) {
      await syncForm(f.id)
    }
    ElMessage.success('全部表單已同步完成')
    fetchForms()
  } finally {
    syncingAll.value = false
  }
}

onMounted(() => {
  fetchForms()
})
</script>

<style scoped>
.form-list-view {
  display: flex;
  flex-direction: column;
}

.table-card {
  border-radius: 8px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;

  .title {
    font-size: 16px;
    font-weight: bold;
    color: var(--el-color-primary);
  }
}

.text-danger {
  color: var(--el-color-danger);
}

.font-bold {
  font-weight: bold;
}
</style>
