<template>
  <div class="panel">
    <el-card shadow="never" class="settings-card">
      <template #header>
        <span class="card-title">據點趟數彙總表</span>
      </template>

      <el-alert type="info" show-icon :closable="false" class="scope-hint">
        本表為管理用趟數統計：列出所選據點底下每一位個案在各月份的搭乘趟數與跨月加總。
        產出即時下載、不留匯出紀錄，也不是政府申報格式。
      </el-alert>

      <el-form label-width="140px" :disabled="!canEdit" class="settings-form">
        <el-row :gutter="16">
          <el-col :xs="24" :sm="12">
            <el-form-item label="據點">
              <el-select
                v-model="selectedSiteIds"
                multiple
                filterable
                collapse-tags
                collapse-tags-tooltip
                placeholder="選擇據點（可多選）"
                style="width: 100%"
                :loading="loadingSites"
              >
                <el-option
                  v-for="site in siteOptions"
                  :key="site.id"
                  :label="site.region ? `${site.name}（${site.region}）` : site.name"
                  :value="site.id"
                />
              </el-select>
            </el-form-item>
          </el-col>

          <el-col :xs="24" :sm="12">
            <el-form-item label="統計月份 (民國)">
              <div class="month-picker">
                <el-date-picker
                  v-model="selectedMonths"
                  type="months"
                  format="YYYY-MM"
                  value-format="YYYY-MM"
                  placeholder="選擇月份（可多選）"
                  style="width: 100%"
                />
                <span class="month-summary">{{ summary }}</span>
              </div>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="16">
          <el-col :span="24">
            <div v-if="canEdit" class="action-buttons">
              <el-button type="primary" :loading="downloading" @click="handleDownload">
                <el-icon><Download /></el-icon>
                下載趟數彙總表
              </el-button>
            </div>
          </el-col>
        </el-row>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Download } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { listAllSites } from '@/api/masters'
import { downloadSiteTripSummary } from '@/api/exports'
import { useRocMonthRange } from '@/composables/useRocMonthRange'
import { downloadBlob } from '@/utils/download'
import type { SiteDTO } from '@/types/api'

defineProps<{ canEdit: boolean }>()

const { selectedMonths, periodYms, summary } = useRocMonthRange()

const allSites = ref<SiteDTO[]>([])
const loadingSites = ref(false)
const selectedSiteIds = ref<string[]>([])
const downloading = ref(false)

// 據點依區域再依名稱排序，讓同一個區域的據點在選單裡排在一起。
const siteOptions = computed(() =>
  [...allSites.value].sort((a, b) => {
    const regionDiff = (a.region || '').localeCompare(b.region || '', 'zh-Hant')
    if (regionDiff !== 0) return regionDiff
    return a.name.localeCompare(b.name, 'zh-Hant')
  })
)

async function fetchSites() {
  loadingSites.value = true
  try {
    allSites.value = await listAllSites()
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  } finally {
    loadingSites.value = false
  }
}

async function handleDownload() {
  if (selectedSiteIds.value.length === 0) {
    ElMessage.warning('請先選擇要統計的據點')
    return
  }
  if (periodYms.value.length === 0) {
    ElMessage.warning('請先選擇要統計的月份')
    return
  }

  downloading.value = true
  try {
    const blob = await downloadSiteTripSummary({
      siteIds: [...selectedSiteIds.value],
      periodYms: [...periodYms.value]
    })
    downloadBlob(blob, buildFileName(periodYms.value))
    ElMessage.success('趟數彙總表下載成功')
  } catch {
    // 全域攔截器負責顯示 API 錯誤（含所選據點底下沒有個案）。
  } finally {
    downloading.value = false
  }
}

// 檔名與後端一致，讓使用者在下載清單裡認得出這是同一份檔案。
function buildFileName(months: string[]): string {
  if (months.length === 0) return 'site-trip-summary.xlsx'
  const first = months[0]
  const last = months[months.length - 1]
  return first === last
    ? `site-trip-summary-${first}.xlsx`
    : `site-trip-summary-${first}-${last}.xlsx`
}

onMounted(fetchSites)
</script>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.settings-card {
  border-radius: 8px;
}

.card-title {
  font-size: 16px;
  font-weight: bold;
  color: var(--app-primary);
}

.scope-hint {
  margin-bottom: 16px;
}

.month-picker {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 100%;

  .month-summary {
    color: var(--app-text-secondary);
    font-size: 13px;
  }
}

.action-buttons {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 12px;
}

@media (max-width: 640px) {
  .action-buttons {
    justify-content: flex-start;
  }
}
</style>
