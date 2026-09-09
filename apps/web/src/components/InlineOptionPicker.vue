<template>
  <el-popover
    v-model:visible="visible"
    placement="bottom-start"
    trigger="click"
    :width="240"
    :disabled="disabled"
    popper-class="inline-picker-popper"
    :show-arrow="false"
  >
    <template #reference>
      <button
        type="button"
        class="inline-picker-trigger"
        :class="{ 'is-open': visible, 'is-disabled': disabled }"
        :disabled="disabled"
      >
        <span class="inline-picker-value" :class="{ 'is-empty': !summaryText }">
          {{ summaryText || placeholder }}
        </span>
        <el-icon v-if="loading" class="inline-picker-icon is-loading"><Loading /></el-icon>
        <el-icon v-else class="inline-picker-icon"><ArrowDown /></el-icon>
      </button>
    </template>

    <div class="inline-picker-panel">
      <el-input
        v-if="multiple && searchable"
        v-model="query"
        placeholder="搜尋"
        clearable
        class="inline-picker-search"
      />
      <div class="inline-picker-list">
        <template v-if="multiple">
          <el-checkbox
            v-for="opt in filteredOptions"
            :key="opt.value"
            :model-value="pendingMultiValue.includes(opt.value)"
            class="inline-picker-item"
            @change="(checked) => toggleValue(opt.value, !!checked)"
          >
            {{ opt.label }}
          </el-checkbox>
          <div v-if="!filteredOptions.length" class="inline-picker-empty">查無符合選項</div>
        </template>
        <template v-else>
          <div
            v-if="clearable"
            class="inline-picker-item inline-picker-item--radio inline-picker-item--clear"
            :class="{ 'is-active': !modelValue }"
            @click="selectSingle('')"
          >
            未指定
          </div>
          <div
            v-for="opt in options"
            :key="opt.value"
            class="inline-picker-item inline-picker-item--radio"
            :class="{ 'is-active': modelValue === opt.value }"
            @click="selectSingle(opt.value)"
          >
            {{ opt.label }}
          </div>
          <div v-if="allowCreate" class="inline-picker-create">
            <el-input
              v-model="createText"
              size="small"
              placeholder="輸入自訂名稱"
              @keyup.enter="handleCreate"
            />
            <el-button size="small" type="primary" text @click="handleCreate">新增</el-button>
          </div>
        </template>
      </div>
    </div>
  </el-popover>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ArrowDown, Loading } from '@element-plus/icons-vue'

interface PickerOption {
  label: string
  value: string
}

const props = withDefaults(
  defineProps<{
    modelValue: string | string[]
    options: PickerOption[]
    multiple?: boolean
    searchable?: boolean
    allowCreate?: boolean
    clearable?: boolean
    placeholder?: string
    disabled?: boolean
    loading?: boolean
  }>(),
  {
    multiple: false,
    searchable: true,
    allowCreate: false,
    clearable: false,
    placeholder: '未選擇',
    disabled: false,
    loading: false
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: string | string[]]
  change: [value: string | string[]]
}>()

const visible = ref(false)
const query = ref('')
const createText = ref('')

// 多選勾選只改本地暫存，避免每點一次就送一次 API；實際送出動作集中在面板關閉時
const pendingMultiValue = ref<string[]>(Array.isArray(props.modelValue) ? [...props.modelValue] : [])

watch(
  () => props.modelValue,
  (val) => {
    if (!visible.value) {
      pendingMultiValue.value = Array.isArray(val) ? [...val] : []
    }
  }
)

const filteredOptions = computed(() => {
  if (!query.value.trim()) return props.options
  const q = query.value.trim().toLowerCase()
  return props.options.filter((opt) => opt.label.toLowerCase().includes(q))
})

const summaryText = computed(() => {
  if (props.multiple) {
    return pendingMultiValue.value
      .map((v) => props.options.find((opt) => opt.value === v)?.label)
      .filter((label): label is string => !!label)
      .join('、')
  }
  const value = props.modelValue as string
  return props.options.find((opt) => opt.value === value)?.label || value || ''
})

