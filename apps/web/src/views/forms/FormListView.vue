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
              :loading="isAuthorizing"
              @click="openGoogleConnectDialog"
            >
              使用 Google 登入並關聯表單
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
            <span v-if="row.region" class="ml-1 text-secondary">
              ({{ (REGION_LABELS as any)[row.region] || row.region }})
            </span>
          </template>
        </el-table-column>

        <el-table-column label="試算表分頁" min-width="240">
          <template #default="{ row }">
            <div class="sheet-tabs-wrap">
              <el-tooltip
                v-for="tab in (row.sheetTabs || ['回覆一'])"
                :key="tab"
                :content="`分頁：${tab}（${getSheetTabStatus(row, tab).statusText}）`"
                placement="top"
              >
                <span
                  class="sheet-tab-badge"
                  :class="{
                    'status-synced': getSheetTabStatus(row, tab).isSynced,
                    'status-active': getSheetTabStatus(row, tab).isActive,
                    'status-unimported': !getSheetTabStatus(row, tab).isSynced && !getSheetTabStatus(row, tab).isActive
                  }"
                >
                  <span class="tab-name">{{ tab }}</span>
                  <span class="tab-status-tag">
                    {{ getSheetTabStatus(row, tab).isSynced ? '已匯入' : '未匯入' }}
                  </span>
                </span>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="同步月份" min-width="150">
          <template #default="{ row }">
            <div v-if="row.syncedMonths && row.syncedMonths.length > 0" class="month-tags-wrap">
              <span
                v-for="m in row.syncedMonths"
                :key="m"
                class="month-label"
              >
                {{ m }}
              </span>
            </div>
            <span v-else class="text-muted">（尚未標記）</span>
          </template>
        </el-table-column>

        <el-table-column label="最後同步時間" width="190" align="center">
          <template #default="{ row }">
            <span :class="{ 'text-danger font-bold': row.hasSyncAlert }">
              {{ formatDateTime(row.lastSyncedAt, '從未同步') }}
            </span>
            <span v-if="row.hasSyncAlert" class="sync-alert-label">
              逾 48h
            </span>
          </template>
        </el-table-column>

        <el-table-column label="待對應欄位" width="130" align="center">
          <template #default="{ row }">
            <span class="mapping-status" :class="row.pendingColumns > 0 ? 'is-pending' : 'is-ready'">
              {{ row.pendingColumns }} 欄待對應
            </span>
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

    <!-- Google 帳號登入與表單選擇關聯彈窗 -->
    <el-dialog
      v-model="googleConnectDialogVisible"
      title="Google 帳號登入與表單關聯"
      width="640px"
      destroy-on-close
    >
      <!-- 步驟 1：尚未授權 Google 帳號 -->
      <div v-if="!currentAccessToken && step === 1" class="google-auth-box">
        <div class="auth-icon-wrap">
          <svg width="48" height="48" viewBox="0 0 24 24">
            <path
              fill="#4285F4"
              d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
            />
            <path
              fill="#34A853"
              d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
            />
            <path
              fill="#FBBC05"
              d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.06H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.94l2.85-2.22.81-.63z"
            />
            <path
              fill="#EA4335"
              d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.06l3.66 2.84c.87-2.6 3.3-4.52 6.16-4.52z"
            />
          </svg>
        </div>
        <div class="auth-desc">
          <h3>連結您的 Google 帳號</h3>
          <p>登入後系統將讀取您 Google 雲端硬碟中可存取的接送回報表單，無須手動複製貼上網址。</p>
        </div>
        <el-button
          type="primary"
          size="large"
          :loading="isAuthorizing"
          @click="startGoogleLogin"
        >
          登入 Google 帳號並選取表單
        </el-button>

        <div style="margin-top: 24px; width: 100%">
          <el-divider>或手動輸入試算表連結</el-divider>
          <el-input
            v-model="manualSheetUrl"
            placeholder="https://docs.google.com/spreadsheets/d/..."
            clearable
          >
            <template #append>
              <el-button :loading="inspectingManual" @click="handleManualInspect">讀取分頁</el-button>
            </template>
          </el-input>
        </div>
      </div>

      <!-- 步驟 2：已登入，呈現 Google 雲端硬碟中的試算表清單 -->
      <div v-else-if="step === 1" class="drive-files-list-step">
        <div class="list-header">
          <span class="user-status-tag">
            <el-tag type="success" size="small">Google 帳號已連線</el-tag>
          </span>
          <el-input
            v-model="fileSearchQuery"
            placeholder="搜尋表單名稱..."
            prefix-icon="Search"
            clearable
            style="width: 260px"
          />
          <el-button icon="Refresh" :loading="loadingFiles" @click="fetchDriveFilesWithToken">
            重新整理
          </el-button>
        </div>

        <div v-loading="loadingFiles" class="files-container">
          <div v-if="filteredFiles.length === 0 && !loadingFiles" class="empty-box">
            <el-empty description="未在此 Google 帳號中找到試算表" />
          </div>

          <div
            v-for="file in filteredFiles"
            :key="file.id"
            class="sheet-file-item"
            @click="selectDriveFile(file)"
          >
            <div class="sheet-icon">
              <el-icon :size="24" color="#0F9D58"><Document /></el-icon>
            </div>
            <div class="sheet-details">
              <div class="sheet-name">{{ file.name }}</div>
              <div class="sheet-meta">
                <span>ID: {{ file.id }}</span>
                <span v-if="file.modifiedTime" style="margin-left: 10px">
                  更新於 {{ file.modifiedTime }}
                </span>
              </div>
            </div>
            <el-button
              type="primary"
              size="small"
              :loading="inspectingFileId === file.id"
              @click.stop="selectDriveFile(file)"
            >
              選取此表單
            </el-button>
          </div>
        </div>
      </div>

      <!-- 步驟 3：確認分頁、關聯車輛並儲存至資料庫 -->
      <div v-else-if="step === 2" class="confirm-association-step">
        <el-descriptions :column="1" border size="small" style="margin-bottom: 16px">
          <el-descriptions-item label="目標表單">
            <span class="font-bold text-primary">{{ associateForm.title }}</span>
          </el-descriptions-item>
          <el-descriptions-item label="自動匯入之分頁">
            <div class="sheet-tabs-wrap">
              <el-tag
                v-for="tab in availableTabs"
                :key="tab"
                size="small"
                type="success"
                effect="plain"
                style="margin-right: 6px; margin-bottom: 4px;"
              >
                {{ tab }}
              </el-tag>
            </div>
          </el-descriptions-item>
        </el-descriptions>

        <el-form
          ref="associateFormRef"
          :model="associateForm"
          :rules="associateRules"
          label-width="120px"
          label-position="right"
        >
          <el-form-item label="表單名稱" prop="title">
            <el-input v-model="associateForm.title" placeholder="如：竹北一車每日接送回報表" />
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
            <el-input v-model="associateForm.vehicleName" placeholder="如：竹北一車" />
          </el-form-item>

          <el-form-item label="預設同步分頁" prop="activeTab">
            <el-select
              v-model="associateForm.activeTab"
              placeholder="請選擇預設分頁"
              style="width: 100%"
            >
              <el-option
                v-for="t in availableTabs"
                :key="t"
                :label="t"
                :value="t"
              />
            </el-select>
          </el-form-item>

          <el-form-item v-if="previewHeaders.length > 0" label="偵測到之欄位">
            <div class="headers-preview-wrap">
              <el-tag
                v-for="(hdr, idx) in previewHeaders"
                :key="idx"
                size="small"
                type="info"
                style="margin-right: 4px; margin-bottom: 4px;"
              >
                {{ hdr }}
              </el-tag>
            </div>
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <div v-if="step === 1" style="display: flex; justify-content: flex-end; width: 100%">
          <el-button @click="googleConnectDialogVisible = false">取消</el-button>
        </div>
        <div v-else style="display: flex; justify-content: space-between; width: 100%">
          <el-button @click="step = 1">返回選擇其他表單</el-button>
          <div>
            <el-button @click="googleConnectDialogVisible = false">取消</el-button>
            <el-button type="primary" :loading="associating" @click="handleCreateAssociation">
              確認匯入並儲存至資料庫
            </el-button>
          </div>
        </div>
      </template>
    </el-dialog>

    <!-- 表單同步與月份時間篩選彈窗 -->
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
import { Refresh, Link, Plus, Edit, Delete, Document } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { resolveErrorMessage } from '@/api/errorCodes'
import {
  listForms,
  syncForm,
  createFormAssociation,
  deleteFormAssociation,
  inspectGoogleSheet,
  listGoogleDriveSheets
} from '@/api/forms'
import { formatDateTime } from '@/utils/formatters'
import { useAuthStore } from '@/stores/auth'
import { REGION_LABELS, type Region } from '@/types/domain'
import type { FormDTO, GoogleDriveSheetDTO } from '@/types/api'
import { useGooglePicker } from '@/composables/useGooglePicker'

