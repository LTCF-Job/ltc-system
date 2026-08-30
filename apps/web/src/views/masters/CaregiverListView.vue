<template>
  <div class="caregiver-list-view">
    <el-tabs v-model="activeTab" type="border-card" class="caregiver-tabs" @tab-change="handleTabChange">
      <el-tab-pane label="照護人員清單" name="list">
        <DataTablePage
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
              style="width: 220px"
              @keyup.enter="handleSearch"
            />
            <el-button type="primary" @click="handleSearch">查詢</el-button>
            <el-button @click="handleReset">重設</el-button>
          </template>

          <template #actions>
            <el-button v-if="authStore.can('staff')" type="info" plain @click="handleDownloadTemplate">
              <el-icon><Download /></el-icon>
              下載匯入範本
            </el-button>

            <el-button v-if="authStore.can('staff')" type="success" plain @click="openImportDialog">
              <el-icon><Upload /></el-icon>
              批次匯入照護人員
            </el-button>

            <el-button v-if="authStore.can('staff')" type="primary" @click="openCreateDialog">
              <el-icon><Plus /></el-icon>
              新增照護人員
            </el-button>
          </template>

          <template #table>
            <el-table :data="caregivers" border stripe style="width: 100%">
              <el-table-column label="類型" width="90" align="center">
                <template #default="{ row }">
                  <span>{{ CAREGIVER_TYPE_LABELS[row.type as CaregiverType] || row.type }}</span>
                </template>
              </el-table-column>
              <el-table-column label="單位" min-width="220">
                <template #default="{ row }">
                  <span v-if="row.siteName">{{ row.siteName }}</span>
                  <div v-else-if="row.siteNameRaw" class="unresolved-slot">
                    <span class="unresolved-raw-name">原始名稱：{{ row.siteNameRaw }}</span>
                    <el-select
                      filterable
                      placeholder="選擇既有據點"
                      style="width: 150px"
                      @change="(val: string) => handleLinkSite(row as CaregiverDTO, val)"
                    >
                      <el-option v-for="site in availableSites" :key="site.id" :value="site.id" :label="site.name" />
                    </el-select>
                    <el-button link type="primary" size="small" @click="openQuickCreateSite(row as CaregiverDTO)">新增據點</el-button>
                  </div>
                  <span v-else class="empty-value">-</span>
                </template>
              </el-table-column>
              <el-table-column prop="name" label="姓名" width="120" />
              <el-table-column label="聯絡方式" min-width="140">
                <template #default="{ row }">
                  <span>{{ row.contact || '-' }}</span>
                </template>
              </el-table-column>
              <el-table-column label="備註" min-width="180" show-overflow-tooltip>
                <template #default="{ row }">
                  <span>{{ row.notes || '-' }}</span>
                </template>
              </el-table-column>

              <el-table-column label="操作" width="140" fixed="right" align="center">
                <template #default="{ row }">
                  <el-button link type="success" size="small" :icon="Edit" @click="openEditDialog(row)">
                    編輯
                  </el-button>
                  <el-button
                    v-if="authStore.can('admin')"
                    link
                    type="danger"
                    size="small"
                    @click="handleDelete(row)"
                  >
                    刪除
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </template>
        </DataTablePage>
      </el-tab-pane>

      <!-- 待維護：單位比對不到既有據點、或聯絡方式／備註缺漏的照護人員資料，統一用「缺少欄位」欄提示，不再分組 -->
      <el-tab-pane label="待維護" name="pending">
        <div v-loading="pendingLoading" class="pending-panel">
          <el-empty v-if="!pendingLoading && pendingCaregivers.length === 0" description="目前沒有待維護的照護人員" />
          <el-table v-else :data="pendingCaregivers" border stripe style="width: 100%">
            <el-table-column prop="name" label="姓名" width="120" />
            <el-table-column label="單位" min-width="220">
              <template #default="{ row }">
                <span v-if="row.siteName">{{ row.siteName }}</span>
                <div v-else-if="row.siteNameRaw" class="unresolved-slot">
                  <span class="unresolved-raw-name">原始名稱：{{ row.siteNameRaw }}</span>
                  <el-select
                    filterable
                    placeholder="選擇既有據點"
                    style="width: 150px"
                    @change="(val: string) => handleLinkSite(row as CaregiverDTO, val)"
                  >
                    <el-option v-for="site in availableSites" :key="site.id" :value="site.id" :label="site.name" />
                  </el-select>
                  <el-button link type="primary" size="small" @click="openQuickCreateSite(row as CaregiverDTO)">新增據點</el-button>
                </div>
                <span v-else class="empty-value">-</span>
              </template>
            </el-table-column>
            <el-table-column label="聯絡方式" min-width="140">
              <template #default="{ row }">
                <span>{{ row.contact || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="備註" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">
                <span>{{ row.notes || '-' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="缺少欄位" min-width="160">
              <template #default="{ row }">
                <span class="missing-fields">缺少：{{ missingFields(row as CaregiverDTO).join('、') }}</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100" align="center">
              <template #default="{ row }">
                <el-button link type="success" size="small" :icon="Edit" @click="openEditDialog(row)">編輯</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 新增據點快速建立彈窗 -->
    <el-dialog v-model="quickCreateSiteVisible" title="新增據點" width="480px">
      <el-form label-width="90px">
        <el-form-item label="據點名稱"><el-input v-model="quickCreateSiteForm.name" /></el-form-item>
        <el-form-item label="區域">
          <el-select v-model="quickCreateSiteForm.region" style="width: 100%">
            <el-option v-for="(label, key) in REGION_LABELS" :key="key" :value="key" :label="label" />
          </el-select>
        </el-form-item>
        <el-form-item label="地址"><el-input v-model="quickCreateSiteForm.address" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="quickCreateSiteVisible = false">取消</el-button>
        <el-button type="primary" :loading="quickCreateSiteSaving" @click="handleQuickCreateSiteAndLink">建立並關聯</el-button>
      </template>
    </el-dialog>

    <!-- 批次匯入彈窗 -->
    <ImportPreviewDialog
      ref="importDialogRef"
      title="批次匯入照護人員 (類型/單位/姓名/聯絡方式/備註.xlsx)"
      :on-dry-run="handleDryRun"
      :on-commit="handleCommitImport"
      :on-download-template="handleDownloadTemplate"
      @success="handleImportSuccess"
    >
      <template #columns="{ checkedDuplicateRows, toggleDuplicateRow }">
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
                  :model-value="checkedDuplicateRows.has(row.rowIndex ?? $index)"
                  label="仍要匯入"
                  @change="(val: string | number | boolean) => toggleDuplicateRow(row.rowIndex ?? $index, !!val)"
                />
              </el-tooltip>
            </template>
            <span v-else class="empty-value">-</span>
          </template>
        </el-table-column>
      </template>
    </ImportPreviewDialog>

    <!-- 新增/編輯彈窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '編輯照護人員資料' : '新增照護人員'"
      width="480px"
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
        <el-form-item label="單位" prop="siteId">
          <el-select v-model="form.siteId" placeholder="請選擇據點" filterable clearable style="width: 100%">
            <el-option v-for="site in availableSites" :key="site.id" :value="site.id" :label="site.name" />
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
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">確認送出</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { Plus, Upload, Download, Edit } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { resolveErrorMessage } from '@/api/errorCodes'
import DataTablePage from '@/components/DataTablePage.vue'
import ImportPreviewDialog from '@/components/ImportPreviewDialog.vue'
import {
  listCaregivers,
  createCaregiver,
  updateCaregiver,
  deleteCaregiver,
  linkCaregiverSite,
  downloadCaregiverTemplate,
  dryRunImportCaregivers,
  commitImportCaregivers
} from '@/api/caregivers'
import { listSites, createSite } from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import { downloadBlob } from '@/utils/download'
import { REGION_LABELS, CAREGIVER_TYPE_LABELS, type Region, type CaregiverType } from '@/types/domain'
import type { CaregiverDTO, SiteDTO } from '@/types/api'

const authStore = useAuthStore()
const activeTab = ref<'list' | 'pending'>('list')
const caregivers = ref<CaregiverDTO[]>([])
const availableSites = ref<SiteDTO[]>([])
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
  defaultFilters: { q: '' },
  onFetch: async () => {
    const res = await listCaregivers({ page: page.value, pageSize: pageSize.value, q: filters.q })
    caregivers.value = res.data
    total.value = res.meta.total
  }
})

async function loadSites() {
  const res = await listSites({ pageSize: 100 })
  availableSites.value = res.data
}

// 下載匯入範本
async function handleDownloadTemplate() {
  try {
    const blob = await downloadCaregiverTemplate()
    downloadBlob(blob, '照護人員批次匯入範本.xlsx')
    ElMessage.success('照護人員匯入範本下載成功')
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '下載範本失敗'))
  }
}

