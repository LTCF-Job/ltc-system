<template>
  <a class="skip-link" href="#main-content">跳至主要內容</a>
  <el-container class="layout-container">
    <div
      v-if="isMobile && isMobileMenuOpen"
      class="navigation-backdrop"
      aria-hidden="true"
      @click="isMobileMenuOpen = false"
    />
    <!-- 側邊選單欄 -->
    <el-aside
      :width="isMobile ? expandedAsideWidth : (isCollapse ? '68px' : expandedAsideWidth)"
      class="aside-menu"
      :class="{ 'is-mobile': isMobile, 'is-mobile-open': isMobileMenuOpen }"
    >
      <el-scrollbar class="menu-scrollbar">
        <el-menu
          id="primary-navigation"
          :default-active="activeRoute"
            :collapse="!isMobile && isCollapse"
          router
          class="el-menu-vertical"
          background-color="#19324d"
          text-color="#d2deea"
          active-text-color="#ffffff"
        >
          <el-menu-item index="/">
            <el-icon><Odometer /></el-icon>
            <template #title>總覽儀表板</template>
          </el-menu-item>

          <el-sub-menu v-if="authStore.hasPermission('rides_calendar') || authStore.hasPermission('rides_issues') || authStore.hasPermission('rides_missing')" index="rides">
            <template #title>
              <el-icon><Calendar /></el-icon>
              <span>搭乘紀錄</span>
            </template>
            <el-menu-item v-if="authStore.hasPermission('rides_calendar')" index="/rides">
              <el-icon><Grid /></el-icon>
              <template #title>搭乘月曆表</template>
            </el-menu-item>
            <el-menu-item v-if="authStore.hasPermission('rides_issues')" index="/rides/issues">
              <el-icon><Warning /></el-icon>
              <template #title>異常集中處理</template>
            </el-menu-item>
            <el-menu-item v-if="authStore.hasPermission('rides_missing')" index="/rides/missing">
              <el-icon><Bell /></el-icon>
              <template #title>未回報清單</template>
            </el-menu-item>
          </el-sub-menu>

          <el-menu-item v-if="authStore.hasPermission('attendance_fuel')" index="/attendance">
            <el-icon><Calendar /></el-icon>
            <template #title>出勤與油資管理</template>
          </el-menu-item>

          <el-menu-item v-if="authStore.hasPermission('vehicles_maintenance')" index="/vehicles/maintenance">
            <el-icon><Management /></el-icon>
            <template #title>車輛維修保養</template>
          </el-menu-item>

          <el-sub-menu v-if="authStore.hasPermission('masters_regions') || authStore.hasPermission('masters_cases') || authStore.hasPermission('masters_sites') || authStore.hasPermission('masters_vehicles') || authStore.hasPermission('masters_drivers') || authStore.hasPermission('masters_caregivers')" index="masters">
            <template #title>
              <el-icon><Management /></el-icon>
              <span>主檔資料</span>
            </template>
            <el-menu-item v-if="authStore.hasPermission('masters_regions')" index="/masters/regions">
              <el-icon><MapLocation /></el-icon>
              <template #title>地區管理</template>
            </el-menu-item>
            <el-menu-item v-if="authStore.hasPermission('masters_cases')" index="/cases">
              <el-icon><User /></el-icon>
              <template #title>個案管理</template>
            </el-menu-item>
            <el-menu-item v-if="authStore.hasPermission('masters_sites')" index="/masters/sites">
              <el-icon><Location /></el-icon>
              <template #title>據點管理</template>
            </el-menu-item>
            <el-menu-item v-if="authStore.hasPermission('masters_vehicles')" index="/masters/vehicles">
              <el-icon><Van /></el-icon>
              <template #title>車輛管理</template>
            </el-menu-item>
            <el-menu-item v-if="authStore.hasPermission('masters_drivers')" index="/masters/drivers">
              <el-icon><Avatar /></el-icon>
              <template #title>司機管理</template>
            </el-menu-item>
            <el-menu-item v-if="authStore.hasPermission('masters_caregivers')" index="/masters/caregivers">
              <el-icon><UserFilled /></el-icon>
              <template #title>照護人員管理</template>
            </el-menu-item>
          </el-sub-menu>

          <el-sub-menu v-if="authStore.hasPermission('driver_reports') || authStore.hasPermission('driver_report_mappings')" index="driver-reports">
            <template #title>
              <el-icon><DocumentCopy /></el-icon>
              <span>司機接送匯報</span>
            </template>
            <el-menu-item v-if="authStore.hasPermission('driver_reports')" index="/driver-reports">
              <el-icon><Upload /></el-icon>
              <template #title>匯報表管理</template>
            </el-menu-item>
            <el-menu-item v-if="authStore.hasPermission('driver_report_mappings')" index="/driver-reports/mappings">
              <el-icon><Connection /></el-icon>
              <template #title>欄位對應設定</template>
            </el-menu-item>
          </el-sub-menu>

          <el-sub-menu v-if="authStore.hasPermission('reports_trip_summary') || authStore.hasPermission('reports_hsinchu_schedule')" index="reports">
            <template #title>
              <el-icon><DataAnalysis /></el-icon>
              <span>報表管理</span>
            </template>
            <el-menu-item v-if="authStore.hasPermission('reports_trip_summary')" index="/reports/trip-summary">
              <el-icon><List /></el-icon>
              <template #title>車輛趟數表</template>
            </el-menu-item>
            <el-menu-item v-if="authStore.hasPermission('reports_hsinchu_schedule')" index="/reports/hsinchu-schedule">
              <el-icon><DocumentCopy /></el-icon>
              <template #title>新竹接送時刻表</template>
            </el-menu-item>
          </el-sub-menu>

          <el-menu-item v-if="authStore.hasPermission('exports')" index="/exports">
            <el-icon><Download /></el-icon>
            <template #title>政府申報匯出</template>
          </el-menu-item>

          <el-menu-item v-if="authStore.hasPermission('audit_logs')" index="/audit">
            <el-icon><DocumentCopy /></el-icon>
            <template #title>系統操作紀錄</template>
          </el-menu-item>

          <el-sub-menu v-if="authStore.hasPermission('settings_users') || authStore.hasPermission('settings_roles') || authStore.hasPermission('settings_notifications')" index="settings">
            <template #title>
              <el-icon><Setting /></el-icon>
              <span>系統設定</span>
            </template>
            <el-menu-item v-if="authStore.hasPermission('settings_users')" index="/settings/users">
              <el-icon><User /></el-icon>
              <template #title>使用者管理</template>
            </el-menu-item>
            <el-menu-item v-if="authStore.hasPermission('settings_roles')" index="/settings/roles">
              <el-icon><Avatar /></el-icon>
              <template #title>角色身分管理</template>
            </el-menu-item>
            <el-menu-item v-if="authStore.hasPermission('settings_notifications')" index="/settings/notifications">
              <el-icon><Bell /></el-icon>
              <template #title>通知收件人</template>
            </el-menu-item>
            <el-menu-item v-if="authStore.hasPermission('settings_notifications')" index="/settings/holidays">
              <el-icon><Calendar /></el-icon>
              <template #title>政府假日與工作日設定</template>
            </el-menu-item>
          </el-sub-menu>
        </el-menu>
      </el-scrollbar>

    </el-aside>

    <!-- 右側主體內容 -->
    <el-container>
      <!-- 頂部導覽列 -->
      <el-header class="layout-header">
        <div class="header-left">
          <el-button
            link
            class="toggle-btn"
            :aria-label="isNavigationOpen ? '收合側邊導覽' : '展開側邊導覽'"
            :aria-expanded="isNavigationOpen"
            aria-controls="primary-navigation"
            @click="toggleNavigation"
          >
            <el-icon :size="18">
              <Fold v-if="isNavigationOpen" />
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
          <el-dropdown trigger="click" @command="handleCommand">
            <div class="user-dropdown-link">
              <div class="avatar-box">
                <el-avatar :size="30" icon="UserFilled" class="user-avatar" />
                <span class="avatar-online-dot"></span>
              </div>
              <span class="user-name">{{ authStore.user?.displayName || '使用者' }}</span>
              <el-icon class="dropdown-arrow"><ArrowDown /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu class="user-popover-menu">
                <el-dropdown-item command="profile" disabled>
                  帳號：{{ authStore.user?.email }}
                </el-dropdown-item>
                <el-dropdown-item command="change-password">
                  <el-icon><Lock /></el-icon>
                  修改個人密碼
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
      <el-main id="main-content" class="layout-main" tabindex="-1" aria-label="主要內容">
        <Transition name="page" mode="out-in">
          <router-view />
        </Transition>
      </el-main>
    </el-container>

    <!-- 修改個人密碼彈窗 -->
    <ChangePasswordDialog ref="changePasswordDialogRef" />
  </el-container>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Van,
  Odometer,
  User,
  Management,
  Location,
  Avatar,
  DocumentCopy,
  Upload,
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
  Setting,
  MapLocation,
  Lock,
  UserFilled
} from '@element-plus/icons-vue'