function toggleValue(value: string, checked: boolean) {
  pendingMultiValue.value = checked
    ? [...pendingMultiValue.value, value]
    : pendingMultiValue.value.filter((v) => v !== value)
}

function selectSingle(value: string) {
  emit('update:modelValue', value)
  emit('change', value)
  visible.value = false
}

function handleCreate() {
  const text = createText.value.trim()
  if (!text) return
  emit('update:modelValue', text)
  emit('change', text)
  createText.value = ''
  visible.value = false
}

watch(visible, (isVisible, wasVisible) => {
  if (isVisible) {
    pendingMultiValue.value = Array.isArray(props.modelValue) ? [...props.modelValue] : []
    return
  }
  // 只在多選面板從開到關的當下才送出，滑鼠離開才觸發一次 API，不是每次勾選都送
  if (wasVisible && props.multiple) {
    const original = Array.isArray(props.modelValue) ? props.modelValue : []
    const changed =
      pendingMultiValue.value.length !== original.length ||
      pendingMultiValue.value.some((v) => !original.includes(v))
    if (changed) {
      emit('update:modelValue', pendingMultiValue.value)
      emit('change', pendingMultiValue.value)
    }
  }
  query.value = ''
  createText.value = ''
})
</script>

<style scoped>
.inline-picker-trigger {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  padding: 5px 8px 5px 6px;
  border-radius: var(--app-radius-xs);
  border: 1px solid transparent;
  background: transparent;
  color: var(--app-text-regular);
  font-size: var(--app-font-sm);
  font-family: inherit;
  cursor: pointer;
  transition: background-color 0.14s ease, border-color 0.14s ease, box-shadow 0.14s ease;
}

.inline-picker-trigger:hover:not(.is-disabled) {
  background: var(--app-border-light);
}

.inline-picker-trigger.is-open {
  background: var(--app-surface);
  border-color: var(--app-primary);
  box-shadow: 0 0 0 3px rgba(12, 112, 237, 0.12);
}

.inline-picker-trigger.is-disabled {
  cursor: not-allowed;
  color: var(--app-text-muted);
}

.inline-picker-value {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.inline-picker-value.is-empty {
  color: var(--app-text-muted);
}

.inline-picker-icon {
  font-size: 11px;
  color: var(--app-text-muted);
  flex-shrink: 0;
}

.inline-picker-icon.is-loading {
  animation: inline-picker-spin 1s linear infinite;
}

@keyframes inline-picker-spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.inline-picker-panel {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.inline-picker-search :deep(.el-input__wrapper) {
  padding: 8px 12px;
}

.inline-picker-search :deep(.el-input__inner) {
  font-size: var(--app-font-lg);
  height: 24px;
  line-height: 24px;
}

.inline-picker-list {
  display: flex;
  flex-direction: column;
  max-height: 220px;
  overflow-y: auto;
}

.inline-picker-item {
  display: flex;
  align-items: center;
  padding: 6px 8px;
  border-radius: var(--app-radius-xs);
  font-size: var(--app-font-sm);
}

.inline-picker-item--radio {
  cursor: pointer;
  color: var(--app-text-regular);
}

.inline-picker-item--radio:hover {
  background: var(--app-primary-light);
}

.inline-picker-item--radio.is-active {
  color: var(--app-primary);
  font-weight: 600;
  background: var(--app-primary-light);
}

.inline-picker-item--clear {
  color: var(--app-text-muted);
  border-bottom: 1px solid var(--app-border-light);
  border-radius: 0;
  margin-bottom: 4px;
}

.inline-picker-empty {
  padding: 10px 8px;
  color: var(--app-text-muted);
  font-size: var(--app-font-xs);
  text-align: center;
}

.inline-picker-create {
  display: flex;
  gap: 6px;
  padding-top: 6px;
  margin-top: 4px;
  border-top: 1px solid var(--app-border-light);
}

.inline-picker-create .el-input {
  flex: 1;
}
</style>