function openImportDialog() {
  importDialogRef.value?.open()
}

// ImportPreviewDialog 是沿用個案匯入的共用元件，其錯誤／警告清單固定以 caseName 顯示
// 姓名欄位；照護人員後端回應的欄位是 name，這裡轉接成元件既有的欄位形狀，元件本身不需改動。
function withCaseNameAlias(items: any[] = []): any[] {
  return items.map((item) => ({ ...item, caseName: item.name }))
}

async function handleDryRun(file: File): Promise<any> {
  const preview: any = await dryRunImportCaregivers(file)
  return {
    ...preview,
    errors: withCaseNameAlias(preview.errors),
    warnings: withCaseNameAlias(preview.warnings)
  }
}

async function handleCommitImport(file: File, includeDuplicateRows: number[]): Promise<any> {
  const result: any = await commitImportCaregivers(file, includeDuplicateRows)
  return {
    importedCount: result.importedCount,
    skippedRows: (result.skippedRows || []).map((row: any) => ({ rowIndex: row.rowIndex, caseName: row.name, reasons: row.reasons })),
    warnings: withCaseNameAlias(result.warnings)
  }
}

// 匯入完成後，若有單位待關聯或資料待補齊的提示，導引使用者前往「待維護」頁籤處理；
// 無論點選哪個按鈕都視為使用者已確認匯入結果，一併關閉匯入視窗
function handleImportSuccess() {
  executeFetch()
  ElMessageBox.confirm(
    '本次匯入若有單位未比對到既有據點，或聯絡方式／備註未填寫，已建立資料並列入「待維護」頁籤，是否立即前往查看？',
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

// 依 siteId／contact／notes 是否有值，列出該筆照護人員缺少的欄位
function missingFields(row: CaregiverDTO): string[] {
  const missing: string[] = []
  if (!row.siteId) missing.push('單位')
  if (!row.contact) missing.push('聯絡方式')
  if (!row.notes) missing.push('備註')
  return missing
}

// 新增/編輯彈窗
const dialogVisible = ref(false)
const saving = ref(false)
const editingId = ref<string | null>(null)
const formRef = ref<FormInstance>()
const form = reactive({ siteId: '' as string | undefined, name: '', type: '' as CaregiverType | '', contact: '', notes: '' })
const rules = {
  name: [{ required: true, message: '請輸入姓名', trigger: 'blur' }],
  type: [{ required: true, message: '請選擇類型', trigger: 'change' }]
}

function openCreateDialog() {
  editingId.value = null
  form.siteId = undefined
  form.name = ''
  form.type = ''
  form.contact = ''
  form.notes = ''
  dialogVisible.value = true
}

function openEditDialog(row: any) {
  editingId.value = row.id
  form.siteId = row.siteId
  form.name = row.name
  form.type = row.type
  form.contact = row.contact || ''
  form.notes = row.notes || ''
  dialogVisible.value = true
}

async function handleSave() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      if (editingId.value) {
        await updateCaregiver(editingId.value, {
          siteId: form.siteId,
          name: form.name,
          type: form.type as CaregiverType,
          contact: form.contact,
          notes: form.notes
        })
        ElMessage.success('照護人員資料已更新')
      } else {
        await createCaregiver({
          siteId: form.siteId,
          name: form.name,
          type: form.type as CaregiverType,
          contact: form.contact,
          notes: form.notes
        })
        ElMessage.success('照護人員建立成功')
      }
      dialogVisible.value = false
      executeFetch()
      if (activeTab.value === 'pending') {
        await fetchPending()
      }
    } catch (err: any) {
      ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '儲存照護人員資料失敗'))
    } finally {
      saving.value = false
    }
  })
}