const authStore = useAuthStore()
const forms = ref<FormDTO[]>([])
const loading = ref(false)
const searchQuery = ref('')
const syncingId = ref<string | null>(null)
const syncingAll = ref(false)

const hasOutdatedForms = computed(() => {
  return forms.value.some((f) => f.hasSyncAlert)
})

// Google 登入與關聯對話框狀態
const googleConnectDialogVisible = ref(false)
const step = ref(1)
const currentAccessToken = ref('')
const driveFiles = ref<GoogleDriveSheetDTO[]>([])
const loadingFiles = ref(false)
const fileSearchQuery = ref('')
const inspectingFileId = ref('')
const manualSheetUrl = ref('')
const inspectingManual = ref(false)

const { openPicker, isAuthorizing, accessToken: pickerAccessToken } = useGooglePicker()

const filteredFiles = computed(() => {
  if (!fileSearchQuery.value) return driveFiles.value
  const q = fileSearchQuery.value.toLowerCase().trim()
  return driveFiles.value.filter((f) => f.name.toLowerCase().includes(q) || f.id.toLowerCase().includes(q))
})

// 關聯表單確認狀態
const associating = ref(false)
const availableTabs = ref<string[]>([])
const previewHeaders = ref<string[]>([])
const associateFormRef = ref<FormInstance>()
const associateForm = reactive({
  title: '',
  sheetUrl: '',
  region: 'hsinchu' as Region,
  vehicleName: '',
  activeTab: ''
})

