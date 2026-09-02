<template>
  <div class="role-management-view">
    <PageHeader title="角色身分管理" />
    <el-alert
      type="info"
      show-icon
      :closable="false"
      class="rule-alert"
    >
      <template #title>
        <span class="font-bold">權限套用規則：個人設定 ＞ 角色身分設定 ＞ 系統預設</span>
      </template>
      系統以使用者為中心進行權限裁決。若使用者擁有個人自訂權限，則優先套用個人設定；若無個人設定，系統自動繼承並套用所屬角色之預設模組權限配置。
    </el-alert>

    <!-- 頂部搜尋與動作列 -->
    <div class="toolbar-card">
      <div class="toolbar-left">
        <el-input
          v-model="searchQuery"
          placeholder="搜尋角色名稱或說明"
          :prefix-icon="Search"
          clearable
          style="width: 240px"
        />
        <el-tag type="info" effect="plain" class="total-tag">
          共 {{ filteredRoles.length }} 個角色身分
        </el-tag>
      </div>

      <div class="toolbar-right">
        <el-button type="primary" :icon="Plus" @click="openCreateDialog">
          新增自訂角色
        </el-button>
      </div>
    </div>

    <!-- 角色清單 + 權限矩陣：左側單欄清單取代原本的彩色方塊網格，
         差異靠字重／描述／統計數字呈現，不再用 5 種角色色分別上色 -->
    <div class="role-layout">
      <el-card shadow="never" class="role-list-card">
        <div class="role-list">
          <div
            v-for="role in filteredRoles"
            :key="role.id || role.key"
            class="role-row"
            :class="{ 'is-selected': isSelectedRole(role) }"
            role="button"
            tabindex="0"
            @click="selectedRole = role"
            @keydown.enter="selectedRole = role"
          >
            <div class="role-row-main">
              <div class="role-row-heading">
                <span class="role-row-name">{{ role.name }}</span>
                <span v-if="role.isSystem" class="role-row-system-badge">系統內建</span>
              </div>
              <p class="role-row-desc" :title="role.description">{{ role.description || '無詳細說明' }}</p>
            </div>

            <div class="role-row-meta">
              <span class="role-row-meta-item">{{ role.userCount || 0 }} 位使用者</span>
              <span class="role-row-meta-sep">·</span>
              <span class="role-row-meta-item">
                {{ countRolePerms(role.permissions).views }} 檢視 / {{ countRolePerms(role.permissions).edits }} 編輯
              </span>
            </div>

            <TableRowActions @click.stop>
              <el-button type="primary" link size="small" @click="openEditDialog(role)">
                編輯設定
              </el-button>
              <el-button type="primary" link size="small" @click="openCopyDialog(role)">
                複製建立
              </el-button>
              <el-tooltip
                v-if="role.isSystem"
                content="系統內建核心角色不可刪除"
                placement="top"
              >
                <span>
                  <el-button type="danger" link size="small" disabled>
                    刪除
                  </el-button>
                </span>
              </el-tooltip>
              <el-button v-else type="danger" link size="small" @click="handleDeleteRole(role)">
                刪除
              </el-button>
            </TableRowActions>
          </div>

          <el-empty v-if="!filteredRoles.length" description="找不到符合條件的角色" />
        </div>
      </el-card>

      <!-- 角色模組權限矩陣檢視卡片 -->
      <el-card v-if="selectedRole" shadow="never" class="matrix-card">
        <template #header>
          <div class="matrix-header">
            <span class="matrix-title">
              【{{ selectedRole.name }}】功能模組權限矩陣
            </span>
            <div class="matrix-header-right">
              <span class="matrix-stat-text">
                支援 {{ countRolePerms(selectedRole.permissions).views }} 項檢視、{{ countRolePerms(selectedRole.permissions).edits }} 項操作
              </span>
              <el-button type="primary" size="small" plain @click="openEditDialog(selectedRole)">
                修改此角色權限
              </el-button>
            </div>
          </div>
        </template>

        <el-table
          :data="SYSTEM_MODULES"
          border
          stripe
          size="small"
          style="width: 100%"
        >
          <el-table-column prop="categoryName" label="分類" width="170" align="center">
            <template #default="{ row }">
              <span>{{ row.categoryName }}</span>
            </template>
          </el-table-column>

          <el-table-column prop="name" label="功能模組名稱" min-width="200">
            <template #default="{ row }">
              <span class="font-bold">{{ row.name }}</span>
            </template>
          </el-table-column>

          <el-table-column label="檢視權限" width="180" align="center">
            <template #default="{ row }">
              <span
                v-if="selectedRole?.permissions?.[row.id]?.view"
                class="perm-status perm-view"
              >
                <el-icon><Check /></el-icon> 允許檢視
              </span>
              <span v-else class="perm-status perm-none">
                <el-icon><Close /></el-icon> 無權限
              </span>
            </template>
          </el-table-column>

          <el-table-column label="編輯/操作權限" width="180" align="center">
            <template #default="{ row }">
              <span
                v-if="selectedRole?.permissions?.[row.id]?.edit"
                class="perm-status perm-edit"
              >
                <el-icon><Check /></el-icon> 允許新增/修改
              </span>
              <span v-else class="perm-status perm-none">
                <el-icon><Close /></el-icon> 僅讀/無權限
              </span>
            </template>
          </el-table-column>

          <el-table-column label="刪除權限" width="180" align="center">
            <template #default="{ row }">
              <span
                v-if="selectedRole?.permissions?.[row.id]?.delete"
                class="perm-status perm-delete"
              >
                <el-icon><Check /></el-icon> 允許刪除
              </span>
              <span v-else class="perm-status perm-none">
                <el-icon><Close /></el-icon> 無權限
              </span>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>

    <!-- 新增 / 編輯 / 複製角色 Dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="min(820px, calc(100vw - 32px))"
      destroy-on-close
      top="5vh"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-width="110px"
        label-position="right"
      >
        <el-form-item label="角色名稱" prop="name">
          <el-input v-model="form.name" placeholder="如：外部稽核員、車隊專員" />
        </el-form-item>

        <el-form-item label="角色說明" prop="description">
          <el-input
            v-model="form.description"
            placeholder="請輸入此角色在系統中之職掌說明"
          />
        </el-form-item>

        <!-- 權限矩陣配置區塊 -->
        <div class="perm-config-box">
          <div class="perm-config-header">
            <div class="perm-config-title">
              <el-icon><Setting /></el-icon>
              <span>模組權限矩陣配置</span>
            </div>

            <div class="perm-quick-actions">
              <el-button size="small" @click="handleBatchSet(true, false, false)">全選檢視</el-button>
              <el-button size="small" type="primary" plain @click="handleBatchSet(true, true, false)">全選編輯</el-button>
              <el-button size="small" type="info" plain @click="handleBatchSet(false, false, false)">全部清空</el-button>

              <el-select
                v-model="templateRoleKey"
                placeholder="複製既有角色..."
                size="small"
                style="width: 160px; margin-left: 8px"
                @change="handleApplyTemplate"
              >
                <el-option
                  v-for="r in roleList"
                  :key="r.id || r.key"
                  :label="r.name"
                  :value="r.id || r.key"
                />
              </el-select>
            </div>
          </div>

          <el-table
            :data="SYSTEM_MODULES"
            border
            stripe
            size="small"
            style="width: 100%"
            max-height="360px"
          >
            <el-table-column prop="categoryName" label="分類" width="170" align="center">
              <template #default="{ row }">
                <el-tag size="small" type="info">{{ row.categoryName }}</el-tag>
              </template>
            </el-table-column>

            <el-table-column prop="name" label="功能模組名稱" min-width="160" />

            <el-table-column label="顯示／檢視" width="120" align="center">
              <template #default="{ row }">
                <el-checkbox
                  v-model="formPermissions[row.id].view"
                  @change="onViewPermChange(row.id)"
                >
                  檢視
                </el-checkbox>
              </template>
            </el-table-column>

            <el-table-column label="操作／編輯" width="120" align="center">
              <template #default="{ row }">
                <el-checkbox
                  v-model="formPermissions[row.id].edit"
                  :disabled="!formPermissions[row.id].view"
                  @change="onEditPermChange(row.id)"
                >
                  編輯
                </el-checkbox>
              </template>
            </el-table-column>

            <el-table-column label="刪除" width="120" align="center">
              <template #default="{ row }">
                <el-checkbox
                  v-model="formPermissions[row.id].delete"
                  :disabled="!formPermissions[row.id].edit"
                  @change="onDeletePermChange(row.id)"
                >
                  刪除
                </el-checkbox>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-form>

      <template #footer>
        <DialogFooter :loading="submitting" @confirm="handleSubmit" @cancel="dialogVisible = false" />
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import PageHeader from '@/components/PageHeader.vue'
import DialogFooter from '@/components/DialogFooter.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import { Check, Close, Plus, Search, Setting } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { resolveErrorMessage } from '@/api/errorCodes'
import {
  listRoles,
  createRole,
  updateRole,
  deleteRole
} from '@/api/roles'
import { SYSTEM_MODULES, type SystemPermissions } from '@/types/domain'
import type { RoleDTO } from '@/types/api'

