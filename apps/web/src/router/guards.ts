import type { Router } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ElMessage } from 'element-plus'

export function setupRouterGuards(router: Router) {
  router.beforeEach(async (to, from, next) => {
    const authStore = useAuthStore()
    const isPublic = to.meta.public === true

    // 保留原始路徑，登入後可導回目標頁面
    if (!isPublic && !authStore.isAuthenticated) {
      next({
        path: '/login',
        query: { redirect: to.fullPath }
      })
      return
    }

    // 避免已登入使用者停留在登入頁
    if (to.path === '/login' && authStore.isAuthenticated) {
      next('/')
      return
    }

    // F5 重整時 /auth/me 可能還沒回來；guard 在此等待既有請求，避免誤判無權限把使用者踢出當前頁面
    if (authStore.isAuthenticated && authStore.permissionState !== 'loaded') {
      await authStore.loadPermissions()
    }

    if (authStore.isAuthenticated && authStore.permissionState === 'error') {
      next(false)
      return
    }

    // 3. 模組權限比對：畫面顯示與 API 放行一律以後端 /auth/me 回傳的權限矩陣為準
    if (to.meta.module && typeof to.meta.module === 'string') {
      if (!authStore.hasPermission(to.meta.module, 'view')) {
        ElMessage.warning('您的帳號未被授權檢視該功能區塊')
        next('/')
        return
      }
    }

    next()
  })
}