const associateRules = {
  title: [{ required: true, message: '請輸入表單名稱', trigger: 'blur' }]
}

function openGoogleConnectDialog() {
  step.value = 1
  fileSearchQuery.value = ''
  manualSheetUrl.value = ''
  googleConnectDialogVisible.value = true
  // 若已登入過或有後端連線，自動載入檔案清單
  fetchDriveFilesWithToken()
}

// 點擊「登入 Google 帳號並選取表單」
async function startGoogleLogin() {
  const clientId = (import.meta.env.VITE_GOOGLE_CLIENT_ID as string) || '108848783424-demo.apps.googleusercontent.com'
  const apiKey = (import.meta.env.VITE_GOOGLE_API_KEY as string) || 'AIzaSyDemo'

  try {
    const picked = await openPicker({
      clientId,
      apiKey,
      scopes: [
        'https://www.googleapis.com/auth/drive.readonly',
        'https://www.googleapis.com/auth/spreadsheets.readonly'
      ]
    })

    if (picked) {
      currentAccessToken.value = picked.accessToken
      await handlePickedFile(picked)
    }
  } catch (err: any) {
    console.warn('[Google OAuth Notice]', err)
    // 降級為呼叫後端 Drive API 讀取
    await fetchDriveFilesWithToken()
  }
}

async function fetchDriveFilesWithToken() {
  loadingFiles.value = true
  try {
    driveFiles.value = await listGoogleDriveSheets()
  } catch {
    driveFiles.value = []
  } finally {
    loadingFiles.value = false
  }
}