const roleList = ref<RoleDTO[]>([])
const loading = ref(false)
const searchQuery = ref('')
const selectedRole = ref<RoleDTO | null>(null)

const dialogVisible = ref(false)
const isEditing = ref(false)
const editingId = ref<string | null>(null)
const editingRoleIsSystem = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()
const templateRoleKey = ref('')

const form = reactive({
  name: '',
  description: ''
})

const formPermissions = ref<SystemPermissions>({})

const formRules = {
  name: [{ required: true, message: '請輸入角色名稱', trigger: 'blur' }]
}

function isSelectedRole(role: RoleDTO) {
  return selectedRole.value?.key === role.key || selectedRole.value?.id === role.id
}

const dialogTitle = computed(() => {
  if (isEditing.value) {
    return `編輯角色身分 - 【${form.name}】`
  }
  return '新增自訂角色身分'
})

const filteredRoles = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return roleList.value
  return roleList.value.filter(
    (r) =>
      r.name.toLowerCase().includes(q) ||
      r.key.toLowerCase().includes(q) ||
      (r.description && r.description.toLowerCase().includes(q))
  )
})

function countRolePerms(perms?: SystemPermissions) {
  if (!perms) return { views: 0, edits: 0 }
  let views = 0
  let edits = 0
  for (const m of SYSTEM_MODULES) {
    if (perms[m.id]?.view) views++
    if (perms[m.id]?.edit) edits++
  }
  return { views, edits }
}

