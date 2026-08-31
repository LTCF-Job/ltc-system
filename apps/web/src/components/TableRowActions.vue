<template>
  <div class="table-row-actions">
    <slot />
  </div>
</template>

<script setup lang="ts">
// 統一表格列操作按鈕的排列間距，按鈕本身仍由呼叫端依規範表決定 type/size
// 規範表見 .agents/skills/ltc-dashboard-visual-language/references/component-contract.md
</script>

<style scoped>
.table-row-actions {
  display: inline-flex;
  align-items: center;
  gap: var(--app-space-1);
  flex-wrap: nowrap;
}
</style>
