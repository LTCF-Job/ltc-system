<template>
  <main class="forbidden-view" aria-labelledby="forbidden-title">
    <el-card shadow="never" class="forbidden-card">
      <el-icon class="forbidden-icon" aria-hidden="true"><WarningFilled /></el-icon>
      <h1 id="forbidden-title">沒有存取權限</h1>
      <p>目前帳號沒有檢視這個功能區塊的權限。若你認為這是錯誤，請聯絡系統管理員。</p>
      <el-button type="primary" @click="router.back">返回上一頁</el-button>
    </el-card>
  </main>
</template>

<script setup lang="ts">
import { WarningFilled } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<style scoped>
.forbidden-view {
  display: flex;
  justify-content: center;
  padding: var(--app-space-8);
}

.forbidden-card {
  width: min(560px, 100%);
  text-align: center;
}

.forbidden-icon {
  color: var(--app-status-warning-fg);
  font-size: 48px;
}

h1 {
  margin: var(--app-space-4) 0 var(--app-space-2);
  color: var(--app-text-primary);
  font-size: var(--app-font-xl);
}

p {
  margin: 0 0 var(--app-space-6);
  color: var(--app-text-secondary);
}
</style>