import { useAuthStore } from '@/stores/auth'
import ChangePasswordDialog from '@/components/ChangePasswordDialog.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const isCollapse = ref(false)
const isMobile = ref(false)
const isMobileMenuOpen = ref(false)
const changePasswordDialogRef = ref<InstanceType<typeof ChangePasswordDialog>>()

const menuLabels = [
  '總覽儀表板',
  '搭乘紀錄',
  '搭乘月曆表',
  '異常集中處理',
  '未回報清單',
  '出勤與油資管理',
  '車輛維修保養',
  '主檔資料',
  '地區管理',
  '個案管理',
  '據點管理',
  '車輛管理',
  '司機管理',
  '司機接送匯報',
  '匯報表管理',
  '欄位對應設定',
  '報表管理',
  '車輛趟數表',
  '新竹接送時刻表',
  '政府申報匯出',
  '系統操作紀錄',
  '系統設定',
  '使用者管理',
  '角色身分管理',
  '通知收件人',
  '政府假日與工作日設定'
]

const maxMenuLabelWidth = Math.max(...menuLabels.map((label) => {
  return Array.from(label).reduce((width, character) => {
    return width + (/^[\u0000-\u00ff]$/.test(character) ? 7 : 13)
  }, 0)
}))

const expandedAsideWidth = computed(() => `${Math.max(184, maxMenuLabelWidth + 88)}px`)

