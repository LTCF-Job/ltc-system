<template>
  <div class="caregiver-list-view">
    <el-tabs v-model="activeTab" type="border-card" class="caregiver-tabs" @tab-change="handleTabChange">
      <el-tab-pane label="照護人員清單" name="list">
        <DataTablePage
          title="照護人員管理"
          v-model:page="page"
          v-model:pageSize="pageSize"
          :total="total"
          :loading="loading"
          @page-change="handlePageChange"
          @size-change="handleSizeChange"
        >
          <template #filter>
            <el-input
              v-model="filters.q"
              placeholder="搜尋姓名"
              clearable
              style="width: 240px"
              @keyup.enter="handleSearch"
            />
            <el-select
              v-model="filters.status"
              placeholder="狀態"
              clearable
              style="width: 130px"
              @change="handleSearch"
            >
              <el-option label="全部狀態" value="" />
              <el-option label="啟用" value="active" />
              <el-option label="停用" value="inactive" />
            </el-select>
            <el-button type="primary" @click="handleSearch">查詢</el-button>
            <el-button @click="handleReset">重設</el-button>
          </template>

          <template #actions>
            <el-button v-if="authStore.hasPermission('masters_caregivers', 'edit')" type="primary" @click="openCreateDialog">
              <el-icon><Plus /></el-icon>
              新增照護人員
            </el-button>
          </template>

          <template #table>
            <el-table :data="caregivers" border stripe style="width: 100%">
              <el-table-column label="類型" min-width="90" align="center" class-name="type-col">
                <template #default="{ row }">
                  <span class="type-value" :class="{ 'empty-value': !row.type }">
                    {{ CAREGIVER_TYPE_LABELS[row.type as CaregiverType] || row.type || '（未填）' }}
                  </span>
                </template>
              </el-table-column>
              <el-table-column prop="siteName" label="單位" min-width="220" class-name="site-col">
                <template #default="{ row }">
                  <InlineOptionPicker
                    v-if="authStore.hasPermission('masters_caregivers', 'edit')"
                    :model-value="row.siteName || ''"
                    :options="siteOptionItems"
                    allow-create
                    clearable
                    placeholder="未指定"
                    :loading="updatingCaregiverId === row.id"
                    :disabled="updatingCaregiverId === row.id || row.status !== 'active'"
                    @change="(val) => handleInlineUpdateSiteName(row as CaregiverDTO, val as string)"
                  />
                  <span v-else-if="row.siteName">{{ row.siteName }}</span>
                  <span v-else class="empty-value">-</span>
                </template>
              </el-table-column>
              <el-table-column prop="name" label="姓名" min-width="120" class-name="name-col" />
              <el-table-column label="聯絡方式" min-width="140" class-name="contact-col">
                <template #default="{ row }">
                  <span class="contact-value">{{ row.contact || '-' }}</span>
                </template>
              </el-table-column>
              <el-table-column label="備註" min-width="180" show-overflow-tooltip class-name="notes-col">
                <template #default="{ row }">
                  <span>{{ row.notes || '-' }}</span>
                </template>
              </el-table-column>

              <el-table-column prop="status" label="狀態" min-width="120" align="center" class-name="status-col">
                <template #default="{ row }">
                  <el-tooltip
                    v-if="authStore.hasPermission('masters_caregivers', 'edit')"
                    :content="row.status === 'active' ? '目前為啟用，點選切換為停用' : '目前為停用，點選切換為啟用'"
                    placement="top"
                    :show-after="300"
                  >
                    <button
                      type="button"
                      class="status-toggle-pill"
                      :class="row.status === 'active' ? 'is-active' : 'is-inactive'"
                      @click="handleToggleStatus(row as CaregiverDTO, row.status !== 'active')"
                    >
                      <span class="status-indicator-dot"></span>
                      <span class="status-label-text">{{ row.status === 'active' ? '啟用' : '停用' }}</span>
                    </button>
                  </el-tooltip>
                  <div
                    v-else
                    class="status-toggle-pill is-readonly"
                    :class="row.status === 'active' ? 'is-active' : 'is-inactive'"
                  >
                    <span class="status-indicator-dot"></span>
                    <span class="status-label-text">{{ row.status === 'active' ? '啟用' : '停用' }}</span>
                  </div>
                </template>
              </el-table-column>

              <el-table-column label="操作" min-width="140" fixed="right" align="center" class-name="action-col">
                <template #default="{ row }">
                  <TableRowActions>
                    <el-button link type="primary" size="small" @click="openEditDialog(row)">
                      編輯
                    </el-button>
                    <el-button
                      v-if="authStore.hasPermission('masters_caregivers', 'delete')"
                      link
                      type="danger"
                      size="small"
                      @click="handleDelete(row)"
                    >
                      刪除
                    </el-button>
                  </TableRowActions>
                </template>
              </el-table-column>
            </el-table>
          </template>
        </DataTablePage>
      </el-tab-pane>

      <!-- 待維護：匯入時姓名或類型未填寫的照護人員資料，統一用「缺少欄位」欄提示 -->
      <el-tab-pane label="待維護" name="pending">
        <div v-loading="pendingLoading" class="pending-panel">
          <el-alert
            v-if="pendingError"
            type="error"
            show-icon
            :closable="false"
            title="待維護照護人員清單載入失敗"
            style="margin-bottom: 12px"
          >
            <template #default>
              <el-button size="small" @click="fetchPending">重試</el-button>
            </template>
          </el-alert>
          <el-empty v-if="!pendingLoading && !pendingError && pendingCaregivers.length === 0" description="目前沒有待維護的照護人員" />
          <el-table v-else-if="!pendingError" v-table-auto-width :data="pendingCaregivers" border stripe style="width: 100%">
            <el-table-column label="姓名" min-width="120" class-name="name-col">
              <template #default="{ row }">
                <span :class="{ 'empty-value': !row.name }">{{ row.name || '（未填寫）' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="類型" min-width="90" align="center" class-name="pending-type-col">
              <template #default="{ row }">
                <span class="type-value" :class="{ 'empty-value': !row.type }">
                  {{ CAREGIVER_TYPE_LABELS[row.type as CaregiverType] || row.type || '（未填）' }}
                </span>
              </template>
            </el-table-column>
            <el-table-column label="聯絡方式" min-width="140" class-name="pending-contact-col">
              <template #default="{ row }">
                <span class="contact-value">{{ row.contact || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="備註" min-width="180" show-overflow-tooltip class-name="pending-notes-col">
              <template #default="{ row }">
                <span>{{ row.notes || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="缺少欄位" min-width="160" show-overflow-tooltip class-name="pending-missing-col">
              <template #default="{ row }">
                <span class="missing-fields">缺少：{{ missingFields(row as CaregiverDTO).join('、') }}</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="150" align="center" class-name="pending-action-col">
              <template #default="{ row }">
                <TableRowActions>
                  <el-button link type="primary" size="small" @click="openEditDialog(row)">編輯</el-button>
                  <el-button
                    v-if="authStore.hasPermission('masters_caregivers', 'delete')"
                    link
                    type="danger"
                    size="small"
                    @click="handleIgnorePending(row as CaregiverDTO)"
                  >
                    忽略此筆
                  </el-button>
                </TableRowActions>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 批次匯入對話框 -->
    <ImportPreviewDialog
      ref="importDialogRef"
      title="批次匯入照護人員 (類型/單位/姓名/聯絡方式/備註.xlsx)"
      instruction-text="依照範本格式填寫姓名等必填欄位，類型請填寫「個管」或「照專」，系統會自動比對現有主檔資料。"
      :on-dry-run="handleDryRun"
      :on-commit="handleCommitImport"
      :on-download-template="handleDownloadTemplate"
      @success="handleImportSuccess"
    >
      <template #columns="{ checkedDuplicateRows, toggleDuplicateRow, getRowId }">
        <el-table-column prop="type" label="類型" width="80" />
        <el-table-column prop="siteName" label="單位" width="140" />
        <el-table-column prop="name" label="姓名" width="110" />
        <el-table-column prop="contact" label="聯絡方式" width="140" />
        <el-table-column prop="notes" label="備註" min-width="160" show-overflow-tooltip />
        <el-table-column label="重複人員" width="150" align="center">
          <template #default="{ row, $index }">
            <template v-if="row.isDuplicate">
              <el-tooltip :content="`與既有照護人員「${row.duplicateOf?.name ?? '未知'}」疑似重複`" placement="top">
                <el-checkbox
                  :model-value="checkedDuplicateRows.has(getRowId(row, $index))"
                  label="仍要匯入"
                  @change="(val: string | number | boolean) => toggleDuplicateRow(getRowId(row, $index), !!val)"
                />
              </el-tooltip>
            </template>
            <span v-else class="empty-value">-</span>
          </template>
        </el-table-column>
      </template>
    </ImportPreviewDialog>

    <!-- 新增/編輯對話框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '編輯照護人員資料' : '新增照護人員'"
      width="min(480px, calc(100vw - 32px))"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="類型" prop="type">
          <el-select v-model="form.type" placeholder="請選擇類型" style="width: 100%">
            <el-option
              v-for="(label, key) in CAREGIVER_TYPE_LABELS"
              :key="key"
              :label="label"
              :value="key"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="單位" prop="siteName">
          <el-select
            v-model="form.siteName"
            placeholder="請選擇或輸入所屬單位（選填）"
            clearable
            filterable
            allow-create
            default-first-option
            style="width: 100%"
          >
            <el-option
              v-for="site in siteOptions"
              :key="site"
              :label="site"
              :value="site"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="姓名" prop="name">
          <el-input v-model="form.name" placeholder="請輸入姓名" />
        </el-form-item>
        <el-form-item label="聯絡方式" prop="contact">
          <el-input v-model="form.contact" placeholder="選填" />
        </el-form-item>
        <el-form-item label="備註" prop="notes">
          <el-input v-model="form.notes" type="textarea" :rows="2" placeholder="選填" />
        </el-form-item>
        <el-form-item label="狀態" prop="status">
          <el-radio-group v-model="form.status" class="status-radio-group">
            <el-radio-button value="active">
              <div class="radio-pill active-pill">
                <span class="radio-dot"></span>
                <span>啟用</span>
              </div>
            </el-radio-button>
            <el-radio-button value="inactive">
              <div class="radio-pill inactive-pill">
                <span class="radio-dot"></span>
                <span>停用</span>
              </div>
            </el-radio-button>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <DialogFooter
          confirm-text="確認送出"
          :loading="saving"
          @confirm="handleSave"
          @cancel="dialogVisible = false"
        />
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import DataTablePage from '@/components/DataTablePage.vue'
import DialogFooter from '@/components/DialogFooter.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import ImportPreviewDialog from '@/components/ImportPreviewDialog.vue'
import InlineOptionPicker from '@/components/InlineOptionPicker.vue'
import {
  listCaregivers,
  createCaregiver,
  updateCaregiver,
  deleteCaregiver,
  downloadCaregiverTemplate,
  dryRunImportCaregivers,
  commitImportCaregivers,
  listAllCaregivers
} from '@/api/caregivers'
import { listAllSites } from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import { downloadBlob } from '@/utils/download'
import { notifyPendingRelinked } from '@/utils/pendingRelink'
import { CAREGIVER_TYPE_LABELS, type CaregiverType } from '@/types/domain'
import type { CaregiverDTO, SiteDTO } from '@/types/api'

const authStore = useAuthStore()
const activeTab = ref<'list' | 'pending'>('list')
const caregivers = ref<CaregiverDTO[]>([])
const allSites = ref<SiteDTO[]>([])
const updatingCaregiverId = ref<string | null>(null)
const importDialogRef = ref<InstanceType<typeof ImportPreviewDialog>>()

const {
  page,
  pageSize,
  total,
  loading,
  filters,
  handlePageChange,
  handleSizeChange,
  handleSearch,
  handleReset,
  executeFetch
} = useListQuery({
  defaultFilters: { q: '', status: '' },
  onFetch: async () => {
    const res = await listCaregivers({
      page: page.value,
      pageSize: pageSize.value,
      q: filters.q,
      status: filters.status || undefined
      // 待維護資料由後端預設排除，主清單不需要另外表態
    })
    caregivers.value = res.data
    total.value = res.meta.total
  }
})

async function handleDownloadTemplate() {
  try {
    const blob = await downloadCaregiverTemplate()
    downloadBlob(blob, '照護人員批次匯入範本.xlsx')
    ElMessage.success('照護人員匯入範本下載成功')
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  }
}

function openImportDialog() {
  importDialogRef.value?.open()
}

// ImportPreviewDialog 是沿用個案匯入的共用元件，其錯誤／警告清單固定以 caseName 顯示
// 姓名欄位；照護人員後端回應的欄位是 name，這裡轉接成元件既有的欄位形狀，元件本身不需改動。
// 後端無錯誤／警告時，Go 的 nil slice 會序列化成 null，預設參數只在 undefined 生效，
// 必須自行擋掉 null，否則解析預覽會丟出 TypeError，畫面看起來像按鈕沒反應。
function withCaseNameAlias(items?: any[] | null): any[] {
  return (items ?? []).map((item) => ({ ...item, caseName: item.name }))
}

async function handleDryRun(file: File): Promise<any> {
  const preview: any = await dryRunImportCaregivers(file)
  return {
    ...preview,
    errors: withCaseNameAlias(preview.errors),
    warnings: withCaseNameAlias(preview.warnings)
  }
}

async function handleCommitImport(file: File, includeDuplicateRows: string[]): Promise<any> {
  const result: any = await commitImportCaregivers(file, includeDuplicateRows)
  // failedCount／failedRows 是後端寫入資料庫失敗（非使用者選擇略過）的列，之前整個
  // 被丟棄，使用者看不出這批匯入有列真的失敗；比照 skippedRows 轉接欄位形狀後保留。
  return {
    importedCount: result.importedCount,
    skippedRows: (result.skippedRows || []).map((row: any) => ({ rowId: row.rowId, rowIndex: row.rowIndex, caseName: row.name, reasons: row.reasons })),
    failedCount: result.failedCount,
    failedRows: (result.failedRows || []).map((row: any) => ({ rowId: row.rowId, rowIndex: row.rowIndex, caseName: row.name, reasons: row.reasons })),
    warnings: withCaseNameAlias(result.warnings)
  }
}

// 匯入完成後，若有姓名或類型待補齊的提示，導引使用者前往「待維護」頁籤處理；
// 無論點選哪個按鈕都視為使用者已確認匯入結果，一併關閉匯入視窗
function handleImportSuccess() {
  executeFetch()
  ElMessageBox.confirm(
    '本次匯入若有姓名或類型未填寫，已以空白建立資料並列入「待維護」頁籤，是否立即前往查看？',
    '匯入完成',
    { confirmButtonText: '前往待維護', cancelButtonText: '稍後再說', type: 'info' }
  )
    .then(() => {
      activeTab.value = 'pending'
      pendingLoaded = true
      fetchPending()
    })
    .catch(() => {})
    .finally(() => {
      importDialogRef.value?.close()
    })
}

// 待維護的判定條件只有姓名與類型，兩者也是編輯時的必填欄位
function missingFields(row: CaregiverDTO): string[] {
  const missing: string[] = []
  if (!row.name) missing.push('姓名')
  if (!row.type) missing.push('類型')
  return missing
}

// 新增/編輯對話框
const dialogVisible = ref(false)
const saving = ref(false)
const editingId = ref<string | null>(null)
const formRef = ref<FormInstance>()
const form = reactive({
  siteName: '' as string | undefined,
  name: '',
  type: '' as CaregiverType | '',
  contact: '',
  notes: '',
  status: 'active' as 'active' | 'inactive'
})
const rules = {
  name: [{ required: true, message: '請輸入姓名', trigger: 'blur' }],
  type: [{ required: true, message: '請選擇類型', trigger: 'change' }]
}

function openCreateDialog() {
  editingId.value = null
  form.siteName = undefined
  form.name = ''
  form.type = ''
  form.contact = ''
  form.notes = ''
  form.status = 'active'
  dialogVisible.value = true
}

function openEditDialog(row: any) {
  editingId.value = row.id
  form.siteName = row.siteName
  form.name = row.name
  form.type = row.type
  form.contact = row.contact || ''
  form.notes = row.notes || ''
  form.status = row.status || 'active'
  dialogVisible.value = true
}

async function handleToggleStatus(row: CaregiverDTO, newActive: boolean) {
  const newStatus = newActive ? 'active' : 'inactive'
  try {
    await updateCaregiver(row.id, { status: newStatus })
    row.status = newStatus
    ElMessage.success(`已將照護人員「${row.name}」切換為 ${newActive ? '啟用' : '停用'}`)
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  }
}

async function loadSites() {
  try {
    allSites.value = await listAllSites({ status: 'active' })
  } catch {
    allSites.value = []
  }
}

const siteOptions = computed(() => {
  const set = new Set<string>()
  for (const s of allSites.value) {
    if (s.name) set.add(s.name)
  }
  for (const c of caregivers.value) {
    if (c.siteName) set.add(c.siteName)
  }
  return Array.from(set)
})

const siteOptionItems = computed(() => siteOptions.value.map((site) => ({ label: site, value: site })))

async function handleInlineUpdateSiteName(row: CaregiverDTO, newSiteName: string) {
  const currentSiteName = row.siteName || ''
  const trimmed = newSiteName ? newSiteName.trim() : ''
  if (trimmed === currentSiteName) return

  updatingCaregiverId.value = row.id
  try {
    await updateCaregiver(row.id, { siteName: trimmed })
    row.siteName = trimmed
    ElMessage.success(`已將「${row.name}」單位更新為 ${trimmed ? `「${trimmed}」` : '（未填）'}`)
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  } finally {
    updatingCaregiverId.value = null
  }
}

async function handleSave() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      if (editingId.value) {
        const { meta } = await updateCaregiver(editingId.value, {
          siteName: form.siteName,
          name: form.name,
          type: form.type as CaregiverType,
          contact: form.contact,
          notes: form.notes,
          status: form.status
        })
        ElMessage.success('照護人員資料已更新')
        notifyPendingRelinked(meta)
      } else {
        const { meta } = await createCaregiver({
          siteName: form.siteName,
          name: form.name,
          type: form.type as CaregiverType,
          contact: form.contact,
          notes: form.notes,
          status: form.status
        })
        ElMessage.success('照護人員建立成功')
        notifyPendingRelinked(meta)
      }
      dialogVisible.value = false
      executeFetch()
      if (activeTab.value === 'pending') {
        await fetchPending()
      }
    } catch {
      // 全域攔截器負責顯示 API 錯誤。
    } finally {
      saving.value = false
    }
  })
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(`確定要刪除照護人員「${row.name}」？此操作無法復原。`, '刪除確認', {
      confirmButtonText: '刪除',
      cancelButtonText: '取消',
      type: 'warning',
      confirmButtonClass: 'el-button--danger'
    })
    await deleteCaregiver(row.id)
    ElMessage.success(`照護人員「${row.name}」已成功刪除`)
    executeFetch()
  } catch {
    // 使用者取消或 API 錯誤皆不在此重複顯示。
  }
}

// 待維護頁籤：匯入時姓名或類型未填寫、待人工補齊的照護人員
const pendingLoading = ref(false)
const pendingCaregivers = ref<CaregiverDTO[]>([])
// 載入失敗與「查無資料」是不同狀態，不應共用同一個「目前沒有待維護的照護人員」空狀態，
// 否則使用者會誤以為真的沒有待維護資料而不會重試。
const pendingError = ref(false)

async function fetchPending() {
  pendingLoading.value = true
  pendingError.value = false
  try {
    pendingCaregivers.value = await listAllCaregivers({ pending: true })
  } catch {
    // 全域攔截器負責顯示 API 錯誤 toast；這裡另外標記狀態以顯示可重試的載入失敗畫面。
    pendingError.value = true
  } finally {
    pendingLoading.value = false
  }
}

// 忽略此筆：使用者判斷這筆本來就不該匯入時，直接把資料從系統刪除。
async function handleIgnorePending(row: CaregiverDTO) {
  const label = row.name || '（未填寫姓名）'
  try {
    await ElMessageBox.confirm(
      `確定要忽略待維護資料「${label}」？將直接刪除這筆照護人員資料，無法復原。若之後重新匯入同一份檔案，這筆仍會再次出現。`,
      '忽略確認',
      { confirmButtonText: '忽略並刪除', cancelButtonText: '取消', type: 'warning' }
    )
    await deleteCaregiver(row.id)
    pendingCaregivers.value = pendingCaregivers.value.filter((c) => c.id !== row.id)
    ElMessage.success('已忽略該筆待維護資料')
    executeFetch()
  } catch {
    // 使用者取消或 API 錯誤皆不在此重複顯示。
  }
}

// 待維護頁籤首次切入時才拉取清單，避免一般清單頁多打一次 API
let pendingLoaded = false
async function handleTabChange(name: string | number) {
  if (name === 'pending' && !pendingLoaded) {
    pendingLoaded = true
    await fetchPending()
  }
}

loadSites()
executeFetch()
</script>

<style scoped>
.caregiver-list-view {
  display: flex;
  flex-direction: column;
}

.caregiver-tabs {
  border-radius: 8px;
}

/* overflow-x: auto 讓表格加總寬度超過版面時把捲軸包在面板內，不外溢到整個頁面
   （這個面板不像 DataTablePage 的 .table-container 內建這條規則，要自己補）。 */
.pending-panel {
  min-height: 120px;
  overflow-x: auto;
}

/* table-layout="auto" 底下 el-table 本體內建 width: 100%，即使拿掉 inline style
   仍會撐滿容器、內容明明很短卻拉滿版面；要顯式蓋成 max-content 才會縮到
   「各欄寬度加總」（見 ltc-dashboard-visual-language skill 表格欄位一節）。 */
.pending-panel :deep(.el-table) {
  width: max-content;
}

/* el-table-column 的 min-width prop 在 table-layout="auto" 底下只會拿去算表格
   總寬度的預算，不會變成該欄真正的 CSS min-width，要另外補一條 :deep() min-width
   才是真的鎖住下限（見 ltc-dashboard-visual-language skill 表格欄位一節）。 */
.pending-panel :deep(.pending-contact-col .cell) { min-width: 140px; }
.pending-panel :deep(.pending-action-col .cell) {
  white-space: nowrap;
  min-width: 150px;
}
.pending-panel :deep(.pending-notes-col .cell) { min-width: 180px; }
.pending-panel :deep(.pending-missing-col .cell) { min-width: 160px; }
.pending-panel :deep(.name-col .cell) { min-width: 120px; }
.pending-panel :deep(.pending-type-col .cell) {
  white-space: nowrap;
  min-width: 90px;
}

.missing-fields {
  color: var(--app-status-danger-fg);
  font-size: 13px;
}

/* 同樣道理：span 沒鎖 nowrap，table-layout: auto 也救不了，聯絡方式還是會被壓成多行。 */
.contact-value,
.type-value {
  white-space: nowrap;
}

/* 類型與姓名欄沒有自訂 template 或在 table-layout="auto" 下需鎖死欄寬，
   使用 class-name 打進 el-table 內部 cell 鎖 nowrap 與 min-width 下限。 */
:deep(.type-col .cell) {
  white-space: nowrap;
  min-width: 90px;
}

:deep(.status-col .cell) {
  white-space: nowrap;
  min-width: 120px;
}

/* 姓名欄沒有自訂 template，只能用 class-name 打進 el-table 內部 cell 鎖 nowrap；
   table-layout="auto" 底下固定 width 只會鎖死欄寬讓文字換行、不會依內容自動撐開，
   min-width 欄位沒鎖 nowrap 一樣會被壓成逐字換行（見 ltc-dashboard-visual-language skill 表格欄位一節）。 */
:deep(.name-col .cell) {
  white-space: nowrap;
  min-width: 120px;
}

/* 主表格「單位／聯絡方式／備註」欄同樣沒有 class-name 鎖 min-width 下限，
   會被 table-layout="auto" 壓窄或被其他欄擠壓（見待維護子表格同一段說明）。 */
:deep(.site-col .cell) {
  min-width: 220px;
}

:deep(.contact-col .cell) {
  min-width: 140px;
}

:deep(.notes-col .cell) {
  min-width: 180px;
}

:deep(.action-col .cell) {
  white-space: nowrap;
  min-width: 140px;
}

.empty-value {
  color: var(--app-text-muted);
}
</style>
