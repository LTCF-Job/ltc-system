<template>
  <div class="user-management-view">
    <DataTablePage
      title="使用者管理"
      :loading="loading"
      :total="total"
      :page="page"
      :page-size="pageSize"
      @page-change="onPageChange"
      @size-change="onSizeChange"
    >
      <template #filter>
        <el-input
          v-model="queryKeyword"
          placeholder="搜尋使用者姓名／電子郵件"
          clearable
          style="width: 240px"
          @keyup.enter="fetchUsers"
        />

        <el-select
          v-model="queryRole"
          placeholder="身分角色"
          clearable
          style="width: 150px"
          @change="fetchUsers"
        >
          <el-option
            v-for="r in roleList"
            :key="r.key"
            :label="r.name"
            :value="r.key"
          >
            <span class="role-text" :class="`role-${r.key}`">
              <span class="role-dot" :class="`dot-${r.key}`"></span>
              {{ r.name }}
            </span>
          </el-option>
        </el-select>

        <el-button type="primary" @click="fetchUsers">查詢</el-button>
        <el-button @click="handleReset">重設</el-button>
      </template>

      <template #actions>
        <el-button type="primary" :icon="Plus" @click="openCreateDialog">
          新增使用者
        </el-button>
      </template>

      <template #table>
        <el-table :data="users" border stripe style="width: 100%">
          <el-table-column prop="displayName" label="使用者姓名" min-width="140">
            <template #default="{ row }">
              <div class="user-name-cell">
                <span class="font-bold">{{ row.displayName }}</span>
              </div>
            </template>
          </el-table-column>

          <el-table-column prop="email" label="電子郵件 / 帳號" min-width="200" show-overflow-tooltip />

          <el-table-column label="身分角色" width="140" align="center">
            <template #default="{ row }">
              <span class="role-text" :class="`role-${(row as any).role}`">
                <span class="role-dot" :class="`dot-${(row as any).role}`"></span>
                {{ getRoleDisplayName((row as any).role) }}
              </span>
            </template>
          </el-table-column>

          <el-table-column label="權限模式" width="140" align="center">
            <template #default="{ row }">
              <span
                v-if="(row as any).customPermissions && Object.keys((row as any).customPermissions).length > 0"
                class="perm-mode-custom"
              >
                個人自訂權限
              </span>
              <span v-else class="perm-mode-default">
                套用角色預設
              </span>
            </template>
          </el-table-column>

          <el-table-column prop="status" label="帳號狀態" width="100" align="center">
            <template #default="{ row }">
              <el-switch
                v-model="(row as any).status"
                active-value="active"
                inactive-value="inactive"
                :disabled="(row as any).role === 'admin' && (row as any).id === currentUserId"
                @change="handleToggleStatus(row as any)"
              />
            </template>
          </el-table-column>

          <el-table-column prop="lastLoginAt" label="最後登入" width="170" align="center">
            <template #default="{ row }">
              <span>{{ (row as any).lastLoginAt ? formatDateTime((row as any).lastLoginAt) : '從未登入' }}</span>
            </template>
          </el-table-column>

          <el-table-column label="操作" width="220" fixed="right" align="center">
            <template #default="{ row }">
              <TableRowActions>
                <el-button link type="primary" size="small" @click="openEditDialog(row as any)">
                  編輯
                </el-button>
                <el-button link type="primary" size="small" @click="openPermissionDrawer(row as any)">
                  設定權限
                </el-button>
                <el-button
                  v-if="(row as any).id !== currentUserId"
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

    <!-- 新增 / 編輯使用者彈窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '編輯使用者基本資料' : '新增系統使用者'"
      width="min(480px, calc(100vw - 32px))"
      destroy-on-close
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-width="110px"
        label-position="right"
      >
        <el-form-item label="電子郵件" prop="email">
          <el-input
            v-model="form.email"
            placeholder="請輸入電子郵件（作為登入帳號）"
            :disabled="!!editingId"
          />
        </el-form-item>

        <el-form-item label="使用者姓名" prop="displayName">
          <el-input v-model="form.displayName" placeholder="如：王小明" />
        </el-form-item>

        <el-form-item label="聯絡電話" prop="phone">
          <el-input v-model="form.phone" placeholder="如：0912-345-678" />
        </el-form-item>

        <el-form-item label="身分角色" prop="role">
          <el-select v-model="form.role" placeholder="請選擇身分角色" style="width: 100%">
            <el-option
              v-for="r in roleList"
              :key="r.key"
              :label="r.name"
              :value="r.key"
            >
              <span class="role-text" :class="`role-${r.key}`">
                <span class="role-dot" :class="`dot-${r.key}`"></span>
                {{ r.name }}
              </span>
            </el-option>
          </el-select>
        </el-form-item>

        <el-form-item v-if="!editingId" label="登入密碼" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="請輸入初始登入密碼（至少 6 碼）"
            show-password
          />
        </el-form-item>

        <el-form-item label="帳號狀態" prop="status">
          <el-radio-group v-model="form.status">
            <el-radio value="active">啟用中</el-radio>
            <el-radio value="inactive">停用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>

      <template #footer>
        <DialogFooter :loading="submitting" @confirm="handleSubmit" @cancel="dialogVisible = false" />
      </template>
    </el-dialog>

    <!-- 個人權限自訂抽屜 -->
    <el-drawer
      v-model="drawerVisible"
      title="自訂功能模組權限"
      size="min(620px, 92vw)"
      destroy-on-close
    >
      <div v-if="selectedUser" class="perm-drawer-content">
        <el-alert
          type="info"
          show-icon
          :closable="false"
          style="margin-bottom: 16px"
        >
          <template #title>
            <span class="font-bold">權限套用規則：個人設定 ＞ 角色預設</span>
          </template>
          當此使用者具備自訂設定時優先採用；若無自訂設定則套用
          <strong>【{{ getRoleDisplayName(selectedUser.role) }}】</strong>
          之角色預設權限。
        </el-alert>

        <div class="perm-user-header">
          <div>
            <span class="perm-user-name">{{ selectedUser.displayName }}</span>
            <span class="perm-user-email">({{ selectedUser.email }})</span>
          </div>
          <el-button size="small" type="warning" plain @click="handleResetToRoleDefault">
            重設為【{{ getRoleDisplayName(selectedUser.role) }}】角色預設
          </el-button>
        </div>

        <el-table
          :data="SYSTEM_MODULES"
          border
          stripe
          size="small"
          style="width: 100%"
          max-height="calc(100vh - 280px)"
        >
          <el-table-column prop="categoryName" label="分類" width="110" align="center">
            <template #default="{ row }">
              <span>{{ row.categoryName }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="name" label="功能區塊模組" min-width="160" />
          <el-table-column label="顯示／檢視" width="105" align="center">
            <template #default="{ row }">
              <el-checkbox
                v-model="tempPermissions[row.id].view"
                @change="onViewPermChange(row.id)"
              />
            </template>
          </el-table-column>
          <el-table-column label="操作／編輯" width="105" align="center">
            <template #default="{ row }">
              <el-checkbox
                v-model="tempPermissions[row.id].edit"
                :disabled="!tempPermissions[row.id].view"
              />
            </template>
          </el-table-column>
        </el-table>
      </div>

      <template #footer>
        <DialogFooter :loading="savingPerms" @confirm="handleSavePermissions" @cancel="drawerVisible = false" />
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import DataTablePage from '@/components/DataTablePage.vue'
import DialogFooter from '@/components/DialogFooter.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import {
  listUsers,
  createUser,
  updateUser,
  updateUserPermissions,
  deleteUser
} from '@/api/users'
import { listRoles } from '@/api/roles'
import { formatDateTime } from '@/utils/formatters'
import { useAuthStore } from '@/stores/auth'
import type { UserDTO, RoleDTO } from '@/types/api'
import {
  ROLE_LABELS,
  SYSTEM_MODULES,
  type UserRole,
  type SystemPermissions
} from '@/types/domain'

