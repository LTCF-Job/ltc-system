<template>
  <div class="dialog-footer">
    <el-button @click="$emit('cancel')">{{ cancelText }}</el-button>
    <el-button
      v-if="showConfirm"
      :type="confirmType"
      :loading="loading"
      :disabled="confirmDisabled"
      @click="$emit('confirm')"
    >
      {{ confirmText }}
    </el-button>
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    confirmText?: string
    cancelText?: string
    confirmType?: 'primary' | 'danger'
    loading?: boolean
    confirmDisabled?: boolean
    showConfirm?: boolean
  }>(),
  {
    confirmText: '儲存',
    cancelText: '取消',
    confirmType: 'primary',
    loading: false,
    confirmDisabled: false,
    showConfirm: true
  }
)

defineEmits<{
  (e: 'confirm'): void
  (e: 'cancel'): void
}>()
</script>

<style scoped>
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--app-space-2);
}
</style>