const activeRoute = computed(() => {
  const path = route.path
  if (path.startsWith('/cases/')) return '/cases'
  return path
})

const isNavigationOpen = computed(() => isMobile.value ? isMobileMenuOpen.value : !isCollapse.value)

function updateViewportMode() {
  isMobile.value = window.matchMedia('(max-width: 640px)').matches
  if (!isMobile.value) isMobileMenuOpen.value = false
}

function toggleNavigation() {
  if (isMobile.value) {
    isMobileMenuOpen.value = !isMobileMenuOpen.value
    return
  }
  isCollapse.value = !isCollapse.value
}

watch(() => route.fullPath, () => {
  if (isMobile.value) isMobileMenuOpen.value = false
})

onMounted(() => {
  updateViewportMode()
  window.addEventListener('resize', updateViewportMode)
})

onUnmounted(() => {
  window.removeEventListener('resize', updateViewportMode)
})

const currentRouteTitle = computed(() => {
  return (route.meta.title as string) || ''
})

async function handleCommand(cmd: string) {
  if (cmd === 'logout') {
    // 等待展示模式清理（停用 mock、重置展示資料）完成後再導頁，避免與下一次登入競態
    await authStore.logout()
    router.push('/login')
  } else if (cmd === 'change-password') {
    changePasswordDialogRef.value?.open()
  }
}
</script>

<style scoped>
.skip-link {
  position: fixed;
  top: 8px;
  left: 8px;
  z-index: 1000;
  padding: 8px 12px;
  border-radius: 8px;
  background: var(--app-surface);
  color: var(--app-text-primary);
  box-shadow: var(--app-shadow-md);
  transform: translateY(-160%);
  transition: transform 0.18s ease-out;
}

.skip-link:focus-visible {
  transform: translateY(0);
  outline: 3px solid var(--app-orange);
  outline-offset: 2px;
}

.layout-container { height: 100vh; min-height: 0; background: var(--app-bg); overflow: hidden; }

.aside-menu {
  background: #19324d;
  border-right: 1px solid #294766;
  height: 100vh;
  min-height: 100vh;
  transition: width 0.25s ease;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  flex: 0 0 auto;

  .menu-scrollbar {
    flex: 1;
    min-height: 0;
    height: 100%;

    :deep(.el-scrollbar__wrap) { overflow-x: hidden; }
    :deep(.el-scrollbar__view) { min-height: 100%; }
  }

  .el-menu-vertical {
    background-color: #19324d !important;
    border-right: none;
    padding: 6px 8px;

    :deep(.el-menu-item),
    :deep(.el-sub-menu__title) {
      position: relative;
      width: 100%;
      min-width: 0;
      box-sizing: border-box;
      overflow: hidden;
      border-radius: 9px;
      margin: 2px 0;
      height: 40px;
      line-height: 40px;
      font-size: 13px;
      font-weight: 500;
      color: #d2deea;
      transition: transform 0.18s ease-out, background-color 0.18s ease-out, color 0.18s ease-out;

      &:hover {
        background-color: #244d71 !important;
        color: #ffffff !important;
        transform: translateX(2px);
      }

      .el-icon {
        flex: 0 0 auto;
        font-size: 17px;
        margin-right: 8px;
        color: #9ab0c4;
      }
    }

    :deep(.el-menu-item > span),
    :deep(.el-sub-menu__title > span:not(.el-sub-menu__icon-arrow)) {
      flex: 0 0 auto;
      white-space: nowrap;
    }

    :deep(.el-menu-item.is-active) {
      background: #2d6593 !important;
      color: #ffffff !important;
      font-weight: 600;

      .el-icon {
        color: #ffffff !important;
      }

      &::before {
        content: '';
        position: absolute;
        left: 0;
        top: 8px;
        bottom: 8px;
        width: 3px;
        border-radius: 0 4px 4px 0;
        background-color: #79c7e8;
        animation: nav-indicator-in 0.2s ease-out;
      }
    }

    :deep(.el-sub-menu .el-menu) {
      background-color: #19324d !important;
      padding: 4px 0 4px 8px;
      border-radius: 8px;
    }
  }

}

