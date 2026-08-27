<template>
  <div class="form-list-view">
    <el-card shadow="never" class="table-card">
      <template #header>
        <div class="card-header">
          <div style="display: flex; align-items: center; gap: 16px; flex-wrap: wrap;">
            <span class="title">Google 表單同步管理 (共 {{ forms.length }} 份表單)</span>
            <el-input
              v-model="searchQuery"
              placeholder="搜尋表單名稱／表單 ID／車輛"
              clearable
              style="width: 240px"
              @keyup.enter="fetchForms"
            />
            <el-button type="primary" icon="Search" @click="fetchForms">查詢</el-button>
            <el-button @click="handleReset">重設</el-button>
          </div>
          <div class="header-actions">
            <el-button
              v-if="authStore.hasPermission('forms_sync', 'edit')"
              type="success"
              icon="Plus"
              @click="openAssociateDialog"
            >
              新增關聯表單
            </el-button>
            <el-button
              v-if="authStore.hasPermission('forms_sync', 'edit')"
              type="primary"
              :loading="syncingAll"
              @click="syncAllForms"
            >
              <el-icon><Refresh /></el-icon>
              全部批次同步
            </el-button>
          </div>
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
        <el-table-column prop="title" label="表單名稱" min-width="170">
          <template #default="{ row }">
            <div class="form-title-cell">
              <span class="font-bold text-gray-800">{{ row.title }}</span>
              <a
                v-if="row.sheetUrl"
                :href="row.sheetUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="sheet-link"
                title="開啟 Google 試算表"
              >
                <el-icon><Link /></el-icon>
              </a>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="所屬車輛／地區" width="160">
          <template #default="{ row }">
            <span>{{ row.vehicleName || row.formId }}</span>
            <el-tag v-if="row.region" size="small" type="info" style="margin-left: 4px">
              {{ (REGION_LABELS as any)[row.region] || row.region }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="試算表分頁 (Sub-sheets)" min-width="180">
          <template #default="{ row }">
            <div class="sheet-tabs-wrap">
              <el-tag
                v-for="tab in (row.sheetTabs || ['回覆一'])"
                :key="tab"
                size="small"
                :effect="row.activeTab === tab ? 'dark' : 'plain'"
                :type="row.activeTab === tab ? 'primary' : 'info'"
                style="margin-right: 4px; margin-bottom: 4px;"
              >
                {{ tab }}
              </el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="已同步月份" min-width="150">
          <template #default="{ row }">
            <div v-if="row.syncedMonths && row.syncedMonths.length > 0" class="month-tags-wrap">
              <el-tag
                v-for="m in row.syncedMonths"
                :key="m"
                size="small"
                type="success"
                effect="plain"
                style="margin-right: 4px; margin-bottom: 4px;"
              >
                {{ m }}
              </el-tag>
            </div>
            <span v-else class="text-muted">（尚未標記）</span>
          </template>
        </el-table-column>

        <el-table-column label="最後同步時間" width="160" align="center">
          <template #default="{ row }">
            <span :class="{ 'text-danger font-bold': row.hasSyncAlert }">
              {{ formatDateTime(row.lastSyncedAt, '從未同步') }}
            </span>
            <el-tag v-if="row.hasSyncAlert" type="danger" size="small" style="margin-left: 4px">
              逾 48h
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="待對應欄位" width="110" align="center">
          <template #default="{ row }">
            <el-tag :type="row.pendingColumns > 0 ? 'warning' : 'success'">
              {{ row.pendingColumns }} 欄待對應
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="240" fixed="right" align="center">
          <template #default="{ row }">
            <el-button
              link
              type="success"
              size="small"
              :icon="Edit"
              @click="$router.push(`/forms/mappings?formId=${row.id}`)"
            >
              欄位對應
            </el-button>
            <el-button
              v-if="authStore.hasPermission('forms_sync', 'edit')"
              link
              type="primary"
              size="small"
              :icon="Refresh"
              :loading="syncingId === (row as any).id"
              @click="openSyncDialog(row as any)"
            >
              同步...
            </el-button>
            <el-button
              v-if="authStore.hasPermission('forms_sync', 'edit')"
              link
              type="danger"
              size="small"
              :icon="Delete"
              @click="handleDeleteAssociation(row as any)"
            >
              解除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 新增關聯 Google 表單彈窗 -->
    <el-dialog
      v-model="associateDialogVisible"
      title="關聯新 Google 表單／試算表"
      width="540px"
      destroy-on-close
    >
      <el-form
        ref="associateFormRef"
        :model="associateForm"
        :rules="associateRules"
        label-width="120px"
        label-position="right"
      >
        <el-form-item label="表單名稱" prop="title">
          <el-input v-model="associateForm.title" placeholder="如：竹北三車每日接送回報表" />
        </el-form-item>

        <el-form-item label="Google 試算表連結" prop="sheetUrl">
          <el-input
            v-model="associateForm.sheetUrl"
            placeholder="https://docs.google.com/spreadsheets/d/..."
          />
        </el-form-item>

        <el-form-item label="所屬營運地區" prop="region">
          <el-select v-model="associateForm.region" placeholder="請選擇地區" style="width: 100%">
            <el-option
              v-for="(name, code) in REGION_LABELS"
              :key="code"
              :label="name"
              :value="code"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="關聯車輛" prop="vehicleName">
          <el-input v-model="associateForm.vehicleName" placeholder="如：竹北三車" />
        </el-form-item>

        <el-form-item label="試算表分頁" prop="sheetTabsInput">
          <el-input
            v-model="associateForm.sheetTabsInput"
            placeholder="多個分頁請以逗號分隔，如：8月回報, 7月回報, 去程, 回程"
          />
          <div class="form-tip">若試算表含多個月份或趟次工作表，請在此設定分頁名稱。</div>
        </el-form-item>

        <el-form-item label="預設同步分頁" prop="activeTab">
          <el-input v-model="associateForm.activeTab" placeholder="如：8月回報" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="associateDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="associating" @click="handleCreateAssociation">
          確認關聯
        </el-button>
      </template>
    </el-dialog>

    <!-- 表單同步與月份時間篩選彈窗 (含防重複防呆) -->
    <el-dialog
      v-model="syncDialogVisible"
      title="表單同步設定與時間篩選"
      width="480px"
      destroy-on-close
    >
      <div v-if="targetForm" class="sync-dialog-body">
        <el-descriptions :column="1" border size="small" style="margin-bottom: 16px">
          <el-descriptions-item label="目標表單">{{ targetForm.title }}</el-descriptions-item>
          <el-descriptions-item label="已同步月份">
            <span v-if="targetForm.syncedMonths && targetForm.syncedMonths.length > 0">
              {{ targetForm.syncedMonths.join('、') }}
            </span>
            <span v-else class="text-muted">（尚未有記錄）</span>
          </el-descriptions-item>
        </el-descriptions>

        <el-form label-width="110px" label-position="right">
          <el-form-item label="同步月份篩選">
            <el-date-picker
              v-model="syncMonth"
              type="month"
              value-format="YYYY-MM"
              placeholder="請選擇欲同步的服務月份"
              style="width: 100%"
            />
          </el-form-item>

          <el-form-item label="指定試算表分頁">
            <el-select v-model="syncTab" placeholder="選擇分頁" style="width: 100%">
              <el-option
                v-for="t in (targetForm.sheetTabs || ['工作表1'])"
                :key="t"
                :label="t"
                :value="t"
              />
            </el-select>
          </el-form-item>

          <!-- 已經處理過之月份警示 -->
          <el-alert
            v-if="isMonthAlreadySynced"
            type="warning"
            show-icon
            :closable="false"
            style="margin-bottom: 12px"
          >
            <template #title>
              <span class="font-bold">提醒：{{ syncMonth }} 月份已經處理/同步過！</span>
            </template>
            此月份之搭乘紀錄可能已進入核定或更正階段。重新同步可能會覆蓋已修正之紀錄，請確認是否強制執行。
          </el-alert>

          <el-form-item v-if="isMonthAlreadySynced" label="強制執行">
            <el-checkbox v-model="forceSyncConfirm">
              我已確認風險，強制重新同步此月份資料
            </el-checkbox>
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <el-button @click="syncDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="syncing"
          :disabled="isMonthAlreadySynced && !forceSyncConfirm"
          @click="executeSyncForm"
        >
          開始同步
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Refresh, Link, Plus, Edit, Delete } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { listForms, syncForm, createFormAssociation, deleteFormAssociation } from '@/api/forms'
import { formatDateTime } from '@/utils/formatters'
import { useAuthStore } from '@/stores/auth'
import { REGION_LABELS, type Region } from '@/types/domain'
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

