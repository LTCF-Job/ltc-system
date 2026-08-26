<template>
  <div class="role-management-view">
    <el-alert
      type="info"
      show-icon
      :closable="false"
      class="rule-alert"
    >
      <template #title>
        <span class="font-bold">權限套用規則：個人設定 ＞ 角色身分設定</span>
      </template>
      系統以使用者為中心進行權限裁決。若使用者擁有個人自訂權限，則優先套用個人設定；若無個人設定，系統自動繼承並套用下方定義之角色預設權限配置。
    </el-alert>

    <!-- 角色卡片清單 -->
    <el-row :gutter="16" class="role-cards-row">
      <el-col
        v-for="role in roleList"
        :key="role.key"
        :xs="24"
        :sm="12"
        :md="6"
      >
        <el-card
          shadow="hover"
          class="role-card"
          :class="{ 'is-selected': selectedRoleKey === role.key }"
          @click="selectedRoleKey = role.key"
        >
          <div class="role-card-header">
            <el-tag :type="role.tagType" effect="dark" size="default">
              {{ role.label }}
            </el-tag>
            <span class="role-key font-mono">{{ role.key }}</span>
          </div>
          <p class="role-desc">{{ role.description }}</p>
          <div class="role-stat">
            <span class="stat-item">
              可檢視模組：<strong>{{ countRolePerms(role.key).views }}</strong> / {{ SYSTEM_MODULES.length }}
            </span>
            <span class="stat-item">
              具編輯權限：<strong>{{ countRolePerms(role.key).edits }}</strong> / {{ SYSTEM_MODULES.length }}
            </span>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 角色權限矩陣清單 -->
    <el-card shadow="never" class="matrix-card">
      <template #header>
        <div class="matrix-header">
          <div>
            <span class="matrix-title">
              【{{ ROLE_LABELS[selectedRoleKey] }}】預設功能模組權限矩陣
            </span>
            <span class="matrix-subtitle font-mono">({{ selectedRoleKey }})</span>
          </div>
          <el-tag :type="getRoleTagType(selectedRoleKey)" effect="plain">
            共支援 {{ countRolePerms(selectedRoleKey).views }} 項檢視、{{ countRolePerms(selectedRoleKey).edits }} 項操作
          </el-tag>
        </div>
      </template>

      <el-table
        :data="SYSTEM_MODULES"
        border
        stripe
        size="small"
        style="width: 100%"
      >
        <el-table-column prop="categoryName" label="分類" width="120" align="center">
          <template #default="{ row }">
            <el-tag size="small" effect="plain" type="info">{{ row.categoryName }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="name" label="功能模組名稱" min-width="180">
          <template #default="{ row }">
            <span class="font-bold">{{ row.name }}</span>
            <span class="font-mono text-muted" style="margin-left: 6px; font-size: 12px">({{ row.id }})</span>
          </template>
        </el-table-column>

        <el-table-column label="檢視權限 (View)" width="160" align="center">
          <template #default="{ row }">
            <el-tag
              v-if="DEFAULT_ROLE_PERMISSIONS[selectedRoleKey]?.[row.id]?.view"
              type="success"
              size="small"
              effect="dark"
            >
              <el-icon><Check /></el-icon> 允許檢視
            </el-tag>
            <el-tag v-else type="info" size="small" effect="plain">
              <el-icon><Close /></el-icon> 無權限
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="編輯權限 (Edit)" width="160" align="center">
          <template #default="{ row }">
            <el-tag
              v-if="DEFAULT_ROLE_PERMISSIONS[selectedRoleKey]?.[row.id]?.edit"
              type="primary"
              size="small"
              effect="dark"
            >
              <el-icon><Check /></el-icon> 允許新增/修改
            </el-tag>
            <el-tag v-else type="info" size="small" effect="plain">
              <el-icon><Close /></el-icon> 僅讀/無權限
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Check, Close } from '@element-plus/icons-vue'
import {
  ROLE_LABELS,
  SYSTEM_MODULES,
  DEFAULT_ROLE_PERMISSIONS,
  type UserRole
} from '@/types/domain'

const selectedRoleKey = ref<UserRole>('dispatcher')

interface RoleInfoItem {
  key: UserRole
  label: string
  tagType: 'danger' | 'primary' | 'success' | 'warning' | 'info'
  description: string
}

const roleList: RoleInfoItem[] = [
  {
    key: 'admin',
    label: '系統管理員',
    tagType: 'danger',
    description: '具備全系統最高權限，可管理使用者帳號、角色、稽核紀錄與所有主檔及申報功能。'
  },
  {
    key: 'dispatcher',
    label: '調度員',
    tagType: 'primary',
    description: '負責日常派車、個案管理、搭乘月曆排程、異常處理、表單同步與申報資料匯出。'
  },
  {
    key: 'driver',
    label: '司機',
    tagType: 'success',
    description: '負責每日出勤登錄、車輛維修紀錄填寫與個人接送趟次狀況檢視。'
  },
  {
    key: 'viewer',
    label: '檢視者',
    tagType: 'info',
    description: '僅具備全系統營運資料之唯讀檢視權限，無法進行任何新增、修改或刪除操作。'
  }
]

function getRoleTagType(role: UserRole): 'danger' | 'primary' | 'success' | 'warning' | 'info' {
  switch (role) {
    case 'admin':
      return 'danger'
    case 'dispatcher':
    case 'staff':
      return 'primary'
    case 'driver':
      return 'success'
    case 'viewer':
    default:
      return 'info'
  }
}

function countRolePerms(roleKey: UserRole) {
  const perms = DEFAULT_ROLE_PERMISSIONS[roleKey] || {}
  let views = 0
  let edits = 0
  for (const m of SYSTEM_MODULES) {
    if (perms[m.id]?.view) views++
    if (perms[m.id]?.edit) edits++
  }
  return { views, edits }
}
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

  &:hover {
    transform: translateY(-2px);
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

    .role-key {
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .role-desc {
    font-size: 13px;
    color: var(--el-text-color-regular);
    line-height: 1.4;
    min-height: 52px;
    margin-bottom: 12px;
  }

  .role-stat {
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    border-top: 1px dashed var(--el-border-color-lighter);
    padding-top: 8px;

    strong {
      color: var(--el-color-primary);
    }
  }
}

.matrix-card {
  border-radius: 8px;

  .matrix-header {
    display: flex;
    justify-content: space-between;
    align-items: center;

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
  }
}
</style>
