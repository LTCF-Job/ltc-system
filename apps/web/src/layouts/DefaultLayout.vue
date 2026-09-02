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
      <!-- 品牌識別區 -->
      <router-link
        to="/"
        class="sidebar-brand-link"
        :class="{ 'is-collapsed': isCollapse && !isMobile }"
        :title="isCollapse && !isMobile ? '好安心關懷協會-後臺系統' : ''"
      >
        <AppLogo
          :size="isCollapse && !isMobile ? 'sm' : 'md'"
          :collapsed="isCollapse && !isMobile"
          variant="light"
        />
      </router-link>

      <el-scrollbar class="menu-scrollbar">
        <el-menu
          id="primary-navigation"
          :default-active="activeRoute"
            :collapse="!isMobile && isCollapse"
          router
          class="el-menu-vertical"
        >
          <el-menu-item-group>
            <template #title><span class="nav-group-label">總覽</span></template>
          <el-menu-item index="/">
            <el-icon><Odometer /></el-icon>
            <template #title>總覽儀表板</template>
          </el-menu-item>

          </el-menu-item-group>

          <el-menu-item-group>
            <template #title><span class="nav-group-label">營運作業</span></template>
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
            <el-icon><Clock /></el-icon>
            <template #title>出勤與油資管理</template>
          </el-menu-item>

          <el-menu-item v-if="authStore.hasPermission('vehicles_maintenance')" index="/vehicles/maintenance">
            <el-icon><Management /></el-icon>
            <template #title>車輛維修保養</template>
          </el-menu-item>

          </el-menu-item-group>

          <el-menu-item-group>
            <template #title><span class="nav-group-label">報表與申報</span></template>
          <el-sub-menu v-if="authStore.hasPermission('driver_reports') || authStore.hasPermission('driver_report_mappings')" index="driver-reports">
            <template #title>
              <el-icon><DocumentCopy /></el-icon>
              <span>司機接送匯報</span>
            </template>
            <el-menu-item v-if="authStore.hasPermission('driver_reports')" index="/driver-reports/status">
              <el-icon><DocumentCopy /></el-icon>
              <template #title>接送匯報總覽</template>
            </el-menu-item>
            <el-menu-item
              v-if="authStore.hasPermission('driver_reports') || authStore.hasPermission('driver_report_mappings')"
              index="/driver-reports/import"
            >
              <el-icon><Connection /></el-icon>
              <template #title>批次上傳與待維護資料</template>
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
              <el-icon><Tickets /></el-icon>
              <template #title>新竹接送時刻表</template>
            </el-menu-item>
          </el-sub-menu>

          <el-menu-item v-if="authStore.hasPermission('exports')" index="/exports">
            <el-icon><Download /></el-icon>
            <template #title>政府申報匯出</template>
          </el-menu-item>

          </el-menu-item-group>

          <el-menu-item-group>
            <template #title><span class="nav-group-label">資料管理</span></template>
          <el-sub-menu v-if="authStore.hasPermission('masters_regions') || authStore.hasPermission('masters_cases') || authStore.hasPermission('masters_sites') || authStore.hasPermission('masters_vehicles') || authStore.hasPermission('masters_drivers') || authStore.hasPermission('masters_caregivers')" index="masters">
            <template #title>
              <el-icon><Folder /></el-icon>
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

          </el-menu-item-group>

          <el-menu-item-group>
            <template #title><span class="nav-group-label">系統</span></template>
          <el-menu-item v-if="authStore.hasPermission('audit_logs')" index="/audit">
            <el-icon><Notebook /></el-icon>
            <template #title>系統操作紀錄</template>
          </el-menu-item>

          <el-sub-menu v-if="authStore.hasPermission('settings_users') || authStore.hasPermission('settings_roles') || authStore.hasPermission('settings_notifications')" index="settings">
            <template #title>
              <el-icon><Setting /></el-icon>
              <span>系統設定</span>
            </template>
            <el-menu-item v-if="authStore.hasPermission('settings_users')" index="/settings/users">
              <el-icon><Postcard /></el-icon>
              <template #title>使用者管理</template>
            </el-menu-item>
            <el-menu-item v-if="authStore.hasPermission('settings_roles')" index="/settings/roles">
              <el-icon><Medal /></el-icon>
              <template #title>角色身分管理</template>
            </el-menu-item>
            <el-menu-item v-if="authStore.hasPermission('settings_notifications')" index="/settings/notifications">
              <el-icon><Message /></el-icon>
              <template #title>通知收件人</template>
            </el-menu-item>
            <el-menu-item v-if="authStore.hasPermission('settings_notifications')" index="/settings/holidays">
              <el-icon><Timer /></el-icon>
              <template #title>政府假日與工作日設定</template>
            </el-menu-item>
          </el-sub-menu>
          </el-menu-item-group>
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
                <el-avatar :size="30" :icon="UserFilled" class="user-avatar" />
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
  UserFilled,
  Tickets,
  Notebook,
  Clock,
  Timer,
  Folder,
  Postcard,
  Medal,
  Message
} from '@element-plus/icons-vue'

