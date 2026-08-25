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

      <!-- 展示模式快速身分切換 -->
      <div class="dev-quick-login">
        <el-divider>
          <span class="divider-tag">✨ 展示模式快速登入</span>
        </el-divider>
        <p class="demo-tip">系統已預載全類型個案、排班、搭乘矩陣、異常裁決與報表資料：</p>
        <div class="quick-btns">
          <el-button size="default" type="danger" plain @click="quickLogin('admin')">
            系統管理員 (Admin)
          </el-button>
          <el-button size="default" type="primary" plain @click="quickLogin('staff')">
            行政人員 (Staff)
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
import type { UserRole } from '@/types/domain'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({
  email: 'admin@ltc.example.com',
  password: 'password123'
})

const rules = {
  email: [{ required: true, message: '請輸入電子郵件', trigger: 'blur' }],
  password: [{ required: true, message: '請輸入密碼', trigger: 'blur' }]
}

async function handleLogin() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    loading.value = true
    try {
      // 依帳號決定展示角色
      let role: UserRole = 'staff'
      if (form.email.includes('admin')) role = 'admin'
      else if (form.email.includes('viewer')) role = 'viewer'

      authStore.setSession(`mock_jwt_${role}`, {
        id: `usr_${role}`,
        email: form.email,
        displayName: role === 'admin' ? '系統管理員' : (role === 'staff' ? '承辦行政' : '主管檢視者'),
        role
      })

      ElMessage.success('登入成功')
      const redirect = (route.query.redirect as string) || '/'
      router.push(redirect)
    } finally {
      loading.value = false
    }
  })
}

function quickLogin(role: UserRole) {
  authStore.setSession(`mock_jwt_${role}`, {
    id: `usr_${role}`,
    email: `${role}@ltc.example.com`,
    displayName: role === 'admin' ? '系統管理員' : (role === 'staff' ? '承辦行政' : '主管檢視者'),
    role
  })
  ElMessage.success(`已快速切換為【${role === 'admin' ? '系統管理員' : (role === 'staff' ? '行政人員' : '檢視者')}】`)
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
