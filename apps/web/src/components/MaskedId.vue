<template>
  <span class="masked-id-container">
    <span class="id-text">{{ revealedId || maskedValue }}</span>
    <el-button
      v-if="canReveal && !revealedId"
      link
      type="primary"
      size="small"
      :loading="loading"
      class="reveal-btn"
      @click="handleReveal"
    >
      <el-icon><View /></el-icon>
      顯示完整
    </el-button>
    <el-tag v-if="revealedId" size="small" type="info" class="timer-tag">
      {{ remainingSec }}s
    </el-tag>
  </span>
</template>

<script setup lang="ts">
import { ref, onUnmounted } from 'vue'
import { View } from '@element-plus/icons-vue'

// 共用元件不綁定特定模組，是否可揭露交由使用它的頁面依自身模組的 edit 權限判斷後傳入
const props = defineProps<{
  maskedValue: string
  canReveal?: boolean
  onReveal?: () => Promise<string>
}>()

const canReveal = props.canReveal ?? false
const revealedId = ref<string | null>(null)
const loading = ref(false)
const remainingSec = ref(30)
let timer: ReturnType<typeof setInterval> | null = null

async function handleReveal() {
  if (!props.onReveal) return
  loading.value = true
  try {
    const plainId = await props.onReveal()
    revealedId.value = plainId
    remainingSec.value = 30

    // 30 秒自動回復遮罩狀態
    timer = setInterval(() => {
      remainingSec.value--
      if (remainingSec.value <= 0) {
        resetMask()
      }
    }, 1000)
  } finally {
    loading.value = false
  }
}

function resetMask() {
  revealedId.value = null
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}

onUnmounted(() => {
  resetMask()
})
</script>

<style scoped>
.masked-id-container {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: monospace;
}

.reveal-btn {
  padding: 0;
  font-size: var(--app-font-xs);
}

.timer-tag {
  font-size: var(--app-font-xs);
}
</style>