function initEmptyPermissions(): SystemPermissions {
  const p: SystemPermissions = {}
  for (const m of SYSTEM_MODULES) {
    p[m.id] = { view: false, edit: false, delete: false }
  }
  return p
}

async function fetchRoles() {
  loading.value = true
  try {
    const list = await listRoles()
    roleList.value = list

    // 保持目前選取的角色；若無選取則預設選取清單第一個
    if (selectedRole.value) {
      const found = list.find((r) => r.key === selectedRole.value?.key)
      selectedRole.value = found || list[0] || null
    } else {
      selectedRole.value = list[0] || null
    }
  } catch (error: any) {
    ElMessage.error(resolveErrorMessage(error.response?.data?.error?.code, '載入角色清單失敗'))
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  isEditing.value = false
  editingId.value = null
  editingRoleIsSystem.value = false
  templateRoleKey.value = ''
  form.name = ''
  form.description = ''
  formPermissions.value = initEmptyPermissions()
  dialogVisible.value = true
}

function openEditDialog(role: RoleDTO) {
  isEditing.value = true
  editingId.value = role.id || role.key
  editingRoleIsSystem.value = role.isSystem
  templateRoleKey.value = ''
  form.name = role.name
  form.description = role.description || ''

  // 複製現有權限並確保 20 個模組都有鍵值
  const p = initEmptyPermissions()
  if (role.permissions) {
    for (const m of SYSTEM_MODULES) {
      if (role.permissions[m.id]) {
        p[m.id] = {
          view: !!role.permissions[m.id].view,
          edit: !!role.permissions[m.id].edit,
          delete: !!role.permissions[m.id].delete
        }
      }
    }
  }
  formPermissions.value = p
  dialogVisible.value = true
}

function openCopyDialog(role: RoleDTO) {
  isEditing.value = false
  editingId.value = null
  editingRoleIsSystem.value = false
  templateRoleKey.value = ''
  form.name = `${role.name} (複製)`
  form.description = role.description || ''

  const p = initEmptyPermissions()
  if (role.permissions) {
    for (const m of SYSTEM_MODULES) {
      if (role.permissions[m.id]) {
        p[m.id] = {
          view: !!role.permissions[m.id].view,
          edit: !!role.permissions[m.id].edit,
          delete: !!role.permissions[m.id].delete
        }
      }
    }
  }
  formPermissions.value = p
  dialogVisible.value = true
}

function handleBatchSet(viewVal: boolean, editVal: boolean, deleteVal: boolean) {
  for (const m of SYSTEM_MODULES) {
    formPermissions.value[m.id] = {
      view: viewVal,
      edit: editVal,
      delete: deleteVal
    }
  }
}

function handleApplyTemplate(roleIdOrKey: string) {
  const target = roleList.value.find((r) => r.id === roleIdOrKey || r.key === roleIdOrKey)
  if (!target || !target.permissions) return
  for (const m of SYSTEM_MODULES) {
    formPermissions.value[m.id] = {
      view: !!target.permissions[m.id]?.view,
      edit: !!target.permissions[m.id]?.edit,
      delete: !!target.permissions[m.id]?.delete
    }
  }
  ElMessage.info(`已套用【${target.name}】之權限配置範本`)
}

// 三個層級是包含關係：delete 需要 edit，edit 需要 view；勾選較高層級時往下補齊，
// 取消較低層級時往上一併取消，避免存出「能刪除但不能檢視」這種無意義組合。
function onViewPermChange(moduleId: string) {
  if (!formPermissions.value[moduleId].view) {
    formPermissions.value[moduleId].edit = false
    formPermissions.value[moduleId].delete = false
  }
}

function onEditPermChange(moduleId: string) {
  if (formPermissions.value[moduleId].edit) {
    formPermissions.value[moduleId].view = true
  } else {
    formPermissions.value[moduleId].delete = false
  }
}

function onDeletePermChange(moduleId: string) {
  if (formPermissions.value[moduleId].delete) {
    formPermissions.value[moduleId].edit = true
    formPermissions.value[moduleId].view = true
  }
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (isEditing.value && editingId.value) {
        await updateRole(editingId.value, {
          name: form.name,
          description: form.description,
          permissions: formPermissions.value
        })
        ElMessage.success(`角色「${form.name}」已成功更新`)
      } else {
        await createRole({
          name: form.name,
          description: form.description,
          permissions: formPermissions.value
        })
        ElMessage.success(`自訂角色「${form.name}」建立成功`)
      }
      dialogVisible.value = false
      await fetchRoles()
    } catch (err: any) {
      const msg = resolveErrorMessage(err.response?.data?.error?.code, '儲存角色失敗')
      ElMessage.error(msg)
    } finally {
      submitting.value = false
    }
  })
}