import { useAuthStore } from '@/stores/auth'
import ChangePasswordDialog from '@/components/ChangePasswordDialog.vue'
import AppLogo from '@/components/AppLogo.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const isCollapse = ref(false)
const isMobile = ref(false)
const isMobileMenuOpen = ref(false)
const userToggledCollapse = ref(false)
const changePasswordDialogRef = ref<InstanceType<typeof ChangePasswordDialog>>()

// 側邊欄展開寬度為固定值，取代原本依選單文字逐字估算寬度的作法；
// 數值需跟 tokens.scss 的 --app-aside-width 同步，這裡是 JS 綁定 el-aside :width，CSS token 改了這裡也要一起改
const expandedAsideWidth = '240px'

const activeRoute = computed(() => {
  const path = route.path
  if (path.startsWith('/cases/')) return '/cases'
  return path
})

const isNavigationOpen = computed(() => isMobile.value ? isMobileMenuOpen.value : !isCollapse.value)

function updateViewportMode() {
  isMobile.value = window.matchMedia('(max-width: 640px)').matches
  if (!isMobile.value) isMobileMenuOpen.value = false
  // 900px 到 640px 之間側欄若不自動收合，240px 側欄會壓縮主內容區
  if (!userToggledCollapse.value) {
    isCollapse.value = window.matchMedia('(max-width: 1024px)').matches
  }
}

function toggleNavigation() {
  if (isMobile.value) {
    isMobileMenuOpen.value = !isMobileMenuOpen.value
    return
  }
  userToggledCollapse.value = true
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
  outline: 3px solid var(--app-primary);
  outline-offset: 2px;
}

.layout-container {
  height: calc(100vh - var(--app-shell-inset-y) * 2);
  min-height: 0;
  margin: var(--app-shell-inset-y) var(--app-shell-inset-x);
  background: var(--app-surface);
  border-radius: var(--app-radius-lg);
  box-shadow: var(--app-shadow-shell);
  overflow: hidden;
}

