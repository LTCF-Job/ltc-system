<template>
  <el-container class="layout-container">
    <!-- 側邊選單欄 -->
    <el-aside :width="isCollapse ? '64px' : '220px'" class="aside-menu">
      <div class="logo-box">
        <el-icon class="logo-icon"><Van /></el-icon>
        <span v-if="!isCollapse" class="logo-title">長照接送管理</span>
      </div>

      <el-menu
        :default-active="activeRoute"
        :collapse="isCollapse"
        router
        class="el-menu-vertical"
        background-color="#1d5b79"
        text-color="#cde1ec"
        active-text-color="#ffffff"
      >
        <el-menu-item index="/">
          <el-icon><Odometer /></el-icon>
          <template #title>總覽儀表板</template>
        </el-menu-item>

        <el-menu-item index="/cases">
          <el-icon><User /></el-icon>
          <template #title>個案管理</template>
        </el-menu-item>

        <el-sub-menu index="masters">
          <template #title>
            <el-icon><Management /></el-icon>
            <span>主檔資料</span>
          </template>
          <el-menu-item index="/masters/sites">
            <el-icon><Location /></el-icon>
            <template #title>據點管理</template>
          </el-menu-item>
          <el-menu-item index="/masters/vehicles">
            <el-icon><Van /></el-icon>
            <template #title>車輛管理</template>
          </el-menu-item>
          <el-menu-item index="/masters/drivers">
            <el-icon><Avatar /></el-icon>
            <template #title>司機管理</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="forms">
          <template #title>
            <el-icon><DocumentCopy /></el-icon>
            <span>表單管理</span>
          </template>
          <el-menu-item index="/forms">
            <el-icon><Refresh /></el-icon>
            <template #title>表單同步狀態</template>
          </el-menu-item>
          <el-menu-item index="/forms/mappings">
            <el-icon><Connection /></el-icon>
            <template #title>欄位對應設定</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="rides">
          <template #title>
            <el-icon><Calendar /></el-icon>
            <span>搭乘紀錄</span>
          </template>
          <el-menu-item index="/rides">
            <el-icon><Grid /></el-icon>
            <template #title>搭乘月曆矩陣</template>
          </el-menu-item>
          <el-menu-item index="/rides/issues">
            <el-icon><Warning /></el-icon>
            <template #title>異常集中處理</template>
          </el-menu-item>
          <el-menu-item index="/rides/missing">
            <el-icon><Bell /></el-icon>
            <template #title>未回報清單</template>
          </el-menu-item>
        </el-sub-menu>

        <el-sub-menu index="reports">
          <template #title>
            <el-icon><DataAnalysis /></el-icon>
            <span>報表管理</span>
          </template>
          <el-menu-item index="/reports/trip-summary">
            <el-icon><List /></el-icon>
            <template #title>車輛趟數表</template>
          </el-menu-item>
        </el-sub-menu>

        <el-menu-item index="/exports">
          <el-icon><Download /></el-icon>
          <template #title>政府申報匯出</template>
        </el-menu-item>

        <el-menu-item v-if="authStore.can('admin')" index="/audit">
          <el-icon><DocumentCopy /></el-icon>
          <template #title>系統稽核日誌</template>
        </el-menu-item>

        <el-sub-menu index="settings">
          <template #title>
            <el-icon><Setting /></el-icon>
            <span>系統設定</span>
          </template>
          <el-menu-item index="/settings/notifications">
            <el-icon><Bell /></el-icon>
            <template #title>通知收件人</template>
          </el-menu-item>
        </el-sub-menu>
      </el-menu>
    </el-aside>


    <!-- 右側主體內容 -->
    <el-container>
      <!-- 頂部導覽列 -->
      <el-header class="layout-header">
        <div class="header-left">
          <el-button
            link
            class="toggle-btn"
            @click="isCollapse = !isCollapse"
          >
            <el-icon :size="20">
              <Fold v-if="!isCollapse" />
              <Expand v-else />
            </el-icon>
          </el-button>
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/' }">首頁</el-breadcrumb-item>
            <el-breadcrumb-item v-if="currentRouteTitle">
              {{ currentRouteTitle }}
            </el-breadcrumb-item>
          </el-breadcrumb>
        </div>

        <div class="header-right">
          <el-tag
            :type="authStore.currentRole === 'admin' ? 'danger' : (authStore.currentRole === 'staff' ? 'primary' : 'info')"
            effect="dark"
            size="small"
          >
            {{ ROLE_LABELS[authStore.currentRole] }}
          </el-tag>
          <el-dropdown trigger="click" @command="handleCommand">
            <span class="user-dropdown-link">
              <el-avatar :size="28" icon="UserFilled" />
              <span class="user-name">{{ authStore.user?.displayName || '使用者' }}</span>
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile" disabled>
                  帳號：{{ authStore.user?.email }}
                </el-dropdown-item>
                <el-dropdown-item divided command="logout">
                  登出系統
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <!-- 主要頁面檢視區 -->
      <el-main class="layout-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Van,
  Odometer,
  User,
  Management,
  Location,
  Avatar,
  DocumentCopy,
  Refresh,
  Connection,
  Calendar,
  Grid,
  Warning,
  Download,
  Fold,
  Expand,
  ArrowDown,
  Bell,
  DataAnalysis,
  List,
  Setting
} from '@element-plus/icons-vue'

import { useAuthStore } from '@/stores/auth'
import { ROLE_LABELS } from '@/types/domain'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const isCollapse = ref(false)

const activeRoute = computed(() => {
  const path = route.path
  if (path.startsWith('/cases/')) return '/cases'
  return path
})

const currentRouteTitle = computed(() => {
  return (route.meta.title as string) || ''
})

function handleCommand(cmd: string) {
  if (cmd === 'logout') {
    authStore.logout()
    router.push('/login')
  }
}
</script>

<style scoped>
.layout-container {
  min-height: 100vh;
}

.aside-menu {
  background-color: #1d5b79;
  transition: width 0.3s;
  overflow-x: hidden;

  .logo-box {
    height: 60px;
    display: flex;
    align-items: center;
    padding: 0 16px;
    background-color: #143d52;
    color: #ffffff;
    gap: 12px;

    .logo-icon {
      font-size: 24px;
      color: #8bc34a;
    }

    .logo-title {
      font-size: 16px;
      font-weight: bold;
      white-space: nowrap;
    }
  }

  .el-menu-vertical {
    border-right: none;
  }
}

.layout-header {
  height: 60px;
  background-color: #ffffff;
  border-bottom: 1px solid var(--el-border-color-lighter);
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 20px;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.05);

  .header-left {
    display: flex;
    align-items: center;
    gap: 16px;

    .toggle-btn {
      color: var(--el-text-color-primary);
    }
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 16px;

    .user-dropdown-link {
      display: flex;
      align-items: center;
      gap: 8px;
      cursor: pointer;
      font-size: 14px;
      color: var(--el-text-color-primary);
    }
  }
}

.layout-main {
  background-color: #f5f7fa;
  padding: 20px;
  min-height: calc(100vh - 60px);
}
</style>
