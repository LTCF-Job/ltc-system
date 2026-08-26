<template>
  <div class="notification-settings-view">
    <el-card shadow="never" class="settings-card">
      <!-- 頂部篩選與操作列 -->
      <el-row :gutter="16" justify="space-between" align="middle" style="margin-bottom: 16px;">
        <el-col :span="19">
          <div class="filter-wrapper">
            <el-input
              v-model="searchQuery"
              placeholder="搜尋信箱／顯示名稱／角色"
              clearable
              style="width: 220px;"
              @keyup.enter="fetchRecipients"
            />

            <el-select
              v-model="selectedTopic"
              placeholder="通知主題篩選"
              clearable
              style="width: 160px;"
              @change="fetchRecipients"
            >
              <el-option
                v-for="(label, key) in NOTIFICATION_TOPIC_LABELS"
                :key="key"
                :label="label"
                :value="key"
              />
            </el-select>

            <el-select
              v-model="selectedRecipientType"
              placeholder="收件類型篩選"
              clearable
              style="width: 150px;"
              @change="fetchRecipients"
            >
              <el-option label="指定角色群組" value="role" />
              <el-option label="系統使用者" value="user" />
              <el-option label="自訂外部信箱" value="custom" />
            </el-select>

            <el-button type="primary" icon="Search" @click="fetchRecipients">
              查詢
            </el-button>
            <el-button @click="handleReset">
              重設
            </el-button>
          </div>
        </el-col>
        <el-col :span="5" class="actions-col">
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
        :data="filteredRecipientList"
        stripe
        border
        style="width: 100%;"
      >
        <el-table-column label="通知主題" width="150">
          <template #default="{ row }">
            <el-tag :type="getTopicTagType(row.topic as NotificationTopic)">
              {{ (NOTIFICATION_TOPIC_LABELS as any)[row.topic] || row.topic }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="收件類型" width="130" align="center">
          <template #default="{ row }">
            <el-tag
              v-if="row.recipientType === 'role' || (!row.recipientType && row.targetRole)"
              type="primary"
              size="small"
            >
              角色群組
            </el-tag>
            <el-tag
              v-else-if="row.recipientType === 'user' || row.userId"
              type="success"
              size="small"
            >
              系統使用者
            </el-tag>
            <el-tag
              v-else
              type="info"
              size="small"
            >
              自訂外部信箱
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="收件對象 / 角色" min-width="220">
          <template #default="{ row }">
            <div class="target-cell">
              <el-tag
                v-if="row.targetRole || row.recipientType === 'role'"
                size="small"
                :type="row.targetRole === 'admin' ? 'danger' : ((row.targetRole === 'dispatcher' || row.targetRole === 'staff') ? 'primary' : 'info')"
              >
                {{ (ROLE_LABELS as any)[row.targetRole] || row.targetRole || '角色' }}
              </el-tag>
              <span class="target-title">{{ row.displayName }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="email" label="通知電子信箱 (Email)" min-width="220" />

        <el-table-column label="啟用狀態" width="110" align="center">
          <template #default="{ row }">
            <el-switch
              :model-value="row.active"
              :disabled="!authStore.can('admin')"
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
      width="560px"
      destroy-on-close
    >
      <el-form
        ref="formRef"
        :model="formModel"
        :rules="formRules"
        label-width="110px"
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

        <el-form-item label="收件類型" prop="recipientType">
          <el-radio-group v-model="formModel.recipientType" @change="onRecipientTypeChange">
            <el-radio-button label="role">指定角色群組</el-radio-button>
            <el-radio-button label="user">系統使用者</el-radio-button>
            <el-radio-button label="custom">自訂外部信箱</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <!-- 類型一：指定角色群組 -->
        <template v-if="formModel.recipientType === 'role'">
          <el-form-item label="選擇目標角色" prop="targetRole">
            <el-select
              v-model="formModel.targetRole"
              placeholder="請選擇目標角色"
              style="width: 100%;"
              @change="onTargetRoleChange"
            >
              <el-option
                v-for="(label, key) in ROLE_LABELS"
                :key="key"
                :label="label"
                :value="key"
              >
                <div style="display: flex; justify-content: space-between; align-items: center;">
                  <span>{{ label }}</span>
                  <el-tag size="small" type="info">{{ getRoleUserCount(key) }} 位成員</el-tag>
                </div>
              </el-option>
            </el-select>
          </el-form-item>

          <el-form-item label="目前角色成員">
            <div class="role-members-preview">
              <span v-if="roleMembersPreview.length === 0" class="text-muted">（目前此角色尚無任何在職使用者）</span>
              <el-tag
                v-for="u in roleMembersPreview"
                :key="u.id"
                size="small"
                type="info"
                style="margin-right: 6px; margin-bottom: 4px;"
              >
                {{ u.displayName || u.email }}
              </el-tag>
            </div>
          </el-form-item>
        </template>

        <!-- 類型二：指定個別使用者 -->
        <template v-if="formModel.recipientType === 'user'">
          <el-form-item label="選擇使用者" prop="userId">
            <el-select
              v-model="formModel.userId"
              placeholder="請搜尋或選擇使用者"
              filterable
              style="width: 100%;"
              @change="onUserSelectChange"
            >
              <el-option
                v-for="u in userList"
                :key="u.id"
                :label="`${u.displayName} (${u.email})`"
                :value="u.id"
              >
                <div style="display: flex; justify-content: space-between; align-items: center;">
                  <span>{{ u.displayName }} <small style="color: #999;">({{ u.email }})</small></span>
                  <el-tag size="small" :type="u.role === 'admin' ? 'danger' : 'primary'">{{ (ROLE_LABELS as any)[u.role] || u.role }}</el-tag>
                </div>
              </el-option>
            </el-select>
          </el-form-item>
        </template>

        <el-form-item label="顯示名稱" prop="displayName">
          <el-input
            v-model="formModel.displayName"
            placeholder="例如 全體調度組 / 王大明"
          />
        </el-form-item>

        <el-form-item label="通知電子信箱" prop="email">
          <el-input
            v-model="formModel.email"
            placeholder="例如 dispatch@ltc.example.com"
          />
          <div v-if="formModel.recipientType === 'role'" class="form-tip">
            💡 此信箱為通知發送目標（可設定群組轉發信箱或個別代表信箱）
          </div>
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
import { ref, reactive, computed, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/utils/formatters'
import {
  listNotificationRecipients,
  createNotificationRecipient,
  updateNotificationRecipient,
  deleteNotificationRecipient
} from '@/api/notifications'
import { listUsers } from '@/api/users'
import type { NotificationRecipientDTO, UserDTO, RecipientTargetType } from '@/types/api'
import {
  NOTIFICATION_TOPIC_LABELS,
  ROLE_LABELS,
  type NotificationTopic,
  type UserRole
} from '@/types/domain'

const authStore = useAuthStore()

const recipientList = ref<NotificationRecipientDTO[]>([])
const userList = ref<UserDTO[]>([])
const loading = ref(false)
const searchQuery = ref('')
const selectedTopic = ref<string | undefined>(undefined)
const selectedRecipientType = ref<string | undefined>(undefined)

const dialogVisible = ref(false)
const isEdit = ref(false)
const currentId = ref<string | null>(null)
const saving = ref(false)
const formRef = ref<FormInstance>()

const formModel = reactive<{
  topic: NotificationTopic
  recipientType: RecipientTargetType
  targetRole?: UserRole
  userId?: string
  email: string
  displayName: string
  active: boolean
}>({
  topic: 'missing_report',
  recipientType: 'role',
  targetRole: 'admin',
  userId: '',
  email: 'admin@ltc.example.com',
  displayName: '全體系統管理員',
  active: true
})

const formRules: FormRules = {
  topic: [{ required: true, message: '請選擇通知主題', trigger: 'change' }],
  targetRole: [
    {
      validator: (_rule, value, callback) => {
        if (formModel.recipientType === 'role' && !value) {
          callback(new Error('請選擇目標角色'))
        } else {
          callback()
        }
      },
      trigger: 'change'
    }
  ],
  userId: [
    {
      validator: (_rule, value, callback) => {
        if (formModel.recipientType === 'user' && !value) {
          callback(new Error('請選擇系統使用者'))
        } else {
          callback()
        }
      },
      trigger: 'change'
    }
  ],
  displayName: [{ required: true, message: '請輸入顯示名稱', trigger: 'blur' }],
  email: [
    { required: true, message: '請輸入電子信箱', trigger: 'blur' },
    { type: 'email', message: '請輸入合法的 Email 格式', trigger: ['blur', 'change'] }
  ]
}

const roleMembersPreview = computed(() => {
  if (!formModel.targetRole) return []
  return userList.value.filter((u) => u.role === formModel.targetRole)
})

function getRoleUserCount(roleKey: string): number {
  return userList.value.filter((u) => u.role === roleKey).length
}

const filteredRecipientList = computed(() => {
  return recipientList.value.filter((r) => {
    if (selectedRecipientType.value) {
      const type = r.recipientType || (r.targetRole ? 'role' : (r.userId ? 'user' : 'custom'))
      if (type !== selectedRecipientType.value) return false
    }
    return true
  })
})

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

async function fetchUsers() {
  try {
    userList.value = await listUsers()
  } catch {
    // 降級處理
  }
}

async function fetchRecipients() {
  loading.value = true
  try {
    const list = await listNotificationRecipients({
      topic: selectedTopic.value,
      recipientType: selectedRecipientType.value,
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
  selectedRecipientType.value = undefined
  fetchRecipients()
}

function onRecipientTypeChange(type: any) {
  if (type === 'role') {
    if (!formModel.targetRole) {
      formModel.targetRole = 'admin'
    }
    onTargetRoleChange(formModel.targetRole)
  } else if (type === 'user') {
    if (formModel.userId) {
      onUserSelectChange(formModel.userId)
    } else if (userList.value.length > 0) {
      formModel.userId = userList.value[0].id
      onUserSelectChange(formModel.userId)
    }
  } else {
    formModel.targetRole = undefined
    formModel.userId = undefined
    if (formModel.displayName.startsWith('全體')) {
      formModel.displayName = ''
    }
  }
}

function onTargetRoleChange(role: UserRole) {
  const roleName = (ROLE_LABELS as any)[role] || role
  formModel.displayName = `全體${roleName}`
  const user = userList.value.find((u) => u.role === role)
  formModel.email = user ? user.email : `${role}@ltc.example.com`
}

function onUserSelectChange(userId: string) {
  const user = userList.value.find((u) => u.id === userId)
  if (user) {
    formModel.displayName = user.displayName
    formModel.email = user.email
    formModel.targetRole = user.role
  }
}

function openCreateDialog() {
  isEdit.value = false
  currentId.value = null
  formModel.topic = 'missing_report'
  formModel.recipientType = 'role'
  formModel.targetRole = 'admin'
  formModel.userId = ''
  formModel.displayName = '全體系統管理員'
  const adminUser = userList.value.find((u) => u.role === 'admin')
  formModel.email = adminUser ? adminUser.email : 'admin@ltc.example.com'
  formModel.active = true
  dialogVisible.value = true
}

function openEditDialog(row: any) {
  isEdit.value = true
  currentId.value = row.id
  formModel.topic = row.topic
  formModel.recipientType = row.recipientType || (row.targetRole ? 'role' : (row.userId ? 'user' : 'custom'))
  formModel.targetRole = row.targetRole || 'admin'
  formModel.userId = row.userId || ''
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
          recipientType: formModel.recipientType,
          targetRole: formModel.recipientType === 'role' ? formModel.targetRole : undefined,
          userId: formModel.recipientType === 'user' ? formModel.userId : undefined,
          email: formModel.email,
          displayName: formModel.displayName,
          active: formModel.active
        })
        ElMessage.success('收件人資料已更新！')
      } else {
        await createNotificationRecipient({
          topic: formModel.topic,
          recipientType: formModel.recipientType,
          targetRole: formModel.recipientType === 'role' ? formModel.targetRole : undefined,
          userId: formModel.recipientType === 'user' ? formModel.userId : undefined,
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
    ? `確定啟用「${topicLabel}」的收件對象 ${row.displayName || row.email}？`
    : `確定停用「${topicLabel}」的收件對象 ${row.displayName || row.email}？停用後此類通知不再寄給此收件對象。`

  try {
    await ElMessageBox.confirm(confirmMsg, '狀態變更確認', {
      confirmButtonText: '確定',
      cancelButtonText: '取消',
      type: targetVal ? 'info' : 'warning'
    })

    await updateNotificationRecipient(row.id, { active: targetVal })
    row.active = targetVal
    ElMessage.success(`收件對象已${targetVal ? '啟用' : '停用'}`)
  } catch (err: any) {
    if (err !== 'cancel') {
      ElMessage.error(err?.message || '操作失敗')
    }
  }
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(
      `確定刪除通知收件對象「${row.displayName || row.email}」？刪除後無法復原。`,
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
  fetchUsers()
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
  flex-wrap: wrap;
}

.actions-col {
  display: flex;
  justify-content: flex-end;
}

.target-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.target-title {
  font-weight: 500;
}

.role-members-preview {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  min-height: 32px;
  padding: 4px 8px;
  background-color: var(--el-fill-color-light);
  border-radius: 4px;
}

.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
  line-height: 1.4;
}

.text-muted {
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}
</style>
