import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  DEFAULT_ROLE_PERMISSIONS,
  type UserRole,
  type SystemPermissions
} from '@/types/domain'
import type { UserDTO } from '@/types/api'
import { clearDemoModeOnLogout } from '@/lib/demoMode'

const TOKEN_KEY = 'ltc_auth_token'
const USER_KEY = 'ltc_auth_user'

const ROLE_HIERARCHY: Record<UserRole, number> = {
  admin: 4,
  dispatcher: 3,
  staff: 3,
  driver: 2,
  viewer: 1
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))
  const user = ref<UserDTO | null>(
    localStorage.getItem(USER_KEY) ? JSON.parse(localStorage.getItem(USER_KEY)!) : null
  )

  const isAuthenticated = computed(() => !!token.value && !!user.value)
  const currentRole = computed<UserRole>(() => user.value?.role || 'viewer')

  // 計算最終有效權限（優先權：個人自訂設定 > 角色預設設定）
  const effectivePermissions = computed<SystemPermissions>(() => {
    const role = currentRole.value
    const roleDefault = DEFAULT_ROLE_PERMISSIONS[role] || DEFAULT_ROLE_PERMISSIONS.viewer
    const custom = user.value?.customPermissions

    if (!custom) {
      return roleDefault
    }

    const merged: SystemPermissions = { ...roleDefault }
    for (const [modKey, perm] of Object.entries(custom)) {
      if (perm) {
        merged[modKey] = { ...perm }
      }
    }
    return merged
  })

  function setSession(newToken: string, newUser: UserDTO) {
    token.value = newToken
    user.value = newUser
    localStorage.setItem(TOKEN_KEY, newToken)
    localStorage.setItem(USER_KEY, JSON.stringify(newUser))
  }

  async function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
    // 必須等待展示模式清理完成，避免緊接著的下一次登入與 mock 停用/重置互相競態
    await clearDemoModeOnLogout()
  }

  // 檢查當前角色是否滿足基本身分限制
  function can(required: UserRole | UserRole[]): boolean {
    if (!user.value) return false
    if (user.value.role === 'admin') return true

    const myLevel = ROLE_HIERARCHY[user.value.role] || 1

    if (Array.isArray(required)) {
      return required.some((role) => myLevel >= (ROLE_HIERARCHY[role as UserRole] || 1))
    }
    return myLevel >= (ROLE_HIERARCHY[required as UserRole] || 1)
  }

  // 檢查特定功能區塊的檢視 (view) 或編輯 (edit) 權限
  function hasPermission(moduleId: string, action: 'view' | 'edit' = 'view'): boolean {
    if (!user.value) return false
    if (user.value.role === 'admin') return true

    const modPerm = effectivePermissions.value[moduleId]
    if (!modPerm) return false
    return !!modPerm[action]
  }

  return {
    token,
    user,
    isAuthenticated,
    currentRole,
    effectivePermissions,
    setSession,
    logout,
    can,
    hasPermission
  }
})