.aside-menu {
  background: var(--app-nav-bg);
  border-right: 1px solid var(--app-nav-border);
  height: 100%;
  min-height: 0;
  transition: width 0.25s ease;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  flex: 0 0 auto;

  .sidebar-brand-link {
    display: flex;
    align-items: center;
    padding: 14px 16px;
    height: 60px;
    box-sizing: border-box;
    text-decoration: none;
    border-bottom: 1px solid var(--app-nav-border);
    background: var(--app-nav-bg);
    overflow: hidden;
    transition: background-color 0.2s ease;

    &:hover {
      background: var(--app-nav-hover-bg);
    }

    &.is-collapsed {
      padding: 14px 0;
      justify-content: center;
    }
  }

  .menu-scrollbar {
    flex: 1;
    min-height: 0;
    height: 100%;

    :deep(.el-scrollbar__wrap) { overflow-x: hidden; }
    :deep(.el-scrollbar__view) { min-height: 100%; }
  }

  .el-menu-vertical {
    /* Element Plus 的選單配色一律由這組變數驅動，template 不再傳色碼 prop */
    --el-menu-bg-color: var(--app-nav-bg);
    --el-menu-text-color: var(--app-nav-fg);
    --el-menu-active-color: var(--app-nav-active-fg);
    --el-menu-hover-bg-color: var(--app-nav-hover-bg);
    background-color: var(--app-nav-bg) !important;
    border-right: none;
    padding: 6px 8px;

    /* 導覽分組標題：淺色側欄靠字距與字重分段，不再加分隔線 */
    :deep(.el-menu-item-group__title) {
      padding: 14px 10px 6px;
    }

    .nav-group-label {
      display: block;
      font-size: var(--app-label-size);
      font-weight: 700;
      letter-spacing: var(--app-label-tracking);
      text-transform: uppercase;
      color: var(--app-nav-fg-muted);
      white-space: nowrap;
    }

    /* 收合時側欄只剩圖示，分組標題會變成截斷的碎字，直接隱藏 */
    &.el-menu--collapse :deep(.el-menu-item-group__title) {
      display: none;
    }

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
      color: var(--app-nav-fg);
      transition: transform 0.18s ease-out, background-color 0.18s ease-out, color 0.18s ease-out;

      &:hover {
        background-color: var(--app-nav-hover-bg) !important;
        color: var(--app-text-primary) !important;
        transform: translateX(2px);
      }

      .el-icon {
        flex: 0 0 auto;
        font-size: 17px;
        margin-right: 8px;
        color: var(--app-nav-fg-muted);
      }
    }

    :deep(.el-menu-item > span),
    :deep(.el-sub-menu__title > span:not(.el-sub-menu__icon-arrow)) {
      flex: 0 0 auto;
      white-space: nowrap;
    }

    :deep(.el-menu-item.is-active) {
      background: var(--app-nav-active-bg) !important;
      color: var(--app-nav-active-fg) !important;
      font-weight: 600;

      .el-icon {
        color: var(--app-nav-active-fg) !important;
      }

      &::before {
        content: '';
        position: absolute;
        left: 0;
        top: 8px;
        bottom: 8px;
        width: 3px;
        border-radius: 0 4px 4px 0;
        background-color: var(--app-primary);
        animation: nav-indicator-in 0.2s ease-out;
      }
    }

    :deep(.el-sub-menu .el-menu) {
      background-color: var(--app-nav-bg) !important;
      padding: 4px 0 4px 8px;
      border-radius: 8px;
    }
  }

}

.navigation-backdrop {
  display: none;
}

.layout-header {
  height: var(--app-header-height);
  background-color: var(--app-surface);
  border-bottom: 1px solid var(--app-border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 22px;
  flex: 0 0 auto;

  .header-left {
    display: flex;
    align-items: center;
    gap: 16px;

    .toggle-btn {
      color: var(--app-text-secondary);
      border-radius: 6px;
      padding: 6px;

      &:hover {
        background-color: var(--app-status-neutral-bg);
        color: var(--app-text-primary);
      }
    }
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 14px;

    /* 使用者資訊常態就是一個淡底膠囊，不是只有 hover 才浮現背景——
       跟頂部列其他互動元件（切換側欄、通知）的「按了才有反饋」語彙不同，
       這是身分識別區，常態可見的邊界能讓使用者一眼找到帳號入口 */
    .user-dropdown-link {
      display: flex;
      align-items: center;
      gap: 9px;
      cursor: pointer;
      font-size: 13.5px;
      font-weight: 600;
      color: var(--app-text-primary);
      padding: 5px 14px 5px 6px;
      border-radius: var(--app-radius-full);
      background-color: var(--app-status-neutral-bg);
      border: 1px solid var(--app-border-light);
      transition: background-color 0.15s ease, border-color 0.15s ease, transform 0.15s ease;

      &:hover {
        background-color: var(--app-primary-light);
        border-color: #c3d9fc;
        transform: scale(1.03);
      }

      &:active {
        transform: scale(0.97);
      }

      .avatar-box {
        position: relative;
        display: flex;
        align-items: center;

        .user-avatar {
          background-color: var(--app-primary);
          box-shadow: 0 0 0 2px #ffffff, 0 0 0 3px var(--app-primary-light);
        }

        .avatar-online-dot {
          position: absolute;
          bottom: -1px;
          right: -1px;
          width: 9px;
          height: 9px;
          border-radius: 50%;
          background-color: var(--app-status-success-fg);
          border: 2px solid #ffffff;
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
        color: var(--app-text-secondary);
        transition: transform 0.15s ease;
      }

      &:hover .dropdown-arrow {
        transform: translateY(1px);
      }
    }
  }
}

.layout-main {
  background-color: var(--app-bg);
  padding: 24px;
  min-height: 0;
  min-width: 0;
  height: calc(100% - var(--app-header-height));
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
  /* 窄螢幕留白比可視面積值錢，應用殼改貼齊視窗 */
  .layout-container {
    height: 100vh;
    margin: 0;
    border-radius: 0;
    box-shadow: none;
  }
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