// 關聯表單彈窗狀態
const associateDialogVisible = ref(false)
const associating = ref(false)
const associateFormRef = ref<FormInstance>()
const associateForm = reactive({
  title: '',
  sheetUrl: '',
  region: 'hsinchu' as Region,
  vehicleName: '',
  sheetTabsInput: '8月回報, 7月回報',
  activeTab: '8月回報'
})

const associateRules = {
  title: [{ required: true, message: '請輸入表單名稱', trigger: 'blur' }],
  sheetUrl: [{ required: true, message: '請輸入 Google 試算表連結', trigger: 'blur' }]
}

// 同步設定彈窗狀態
const syncDialogVisible = ref(false)
const targetForm = ref<FormDTO | null>(null)
const syncMonth = ref('2026-08')
const syncTab = ref('')
const forceSyncConfirm = ref(false)
const syncing = ref(false)

const isMonthAlreadySynced = computed(() => {
  if (!targetForm.value || !syncMonth.value) return false
  return targetForm.value.syncedMonths?.includes(syncMonth.value) || false
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

function openAssociateDialog() {
  associateForm.title = ''
  associateForm.sheetUrl = ''
  associateForm.region = 'hsinchu'
  associateForm.vehicleName = ''
  associateForm.sheetTabsInput = '8月回報, 7月回報'
  associateForm.activeTab = '8月回報'
  associateDialogVisible.value = true
}

async function handleCreateAssociation() {
  if (!associateFormRef.value) return
  await associateFormRef.value.validate(async (valid) => {
    if (!valid) return
    associating.value = true
    try {
      const tabs = associateForm.sheetTabsInput
        ? associateForm.sheetTabsInput.split(',').map((s) => s.trim()).filter(Boolean)
        : ['工作表1']

      await createFormAssociation({
        title: associateForm.title,
        sheetUrl: associateForm.sheetUrl,
        region: associateForm.region,
        sheetTabs: tabs,
        activeTab: associateForm.activeTab || tabs[0]
      })

      ElMessage.success(`已成功建立與 Google 表單「${associateForm.title}」之關聯`)
      associateDialogVisible.value = false
      fetchForms()
    } finally {
      associating.value = false
    }
  })
}

function openSyncDialog(form: FormDTO) {
  targetForm.value = form
  syncMonth.value = '2026-08'
  syncTab.value = form.activeTab || form.sheetTabs?.[0] || '工作表1'
  forceSyncConfirm.value = false
  syncDialogVisible.value = true
}

async function executeSyncForm() {
  if (!targetForm.value) return
  syncing.value = true
  try {
    const res = await syncForm(targetForm.value.id, {
      month: syncMonth.value,
      sheetTab: syncTab.value,
      force: forceSyncConfirm.value
    })
    ElMessage.success(`【${targetForm.value.title}】(${syncMonth.value} / ${syncTab.value}) 同步完成：新增 ${res.syncedRows} 筆紀錄、${res.newColumns} 個新欄位`)
    syncDialogVisible.value = false
    fetchForms()
  } finally {
    syncing.value = false
  }
}

async function syncAllForms() {
  syncingAll.value = true
  try {
    for (const f of forms.value) {
      await syncForm(f.id, { month: '2026-08' })
    }
    ElMessage.success('全部表單已批次同步完成')
    fetchForms()
  } finally {
    syncingAll.value = false
  }
}

async function handleDeleteAssociation(form: FormDTO) {
  try {
    await ElMessageBox.confirm(
      `確定要解除表單「${form.title}」的關聯嗎？解除後系統將停止該表單之定時排程同步。`,
      '確認解除表單關聯',
      {
        confirmButtonText: '確定解除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    await deleteFormAssociation(form.id)
    ElMessage.success(`已成功解除「${form.title}」之表單關聯`)
    fetchForms()
  } catch {
    // 使用者取消
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
  flex-wrap: wrap;
  gap: 12px;

  .title {
    font-size: 16px;
    font-weight: bold;
    color: var(--el-color-primary);
  }
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.form-title-cell {
  display: flex;
  align-items: center;
  gap: 6px;

  .sheet-link {
    color: var(--el-color-primary);
    font-size: 15px;
    display: flex;
    align-items: center;

    &:hover {
      color: var(--el-color-primary-dark-2);
    }
  }
}

.sheet-tabs-wrap,
.month-tags-wrap {
  display: flex;
  flex-wrap: wrap;
}

.font-bold {
  font-weight: bold;
}

.text-danger {
  color: var(--el-color-danger);
}

.text-muted {
  color: var(--el-text-color-placeholder);
}

.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

.sync-dialog-body {
  display: flex;
  flex-direction: column;
}
</style>

