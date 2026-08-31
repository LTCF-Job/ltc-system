<template>
  <div class="holiday-calendar-view">
    <DataTablePage title="政府假日與工作日設定" :max-width="740" :loading="loading">
      <template #filter>
        <el-date-picker
          v-model="year"
          type="year"
          value-format="YYYY"
          :clearable="false"
          style="width: 130px"
        />
        <el-select
          v-model="filter"
          aria-label="行事曆篩選"
          style="width: 180px"
        >
          <el-option label="全部" value="all" />
          <el-option label="休假日" value="holidays" />
          <el-option label="手動新增的上班日" value="manual-workdays" />
        </el-select>
      </template>

      <template #actions>
        <el-button plain :loading="syncing" @click="syncYear">
          從政府行事曆匯入
        </el-button>
        <el-button type="primary" :icon="Plus" @click="openCreate">
          新增休假日／上班日
        </el-button>
      </template>

      <template #table>
        <el-alert type="info" :closable="false" show-icon style="margin-bottom: 12px;">
          休假日會套用到排班與搭乘月曆；補班日保留在清單中，但不會被視為休假。
        </el-alert>

        <el-table :data="filteredHolidays" stripe style="width: 100%">
          <el-table-column prop="holidayDate" label="日期" width="150" />
          <el-table-column prop="name" label="名稱" min-width="180" show-overflow-tooltip />
          <el-table-column label="類型" width="120">
            <template #default="{ row }">
              <el-tag :type="row.isDayOff === false ? 'warning' : 'success'">
                {{ row.isDayOff === false && row.source === 'manual' ? '上班日' : row.isDayOff === false ? '補班日' : '休假日' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="來源" width="140">
            <template #default="{ row }">
              {{ sourceLabel(row.source) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" align="center">
            <template #default="{ row }">
              <TableRowActions>
                <el-button link type="danger" size="small" @click="removeHoliday(row.holidayDate)">
                  刪除
                </el-button>
              </TableRowActions>
            </template>
          </el-table-column>
        </el-table>
      </template>
    </DataTablePage>

    <!-- 新增休假日或上班日 Dialog -->
    <el-dialog
      v-model="createVisible"
      title="新增休假日或上班日"
      width="min(480px, calc(100vw - 32px))"
      destroy-on-close
    >
      <el-form label-width="90px">
        <el-form-item label="日期">
          <el-date-picker
            v-model="form.holidayDate"
            type="date"
            value-format="YYYY-MM-DD"
            placeholder="請選擇日期"
            style="width: 100%"
          />
        </el-form-item>

        <el-form-item label="類型">
          <el-radio-group v-model="form.isDayOff">
            <el-radio :value="true">休假日</el-radio>
            <el-radio :value="false">上班日</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="名稱">
          <el-input
            v-model="form.name"
            placeholder="請輸入或點選下方快捷名稱"
            clearable
          />
          <div class="quick-names-wrap">
            <span class="quick-names-label">快捷選取：</span>
            <div class="quick-names-tags">
              <el-tag
                v-for="name in commonNameOptions"
                :key="name"
                class="quick-name-tag"
                size="small"
                effect="plain"
                @click="form.name = name"
              >
                {{ name }}
              </el-tag>
            </div>
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <DialogFooter :loading="saving" @confirm="create" @cancel="createVisible = false" />
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import DataTablePage from '@/components/DataTablePage.vue'
import DialogFooter from '@/components/DialogFooter.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import {
  createHoliday,
  deleteHoliday,
  importGovHolidays,
  listHolidays,
  type HolidayItem
} from '@/api/holidays'

type HolidayFilter = 'all' | 'holidays' | 'manual-workdays'

const year = ref(String(new Date().getFullYear()))
const filter = ref<HolidayFilter>('all')
const holidays = ref<HolidayItem[]>([])
const loading = ref(false)
const syncing = ref(false)
const saving = ref(false)
const createVisible = ref(false)

const form = reactive({
  holidayDate: '',
  name: '',
  isDayOff: true
})

const range = computed(() => ({
  startDate: `${year.value}-01-01`,
  endDate: `${year.value}-12-31`
}))

const filteredHolidays = computed(() =>
  holidays.value.filter(
    (holiday) =>
      filter.value === 'all' ||
      (filter.value === 'holidays' && holiday.isDayOff) ||
      (filter.value === 'manual-workdays' && holiday.source === 'manual' && !holiday.isDayOff)
  )
)

const commonNameOptions = computed(() =>
  form.isDayOff
    ? ['春節', '元旦', '和平紀念日', '清明節', '兒童節', '端午節', '中秋節', '國慶日', '颱風停班停課', '自訂公休', '補假']
    : ['補行上班日', '調整上班日', '彈性上班日', '專案出勤日']
)

function sourceLabel(source: string) {
  return source === 'gov_calendar' ? '政府行事曆' : source === 'manual' ? '手動加入' : source
}

async function load() {
  loading.value = true
  try {
    holidays.value = (await listHolidays(range.value)).data || []
  } finally {
    loading.value = false
  }
}

async function syncYear() {
  syncing.value = true
  try {
    const result = await importGovHolidays(Number(year.value))
    ElMessage.success(`已匯入 ${result.data?.importedCount || 0} 筆行事曆資料`)
    await load()
  } finally {
    syncing.value = false
  }
}

function openCreate() {
  form.holidayDate = ''
  form.name = ''
  form.isDayOff = true
  createVisible.value = true
}

async function create() {
  if (!form.holidayDate || !form.name.trim()) {
    return ElMessage.warning('請填寫日期與名稱')
  }
  saving.value = true
  try {
    await createHoliday({
      holidayDate: form.holidayDate,
      name: form.name.trim(),
      source: 'manual',
      isDayOff: form.isDayOff
    })
    createVisible.value = false
    await load()
    ElMessage.success(form.isDayOff ? '休假日已儲存' : '上班日已儲存')
  } finally {
    saving.value = false
  }
}

async function removeHoliday(date: string) {
  await ElMessageBox.confirm(`確定刪除 ${date} 的假日設定嗎？`, '請確認', {
    type: 'warning'
  })
  await deleteHoliday(date)
  await load()
  ElMessage.success('假日設定已刪除')
}

watch(year, load)
onMounted(load)
</script>

<style scoped>
.holiday-calendar-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.quick-names-wrap {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.quick-names-label {
  font-size: var(--app-font-xs);
  color: var(--el-text-color-secondary);
}

.quick-names-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.quick-name-tag {
  cursor: pointer;
  user-select: none;
  transition: all 0.15s ease;

  &:hover {
    color: var(--el-color-primary);
    border-color: var(--el-color-primary-light-5);
    background-color: var(--el-color-primary-light-9);
  }
}
</style>