async function selectDriveFile(file: GoogleDriveSheetDTO) {
  inspectingFileId.value = file.id
  try {
    const sheetUrl = `https://docs.google.com/spreadsheets/d/${file.id}/edit`
    const res = await inspectGoogleSheet({
      spreadsheetId: file.id,
      accessToken: currentAccessToken.value || undefined
    })

    associateForm.title = file.name
    associateForm.sheetUrl = sheetUrl
    associateForm.vehicleName = file.name.replace(/\(回覆\)|\(Responses\)/gi, '').trim()
    associateForm.region = 'hsinchu'

    if (res.sheetTabs && res.sheetTabs.length > 0) {
      availableTabs.value = res.sheetTabs
      associateForm.activeTab = res.sheetTabs[0]
    } else {
      availableTabs.value = ['工作表1']
      associateForm.activeTab = '工作表1'
    }

    if (res.previewHeaders) {
      previewHeaders.value = res.previewHeaders
    }

    step.value = 2
    ElMessage.success(`已載入【${file.name}】之 ${availableTabs.value.length} 個工作表分頁`)
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '讀取試算表分頁失敗'))
  } finally {
    inspectingFileId.value = ''
  }
}

async function handlePickedFile(picked: any) {
  associateForm.title = picked.name
  associateForm.sheetUrl = picked.url
  associateForm.vehicleName = picked.name.replace(/\(回覆\)|\(Responses\)/gi, '').trim()
  associateForm.region = 'hsinchu'

  try {
    const res = await inspectGoogleSheet({
      spreadsheetId: picked.id,
      accessToken: picked.accessToken
    })

    if (res.sheetTabs && res.sheetTabs.length > 0) {
      availableTabs.value = res.sheetTabs
      associateForm.activeTab = res.sheetTabs[0]
    } else {
      availableTabs.value = ['工作表1']
      associateForm.activeTab = '工作表1'
    }

    if (res.previewHeaders) {
      previewHeaders.value = res.previewHeaders
    }

    step.value = 2
    ElMessage.success(`已自動載入【${picked.name}】之 ${availableTabs.value.length} 個工作表分頁`)
  } catch (err: any) {
    ElMessage.warning('無法自動讀取分頁，請手動確認')
    step.value = 2
  }
}

async function handleManualInspect() {
  if (!manualSheetUrl.value) {
    ElMessage.warning('請輸入 Google 試算表連結')
    return
  }
  inspectingManual.value = true
  try {
    const res = await inspectGoogleSheet({
      sheetUrl: manualSheetUrl.value,
      accessToken: currentAccessToken.value || undefined
    })

    associateForm.title = res.title || '新關聯試算表'
    associateForm.sheetUrl = manualSheetUrl.value
    associateForm.vehicleName = ''
    associateForm.region = 'hsinchu'

    if (res.sheetTabs && res.sheetTabs.length > 0) {
      availableTabs.value = res.sheetTabs
      associateForm.activeTab = res.sheetTabs[0]
    } else {
      availableTabs.value = ['工作表1']
      associateForm.activeTab = '工作表1'
    }

    if (res.previewHeaders) {
      previewHeaders.value = res.previewHeaders
    }

    step.value = 2
    ElMessage.success(`已成功解析 ${availableTabs.value.length} 個分頁`)
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '解析試算表失敗，請確認連結與權限'))
  } finally {
    inspectingManual.value = false
  }
}