.navigation-backdrop {
  display: none;
}

.layout-header {
  height: 56px;
  background-color: var(--app-surface);
  border-bottom: 1px solid var(--app-border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 22px;
  box-shadow: var(--app-shadow-sm);

  .header-left {
    display: flex;
    align-items: center;
    gap: 16px;

    .toggle-btn {
      color: var(--app-text-secondary);
      border-radius: 6px;
      padding: 6px;

      &:hover {
        background-color: #fff8f4;
        color: var(--app-text-primary);
      }
    }
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 14px;

    .user-dropdown-link {
      display: flex;
      align-items: center;
      gap: 8px;
      cursor: pointer;
      font-size: 13.5px;
      font-weight: 500;
      color: var(--app-text-primary);
      padding: 4px 8px;
      border-radius: 8px;
      transition: background-color 0.15s ease;

      &:hover {
        background-color: #fff8f4;
        transform: translateY(-1px);
      }

      .avatar-box {
        position: relative;
        display: flex;
        align-items: center;

        .user-avatar {
          background-color: var(--app-orange);
        }

        .avatar-online-dot {
          position: absolute;
          bottom: -1px;
          right: -1px;
          width: 8px;
          height: 8px;
          border-radius: 50%;
          background-color: #16c889;
          border: 1.5px solid #ffffff;
        }
      }

      .user-name {
        max-width: 100px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .dropdown-arrow {
        font-size: 12px;
        color: var(--app-text-muted);
      }
    }
  }
}

.layout-main {
  background-color: var(--app-bg);
  padding: 24px;
  min-height: calc(100vh - 56px);
  min-width: 0;
  height: calc(100vh - 56px);
  overflow: auto;
}

.layout-container > .el-container { min-width: 0; min-height: 0; }

.page-enter-active, .page-leave-active { transition: opacity 0.18s ease-out, transform 0.18s ease-out; }
.page-enter-from { opacity: 0; transform: translateY(5px); }
.page-leave-to { opacity: 0; transform: translateY(-3px); }

@keyframes nav-indicator-in {
  from { opacity: 0; transform: scaleY(0.5); }
  to { opacity: 1; transform: scaleY(1); }
}

@media (max-width: 900px) {
  .layout-header { padding: 0 16px; }
  .layout-main { padding: 18px; }
}

@media (max-width: 640px) {
  .navigation-backdrop {
    display: block;
    position: fixed;
    inset: 0;
    z-index: 19;
    background: rgba(16, 21, 34, 0.34);
  }

  .aside-menu.is-mobile {
    position: fixed;
    inset: 0 auto 0 0;
    z-index: 20;
    box-shadow: 8px 0 24px rgba(16, 21, 34, 0.16);
    transform: translateX(-105%);
    transition: transform 0.22s ease-out;
  }

  .aside-menu.is-mobile.is-mobile-open { transform: translateX(0); }
  .layout-header { padding-left: 12px; }
  .layout-header .header-right { gap: 8px; }
  .layout-header .user-name, .layout-header .dropdown-arrow { display: none; }
  .layout-main { padding: 14px; }
}

@media (prefers-reduced-motion: reduce) {
  .aside-menu, .el-menu-vertical :deep(.el-menu-item), .el-menu-vertical :deep(.el-sub-menu__title),
  .user-dropdown-link, .page-enter-active, .page-leave-active, .skip-link { transition: none !important; }
  .el-menu-item.is-active::before { animation: none !important; }
}
</style>
