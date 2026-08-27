<template>
  <div class="precheck-result-wrapper">
    <!-- 總體摘要橫條 -->
    <div class="summary-banner" :class="{ 'has-errors': result.hasErrors, 'has-warnings': result.hasWarnings && !result.hasErrors, 'all-pass': !result.hasErrors && !result.hasWarnings }">
      <div class="summary-left">
        <el-icon v-if="result.hasErrors" class="icon-error"><CircleCloseFilled /></el-icon>
        <el-icon v-else-if="result.hasWarnings" class="icon-warning"><WarningFilled /></el-icon>
        <el-icon v-else class="icon-success"><SuccessFilled /></el-icon>
        <span class="summary-title">
          {{ result.hasErrors ? '檢核未通過：發現阻斷性錯誤，無法進行匯出' : (result.hasWarnings ? '檢核通過但有警告事項：請確認後再行匯出' : '檢核通過：所有資料皆符合申報規格') }}
        </span>
      </div>
      <div class="summary-counts">
        <el-tag v-if="result.summary.totalErrors > 0" type="danger">
          錯誤 {{ result.summary.totalErrors }} 項
        </el-tag>
        <el-tag v-if="result.summary.totalWarnings > 0" type="warning">
          警告 {{ result.summary.totalWarnings }} 項
        </el-tag>
        <el-tag v-if="result.summary.totalInfos > 0" type="info" effect="plain">
          提示 {{ result.summary.totalInfos }} 項
        </el-tag>
      </div>
    </div>

    <!-- 檢核項目條列清單 -->
    <div class="items-list">
      <el-collapse v-model="activeCollapse">
        <el-collapse-item
          v-for="(item, idx) in result.items"
          :key="idx"
          :name="String(idx)"
        >
          <template #title>
            <div class="item-title-row">
              <el-tag
                :type="item.level === 'error' ? 'danger' : (item.level === 'warning' ? 'warning' : 'info')"
                size="small"
                class="level-tag"
              >
                {{ item.level.toUpperCase() }}
              </el-tag>
              <span class="item-message">{{ item.message }}</span>
              <span v-if="item.details && item.details.length > 0" class="item-count">
                ({{ item.details.length }} 筆)
              </span>
            </div>
          </template>

          <div v-if="item.details && item.details.length > 0" class="details-table-wrapper">
            <el-table :data="item.details" size="small" border stripe max-height="250">
              <el-table-column v-if="item.details[0].caseName" prop="caseName" label="個案姓名" width="120" />
              <el-table-column v-if="item.details[0].serviceDate" prop="serviceDate" label="服務日期" width="110" />
              <el-table-column v-if="item.details[0].field" prop="field" label="缺漏/問題欄位" width="140" />
              <el-table-column prop="description" label="問題說明" min-width="200" />
              <el-table-column v-if="item.details[0].caseId" label="操作" width="90" align="center">
                <template #default="{ row }">
                  <el-button
                    link
                    type="primary"
                    size="small"
                    @click="$router.push(`/cases/${row.caseId}`)"
                  >
                    檢視個案
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-collapse-item>
      </el-collapse>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { CircleCloseFilled, WarningFilled, SuccessFilled } from '@element-plus/icons-vue'
import type { PrecheckResultDTO } from '@/types/api'

const props = defineProps<{
  result: PrecheckResultDTO
}>()

// 預設展開所有 error 與 warning 項目
const activeCollapse = ref<string[]>([])

watch(
  () => props.result,
  (newVal) => {
    if (newVal?.items) {
      activeCollapse.value = newVal.items
        .map((it, idx) => (it.level !== 'info' ? String(idx) : null))
        .filter(Boolean) as string[]
    }
  },
  { immediate: true }
)
</script>

<style scoped>
.precheck-result-wrapper {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.summary-banner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-radius: 6px;
  font-weight: 500;

  &.has-errors {
    background-color: var(--el-color-danger-light-9);
    border: 1px solid var(--el-color-danger-light-7);
    color: var(--el-color-danger-dark-2);
  }

  &.has-warnings {
    background-color: var(--el-color-warning-light-9);
    border: 1px solid var(--el-color-warning-light-7);
    color: var(--el-color-warning-dark-2);
  }

  &.all-pass {
    background-color: var(--el-color-success-light-9);
    border: 1px solid var(--el-color-success-light-7);
    color: var(--el-color-success-dark-2);
  }

  .summary-left {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 15px;

    .icon-error {
      color: var(--el-color-danger);
      font-size: 20px;
    }
    .icon-warning {
      color: var(--el-color-warning);
      font-size: 20px;
    }
    .icon-success {
      color: var(--el-color-success);
      font-size: 20px;
    }
  }

  .summary-counts {
    display: flex;
    gap: 6px;
  }
}

.item-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;

  .level-tag {
    font-weight: bold;
  }

  .item-message {
    font-size: 14px;
    font-weight: 500;
  }

  .item-count {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
}

.details-table-wrapper {
  padding: 4px 0;
}
</style>
