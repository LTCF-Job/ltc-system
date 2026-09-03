<template>
  <el-tag
    v-if="variant === 'tag'"
    :type="tagType"
    size="small"
    :class="{ 'is-neutral': entry.variant === 'neutral' }"
  >
    {{ entry.label }}
  </el-tag>
  <span v-else-if="variant === 'chip'" class="status-chip" :class="`status-chip-${entry.variant}`">
    <span class="status-dot" :class="`status-dot-${entry.variant}`" />
    {{ entry.label }}
  </span>
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
    variant?: 'tag' | 'dot' | 'chip'
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

/* chip：淡色圓角底 + 小圓點，用於表格狀態欄但不想要 el-tag 的方角觀感；
   邊框跟 el-tag／status-toggle-pill 用同一套淡色描邊語言，三者觀感統一 */
.status-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 2px 9px;
  border-radius: var(--app-radius-full);
  border: 1px solid transparent;
  font-size: var(--app-font-xs);
  font-weight: 700;
  line-height: 1.6;
  white-space: nowrap;
}

.status-chip .status-dot {
  width: 6px;
  height: 6px;
}

.status-chip-success {
  background: var(--app-status-success-bg);
  border-color: #a8e6cd;
  color: var(--app-status-success-fg);
}

.status-chip-warning {
  background: var(--app-status-warning-bg);
  border-color: #f0cf8e;
  color: var(--app-status-warning-fg);
}

.status-chip-danger {
  background: var(--app-status-danger-bg);
  border-color: #f3a8b6;
  color: var(--app-status-danger-fg);
}

.status-chip-info {
  background: var(--app-status-info-bg);
  border-color: #a9d3e8;
  color: var(--app-status-info-fg);
}

.status-chip-neutral {
  background: var(--app-status-neutral-bg);
  border-color: #dde2e8;
  color: var(--app-status-neutral-fg);
}
</style>