async function handleDelete(row: any) {
  try {
    await ElMessageBox.confirm(`確定要刪除照護人員「${row.name}」？此操作無法復原。`, '刪除確認', {
      confirmButtonText: '確定刪除',
      cancelButtonText: '取消',
      type: 'warning',
      confirmButtonClass: 'el-button--danger'
    })
    await deleteCaregiver(row.id)
    ElMessage.success(`照護人員「${row.name}」已成功刪除`)
    executeFetch()
  } catch (err: any) {
    if (err !== 'cancel') {
      ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '刪除照護人員失敗'))
    }
  }
}

// 待維護頁籤：單位未比對到既有據點與資料缺漏的照護人員合併為單一清單，用「缺少欄位」呈現而非分組
const pendingLoading = ref(false)
const pendingCaregivers = ref<CaregiverDTO[]>([])

async function fetchPending() {
  pendingLoading.value = true
  try {
    const [unresolvedRes, incompleteRes] = await Promise.all([
      listCaregivers({ unresolvedLink: true, pageSize: 100 }),
      listCaregivers({ incomplete: true, pageSize: 100 })
    ])
    const merged = new Map<string, CaregiverDTO>()
    for (const row of [...unresolvedRes.data, ...incompleteRes.data]) {
      merged.set(row.id, row)
    }
    pendingCaregivers.value = Array.from(merged.values())
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '載入待維護清單失敗'))
  } finally {
    pendingLoading.value = false
  }
}

