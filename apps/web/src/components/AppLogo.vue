<template>
  <div class="app-logo" :class="[`app-logo--${size}`, `app-logo--${variant}`, { 'is-collapsed': collapsed }]">
    <img src="/project-logo.png" alt="" class="app-logo__image" />
    <span v-if="showText && !collapsed" class="app-logo__text">
      <span class="app-logo__name">好安心關懷協會</span>
      <span class="app-logo__system">後臺系統</span>
    </span>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  size?: 'sm' | 'md' | 'xl'
  collapsed?: boolean
  showText?: boolean
  variant?: 'dark' | 'light'
}>(), {
  size: 'md',
  collapsed: false,
  showText: true,
  variant: 'dark'
})
</script>

<style scoped>
.app-logo {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.app-logo__image {
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  object-fit: contain;
}

.app-logo--sm .app-logo__image {
  width: 32px;
  height: 32px;
}

.app-logo--xl .app-logo__image {
  width: 160px;
  height: 160px;
}

.app-logo__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
  white-space: nowrap;
}

.app-logo__name {
  color: #ffffff;
  font-size: 13.5px;
  font-weight: 700;
  line-height: 1.3;
}

.app-logo__system {
  color: #8faec8;
  font-size: 11px;
  font-weight: 500;
}

.app-logo--light .app-logo__name {
  color: var(--app-text-primary);
}

.app-logo--light .app-logo__system {
  color: var(--app-text-secondary);
}

.app-logo--xl .app-logo__text {
  display: none;
}
</style>
