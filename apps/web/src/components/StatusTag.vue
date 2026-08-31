<template>
  <el-tag
    v-if="variant === 'tag'"
    :type="tagType"
    size="small"
    :class="{ 'is-neutral': entry.variant === 'neutral' }"
  >
    {{ entry.label }}
  </el-tag>
  <span v-else class="status-dot-group">
    <span class="status-dot" :class="`status-dot-${entry.variant}`" />
    {{ entry.label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { resolveStatusEntry, type StatusPresetName } from '@/lib/statusPresets'

const props = withDefaults(
  defineProps<{
    status: string
    preset: StatusPresetName
    variant?: 'tag' | 'dot'
  }>(),
  {
    variant: 'tag'
  }
)

const entry = computed(() => resolveStatusEntry(props.preset, props.status))

// el-tag 沒有 neutral type，先給 info 佔位，實際顏色由 .is-neutral 覆寫成灰色
const tagType = computed(() => (entry.value.variant === 'neutral' ? 'info' : entry.value.variant))
</script>

<style scoped>
.status-dot-group {
  display: inline-flex;
  align-items: center;
  gap: var(--app-space-1);
  font-size: var(--app-font-sm);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot-success {
  background: var(--app-status-success-fg);
}

.status-dot-warning {
  background: var(--app-status-warning-fg);
}

.status-dot-danger {
  background: var(--app-status-danger-fg);
}

.status-dot-info {
  background: var(--app-status-info-fg);
}

.status-dot-neutral {
  background: var(--app-text-muted);
}

.el-tag.is-neutral {
  background: var(--app-status-neutral-bg) !important;
  color: var(--app-status-neutral-fg) !important;
}
</style>
