import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { Session } from '@supabase/supabase-js'
import type { UserRole, SystemPermissions } from '@/types/domain'
import type { UserDTO } from '@/types/api'
import { getAuthMe } from '@/api/auth'
import { supabase } from '@/lib/supabase'

const TOKEN_KEY = 'ltc_auth_token'
const USER_KEY = 'ltc_auth_user'

type PermissionState = 'idle' | 'loading' | 'loaded' | 'error'

function readStoredUser(): UserDTO | null {
  try {
    const raw = localStorage.getItem(USER_KEY)
    return raw ? (JSON.parse(raw) as UserDTO) : null
  } catch {
    localStorage.removeItem(USER_KEY)
    return null
  }
}

function isLocalMockToken(value: string | null): boolean {
  return value?.startsWith('mock_jwt_') === true
}

function mapSupabaseUser(session: Session): UserDTO {
  const metadata = session.user.user_metadata || {}
  const role = (session.user.app_metadata?.role || 'viewer') as UserRole
  return {
    id: session.user.id,
    email: session.user.email || '',
    displayName: metadata.display_name || session.user.email || '使用者',
    role
  }
}

export const useAuthStore = defineStore('auth', () => {
  // Supabase session 是正式環境的唯一 token 來源；localStorage 只保留本機明確建立的 mock session。
  const storedToken = localStorage.getItem(TOKEN_KEY)
  const token = ref<string | null>(isLocalMockToken(storedToken) ? storedToken : null)
  const user = ref<UserDTO | null>(isLocalMockToken(storedToken) ? readStoredUser() : null)

  const permissions = ref<SystemPermissions>({})
  const permissionState = ref<PermissionState>('idle')
  const permissionsLoaded = computed(() => permissionState.value === 'loaded')
  let permissionsRequest: Promise<void> | null = null
  let initialized = false

  const isAuthenticated = computed(() => !!token.value && !!user.value)
  const currentRole = computed<UserRole>(() => user.value?.role || 'viewer')

  function resetPermissions(state: PermissionState = 'idle') {
    permissions.value = {}
    permissionState.value = state
    permissionsRequest = null
  }

  function clearSession() {
    token.value = null
    user.value = null
    resetPermissions()
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
  }

  function syncSession(session: Session | null) {
    const nextToken = session?.access_token || null
    const nextUser = session ? mapSupabaseUser(session) : null
    const changed = token.value !== nextToken || user.value?.id !== nextUser?.id

    token.value = nextToken
    user.value = nextUser
    if (!session) {
      clearSession()
      return
    }

    // 避免把 Supabase access token 寫入自有 localStorage；Supabase SDK 會自行管理 session。
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
    if (changed) resetPermissions()
  }

  function loadPermissions(): Promise<void> {
    if (!token.value || !user.value) {
      permissions.value = {}
      permissionState.value = 'loaded'
      return Promise.resolve()
    }
    if (permissionsRequest) return permissionsRequest

    permissionState.value = 'loading'
    permissionsRequest = getAuthMe()
      .then((me) => {
        permissions.value = me.permissions || {}
        permissionState.value = 'loaded'
      })
      .catch(() => {
        permissions.value = {}
        permissionState.value = token.value ? 'error' : 'idle'
      })
      .finally(() => {
        permissionsRequest = null
      })
    return permissionsRequest
  }

  async function initializeAuth() {
    if (initialized) return
    initialized = true

    if (!supabase) {
      if (token.value && user.value) await loadPermissions()
      return
    }

    supabase.auth.onAuthStateChange((_event, session) => {
      syncSession(session)
      if (session) setTimeout(() => void loadPermissions(), 0)
    })

    const { data, error } = await supabase.auth.getSession()
    if (error) {
      clearSession()
      permissionState.value = 'error'
      return
    }

    syncSession(data.session)
    if (data.session) await loadPermissions()
  }

  async function setSession(newToken: string, newUser: UserDTO) {
    // 只提供給本機 mock 登入與測試；正式 Supabase session 必須透過 syncSession 建立。
    token.value = newToken
    user.value = newUser
    localStorage.setItem(TOKEN_KEY, newToken)
    localStorage.setItem(USER_KEY, JSON.stringify(newUser))
    resetPermissions()
    await loadPermissions()
  }

  async function logout() {
    if (supabase && !isLocalMockToken(token.value)) {
      try {
        await supabase.auth.signOut({ scope: 'local' })
      } catch {
        // 即使遠端 session 清除失敗，也要清除本機的畫面狀態。
      }
    }
    clearSession()
  }

  function hasPermission(moduleId: string, action: 'view' | 'edit' | 'delete' = 'view'): boolean {
    if (!user.value || permissionState.value !== 'loaded') return false
    const modPerm = permissions.value[moduleId]
    return !!modPerm?.[action]
  }

  return {
    token,
    user,
    isAuthenticated,
    currentRole,
    permissions,
    permissionState,
    permissionsLoaded,
    initializeAuth,
    syncSession,
    loadPermissions,
    setSession,
    logout,
    hasPermission
  }
})
