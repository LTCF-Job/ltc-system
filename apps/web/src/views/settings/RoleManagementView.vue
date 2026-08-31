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

    <!-- 角色卡片清單 -->
    <el-row :gutter="16" class="role-cards-row">
      <el-col
        v-for="role in filteredRoles"
        :key="role.id || role.key"
        :xs="24"
        :sm="12"
        :md="8"
        :lg="6"
        style="margin-bottom: 16px"
      >
        <el-card
          shadow="hover"
          class="role-card"
          :class="{ 'is-selected': selectedRole?.key === role.key || selectedRole?.id === role.id }"
          @click="selectedRole = role"
        >
          <div class="role-card-header">
            <div class="header-tag-group">
              <span class="role-title-text" :class="`role-${role.key || role.id}`">
                <span class="role-dot" :class="`dot-${role.key || role.id}`"></span>
                {{ role.name }}
              </span>
            </div>
          </div>

          <p class="role-desc" :title="role.description">{{ role.description || '無詳細說明' }}</p>

          <div class="role-stat">
            <div class="stat-row">
              <span class="stat-label">綁定使用者：</span>
              <el-tag size="small" :type="(role.userCount || 0) > 0 ? 'success' : 'info'" effect="plain">
                {{ role.userCount || 0 }} 人
              </el-tag>
            </div>
            <div class="stat-row">
              <span class="stat-label">模組檢視／操作：</span>
              <span class="stat-val">
                <strong>{{ countRolePerms(role.permissions).views }}</strong> 檢視 /
                <strong>{{ countRolePerms(role.permissions).edits }}</strong> 編輯
              </span>
            </div>
          </div>

          <div class="role-actions" @click.stop>
            <el-button
              type="primary"
              link
              size="small"
              @click="openEditDialog(role)"
            >
              編輯設定
            </el-button>
            <el-button
              type="primary"
              link
              size="small"
              @click="openCopyDialog(role)"
            >
              複製建立
            </el-button>
            <el-tooltip
              v-if="role.isSystem"
              content="系統內建核心角色不可刪除"
              placement="top"
            >
              <span>
                <el-button
                  type="danger"
                  link
                  size="small"
                  disabled
                >
                  刪除
                </el-button>
              </span>
            </el-tooltip>
            <el-button
              v-else
              type="danger"
              link
              size="small"
              @click="handleDeleteRole(role)"
            >
              刪除
            </el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 角色模組權限矩陣檢視卡片 -->
    <el-card v-if="selectedRole" shadow="never" class="matrix-card">
      <template #header>
        <div class="matrix-header">
          <div class="matrix-title-box">
            <span class="matrix-title">
              【{{ selectedRole.name }}】功能模組權限矩陣
            </span>
          </div>
          <div class="matrix-header-right">
            <span class="matrix-stat-text">
              支援 {{ countRolePerms(selectedRole.permissions).views }} 項檢視、{{ countRolePerms(selectedRole.permissions).edits }} 項操作
            </span>
            <el-button
              type="primary"
              size="small"
              plain
              @click="openEditDialog(selectedRole)"
            >
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
        <el-table-column prop="categoryName" label="分類" width="130" align="center">
          <template #default="{ row }">
            <span>{{ row.categoryName }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="name" label="功能模組名稱" min-width="200">
          <template #default="{ row }">
            <span class="font-bold">{{ row.name }}</span>
            <span class="font-mono text-muted" style="margin-left: 6px; font-size: 12px">({{ row.id }})</span>
          </template>
        </el-table-column>

        <el-table-column label="檢視權限 (View)" width="180" align="center">
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

        <el-table-column label="編輯／操作權限 (Edit)" width="180" align="center">
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
      </el-table>
    </el-card>

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
        <el-row :gutter="16">
          <el-col :xs="24" :sm="12">
            <el-form-item label="角色名稱" prop="name">
              <el-input v-model="form.name" placeholder="如：外部稽核員、車隊專員" />
            </el-form-item>
          </el-col>

          <el-col :xs="24" :sm="12">
            <el-form-item label="標籤色彩" prop="tagType">
              <el-select v-model="form.tagType" placeholder="請選擇標籤樣式" style="width: 100%">
                <el-option
                  v-for="color in tagColorOptions"
                  :key="color.value"
                  :label="color.label"
                  :value="color.value"
                >
                  <el-tag :type="color.value" size="small" effect="dark" style="margin-right: 8px">
                    {{ color.label }}
                  </el-tag>
                  <span style="color: var(--el-text-color-secondary); font-size: 12px">{{ color.desc }}</span>
                </el-option>
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

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
              <el-button size="small" @click="handleBatchSet(true, false)">全選檢視</el-button>
              <el-button size="small" type="primary" plain @click="handleBatchSet(true, true)">全選編輯</el-button>
              <el-button size="small" type="info" plain @click="handleBatchSet(false, false)">全部清空</el-button>

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
            <el-table-column prop="categoryName" label="分類" width="110" align="center">
              <template #default="{ row }">
                <el-tag size="small" effect="plain" type="info">{{ row.categoryName }}</el-tag>
              </template>
            </el-table-column>

            <el-table-column prop="name" label="功能區塊模組" min-width="160" />

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
import { Check, Close, Plus, Search, Setting } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { resolveErrorMessage } from '@/api/errorCodes'
import {
  listRoles,
  createRole,
  updateRole,
  deleteRole
} from '@/api/roles'
import {
  SYSTEM_MODULES,
  DEFAULT_ROLE_PERMISSIONS,
  type RoleTagType,
  type RoleItem,
  type SystemPermissions
} from '@/types/domain'
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

const tagColorOptions: { label: string; value: RoleTagType; desc: string }[] = [
  { label: '藍色 (Primary)', value: 'primary', desc: '適用於核心調度與專員' },
  { label: '綠色 (Success)', value: 'success', desc: '適用於司機與執行人員' },
  { label: '橘色 (Warning)', value: 'warning', desc: '適用於外部稽核與臨時身分' },
  { label: '紅色 (Danger)', value: 'danger', desc: '適用於管理人員與高權限' },
  { label: '灰色 (Info)', value: 'info', desc: '適用於唯讀檢視者' }
]

const form = reactive({
  name: '',
  description: '',
  tagType: 'primary' as RoleTagType
})

const formPermissions = ref<SystemPermissions>({})

const formRules = {
  name: [{ required: true, message: '請輸入角色名稱', trigger: 'blur' }],
  tagType: [{ required: true, message: '請選擇標籤色彩', trigger: 'change' }]
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
    p[m.id] = { view: false, edit: false }
  }
  return p
}

