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

          <!-- 登入表單 -->
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

          <!-- 展示模式快速身分切換：僅在明確啟用 mock 的開發／展示環境顯示 -->
          <div v-if="isMockLoginEnabled" class="dev-quick-login">
            <el-divider>
              <span class="divider-tag">展示模式快速登入</span>
            </el-divider>
            <p class="demo-tip">系統已預載展示資料，可快速切換測試身分。</p>
            <div class="quick-btns">
              <el-button size="default" type="danger" plain @click="quickLogin('admin')">
                系統管理員
              </el-button>
              <el-button size="default" type="primary" plain @click="quickLogin('dispatcher')">
                調度員
              </el-button>
              <el-button size="default" type="success" plain @click="quickLogin('driver')">
                司機
              </el-button>
              <el-button size="default" type="info" plain @click="quickLogin('viewer')">
                檢視者
              </el-button>
            </div>
          </div>
        </section>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { User, Lock } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance } from 'element-plus'
import AppLogo from '@/components/AppLogo.vue'
import { useAuthStore } from '@/stores/auth'
import { supabase } from '@/lib/supabase'
import { isDemoCredentials, enterDemoMode, exitDemoModeIfActive, isMockRuntimeEnabled } from '@/lib/demoMode'
import { ROLE_LABELS, type UserRole } from '@/types/domain'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const formRef = ref<FormInstance>()
const loading = ref(false)
const isMockLoginEnabled = isMockRuntimeEnabled()

const form = reactive({
  email: isMockLoginEnabled ? 'admin@ltc.example.com' : '',
  password: isMockLoginEnabled ? 'password123' : ''
})

const rules = {
  email: [{ required: true, message: '請輸入電子郵件', trigger: 'blur' }],
  password: [{ required: true, message: '請輸入密碼', trigger: 'blur' }]
}

async function handleLogin() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return

    // 帳號密碼皆為 demo：略過真實 Supabase 登入，直接進展示模式
    if (isDemoCredentials(form.email, form.password)) {
      loading.value = true
      try {
        await enterDemoMode()
        authStore.setSession('mock_jwt_demo', {
          id: 'usr_demo',
          email: 'demo',
          displayName: '展示帳號',
          role: 'admin'
        })
        ElMessage.success('已進入展示模式')
        router.push((route.query.redirect as string) || '/')
      } finally {
        loading.value = false
      }
      return
    }

    if (!supabase) {
      ElMessage.error('帳號密碼錯誤或無此使用者')
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
        ElMessage.error('帳號密碼錯誤或無此使用者')
        return
      }

      const role = (data.user.user_metadata?.role ?? data.user.app_metadata?.role ?? 'viewer') as UserRole
      authStore.setSession(data.session.access_token, {
        id: data.user.id,
        email: data.user.email || form.email,
        displayName: data.user.user_metadata?.display_name || data.user.email || form.email,
        role
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

function quickLogin(role: UserRole) {
  const nameMap: Record<UserRole, string> = {
    admin: '系統管理員 (王大明)',
    dispatcher: '調度員 (李調度)',
    driver: '司機 (張司機)',
    staff: '行政人員 (陳專員)',
    viewer: '主管檢視者 (林督導)'
  }

  authStore.setSession(`mock_jwt_${role}`, {
    id: `usr_${role}`,
    email: `${role}@ltc.example.com`,
    displayName: nameMap[role] || '測試使用者',
    role
  })
  ElMessage.success(`已快速切換為【${ROLE_LABELS[role] || role}】身分`)
  const redirect = (route.query.redirect as string) || '/'
  router.push(redirect)
}
</script>

<style scoped>
.login-card {
  width: min(900px, 100%);
  overflow: hidden;
  border: 1px solid var(--app-border);
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
  font-size: 11px;
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
    font-size: 12px;
    font-weight: 600;
  }

  .demo-tip {
    margin: 0 0 12px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
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