async function handleDeleteRole(role: RoleDTO) {
  if (role.isSystem) {
    ElMessage.warning('系統內建角色受系統保護，不可刪除')
    return
  }

  if ((role.userCount || 0) > 0) {
    ElMessageBox.alert(
      `目前尚有 ${role.userCount} 位使用者正在使用「${role.name}」角色。請先前往「使用者管理」將這些使用者調派至其他角色後，方可刪除此角色。`,
      '無法刪除角色',
      {
        type: 'warning',
        confirmButtonText: '我知道了'
      }
    )
    return
  }

  try {
    await ElMessageBox.confirm(
      `確定要刪除角色「${role.name}」嗎？此操作將同時記錄於系統操作紀錄。`,
      '確認刪除自訂角色',
      {
        confirmButtonText: '刪除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await deleteRole(role.id || role.key)
    ElMessage.success(`角色「${role.name}」已成功刪除`)
    if (selectedRole.value?.key === role.key || selectedRole.value?.id === role.id) {
      selectedRole.value = null
    }
    await fetchRoles()
  } catch (err: any) {
    if (err !== 'cancel') {
      const msg = resolveErrorMessage(err.response?.data?.error?.code, '刪除失敗')
      ElMessage.error(msg)
    }
  }
}

onMounted(() => {
  fetchRoles()
})
</script>

<style scoped>
.role-management-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.rule-alert {
  border-radius: var(--app-radius-md);
}

.toolbar-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: var(--app-surface);
  padding: 12px 16px;
  border-radius: var(--app-radius-md);
  border: 1px solid var(--app-border-light);

  .toolbar-left {
    display: flex;
    align-items: center;
    gap: 12px;

    .total-tag {
      font-size: 13px;
    }
  }
}

/* 角色清單 + 權限矩陣：兩欄版面，清單窄、矩陣寬；窄螢幕收成單欄堆疊 */
.role-layout {
  display: grid;
  grid-template-columns: minmax(280px, 380px) 1fr;
  align-items: start;
  gap: 16px;
}

.role-list-card {
  border-radius: var(--app-radius-md);

  :deep(.el-card__body) {
    padding: 0;
  }
}

.role-list {
  display: flex;
  flex-direction: column;
}

/* 清單列：差異靠字重／描述／統計數字呈現，不靠角色專屬色，
   選取狀態沿用側欄導覽同一套「左側色條 + 淡底」語言，維持全站一致 */
.role-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--app-border-light);
  cursor: pointer;
  transition: background-color 0.15s ease;

  &:last-child {
    border-bottom: none;
  }

  &:hover {
    background-color: var(--app-status-neutral-bg);
  }

  &.is-selected {
    background-color: var(--app-primary-light);
    box-shadow: inset 3px 0 0 var(--app-primary);
  }

  .role-row-heading {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .role-row-name {
    font-size: 15px;
    font-weight: 700;
    color: var(--app-text-primary);
  }

  .role-row-system-badge {
    font-size: var(--app-label-size);
    font-weight: 700;
    letter-spacing: var(--app-label-tracking);
    text-transform: uppercase;
    color: var(--app-text-secondary);
    background: var(--app-status-neutral-bg);
    border-radius: var(--app-radius-full);
    padding: 1px 8px;
  }

  .role-row-desc {
    font-size: 13px;
    color: var(--app-text-secondary);
    line-height: 1.4;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .role-row-meta {
    font-size: var(--app-font-xs);
    color: var(--app-text-muted);

    .role-row-meta-sep {
      margin: 0 4px;
    }
  }
}

.matrix-card {
  border-radius: var(--app-radius-md);

  .matrix-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
  }

  .matrix-title {
    font-size: 16px;
    font-weight: 700;
    color: var(--app-text-primary);
  }

  .matrix-header-right {
    display: flex;
    align-items: center;
    gap: 12px;
  }
}

.perm-config-box {
  margin-top: 16px;
  border: 1px solid var(--app-border-light);
  border-radius: var(--app-radius-sm);
  padding: 12px;
  background-color: var(--app-status-neutral-bg);

  .perm-config-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 10px;

    .perm-config-title {
      display: flex;
      align-items: center;
      gap: 6px;
      font-weight: 700;
      font-size: 14px;
      color: var(--app-text-primary);
    }

    .perm-quick-actions {
      display: flex;
      align-items: center;
      gap: 6px;
    }
  }
}

.matrix-stat-text {
  font-size: 13px;
  color: var(--app-text-regular);
}

.perm-status {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  font-weight: 500;

  &.perm-view {
    color: var(--app-status-success-fg);
  }

  &.perm-edit {
    color: var(--app-primary);
  }

  &.perm-delete {
    color: var(--app-status-danger-fg);
  }

  &.perm-none {
    color: var(--app-text-muted);
  }
}

@media (max-width: 900px) {
  .role-layout {
    grid-template-columns: 1fr;
  }
}
</style>