const authStore = useAuthStore()
const currentUserId = computed(() => authStore.user?.id)

const users = ref<UserDTO[]>([])
const roleList = ref<RoleDTO[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const queryKeyword = ref('')
const queryRole = ref<string | undefined>(undefined)

const dialogVisible = ref(false)
const editingId = ref<string | null>(null)
const submitting = ref(false)
const formRef = ref<FormInstance>()

const form = reactive({
  email: '',
  displayName: '',
  phone: '',
  role: 'dispatcher' as UserRole,
  password: '',
  status: 'active' as 'active' | 'inactive'
})

const formRules = {
  email: [
    { required: true, message: '請輸入電子郵件', trigger: 'blur' },
    { type: 'email', message: '請輸入正確的電子郵件格式', trigger: 'blur' }
  ],
  displayName: [{ required: true, message: '請輸入姓名', trigger: 'blur' }],
  role: [{ required: true, message: '請選擇身分角色', trigger: 'change' }],
  password: [
    {
      validator: (_rule: any, val: string, callback: any) => {
        if (!editingId.value && (!val || val.length < 6)) {
          return callback(new Error('初始密碼長度至少需為 6 個字元'))
        }
        callback()
      },
      trigger: 'blur'
    }
  ]
}

// 權限自訂抽屜狀態
const drawerVisible = ref(false)
const selectedUser = ref<UserDTO | null>(null)
const tempPermissions = ref<SystemPermissions>({})
const savingPerms = ref(false)

function getRoleDisplayName(roleKey?: string): string {
  if (!roleKey) return '未知角色'
  const role = roleList.value.find((r) => r.key === roleKey)
  if (role) return role.name
  return (ROLE_LABELS as any)[roleKey] || roleKey
}

const rolesLoadError = ref(false)

async function fetchRoles() {
  rolesLoadError.value = false
  try {
    const list = await listRoles()
    roleList.value = list
  } catch {
    rolesLoadError.value = true
    ElMessage.error('載入角色清單失敗，個人自訂權限暫時無法設定，請重試')
  }
}

async function fetchUsers() {
  loading.value = true
  try {
    const list = await listUsers({
      q: queryKeyword.value || undefined,
      role: queryRole.value || undefined
    })
    users.value = list
    total.value = list.length
  } finally {
    loading.value = false
  }
}

function handleReset() {
  queryKeyword.value = ''
  queryRole.value = undefined
  page.value = 1
  fetchUsers()
}

function onPageChange(p: number) {
  page.value = p
}

function onSizeChange(size: number) {
  pageSize.value = size
  page.value = 1
}

function openCreateDialog() {
  editingId.value = null
  form.email = ''
  form.displayName = ''
  form.phone = ''
  form.role = 'dispatcher'
  form.password = ''
  form.status = 'active'
  dialogVisible.value = true
}

function openEditDialog(user: UserDTO) {
  editingId.value = user.id
  form.email = user.email
  form.displayName = user.displayName
  form.phone = user.phone || ''
  form.role = user.role
  form.password = ''
  form.status = user.status || 'active'
  dialogVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (editingId.value) {
        await updateUser(editingId.value, {
          displayName: form.displayName,
          phone: form.phone,
          role: form.role,
          status: form.status
        })
        ElMessage.success('使用者資料更新成功')
      } else {
        await createUser({
          email: form.email,
          displayName: form.displayName,
          phone: form.phone,
          role: form.role,
          password: form.password,
          status: form.status
        })
        ElMessage.success('使用者建立成功')
      }
      dialogVisible.value = false
      fetchUsers()
    } finally {
      submitting.value = false
    }
  })
}

