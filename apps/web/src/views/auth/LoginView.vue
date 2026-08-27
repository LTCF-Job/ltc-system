<template>
  <div class="login-container">
    <el-card class="login-card" shadow="always">
      <template #header>
        <div class="card-header">
          <h2>長照交通接送後台系統</h2>
          <p class="subtitle">後台登入管理介面</p>
        </div>
      </template>

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
          <span class="divider-tag">✨ 展示模式快速登入</span>
        </el-divider>
        <p class="demo-tip">系統已預載全類型個案、排班、搭乘矩陣、異常裁決與報表資料：</p>
        <div class="quick-btns">
          <el-button size="default" type="danger" plain @click="quickLogin('admin')">
            系統管理員 (Admin)
          </el-button>
          <el-button size="default" type="primary" plain @click="quickLogin('dispatcher')">
            調度員 (Dispatcher)
          </el-button>
          <el-button size="default" type="success" plain @click="quickLogin('driver')">
            司機 (Driver)
          </el-button>
          <el-button size="default" type="info" plain @click="quickLogin('viewer')">
            檢視者 (Viewer)
          </el-button>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { User, Lock } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance } from 'element-plus'
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
    if (isMockLoginEnabled && isDemoCredentials(form.email, form.password)) {
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
      ElMessage.error('尚未設定 Supabase 登入環境變數，請聯絡系統管理員')
      return
    }
    loading.value = true
    try {
      const { data, error } = await supabase.auth.signInWithPassword({
        email: form.email,
        password: form.password
      })
      if (error || !data.session || !data.user) {
        ElMessage.error(error?.message || '帳號或密碼錯誤')
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
.card-header {
  text-align: center;

  h2 {
    color: var(--el-color-primary);
    font-size: 22px;
    font-weight: bold;
    margin-bottom: 6px;
  }

  .subtitle {
    color: var(--el-text-color-secondary);
    font-size: 14px;
  }
}

.submit-btn {
  width: 100%;
  margin-top: 10px;
}

.dev-quick-login {
  margin-top: 24px;

  .divider-tag {
    font-weight: bold;
    color: var(--el-color-warning-dark-2);
  }

  .demo-tip {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-align: center;
    margin-bottom: 12px;
  }

  .quick-btns {
    display: flex;
    justify-content: space-between;
    gap: 8px;
  }
}
</style>
