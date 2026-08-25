import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { UserRole } from '@/types/domain'
import type { UserDTO } from '@/types/api'

const TOKEN_KEY = 'ltc_auth_token'
const USER_KEY = 'ltc_auth_user'

const ROLE_HIERARCHY: Record<UserRole, number> = {
  admin: 3,
  staff: 2,
  viewer: 1
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))
  const user = ref<UserDTO | null>(
    localStorage.getItem(USER_KEY) ? JSON.parse(localStorage.getItem(USER_KEY)!) : null
  )

  const isAuthenticated = computed(() => !!token.value && !!user.value)
  const currentRole = computed<UserRole>(() => user.value?.role || 'viewer')

  function setSession(newToken: string, newUser: UserDTO) {
    token.value = newToken
    user.value = newUser
    localStorage.setItem(TOKEN_KEY, newToken)
    localStorage.setItem(USER_KEY, JSON.stringify(newUser))
  }

  function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
  }

  // 檢查當前角色權限是否滿足要求（支援階層比較與多角色列舉）
  function can(required: UserRole | UserRole[]): boolean {
    if (!user.value) return false
    const myLevel = ROLE_HIERARCHY[user.value.role]

    if (Array.isArray(required)) {
      return required.some((role) => myLevel >= ROLE_HIERARCHY[role as UserRole])
    }
    return myLevel >= ROLE_HIERARCHY[required as UserRole]
  }

  return {
    token,
    user,
    isAuthenticated,
    currentRole,
    setSession,
    logout,
    can
  }
})