async function handleLinkSite(row: CaregiverDTO, siteId: string) {
  if (!siteId) return
  try {
    await linkCaregiverSite(row.id, siteId)
    ElMessage.success(`照護人員「${row.name}」已完成單位關聯`)
    executeFetch()
    if (activeTab.value === 'pending') {
      await fetchPending()
    }
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '更新單位關聯失敗'))
  }
}

// 新增據點並立即關聯
const quickCreateSiteVisible = ref(false)
const quickCreateSiteSaving = ref(false)
const quickCreateTarget = ref<CaregiverDTO | null>(null)
const quickCreateSiteForm = reactive({ name: '', region: 'miaoli' as Region, address: '', openDays: [1, 2, 3, 4, 5] })

// 據點名稱預先帶入匯入時的原始名稱，使用者只需確認區域與地址即可送出，不必重打一次名稱
function openQuickCreateSite(row: CaregiverDTO) {
  quickCreateTarget.value = row
  quickCreateSiteForm.name = row.siteNameRaw || ''
  quickCreateSiteForm.region = 'miaoli'
  quickCreateSiteForm.address = ''
  quickCreateSiteVisible.value = true
}

async function handleQuickCreateSiteAndLink() {
  if (!quickCreateTarget.value) return
  quickCreateSiteSaving.value = true
  try {
    const site = await createSite(quickCreateSiteForm)
    availableSites.value.push(site)
    await handleLinkSite(quickCreateTarget.value, site.id)
    quickCreateSiteVisible.value = false
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '新增並關聯據點失敗'))
  } finally {
    quickCreateSiteSaving.value = false
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

// 初始載入
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

.pending-panel {
  min-height: 120px;
}

.unresolved-slot {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.unresolved-raw-name {
  color: var(--el-color-warning);
  font-size: 13px;
}

.missing-fields {
  color: var(--el-color-danger);
  font-size: 13px;
}

.empty-value {
  color: var(--el-text-color-placeholder);
}
</style>
