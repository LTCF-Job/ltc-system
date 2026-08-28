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
            placeholder="請輸入電子郵件或 demo"
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

      <!-- Demo 展示模式快速體驗區 -->
      <div class="demo-experience-box">
        <el-divider>
          <span class="divider-tag">✨ 展示模式快速體驗</span>
        </el-divider>
        <p class="demo-tip">帳號 <code>demo</code> / 密碼 <code>demo</code>，預載完整模擬資料且變更不寫入資料庫：</p>
        <el-button
          type="warning"
          plain
          class="demo-fast-btn"
          size="default"
          :loading="loading"
          @click="handleDemoLogin('admin')"
        >
          ✨ 一鍵以 Demo 管理員登入體驗
        </el-button>

        <div v-if="isMockLoginEnabled" class="dev-quick-login">
          <p class="role-switch-title">切換其他角色登入：</p>
          <div class="quick-btns">
            <el-button size="small" type="danger" plain @click="handleDemoLogin('admin')">
              管理員
            </el-button>
            <el-button size="small" type="primary" plain @click="handleDemoLogin('dispatcher')">
              調度員
            </el-button>
            <el-button size="small" type="success" plain @click="handleDemoLogin('driver')">
              司機
            </el-button>
            <el-button size="small" type="info" plain @click="handleDemoLogin('viewer')">
              檢視者
            </el-button>
          </div>
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
  email: [{ required: true, message: '請輸入電子郵件或 demo', trigger: 'blur' }],
  password: [{ required: true, message: '請輸入密碼', trigger: 'blur' }]
}

async function handleLogin() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return

    // 檢查是否使用 demo 帳號登入
    if (isDemoCredentials(form.email, form.password)) {
      await handleDemoLogin('admin')
      return
    }

    if (!supabase) {
      ElMessage.error('尚未設定 Supabase 登入環境變數，請輸入 demo 體驗展示模式或聯絡系統管理員')
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
      // 確保清除前一次展示模式的攔截狀態
      await exitDemoModeIfActive()

      ElMessage.success('登入成功')
      const redirect = (route.query.redirect as string) || '/'
      router.push(redirect)
    } finally {
      loading.value = false
    }
  })
}

// 執行展示模式登入流程
async function handleDemoLogin(role: UserRole = 'admin') {
  loading.value = true
  try {
    await enterDemoMode()
    const nameMap: Record<UserRole, string> = {
      admin: '展示管理員 (王大明)',
      dispatcher: '展示調度員 (李調度)',
      driver: '展示司機 (張司機)',
      staff: '展示行政 (陳專員)',
      viewer: '展示檢視者 (林督導)'
    }

    authStore.setSession(`mock_jwt_${role}`, {
      id: `usr_${role}`,
      email: `${role}@ltc.example.com`,
      displayName: nameMap[role] || '展示使用者',
      role
    })

    ElMessage.success({
      message: '已進入展示模式（所有操作僅在前端模擬，不會寫入資料庫）',
      duration: 4000
    })
    const redirect = (route.query.redirect as string) || '/'
    router.push(redirect)
  } finally {
    loading.value = false
  }
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

.demo-experience-box {
  margin-top: 20px;

  .divider-tag {
    font-weight: bold;
    color: var(--el-color-warning-dark-2);
  }

  .demo-tip {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-align: center;
    margin-bottom: 12px;
    line-height: 1.5;

    code {
      background: #f0f2f5;
      padding: 2px 6px;
      border-radius: 4px;
      color: var(--el-color-primary);
      font-weight: bold;
    }
  }

  .demo-fast-btn {
    width: 100%;
    font-weight: 500;
  }

  .dev-quick-login {
    margin-top: 14px;

    .role-switch-title {
      font-size: 12px;
      color: var(--el-text-color-secondary);
      margin-bottom: 6px;
      text-align: center;
    }

    .quick-btns {
      display: flex;
      justify-content: space-between;
      gap: 6px;
    }
  }
}
</style>
