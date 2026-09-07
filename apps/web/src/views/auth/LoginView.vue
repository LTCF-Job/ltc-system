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
import { isLocalEnvironment as resolveLocalEnvironment } from '@/lib/authMode'
import { type UserRole } from '@/types/domain'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const formRef = ref<FormInstance>()
const loading = ref(false)

// local 一律使用本機 mock JWT；預覽與正式環境才使用 Supabase。
const isLocalEnvironment = computed(
  () =>
    resolveLocalEnvironment({
      isDev: import.meta.env.DEV,
      appEnv: import.meta.env.VITE_APP_ENV
    })
)

const form = reactive({
  email: '',
  password: ''
})

const rules = {
  email: [{ required: true, message: '請輸入電子郵件', trigger: 'blur' }],
  password: [{ required: true, message: '請輸入密碼', trigger: 'blur' }]
}

function goAfterLogin() {
  ElMessage.success('登入成功')
  router.push((route.query.redirect as string) || '/')
}

// describeSignInFailure 把 Supabase 的登入失敗分成使用者能處理的幾類。
// 全部說成「帳號密碼錯誤」會讓斷線、帳號未啟用或服務異常的人一直重打密碼；
// 這裡只依 Supabase 的錯誤分類決定顯示文字，不把原始錯誤字串帶到畫面上。
function describeSignInFailure(error: { status?: number; code?: string; message?: string } | null): string {
  if (!error) return '登入失敗，請稍後再試'
  if (error.status === 400 || error.status === 401) {
    if (error.code === 'email_not_confirmed') return '此帳號尚未完成信箱驗證，請先完成驗證再登入'
    return '帳號或密碼錯誤，請確認後再試'
  }
  if (error.status === 429) return '登入嘗試次數過多，請稍候幾分鐘後再試'
  if (error.status && error.status >= 500) return '認證服務暫時無法使用，請稍後再試'
  if (!error.status) return '無法連線到認證服務，請確認網路狀態後再試'
  return '登入失敗，請稍後再試'
}

async function handleLogin() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return

    if (!supabase) {
      if (!isLocalEnvironment.value) {
        // 非本機環境未連線 Supabase 時，嚴禁隨意登入
        ElMessage.error('系統認證服務未設定，無法登入')
        return
      }
      // 本機不接 Supabase，改發後端 local 分支認得的 mock JWT，讓登入流程與正式環境一致；
      // 帳號含 "viewer" 字樣即以檢視人員身分登入，其餘一律管理員
      const role: UserRole = form.email.includes('viewer') ? 'viewer' : 'admin'
      const permissionsReady = await authStore.setSession(`mock_jwt_${role}`, {
        id: '00000000-0000-0000-0000-000000000001',
        email: form.email,
        displayName: form.email,
        role
      })
      if (!permissionsReady) {
        await authStore.logout()
        return
      }
      goAfterLogin()
      return
    }

    loading.value = true
    try {
      const authEmail = form.email === 'ltcf-admin' ? 'ltcf-admin@ltc.example.com' : form.email
      const { data, error } = await supabase.auth.signInWithPassword({
        email: authEmail,
        password: form.password
      })
      if (error || !data.session || !data.user) {
        ElMessage.error(describeSignInFailure(error))
        return
      }

      // 角色只接受 Supabase app_metadata；user_metadata 可由使用者自行修改，不可作為授權來源。
      await authStore.syncSession(data.session)
      const permissionsReady = await authStore.loadPermissions()
      if (!permissionsReady) {
        await authStore.logout()
        return
      }
      goAfterLogin()
    } finally {
      loading.value = false
    }
  })
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
