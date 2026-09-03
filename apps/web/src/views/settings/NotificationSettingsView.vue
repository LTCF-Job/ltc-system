<template>
  <div class="notification-settings-view">
    <DataTablePage title="通知收件人管理" :max-width="1160" :loading="loading">
      <template #filter>
        <el-input
          v-model="searchQuery"
          placeholder="搜尋信箱／顯示名稱"
          clearable
          style="width: 240px;"
          @keyup.enter="fetchRecipients"
        />

        <el-select
          v-model="selectedTopic"
          placeholder="通知主題篩選"
          clearable
          style="width: 180px;"
          @change="fetchRecipients"
        >
          <el-option
            v-for="(label, key) in NOTIFICATION_TOPIC_LABELS"
            :key="key"
            :label="label"
            :value="key"
          />
        </el-select>

        <el-button type="primary" @click="fetchRecipients">
          查詢
        </el-button>
        <el-button @click="handleReset">
          重設
        </el-button>
      </template>

      <template #actions>
        <el-button
          v-if="authStore.hasPermission('settings_notifications', 'delete') && selectedTableRows.length > 0"
          type="danger"
          plain
          @click="handleBatchDelete"
        >
          批次刪除 ({{ selectedTableRows.length }})
        </el-button>

        <el-button
          v-if="authStore.hasPermission('settings_notifications', 'edit')"
          type="primary"
          :icon="Plus"
          @click="openAddDialog"
        >
          新增外部信箱
        </el-button>
      </template>

      <template #table>
      <el-table
        :data="filteredRecipientList"
        stripe
        border
        table-layout="auto"
        style="width: 100%;"
        @selection-change="handleTableSelectionChange"
      >
        <el-table-column
          v-if="authStore.hasPermission('settings_notifications', 'delete')"
          type="selection"
          width="48"
          align="center"
        />

        <el-table-column label="通知主題" min-width="140" class-name="topic-col">
          <template #default="{ row }">
            <span class="topic-label">{{ (NOTIFICATION_TOPIC_LABELS as any)[row.topic] || row.topic }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="email" label="通知電子信箱" min-width="240" show-overflow-tooltip class-name="email-col">
          <template #default="{ row }">
            <span class="email-text">{{ row.email }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="displayName" label="顯示名稱 / 備註" min-width="180" show-overflow-tooltip class-name="display-name-col">
          <template #default="{ row }">
            <span>{{ row.displayName || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="啟用狀態" width="110" align="center">
          <template #default="{ row }">
            <el-switch
              :model-value="row.active"
              :disabled="!authStore.hasPermission('settings_notifications', 'edit')"
              @change="(val: string | number | boolean) => handleToggleActive(row as any, Boolean(val))"
            />
          </template>
        </el-table-column>

        <el-table-column prop="createdAt" label="建立時間" width="170" align="center">
          <template #default="{ row }">
            <span>{{ formatDateTime(row.createdAt) }}</span>
          </template>
        </el-table-column>

        <el-table-column
          v-if="authStore.hasPermission('settings_notifications', 'edit') || authStore.hasPermission('settings_notifications', 'delete')"
          label="操作"
          width="150"
          fixed="right"
          align="center"
        >
          <template #default="{ row }">
            <TableRowActions>
              <el-button
                v-if="authStore.hasPermission('settings_notifications', 'edit')"
                link
                type="primary"
                size="small"
                @click="openEditDialog(row as any)"
              >
                編輯
              </el-button>
              <el-button
                v-if="authStore.hasPermission('settings_notifications', 'delete')"
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

    <!-- 新增外部信箱對話框（支援多行換行輸入） -->
    <el-dialog
      v-model="addDialogVisible"
      title="新增外部信箱"
      width="min(600px, calc(100vw - 32px))"
      destroy-on-close
      top="6vh"
    >
      <div class="add-dialog-content">
        <el-form label-position="top">
          <el-form-item label="通知主題" required>
            <el-select
              v-model="addTopic"
              placeholder="請選擇通知主題"
              style="width: 100%;"
            >
              <el-option
                v-for="(label, key) in NOTIFICATION_TOPIC_LABELS"
                :key="key"
                :label="label"
                :value="key"
              />
            </el-select>
          </el-form-item>

          <el-form-item label="輸入外部信箱（支援直接貼上多行 Email，或「姓名 <信箱>」格式）：" required>
            <el-input
              v-model="rawEmailsInput"
              type="textarea"
              :rows="7"
              placeholder="例如：&#10;supervisor@external.org&#10;王顧問 <consultant@example.com>&#10;李專員 officer@gov.tw&#10;a email@example.com"
              class="email-textarea"
            />
          </el-form-item>
        </el-form>

        <!-- 即時解析結果預覽 -->
        <div v-if="parsedEmailItems.length > 0" class="preview-panel">
          <div class="preview-header">
            <span class="preview-title">解析預覽結果</span>
            <div class="preview-counts">
              <el-tag type="success" size="small" effect="plain">
                有效 {{ validParsedEmails.length }} 筆
              </el-tag>
              <el-tag v-if="invalidCount > 0" type="danger" size="small" effect="plain">
                格式不符 {{ invalidCount }} 筆
              </el-tag>
              <el-tag v-if="duplicateCount > 0" type="warning" size="small" effect="plain">
                重複 {{ duplicateCount }} 筆
              </el-tag>
            </div>
          </div>

          <div class="parsed-list">
            <div
              v-for="(item, idx) in parsedEmailItems"
              :key="idx"
              class="parsed-row"
              :class="{
                'is-valid': item.isValid && !item.isDuplicate,
                'is-duplicate': item.isDuplicate,
                'is-invalid': !item.isValid
              }"
            >
              <template v-if="item.isValid && !item.isDuplicate">
                <el-icon class="status-icon success"><CircleCheckFilled /></el-icon>
                <span class="parsed-name font-bold">{{ item.displayName }}</span>
                <span class="parsed-email">&lt;{{ item.email }}&gt;</span>
              </template>
              <template v-else-if="item.isDuplicate">
                <el-icon class="status-icon warning"><WarningFilled /></el-icon>
                <span class="parsed-name">{{ item.displayName }}</span>
                <span class="parsed-email">&lt;{{ item.email }}&gt;</span>
                <el-tag type="warning" size="small" class="badge-tag">重複信箱</el-tag>
              </template>
              <template v-else>
                <el-icon class="status-icon danger"><CircleCloseFilled /></el-icon>
                <span class="parsed-raw">{{ item.raw }}</span>
                <el-tag type="danger" size="small" class="badge-tag">格式無效</el-tag>
              </template>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <DialogFooter
          :confirm-text="`確認新增 (共 ${validParsedEmails.length} 筆)`"
          :loading="addSubmitting"
          :confirm-disabled="validParsedEmails.length === 0"
          @confirm="handleSaveAdd"
          @cancel="addDialogVisible = false"
        />
      </template>
    </el-dialog>

    <!-- 單筆編輯外部信箱對話框 -->
    <el-dialog
      v-model="editDialogVisible"
      title="編輯外部信箱"
      width="min(480px, calc(100vw - 32px))"
      destroy-on-close
    >
      <el-form
        ref="editFormRef"
        :model="editFormModel"
        :rules="editFormRules"
        label-width="110px"
      >
        <el-form-item label="通知主題" prop="topic">
          <el-select v-model="editFormModel.topic" placeholder="請選擇主題" style="width: 100%;">
            <el-option
              v-for="(label, key) in NOTIFICATION_TOPIC_LABELS"
              :key="key"
              :label="label"
              :value="key"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="顯示名稱" prop="displayName">
          <el-input
            v-model="editFormModel.displayName"
            placeholder="例如：衛生局承辦人 / 王督導"
          />
        </el-form-item>

        <el-form-item label="通知電子信箱" prop="email">
          <el-input
            v-model="editFormModel.email"
            placeholder="例如：officer@gov.example.tw"
          />
        </el-form-item>

        <el-form-item label="啟用狀態" prop="active">
          <el-switch v-model="editFormModel.active" />
        </el-form-item>
      </el-form>

      <template #footer>
        <DialogFooter :loading="editSaving" @confirm="handleSaveEdit" @cancel="editDialogVisible = false" />
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import DataTablePage from '@/components/DataTablePage.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import DialogFooter from '@/components/DialogFooter.vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, CircleCheckFilled, CircleCloseFilled, WarningFilled } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/utils/formatters'
import {
  listNotificationRecipients,
  batchCreateNotificationRecipients,
  updateNotificationRecipient,
  deleteNotificationRecipient,
  batchDeleteNotificationRecipients
} from '@/api/notifications'
import type { NotificationRecipientDTO } from '@/types/api'
import {
  NOTIFICATION_TOPIC_LABELS,
  type NotificationTopic
} from '@/types/domain'

interface ParsedEmailItem {
  email: string
  displayName: string
  raw: string
  isValid: boolean
  isDuplicate: boolean
}

const authStore = useAuthStore()

const recipientList = ref<NotificationRecipientDTO[]>([])
const loading = ref(false)
const searchQuery = ref('')
const selectedTopic = ref<string | undefined>(undefined)
const selectedTableRows = ref<NotificationRecipientDTO[]>([])

// 新增外部信箱對話框狀態
const addDialogVisible = ref(false)
const addTopic = ref<NotificationTopic>('missing_report')
const rawEmailsInput = ref('')
const addSubmitting = ref(false)

// 編輯外部信箱對話框狀態
const editDialogVisible = ref(false)
const currentEditId = ref<string | null>(null)
const editSaving = ref(false)
const editFormRef = ref<FormInstance>()

const editFormModel = reactive<{
  topic: NotificationTopic
  email: string
  displayName: string
  active: boolean
}>({
  topic: 'missing_report',
  email: '',
  displayName: '',
  active: true
})

const editFormRules: FormRules = {
  topic: [{ required: true, message: '請選擇通知主題', trigger: 'change' }],
  displayName: [{ required: true, message: '請輸入顯示名稱', trigger: 'blur' }],
  email: [
    { required: true, message: '請輸入電子信箱', trigger: 'blur' },
    { type: 'email', message: '請輸入合法的 Email 格式', trigger: ['blur', 'change'] }
  ]
}

// 解析外部信箱輸入（支援換行與姓名信箱格式）
const parsedEmailItems = computed<ParsedEmailItem[]>(() => {
  if (!rawEmailsInput.value.trim()) return []

  const lines = rawEmailsInput.value.split(/\r?\n/)
  const results: ParsedEmailItem[] = []
  const seenEmails = new Set<string>()
  const emailRegex = /([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})/

  for (const line of lines) {
    const trimmed = line.trim()
    if (!trimmed) continue

    const match = trimmed.match(emailRegex)
    if (match) {
      const email = match[1].trim().toLowerCase()
      // 移除 Email 與外圍符號以提取自訂姓名
      let namePart = trimmed
        .replace(match[1], '')
        .replace(/[<>()[\]:;,]/g, ' ')
        .trim()

      const displayName = namePart || email
      const isDuplicate = seenEmails.has(email)
      seenEmails.add(email)

      results.push({
        email,
        displayName,
        raw: trimmed,
        isValid: true,
        isDuplicate
      })
    } else {
      results.push({
        email: '',
        displayName: trimmed,
        raw: trimmed,
        isValid: false,
        isDuplicate: false
      })
    }
  }

  return results
})

const validParsedEmails = computed(() => {
  return parsedEmailItems.value.filter((item) => item.isValid && !item.isDuplicate)
})

const invalidCount = computed(() => {
  return parsedEmailItems.value.filter((item) => !item.isValid).length
})

const duplicateCount = computed(() => {
  return parsedEmailItems.value.filter((item) => item.isDuplicate).length
})

const filteredRecipientList = computed(() => {
  return recipientList.value
})

// 載入收件信箱清單
async function fetchRecipients() {
  loading.value = true
  try {
    const list = await listNotificationRecipients({
      topic: selectedTopic.value,
      q: searchQuery.value || undefined
    })
    recipientList.value = list
  } catch (error: any) {
    ElMessage.error(error?.message || '載入收件人失敗')
  } finally {
    loading.value = false
  }
}

function handleReset() {
  searchQuery.value = ''
  selectedTopic.value = undefined
  fetchRecipients()
}

function handleTableSelectionChange(selection: NotificationRecipientDTO[]) {
  selectedTableRows.value = selection
}

function openAddDialog() {
  addTopic.value = (selectedTopic.value as NotificationTopic) || 'missing_report'
  rawEmailsInput.value = ''
  addDialogVisible.value = true
}

async function handleSaveAdd() {
  if (validParsedEmails.value.length === 0) return

  addSubmitting.value = true
  try {
    const payload = validParsedEmails.value.map((item) => ({
      email: item.email,
      displayName: item.displayName
    }))

    await batchCreateNotificationRecipients({
      topic: addTopic.value,
      recipients: payload
    })

    ElMessage.success(`成功新增 ${payload.length} 筆外部收件信箱！`)
    addDialogVisible.value = false
    fetchRecipients()
  } catch (error: any) {
    ElMessage.error(error?.message || '新增收件信箱失敗')
  } finally {
    addSubmitting.value = false
  }
}

function openEditDialog(row: any) {
  currentEditId.value = row.id
  editFormModel.topic = row.topic
  editFormModel.email = row.email
  editFormModel.displayName = row.displayName || ''
  editFormModel.active = row.active
  editDialogVisible.value = true
}

async function handleSaveEdit() {
  if (!editFormRef.value) return
  await editFormRef.value.validate(async (valid) => {
    if (!valid) return
    editSaving.value = true
    try {
      if (currentEditId.value) {
        await updateNotificationRecipient(currentEditId.value, {
          topic: editFormModel.topic,
          email: editFormModel.email,
          displayName: editFormModel.displayName,
          active: editFormModel.active
        })
        ElMessage.success('外部收件信箱已更新！')
        editDialogVisible.value = false
        fetchRecipients()
      }
    } catch (error: any) {
      ElMessage.error(error?.message || '儲存失敗')
    } finally {
      editSaving.value = false
    }
  })
}

async function handleToggleActive(row: any, targetVal: boolean) {
  const topicLabel = (NOTIFICATION_TOPIC_LABELS as any)[row.topic] || row.topic
  const confirmMsg = targetVal
    ? `確定啟用「${topicLabel}」的收件信箱 ${row.displayName || row.email}？`
    : `確定停用「${topicLabel}」的收件信箱 ${row.displayName || row.email}？停用後此類通知不再發送至此信箱。`

  try {
    await ElMessageBox.confirm(confirmMsg, '狀態變更確認', {
      confirmButtonText: '確定',
      cancelButtonText: '取消',
      type: targetVal ? 'info' : 'warning'
    })

    await updateNotificationRecipient(row.id, { active: targetVal })
    row.active = targetVal
    ElMessage.success(`收件信箱已${targetVal ? '啟用' : '停用'}`)
  } catch (err: any) {
    if (err !== 'cancel') {
      ElMessage.error(err?.message || '操作失敗')
    }
  }
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(
      `確定刪除通知收件信箱「${row.displayName || row.email}」？刪除後無法復原。`,
      '刪除確認',
      {
        confirmButtonText: '刪除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await deleteNotificationRecipient(row.id)
    ElMessage.success('收件信箱已成功刪除！')
    fetchRecipients()
  } catch (err: any) {
    if (err !== 'cancel') {
      ElMessage.error(err?.message || '刪除失敗')
    }
  }
}

async function handleBatchDelete() {
  const count = selectedTableRows.value.length
  if (count === 0) return

  try {
    await ElMessageBox.confirm(
      `確定要批次刪除選取的 ${count} 筆通知收件信箱嗎？刪除後無法復原。`,
      '批次刪除確認',
      {
        confirmButtonText: '刪除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    const ids = selectedTableRows.value.map((r) => r.id)
    await batchDeleteNotificationRecipients(ids)
    ElMessage.success(`已成功批次刪除 ${count} 筆收件信箱！`)
    selectedTableRows.value = []
    fetchRecipients()
  } catch (err: any) {
    if (err !== 'cancel') {
      ElMessage.error(err?.message || '批次刪除失敗')
    }
  }
}

onMounted(() => {
  fetchRecipients()
})
</script>

<style scoped>
.notification-settings-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.email-text {
  font-weight: 500;
}

.topic-label {
  white-space: nowrap;
}

:deep(.topic-col .cell) {
  min-width: 140px;
}

:deep(.email-col .cell) {
  min-width: 240px;
}

:deep(.display-name-col .cell) {
  min-width: 180px;
}

/* 新增外部信箱對話框樣式 */
.add-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 12px;

  :deep(.el-form-item__label) {
    font-weight: 600;
    line-height: 1.5;
    margin-bottom: 6px;
  }
}

.email-textarea {
  font-family: 'Consolas', 'Courier New', monospace;
  font-size: 13px;
}

.preview-panel {
  border: 1px solid var(--app-border-light);
  background-color: var(--app-status-neutral-bg);
  border-radius: 6px;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.preview-header {
  display: flex;
  justify-content: space-between;
  align-items: center;

  .preview-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--app-text-primary);
  }

  .preview-counts {
    display: flex;
    gap: 6px;
  }
}

.parsed-list {
  max-height: 180px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.parsed-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  background-color: var(--app-surface);
  border: 1px solid var(--app-border-light);
  border-radius: 4px;
  font-size: 13px;

  .status-icon {
    font-size: 14px;

    &.success {
      color: var(--app-status-success-fg);
    }
    &.warning {
      color: var(--app-status-warning-fg);
    }
    &.danger {
      color: var(--app-status-danger-fg);
    }
  }

  .parsed-name {
    color: var(--app-text-primary);
  }

  .parsed-email {
    color: var(--app-text-secondary);
    font-family: 'Consolas', monospace;
    font-size: var(--app-font-xs);
  }

  .parsed-raw {
    color: var(--app-text-secondary);
  }

  .badge-tag {
    margin-left: auto;
  }

  &.is-invalid {
    background-color: var(--app-status-danger-bg);
    border-color: var(--app-status-danger-border);
  }

  &.is-duplicate {
    background-color: var(--app-status-warning-bg);
    border-color: var(--app-status-warning-border);
  }
}

</style>