async function fetchRoles() {
  loading.value = true
  try {
    const list = await listRoles()
    roleList.value = list

    // 保持目前選取的角色；若無選取則預設選取第一個 (或 dispatcher)
    if (selectedRole.value) {
      const found = list.find((r) => r.key === selectedRole.value?.key)
      selectedRole.value = found || list[0] || null
    } else {
      const defaultItem = list.find((r) => r.key === 'dispatcher') || list[0] || null
      selectedRole.value = defaultItem
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
  form.tagType = 'primary'
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
  form.tagType = role.tagType || 'primary'

  // 複製現有權限並確保 20 個模組都有鍵值
  const p = initEmptyPermissions()
  if (role.permissions) {
    for (const m of SYSTEM_MODULES) {
      if (role.permissions[m.id]) {
        p[m.id] = {
          view: !!role.permissions[m.id].view,
          edit: !!role.permissions[m.id].edit
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
  form.tagType = role.tagType || 'primary'

  const p = initEmptyPermissions()
  if (role.permissions) {
    for (const m of SYSTEM_MODULES) {
      if (role.permissions[m.id]) {
        p[m.id] = {
          view: !!role.permissions[m.id].view,
          edit: !!role.permissions[m.id].edit
        }
      }
    }
  }
  formPermissions.value = p
  dialogVisible.value = true
}

function handleBatchSet(viewVal: boolean, editVal: boolean) {
  for (const m of SYSTEM_MODULES) {
    formPermissions.value[m.id] = {
      view: viewVal,
      edit: editVal
    }
  }
}

function handleApplyTemplate(roleIdOrKey: string) {
  const target = roleList.value.find((r) => r.id === roleIdOrKey || r.key === roleIdOrKey)
  if (!target || !target.permissions) return
  for (const m of SYSTEM_MODULES) {
    formPermissions.value[m.id] = {
      view: !!target.permissions[m.id]?.view,
      edit: !!target.permissions[m.id]?.edit
    }
  }
  ElMessage.info(`已套用【${target.name}】之權限配置範本`)
}

function onViewPermChange(moduleId: string) {
  if (!formPermissions.value[moduleId].view) {
    formPermissions.value[moduleId].edit = false
  }
}

function onEditPermChange(moduleId: string) {
  if (formPermissions.value[moduleId].edit) {
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
          tagType: form.tagType,
          permissions: formPermissions.value
        })
        ElMessage.success(`角色「${form.name}」已成功更新`)
      } else {
        await createRole({
          name: form.name,
          description: form.description,
          tagType: form.tagType,
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
  border-radius: 8px;
}

.toolbar-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: var(--el-bg-color);
  padding: 12px 16px;
  border-radius: 8px;
  border: 1px solid var(--el-border-color-light);

  .toolbar-left {
    display: flex;
    align-items: center;
    gap: 12px;

    .total-tag {
      font-size: 13px;
    }
  }
}

.font-bold {
  font-weight: bold;
}

.font-mono {
  font-family: 'Consolas', 'Courier New', monospace;
}

.text-muted {
  color: var(--el-text-color-secondary);
}

.role-cards-row {
  margin-bottom: 8px;
}

.role-card {
  cursor: pointer;
  border-radius: 8px;
  transition: all 0.2s ease;
  height: 100%;
  display: flex;
  flex-direction: column;

  &:hover {
    transform: translateY(-2px);
    box-shadow: var(--el-box-shadow-light);
  }

  &.is-selected {
    border-color: var(--el-color-primary);
    background-color: var(--el-color-primary-light-9);
  }

  .role-card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;

    .header-tag-group {
      display: flex;
      align-items: center;
      gap: 6px;

      .role-title-text {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        font-size: 15px;
        font-weight: 600;

        &.role-admin { color: var(--app-role-admin-fg); }
        &.role-dispatcher { color: var(--app-role-dispatcher-fg); }
        &.role-staff { color: var(--app-role-staff-fg); }
        &.role-driver { color: var(--app-role-driver-fg); }
        &.role-viewer { color: var(--app-role-viewer-fg); }
      }

      .role-dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        display: inline-block;

        &.dot-admin { background-color: var(--app-role-admin-dot); }
        &.dot-dispatcher { background-color: var(--app-role-dispatcher-dot); }
        &.dot-staff { background-color: var(--app-role-staff-dot); }
        &.dot-driver { background-color: var(--app-role-driver-dot); }
        &.dot-viewer { background-color: var(--app-role-viewer-dot); }
      }
    }

    .role-key {
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .role-desc {
    font-size: 13px;
    color: var(--el-text-color-regular);
    line-height: 1.4;
    height: 40px;
    margin-bottom: 12px;
    overflow: hidden;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }

  .role-stat {
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    border-top: 1px dashed var(--el-border-color-lighter);
    padding-top: 8px;
    margin-bottom: 10px;

    .stat-row {
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .stat-val strong {
      color: var(--el-color-primary);
    }
  }

  .role-actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-top: 1px solid var(--el-border-color-lighter);
    padding-top: 8px;
    margin-top: auto;
  }
}

.matrix-card {
  border-radius: 8px;

  .matrix-header {
    display: flex;
    justify-content: space-between;
    align-items: center;

    .matrix-title-box {
      display: flex;
      align-items: center;
    }

    .matrix-title {
      font-size: 16px;
      font-weight: bold;
      color: var(--el-color-primary);
    }

    .matrix-subtitle {
      font-size: 13px;
      color: var(--el-text-color-secondary);
      margin-left: 6px;
    }

    .matrix-header-right {
      display: flex;
      align-items: center;
      gap: 12px;
    }
  }
}

.perm-config-box {
  margin-top: 16px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  padding: 12px;
  background-color: var(--el-fill-color-lighter);

  .perm-config-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 10px;

    .perm-config-title {
      display: flex;
      align-items: center;
      gap: 6px;
      font-weight: bold;
      font-size: 14px;
      color: var(--el-text-color-primary);
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
  color: var(--el-text-color-regular);
}

.perm-status {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  font-weight: 500;

  &.perm-view {
    color: var(--el-color-success);
  }

  &.perm-edit {
    color: var(--el-color-primary);
  }

  &.perm-none {
    color: var(--el-text-color-placeholder);
  }
}
</style>

