<template>
  <div class="login-container">
    <el-card class="login-card" shadow="never">
      <div class="login-grid">
        <section class="brand-panel" aria-labelledby="brand-title">
          <AppLogo size="xl" :show-text="false" variant="light" />
          <h1 id="brand-title" aria-label="好安心關懷協會-後臺系統">
            <span>好安心關懷協會</span>
            <span>後臺系統</span>
          </h1>
        </section>

        <section class="form-panel" aria-labelledby="login-title">
          <div class="form-heading">
            <p class="form-kicker">WELCOME BACK</p>
            <h2 id="login-title">登入系統</h2>
          </div>

          <el-form
            ref="formRef"
            :model="form"
            :rules="rules"
            label-position="top"
            @submit.prevent="handleLogin"
          >
            <el-form-item label="帳號 / 電子郵件" prop="email">
              <el-input
                v-model="form.email"
                placeholder="請輸入電子郵件"
                :prefix-icon="User"
                autocomplete="username"
              />
            </el-form-item>

            <el-form-item label="密碼" prop="password">
              <el-input
                v-model="form.password"
                type="password"
                placeholder="請輸入密碼"
                :prefix-icon="Lock"
                show-password
                autocomplete="current-password"
              />
            </el-form-item>

            <el-button
              type="primary"
              native-type="submit"
              :loading="loading"
              class="submit-btn"
              size="large"
            >
              登入系統
            </el-button>
          </el-form>

          <!-- 本機開發快速身分切換：僅在 LOCAL 環境下顯示，其餘環境絕不出現 -->
          <div v-if="isLocalEnvironment" class="dev-quick-login">
            <el-divider>
              <span class="divider-tag">本機快速登入</span>
            </el-divider>
            <p class="demo-tip">
              {{ !supabase ? '本機未連線 Supabase，可直接以測試身分登入操作本機資料庫。' : '本機開發模式，可快速切換測試身分。' }}
            </p>
            <div class="quick-btns">
              <el-button size="default" type="primary" plain @click="quickLogin('admin')">
                系統管理員
              </el-button>
              <el-button size="default" type="info" plain @click="quickLogin('viewer')">
                檢視人員
              </el-button>
            </div>
          </div>
        </section>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { User, Lock } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance } from 'element-plus'
import AppLogo from '@/components/AppLogo.vue'
import { useAuthStore } from '@/stores/auth'
import { supabase } from '@/lib/supabase'
import { isDemoCredentials, exitDemoModeIfActive, isMockRuntimeEnabled } from '@/lib/demoMode'
import { ROLE_LABELS, type UserRole } from '@/types/domain'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const formRef = ref<FormInstance>()
const loading = ref(false)

// 嚴格判定是否為本機環境（Vite DEV 模式、E2E Mock、或明確指定 VITE_APP_ENV=local）
// 非本機環境（正式、預覽、雲端容器等）一律為 false，不論 Supabase 是否設定
const isLocalEnvironment = computed(() => {
  if (isMockRuntimeEnabled()) return true
  if (import.meta.env.DEV) return true
  if (import.meta.env.VITE_APP_ENV === 'local') return true
  return false
})

const form = reactive({
  email: isLocalEnvironment.value ? 'admin@ltc.example.com' : '',
  password: isLocalEnvironment.value ? 'password123' : ''
})

const rules = {
  email: [{ required: true, message: '請輸入電子郵件', trigger: 'blur' }],
  password: [{ required: true, message: '請輸入密碼', trigger: 'blur' }]
}

async function handleLogin() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return

    if (!supabase) {
      if (isLocalEnvironment.value) {
        // 本機未設定 Supabase 時，自動以本機開發身分登入直通本機資料庫
        const role: UserRole = form.email.includes('viewer') ? 'viewer' : 'admin'
        await quickLogin(role)
        return
      }
      // 非本機環境未連線 Supabase 時，嚴禁隨意登入
      ElMessage.error('系統認證服務未設定，無法登入')
      return
    }
    loading.value = true
    try {
      // "demo" 帳號代稱固定對應到 Supabase 上的 Demo 測試帳號，密碼仍走真實驗證，不再略過 Supabase
      const usesDemoAlias = isDemoCredentials(form.email, form.password)
      const authEmail = usesDemoAlias
        ? 'demo@ltc.example.com'
        : form.email === 'ltcf-admin'
          ? 'ltcf-admin@ltc.example.com'
          : form.email
      const { data, error } = await supabase.auth.signInWithPassword({
        email: authEmail,
        password: form.password
      })
      if (error || !data.session || !data.user) {
        ElMessage.error('帳號密碼錯誤或無此使用者')
        return
      }

      const role = (data.user.app_metadata?.role ?? data.user.user_metadata?.role ?? 'viewer') as UserRole
      const dataPlane = (data.user.app_metadata?.data_plane ?? 'production') as 'production' | 'demo'
      authStore.setSession(data.session.access_token, {
        id: data.user.id,
        email: data.user.email || form.email,
        displayName: data.user.user_metadata?.display_name || data.user.email || form.email,
        role,
        dataPlane
      })
      // 確保不殘留前一次展示模式的攔截
      await exitDemoModeIfActive()

      ElMessage.success('登入成功')
      const redirect = (route.query.redirect as string) || '/'
      router.push(redirect)
    } finally {
      loading.value = false
    }
  })
}

