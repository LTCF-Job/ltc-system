<template>
  <div class="data-table-page">
    <!-- 頂部查詢與操作列 -->
    <el-card v-if="$slots.filter || $slots.actions" class="filter-card" shadow="never">
      <el-row :gutter="16" justify="space-between" align="middle">
        <el-col :span="18">
          <div class="filter-wrapper">
            <slot name="filter" />
          </div>
        </el-col>
        <el-col :span="6" class="actions-col">
          <div class="actions-wrapper">
            <slot name="actions" />
          </div>
        </el-col>
      </el-row>
    </el-card>

    <!-- 資料表格區塊 -->
    <el-card class="table-card" shadow="never">
      <div v-loading="loading" class="table-container">
        <slot name="table" />
      </div>

      <!-- 分頁器 -->
      <div v-if="(total || 0) > 0" class="pagination-container">
        <el-pagination
          :current-page="page || 1"
          :page-size="pageSize || 20"
          :total="total || 0"
          :page-sizes="[10, 20, 50, 100]"
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
defineProps<{
  loading?: boolean
  total?: number
  page?: number
  pageSize?: number
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
}

.filter-card {
  border-radius: 8px;
}

.table-card {
  border-radius: 8px;
}

.filter-wrapper {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}

.actions-col {
  display: flex;
  justify-content: flex-end;
}

.actions-wrapper {
  display: flex;
  gap: 8px;
}

.table-container {
  min-height: 200px;
}

.pagination-container {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
