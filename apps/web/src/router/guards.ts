import type { Router } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ElMessage } from 'element-plus'

export function setupRouterGuards(router: Router) {
  router.beforeEach((to, from, next) => {
    const authStore = useAuthStore()
    const isPublic = to.meta.public === true

    // 1. 未登入驗證
    if (!isPublic && !authStore.isAuthenticated) {
      next({
        path: '/login',
        query: { redirect: to.fullPath }
      })
      return
    }

    // 2. 已登入者進入登入頁自動導向首頁
    if (to.path === '/login' && authStore.isAuthenticated) {
      next('/')
      return
    }

    // 3. 模組與角色權限比對
    if (to.meta.module && typeof to.meta.module === 'string') {
      if (!authStore.hasPermission(to.meta.module, 'view')) {
        ElMessage.warning('您的帳號未被授權檢視該功能區塊')
        next('/')
        return
      }
    } else if (to.meta.roles && Array.isArray(to.meta.roles)) {
      if (!authStore.can(to.meta.roles as any)) {
        ElMessage.warning('權限不足，無法存取該頁面')
        next('/')
        return
      }
    }

    next()
  })
}
