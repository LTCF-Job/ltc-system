import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { UserRole, SystemPermissions } from '@/types/domain'
import type { UserDTO } from '@/types/api'
import { getAuthMe } from '@/api/auth'

const TOKEN_KEY = 'ltc_auth_token'
const USER_KEY = 'ltc_auth_user'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))
  const user = ref<UserDTO | null>(
    localStorage.getItem(USER_KEY) ? JSON.parse(localStorage.getItem(USER_KEY)!) : null
  )

  // 權限矩陣一律來自後端 /auth/me（已合併個人 customPermissions），不快取到 localStorage，
  // 避免權限異動後舊分頁仍讀到舊快取；F5 還原 session 一律重新打一次
  const permissions = ref<SystemPermissions>({})
  const permissionsLoaded = ref(false)
  let permissionsRequest: Promise<void> | null = null

  const isAuthenticated = computed(() => !!token.value && !!user.value)
  const currentRole = computed<UserRole>(() => user.value?.role || 'viewer')

  // 供 router guard 與版面在權限尚未回來前等待，避免誤判無權限或選單瞬間全滅
  function loadPermissions(): Promise<void> {
    if (!token.value) {
      permissions.value = {}
      permissionsLoaded.value = true
      return Promise.resolve()
    }
    if (permissionsRequest) return permissionsRequest

    permissionsRequest = getAuthMe()
      .then((me) => {
        permissions.value = me.permissions || {}
      })
      .catch(() => {
        // 攔截器已處理 401/錯誤提示；這裡保持安全預設（全部視為無權限）
        permissions.value = {}
      })
      .finally(() => {
        permissionsLoaded.value = true
        permissionsRequest = null
      })
    return permissionsRequest
  }

  // 頁面重整時以現有 token 立即補打一次 /auth/me，讓 router guard 有 promise 可等待
  if (token.value && user.value) {
    loadPermissions()
  }

  async function setSession(newToken: string, newUser: UserDTO) {
    token.value = newToken
    user.value = newUser
    localStorage.setItem(TOKEN_KEY, newToken)
    localStorage.setItem(USER_KEY, JSON.stringify(newUser))
    permissionsLoaded.value = false
    permissionsRequest = null
    await loadPermissions()
  }

  async function logout() {
    token.value = null
    user.value = null
    permissions.value = {}
    permissionsLoaded.value = false
    permissionsRequest = null
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
  }

  // 檢查特定功能區塊的檢視 (view)／編輯 (edit)／刪除 (delete) 權限；權限未載入完成前一律回傳 false（安全預設）
  function hasPermission(moduleId: string, action: 'view' | 'edit' | 'delete' = 'view'): boolean {
    if (!user.value || !permissionsLoaded.value) return false
    const modPerm = permissions.value[moduleId]
    if (!modPerm) return false
    return !!modPerm[action]
  }

  return {
    token,
    user,
    isAuthenticated,
    currentRole,
    permissions,
    permissionsLoaded,
    loadPermissions,
    setSession,
    logout,
    hasPermission
  }
})
