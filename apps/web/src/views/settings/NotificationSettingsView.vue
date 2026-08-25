<template>
  <div class="notification-settings-view">
    <el-card shadow="never" class="settings-card">
      <!-- 頂部篩選與操作列 -->
      <el-row :gutter="16" justify="space-between" align="middle" style="margin-bottom: 16px;">
        <el-col :span="18">
          <div class="filter-wrapper">
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
          </div>
        </el-col>
        <el-col :span="6" class="actions-col">
          <el-button
            v-if="authStore.can('admin')"
            type="primary"
            icon="Plus"
            @click="openCreateDialog"
          >
            新增收件人
          </el-button>
        </el-col>
      </el-row>

      <!-- 收件人清單表格 -->
      <el-table
        v-loading="loading"
        :data="recipientList"
        stripe
        border
        style="width: 100%;"
      >
        <el-table-column label="通知主題" width="160">
          <template #default="{ row }">
            <el-tag :type="getTopicTagType(row.topic as NotificationTopic)">
              {{ (NOTIFICATION_TOPIC_LABELS as any)[row.topic] || row.topic }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="email" label="電子信箱 (Email)" min-width="220" />
        <el-table-column prop="displayName" label="顯示名稱 / 職稱" width="180" />
        <el-table-column label="啟用狀態" width="120" align="center">
          <template #default="{ row }">
            <el-switch
              :model-value="row.active"
              :disabled="!authStore.can('admin')"
              @change="(val: string | number | boolean) => handleToggleActive(row as any, Boolean(val))"
            />
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="建立時間" width="170" />
        <el-table-column
          v-if="authStore.can('admin')"
          label="操作"
          width="150"
          fixed="right"
          align="center"
        >
          <template #default="{ row }">
            <el-button type="primary" link icon="Edit" @click="openEditDialog(row as any)">
              編輯
            </el-button>
            <el-button type="danger" link icon="Delete" @click="handleDelete(row as any)">
              刪除
            </el-button>
          </template>
        </el-table-column>

      </el-table>
    </el-card>

    <!-- 新增 / 編輯彈窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '編輯通知收件人' : '新增通知收件人'"
      width="500px"
      destroy-on-close
    >
      <el-form
        ref="formRef"
        :model="formModel"
        :rules="formRules"
        label-width="100px"
      >
        <el-form-item label="通知主題" prop="topic">
          <el-select v-model="formModel.topic" placeholder="請選擇主題" style="width: 100%;">
            <el-option
              v-for="(label, key) in NOTIFICATION_TOPIC_LABELS"
              :key="key"
              :label="label"
              :value="key"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="電子信箱" prop="email">
          <el-input
            v-model="formModel.email"
            placeholder="例如 admin@example.com"
          />
        </el-form-item>

        <el-form-item label="顯示名稱" prop="displayName">
          <el-input
            v-model="formModel.displayName"
            placeholder="例如 系統管理員 / 苗栗調度組"
          />
        </el-form-item>

        <el-form-item label="啟用狀態" prop="active">
          <el-switch v-model="formModel.active" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">
          確定存檔
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import {
  listNotificationRecipients,
  createNotificationRecipient,
  updateNotificationRecipient,
  deleteNotificationRecipient
} from '@/api/notifications'
import type { NotificationRecipientDTO } from '@/types/api'
import {
  NOTIFICATION_TOPIC_LABELS,
  type NotificationTopic
} from '@/types/domain'

const authStore = useAuthStore()

const recipientList = ref<NotificationRecipientDTO[]>([])
const loading = ref(false)
const selectedTopic = ref<string | undefined>(undefined)

const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref<string | null>(null)
const saving = ref(false)
const formRef = ref<FormInstance>()

const formModel = reactive<{
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

const formRules: FormRules = {
  topic: [{ required: true, message: '請選擇通知主題', trigger: 'change' }],
  email: [
    { required: true, message: '請輸入電子信箱', trigger: 'blur' },
    { type: 'email', message: '請輸入合法的 Email 格式', trigger: ['blur', 'change'] }
  ]
}

function getTopicTagType(topic: NotificationTopic): 'primary' | 'success' | 'warning' | 'info' | 'danger' {
  switch (topic) {
    case 'missing_report':
      return 'warning'
    case 'driver_leave':
      return 'primary'
    case 'month_end':
      return 'danger'
    case 'export_failed':
      return 'danger'
    default:
      return 'info'
  }
}

async function fetchRecipients() {
  loading.value = true
  try {
    const list = await listNotificationRecipients({
      topic: selectedTopic.value
    })
    recipientList.value = list
  } catch (error: any) {
    ElMessage.error(error?.message || '載入收件人失敗')
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  isEdit.value = false
  currentId.value = null
  formModel.topic = 'missing_report'
  formModel.email = ''
  formModel.displayName = ''
  formModel.active = true
  dialogVisible.value = true
}

function openEditDialog(row: any) {
  isEdit.value = true
  currentId.value = row.id
  formModel.topic = row.topic
  formModel.email = row.email
  formModel.displayName = row.displayName || ''
  formModel.active = row.active
  dialogVisible.value = true
}

async function handleSave() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      if (isEdit.value && currentId.value) {
        await updateNotificationRecipient(currentId.value, {
          topic: formModel.topic,
          email: formModel.email,
          displayName: formModel.displayName,
          active: formModel.active
        })
        ElMessage.success('收件人資料已更新！')
      } else {
        await createNotificationRecipient({
          topic: formModel.topic,
          email: formModel.email,
          displayName: formModel.displayName,
          active: formModel.active
        })
        ElMessage.success('成功新增收件人！')
      }
      dialogVisible.value = false
      fetchRecipients()
    } catch (error: any) {
      ElMessage.error(error?.message || '儲存失敗')
    } finally {
      saving.value = false
    }
  })
}

async function handleToggleActive(row: any, targetVal: boolean) {
  const topicLabel = (NOTIFICATION_TOPIC_LABELS as any)[row.topic] || row.topic
  const confirmMsg = targetVal
    ? `確定啟用「${topicLabel}」的收件人 ${row.email}？`
    : `確定停用「${topicLabel}」的收件人 ${row.email}？停用後此類通知不再寄給他。`


  try {
    await ElMessageBox.confirm(confirmMsg, '狀態變更確認', {
      confirmButtonText: '確定',
      cancelButtonText: '取消',
      type: targetVal ? 'info' : 'warning'
    })

    await updateNotificationRecipient(row.id, { active: targetVal })
    row.active = targetVal
    ElMessage.success(`收件人已${targetVal ? '啟用' : '停用'}`)
  } catch (err: any) {
    if (err !== 'cancel') {
      ElMessage.error(err?.message || '操作失敗')
    }
  }
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(
      `確定刪除通知收件人 ${row.email}？刪除後無法復原。`,
      '刪除確認',
      {
        confirmButtonText: '確定刪除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )


    await deleteNotificationRecipient(row.id)
    ElMessage.success('收件人已成功刪除！')
    fetchRecipients()
  } catch (err: any) {
    if (err !== 'cancel') {
      ElMessage.error(err?.message || '刪除失敗')
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

.settings-card {
  border-radius: 8px;
}

.filter-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
}

.actions-col {
  display: flex;
  justify-content: flex-end;
}
</style>
