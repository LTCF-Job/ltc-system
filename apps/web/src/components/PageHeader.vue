<template>
  <div class="page-header">
    <div class="page-header-text">
      <h1 class="page-header-title">{{ title }}</h1>
      <p v-if="description" class="page-header-description">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="page-header-actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()
</script>

<style scoped>
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--app-space-4);
  margin-bottom: var(--app-space-4);
}

.page-header-text {
  min-width: 0;
}

.page-header-title {
  font-size: var(--app-font-xl);
  font-weight: 600;
  color: var(--app-text-primary);
}

.page-header-description {
  margin-top: var(--app-space-1);
  font-size: var(--app-font-sm);
  color: var(--app-text-secondary);
}

.page-header-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--app-space-2);
  flex-shrink: 0;
}
</style>