async function handleCreateAssociation() {
  if (!associateFormRef.value) return
  await associateFormRef.value.validate(async (valid) => {
    if (!valid) return
    associating.value = true
    try {
      await createFormAssociation({
        title: associateForm.title,
        sheetUrl: associateForm.sheetUrl,
        region: associateForm.region,
        vehicleName: associateForm.vehicleName,
        sheetTabs: availableTabs.value,
        activeTab: associateForm.activeTab || availableTabs.value[0],
        accessToken: currentAccessToken.value || undefined
      })

      ElMessage.success(`已成功建立與 Google 表單「${associateForm.title}」之關聯並儲存至資料庫`)
      googleConnectDialogVisible.value = false
      fetchForms()
    } finally {
      associating.value = false
    }
  })
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

function getSheetTabStatus(row: any, tab: string): { isSynced: boolean; isActive: boolean; statusText: string } {
  const isActive = row.activeTab === tab
  const months: string[] = row.syncedMonths || []

  let isSynced = false
  const match = tab.match(/(\d+)月/)
  if (match) {
    const monthNum = match[1].padStart(2, '0')
    isSynced = months.some((m) => m.endsWith(`-${monthNum}`) || m.includes(monthNum))
  } else if (tab.includes('去程') || tab.includes('回程') || tab.includes('回覆')) {
    isSynced = months.length > 0
  }

  if (isActive && row.lastSyncedAt) {
    isSynced = true
  }

  let statusText = '未匯入'
  if (isSynced) {
    statusText = isActive ? '已匯入 (當前分頁)' : '已匯入'
  } else if (isActive) {
    statusText = '當前分頁'
  }

  return {
    isSynced,
    isActive,
    statusText
  }
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
      force: forceSyncConfirm.value,
      spreadsheetId: targetForm.value.sheetUrl,
      accessToken: currentAccessToken.value || pickerAccessToken.value || undefined
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
      await syncForm(f.id, {
        month: '2026-08',
        spreadsheetId: f.sheetUrl,
        accessToken: currentAccessToken.value || pickerAccessToken.value || undefined
      })
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
    color: var(--el-text-color-primary);
  }
}

.header-actions {
  display: flex;
  gap: 8px;
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

.text-primary {
  color: var(--el-color-primary);
}

.text-danger {
  color: var(--el-color-danger);
}

.text-muted {
  color: var(--el-text-color-placeholder);
}

.text-secondary {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.month-tags-wrap {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.sheet-tabs-wrap {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.sheet-tab-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
  border: 1px solid var(--el-border-color-light);
  background-color: var(--el-fill-color-light);

  .tab-name {
    font-weight: 500;
  }

  .tab-status-tag {
    font-size: 10px;
    padding: 0 3px;
    border-radius: 2px;
    border-color: #bbf7d0;
    color: #15803d;
  }

  &.status-unimported {
    background-color: #f8fafc;
    border-color: #e2e8f0;
    color: #64748b;

    .tab-status-tag {
      background-color: #f1f5f9;
    }
  }
}

.month-label {
  display: inline-flex;
  align-items: center;
  padding: 2px 6px;
  font-size: 12px;
  border-radius: 4px;
  background-color: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-light);
}

.google-auth-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 16px 20px;

  .auth-icon-wrap {
    margin-bottom: 12px;
  }

  .auth-desc {
    margin-bottom: 16px;

    h3 {
      font-weight: 600;
      color: var(--el-text-color-primary);
      margin: 0 0 6px 0;
    }

    p {
      font-size: 14px;
      color: var(--el-text-color-secondary);
      margin: 0;
    }
  }
}

.drive-files-list-step {
  display: flex;
  flex-direction: column;
  gap: 12px;

  .list-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
  }

  .files-container {
    max-height: 320px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .sheet-file-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 14px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    background-color: var(--el-bg-color);
    cursor: pointer;
    transition: all 0.2s ease;

    &:hover {
      border-color: var(--el-color-primary);
      background-color: var(--el-color-primary-light-9);
    }

    .sheet-details {
      flex: 1;
      min-width: 0;

      .sheet-name {
        font-size: 14px;
        font-weight: 600;
        color: var(--el-text-color-primary);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      }

      .sheet-meta {
        font-size: 12px;
        color: var(--el-text-color-secondary);
        margin-top: 2px;
      }
    }
  }
}

.sync-dialog-body,
.confirm-association-step {
  display: flex;
  flex-direction: column;
}

.headers-preview-wrap {
  display: flex;
  flex-wrap: wrap;
  max-height: 120px;
  overflow-y: auto;
  padding: 4px;
  background-color: var(--el-fill-color-light);
  border-radius: 4px;
  width: 100%;
}
</style>