async function handleToggleStatus(user: UserDTO) {
  const previousStatus = user.status === 'active' ? 'inactive' : 'active'
  try {
    await updateUser(user.id, {
      status: user.status
    })
    ElMessage.success(`已將「${user.displayName}」帳號設定為 ${user.status === 'active' ? '啟用' : '停用'}`)
  } catch {
    user.status = previousStatus
    ElMessage.error(`更新「${user.displayName}」帳號狀態失敗，請重試`)
  }
}

async function handleDelete(user: UserDTO) {
  try {
    await ElMessageBox.confirm(
      `確定要刪除使用者「${user.displayName} (${user.email})」嗎？此動作無法復原。`,
      '確認刪除使用者',
      {
        confirmButtonText: '刪除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    await deleteUser(user.id)
    ElMessage.success(`已成功刪除使用者「${user.displayName}」`)
    fetchUsers()
  } catch {
    // 使用者取消操作
  }
}

// 權限設定邏輯：角色預設權限一律以後端 /roles 回傳為準，找不到時視為資料不足，不得用前端猜測值頂替
function getRoleDefaultPermissions(roleKey: string): SystemPermissions | null {
  const role = roleList.value.find((r) => r.key === roleKey)
  return role?.permissions || null
}

function openPermissionDrawer(user: UserDTO) {
  const roleDefault = getRoleDefaultPermissions(user.role)
  if (!roleDefault) {
    ElMessage.error('無法取得該角色的預設權限，請重新載入角色清單後再試')
    return
  }

  selectedUser.value = user
  const initialPerms: SystemPermissions = {}
  const custom = user.customPermissions || {}

  for (const m of SYSTEM_MODULES) {
    const isCustomized = custom[m.id] !== undefined
    const base = isCustomized ? custom[m.id] : roleDefault[m.id] || { view: false, edit: false }
    initialPerms[m.id] = {
      view: !!base.view,
      edit: !!base.edit
    }
  }

  tempPermissions.value = initialPerms
  drawerVisible.value = true
}

function onViewPermChange(modId: string) {
  // 若關閉檢視權限，自動關閉編輯權限
  if (!tempPermissions.value[modId].view) {
    tempPermissions.value[modId].edit = false
  }
}

function handleResetToRoleDefault() {
  if (!selectedUser.value) return
  const roleDefault = getRoleDefaultPermissions(selectedUser.value.role)
  if (!roleDefault) {
    ElMessage.error('無法取得該角色的預設權限，請重新載入角色清單後再試')
    return
  }
  const resetPerms: SystemPermissions = {}
  for (const m of SYSTEM_MODULES) {
    const base = roleDefault[m.id] || { view: false, edit: false }
    resetPerms[m.id] = { view: !!base.view, edit: !!base.edit }
  }
  tempPermissions.value = resetPerms
  ElMessage.info(`已載入【${getRoleDisplayName(selectedUser.value.role)}】角色之預設權限配置`)
}

async function handleSavePermissions() {
  if (!selectedUser.value) return
  savingPerms.value = true
  try {
    await updateUserPermissions(selectedUser.value.id, tempPermissions.value)
    ElMessage.success(`已儲存「${selectedUser.value.displayName}」之個人權限設定`)
    drawerVisible.value = false
    fetchUsers()
  } finally {
    savingPerms.value = false
  }
}

onMounted(() => {
  fetchRoles()
  fetchUsers()
})
</script>

<style scoped>
.user-management-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.user-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;

  .user-avatar {
    background-color: var(--el-color-primary-light-8);
    color: var(--el-color-primary);
  }
}

.font-bold {
  font-weight: bold;
}

.role-text {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 500;

  &.role-admin { color: var(--app-role-admin-fg); }
  &.role-dispatcher { color: var(--app-role-dispatcher-fg); }
  &.role-staff { color: var(--app-role-staff-fg); }
  &.role-driver { color: var(--app-role-driver-fg); }
  &.role-viewer { color: var(--app-role-viewer-fg); }
}

.role-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  display: inline-block;

  &.dot-admin { background-color: var(--app-role-admin-dot); }
  &.dot-dispatcher { background-color: var(--app-role-dispatcher-dot); }
  &.dot-staff { background-color: var(--app-role-staff-dot); }
  &.dot-driver { background-color: var(--app-role-driver-dot); }
  &.dot-viewer { background-color: var(--app-role-viewer-dot); }
}

.perm-mode-custom {
  color: #d97706;
  font-size: 13px;
  font-weight: 500;
}

.perm-mode-default {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.perm-drawer-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.perm-user-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background-color: var(--el-fill-color-light);
  border-radius: 6px;

  .perm-user-name {
    font-size: 15px;
    font-weight: bold;
    color: var(--el-text-color-primary);
  }

  .perm-user-email {
    font-size: 13px;
    color: var(--el-text-color-secondary);
    margin-left: 6px;
  }
}
</style>
