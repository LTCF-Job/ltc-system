<template>
  <div class="holiday-calendar-view">
    <el-card shadow="never">
      <template #header><div class="toolbar"><span class="title">政府假日與工作日設定</span><div class="actions"><el-date-picker v-model="year" type="year" value-format="YYYY" :clearable="false" /><el-select v-model="filter" aria-label="行事曆篩選" style="width: 180px"><el-option label="全部" value="all" /><el-option label="休假日" value="holidays" /><el-option label="手動新增的上班日" value="manual-workdays" /></el-select><el-button type="primary" :loading="syncing" @click="syncYear">從政府行事曆匯入</el-button><el-button @click="openCreate">新增休假日／上班日</el-button></div></div></template>
      <el-alert type="info" :closable="false" show-icon>休假日會套用到排班與搭乘月曆；補班日保留在清單中，但不會被視為休假。</el-alert>
      <el-table v-loading="loading" :data="filteredHolidays" stripe style="width: 100%; margin-top: 16px"><el-table-column prop="holidayDate" label="日期" width="150" /><el-table-column prop="name" label="名稱" min-width="180" /><el-table-column label="類型" width="120"><template #default="{ row }"><el-tag :type="row.isDayOff === false ? 'warning' : 'success'">{{ row.isDayOff === false && row.source === 'manual' ? '上班日' : row.isDayOff === false ? '補班日' : '休假日' }}</el-tag></template></el-table-column><el-table-column label="來源" width="140"><template #default="{ row }">{{ sourceLabel(row.source) }}</template></el-table-column><el-table-column label="操作" width="100" align="center"><template #default="{ row }"><el-button link type="danger" @click="removeHoliday(row.holidayDate)">刪除</el-button></template></el-table-column></el-table>
    </el-card>
    <el-dialog v-model="createVisible" title="新增休假日或上班日" width="420px"><el-form label-width="90px"><el-form-item label="日期"><el-date-picker v-model="form.holidayDate" type="date" value-format="YYYY-MM-DD" /></el-form-item><el-form-item label="類型"><el-radio-group v-model="form.isDayOff"><el-radio :value="true">休假日</el-radio><el-radio :value="false">上班日</el-radio></el-radio-group></el-form-item><el-form-item label="名稱"><el-select v-model="form.name" filterable allow-create default-first-option placeholder="選擇或輸入名稱" style="width: 100%"><el-option v-for="name in commonNameOptions" :key="name" :label="name" :value="name" /></el-select></el-form-item></el-form><template #footer><el-button @click="createVisible = false">取消</el-button><el-button type="primary" :loading="saving" @click="create">儲存</el-button></template></el-dialog>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { createHoliday, deleteHoliday, importGovHolidays, listHolidays, type HolidayItem } from '@/api/holidays'
type HolidayFilter = 'all' | 'holidays' | 'manual-workdays'
const year = ref(String(new Date().getFullYear())); const filter = ref<HolidayFilter>('all'); const holidays = ref<HolidayItem[]>([]); const loading = ref(false); const syncing = ref(false); const saving = ref(false); const createVisible = ref(false); const form = reactive({ holidayDate: '', name: '', isDayOff: true }); const range = computed(() => ({ startDate: `${year.value}-01-01`, endDate: `${year.value}-12-31` }))
const filteredHolidays = computed(() => holidays.value.filter((holiday) => filter.value === 'all' || (filter.value === 'holidays' && holiday.isDayOff) || (filter.value === 'manual-workdays' && holiday.source === 'manual' && !holiday.isDayOff)))
const commonNameOptions = computed(() => form.isDayOff ? ['元旦', '春節', '和平紀念日', '兒童節', '清明節', '勞動節', '端午節', '中秋節', '國慶日', '行憲紀念日', '颱風停班停課'] : ['補行上班日', '調整上班日', '彈性上班日'])
function sourceLabel(source: string) { return source === 'gov_calendar' ? '政府行事曆' : source === 'manual' ? '手動加入' : source }
async function load() { loading.value = true; try { holidays.value = (await listHolidays(range.value)).data || [] } finally { loading.value = false } }
async function syncYear() { syncing.value = true; try { const result = await importGovHolidays(Number(year.value)); ElMessage.success(`已匯入 ${result.data?.importedCount || 0} 筆行事曆資料`); await load() } finally { syncing.value = false } }
function openCreate() { form.holidayDate = ''; form.name = ''; form.isDayOff = true; createVisible.value = true }
async function create() { if (!form.holidayDate || !form.name.trim()) return ElMessage.warning('請填寫日期與名稱'); saving.value = true; try { await createHoliday({ holidayDate: form.holidayDate, name: form.name, source: 'manual', isDayOff: form.isDayOff }); createVisible.value = false; await load(); ElMessage.success(form.isDayOff ? '休假日已儲存' : '上班日已儲存') } finally { saving.value = false } }
async function removeHoliday(date: string) { await ElMessageBox.confirm(`確定刪除 ${date} 的假日設定嗎？`, '請確認', { type: 'warning' }); await deleteHoliday(date); await load(); ElMessage.success('假日設定已刪除') }
watch(year, load); onMounted(load)
</script>
<style scoped>.holiday-calendar-view { display: flex; flex-direction: column; gap: 16px; }.toolbar, .actions { display: flex; align-items: center; gap: 12px; }.toolbar { justify-content: space-between; flex-wrap: wrap; }.title { font-size: 16px; font-weight: 600; }</style>
