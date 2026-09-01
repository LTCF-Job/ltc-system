<template>
  <div class="data-table-page" :style="maxWidth ? { maxWidth: `${maxWidth}px` } : undefined">
    <PageHeader v-if="title" :title="title" :description="description">
      <template v-if="$slots.header" #actions>
        <slot name="header" />
      </template>
    </PageHeader>
    <!-- 頂部查詢與操作列 -->
    <el-card v-if="$slots.filter || $slots.actions" class="filter-card" shadow="never">
      <div class="filter-header-container">
        <div v-if="$slots.filter" class="filter-wrapper">
          <slot name="filter" />
        </div>
        <div v-if="$slots.actions" class="actions-wrapper">
          <slot name="actions" />
        </div>
      </div>
    </el-card>

    <!-- 資料表格區塊 -->
    <el-card class="table-card" shadow="never">
      <div
        v-loading="!!loading"
        class="table-container"
        :class="{ 'table-container--capped': !!maxWidth }"
        :aria-busy="loading ? 'true' : 'false'"
      >
        <slot name="table" />
      </div>

      <!-- 分頁器 -->
      <div v-if="(total || 0) > 0" class="pagination-container">
        <el-pagination
          :current-page="page || 1"
          :page-size="pageSize || 20"
          :total="total || 0"
          :page-sizes="pageSizes || [10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @update:current-page="$emit('update:page', $event)"
          @update:page-size="$emit('update:pageSize', $event)"
          @current-change="$emit('page-change', $event)"
          @size-change="$emit('size-change', $event)"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import PageHeader from '@/components/PageHeader.vue'

defineProps<{
  title?: string
  description?: string
  loading?: boolean
  total?: number
  page?: number
  pageSize?: number
  pageSizes?: number[]
  // 欄位少、內容短的主檔列表可傳入頁面實際欄寬總和（含操作欄，建議加 30px 緩衝），避免整頁被硬撐滿
  maxWidth?: number
}>()

defineEmits<{
  (e: 'update:page', val: number): void
  (e: 'update:pageSize', val: number): void
  (e: 'page-change', val: number): void
  (e: 'size-change', val: number): void
}>()
</script>

<style scoped>
.data-table-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
  min-width: 0;
}

.filter-card {
  border-radius: var(--app-radius-md, 12px);
  border: 1px solid var(--app-border-color, #e2e8f0);
  background: var(--app-card-bg, #f7f9fa);

  :deep(.el-card__body) {
    padding: 14px 16px;
  }
}

.table-card {
  border-radius: var(--app-radius-md, 12px);
  border: 1px solid var(--app-border-color, #e2e8f0);
  min-width: 0;
}

.filter-header-container {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
}

.filter-wrapper {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  flex: 1;
  min-width: 280px;
}

.actions-wrapper {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  justify-content: flex-end;
  flex-shrink: 0;
}

.table-container {
  min-height: 200px;
  min-width: 0;
  width: 100%;
  overflow-x: auto;
  overflow-y: hidden;
}

/* 只有帶 max-width（欄寬已依內容算好）的表格才鎖 min-width:max-content 讓表格內縮版滾動；
   沒有 max-width 的表格要讓欄位隨版面伸展，鎖住反而會在版面還有空間時擠出多餘的橫向卷軸 */
.table-container--capped {
  :deep(.el-table) {
    min-width: max-content;
    max-width: none;
  }
}

.pagination-container {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 12px;
  border-top: 1px solid var(--app-border-light, #f0f2f4);
  padding-top: 16px;
  margin-top: 16px;
}

@media (max-width: 720px) {
  .filter-header-container,
  .filter-wrapper,
  .actions-wrapper {
    align-items: stretch;
    width: 100%;
  }

  .filter-wrapper,
  .actions-wrapper {
    min-width: 0;
  }

  .actions-wrapper {
    justify-content: flex-start;
  }

  .pagination-container {
    justify-content: flex-start;
  }
}
</style>