async function quickLogin(role: UserRole) {
  if (!isLocalEnvironment.value) {
    ElMessage.error('非本機環境禁止使用快速登入')
    return
  }

  const nameMap: Record<UserRole, string> = {
    admin: '系統管理員 (王大明)',
    viewer: '檢視人員 (林督導)'
  }

  authStore.setSession(`mock_jwt_${role}`, {
    id: `usr_${role}`,
    email: `${role}@ltc.example.com`,
    displayName: nameMap[role] || '測試使用者',
    role,
    dataPlane: 'production'
  })
  // 退出前端展示模式攔截，讓請求直通本機後端 API 與本機資料庫
  await exitDemoModeIfActive()

  ElMessage.success(`已快速切換為【${ROLE_LABELS[role] || role}】身分`)
  const redirect = (route.query.redirect as string) || '/'
  router.push(redirect)
}
</script>

<style scoped>
.login-card {
  width: min(900px, 100%);
  overflow: hidden;
  border: 1px solid var(--app-border-color);
  border-radius: 20px;
  background: var(--app-surface);
  box-shadow: 0 18px 50px rgba(25, 50, 77, 0.12);

  :deep(.el-card__body) {
    padding: 0;
  }
}

.login-grid {
  display: grid;
  grid-template-columns: minmax(260px, 0.9fr) minmax(360px, 1.1fr);
  min-height: 560px;
}

.brand-panel {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  padding: 52px 44px;
  text-align: center;
  background: linear-gradient(155deg, #edf8fb 0%, #f7fbfc 58%, #eef5fb 100%);
}

.brand-panel :deep(.app-logo) {
  margin-bottom: 40px;
}

.form-kicker {
  margin: 0 0 10px;
  color: var(--app-primary);
  font-size: var(--app-font-xs);
  font-weight: 700;
  letter-spacing: 0.18em;
}

.brand-panel h1 {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-width: none;
  margin: 0;
  color: var(--app-text-primary);
  font-size: clamp(26px, 3vw, 34px);
  font-weight: 700;
  line-height: 1.3;
}

.form-panel {
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 52px 56px;
}

.form-heading {
  margin-bottom: 30px;
}

.form-heading h2 {
  margin: 0 0 8px;
  color: var(--app-text-primary);
  font-size: 25px;
  font-weight: 700;
}

.form-panel :deep(.el-form-item) {
  margin-bottom: 22px;
}

.submit-btn {
  width: 100%;
  height: 46px;
  margin-top: 4px;
  border-radius: 9px;
}

.dev-quick-login {
  margin-top: 30px;

  .divider-tag {
    color: var(--app-text-secondary);
    font-size: var(--app-font-xs);
    font-weight: 600;
  }

  .demo-tip {
    margin: 0 0 12px;
    font-size: var(--app-font-xs);
    color: var(--app-text-secondary);
    text-align: center;
  }

  .quick-btns {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;

    :deep(.el-button) {
      flex: 1 1 calc(50% - 8px);
      margin-left: 0;
    }
  }
}

@media (max-width: 720px) {
  .login-grid {
    grid-template-columns: 1fr;
    min-height: auto;
  }

  .brand-panel,
  .form-panel {
    padding: 34px 28px;
  }

  .brand-panel {
    align-items: center;
    text-align: center;
  }

  .brand-panel :deep(.app-logo) {
    margin-bottom: 20px;
  }

  .brand-panel h1 {
    max-width: none;
  }

  .form-panel {
    padding-top: 36px;
  }
}
</style>
