<template>
  <div class="case-list-view">
    <el-tabs v-model="activeTab" type="border-card" class="case-tabs" @tab-change="handleTabChange">
    <el-tab-pane label="個案清單" name="list">
    <DataTablePage
      :max-width="1680"
      v-model:page="page"
      v-model:pageSize="pageSize"
      :total="total"
      :loading="loading"
      @page-change="handlePageChange"
      @size-change="handleSizeChange"
    >
      <!-- 篩選器列 -->
      <template #filter>
        <el-input
          v-model="filters.q"
          placeholder="搜尋姓名／編號／身分證／地址"
          clearable
          style="width: 240px"
          @keyup.enter="handleSearch"
        />

        <el-select
          v-model="filters.region"
          placeholder="全部區域"
          clearable
          filterable
          style="width: 140px"
          @change="handleSearch"
        >
          <el-option label="全部區域" value="" />
          <el-option
            v-for="(label, key) in REGION_LABELS"
            :key="key"
            :label="label"
            :value="key"
          />
        </el-select>

        <el-select
          v-model="filters.status"
          placeholder="個案狀態"
          clearable
          style="width: 130px"
          @change="handleSearch"
        >
          <el-option label="全部狀態" value="" />
          <el-option label="在案" value="active" />
          <el-option label="暫停" value="suspended" />
          <el-option label="停案" value="closed" />
        </el-select>

        <el-button type="primary" @click="handleSearch">查詢</el-button>
        <el-button @click="handleReset">重設</el-button>
      </template>

      <!-- 操作按鈕列 -->
      <template #actions>
        <!-- 下載範本／匯出實際呼叫 GET /cases/template、/cases/export，後端僅要求 masters_cases:view -->
        <el-button v-if="authStore.hasPermission('masters_cases', 'view')" plain @click="handleDownloadTemplate">
          下載匯入範本
        </el-button>

        <el-button v-if="authStore.hasPermission('masters_cases', 'edit')" plain @click="openImportDialog">
          批次匯入個案
        </el-button>

        <el-button v-if="authStore.hasPermission('masters_cases', 'view')" plain @click="openExportDialog">
          匯出個案資料
        </el-button>

        <el-button
          v-if="authStore.hasPermission('masters_cases', 'edit')"
          type="primary"
          @click="openCreateDialog"
        >
          <el-icon><Plus /></el-icon>
          新增個案
        </el-button>
      </template>

      <!-- 表格內容 -->
      <template #table>
        <el-table :data="cases" border stripe style="width: 100%">
          <el-table-column prop="name" label="姓名" width="110" align="center" />
          <el-table-column prop="nationalId" label="身分證字號" min-width="150" align="center">
            <template #default="{ row }">
              <span class="font-mono text-id">{{ row.nationalId || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="region" label="區域" width="115" align="center">
            <template #default="{ row }">
              <el-dropdown
                v-if="authStore.hasPermission('masters_cases', 'edit')"
                trigger="click"
                @command="(val: Region) => handleQuickUpdateRegion(row as any, val)"
              >
                <span class="cursor-pointer">
                  <span class="inline-value inline-value-clickable">
                    {{ REGION_LABELS[row.region as Region] || row.region }}
                    <el-icon class="el-icon--right"><ArrowDown /></el-icon>
                  </span>
                </span>
                <template #dropdown>
                  <el-dropdown-menu style="max-height: 240px; overflow-y: auto;">
                    <el-dropdown-item
                      v-for="(label, key) in REGION_LABELS"
                      :key="key"
                      :command="key"
                    >
                      <span>{{ label }}</span>
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
              <span v-else class="inline-value">
                {{ REGION_LABELS[row.region as Region] || row.region }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="status" label="狀態" width="115" align="center">
            <template #default="{ row }">
              <el-dropdown
                v-if="authStore.hasPermission('masters_cases', 'edit')"
                trigger="click"
                @command="(val: CaseStatus) => handleQuickUpdateStatus(row as any, val)"
              >
                <span class="cursor-pointer">
                  <span class="case-status inline-value-clickable">
                    <span class="status-dot" :class="`status-dot-${row.status}`"></span>
                    {{ CASE_STATUS_LABELS[row.status as CaseStatus] || row.status }}
                    <el-icon class="el-icon--right"><ArrowDown /></el-icon>
                  </span>
                </span>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="active">在案</el-dropdown-item>
                    <el-dropdown-item command="suspended">暫停</el-dropdown-item>
                    <el-dropdown-item command="closed">停案</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
              <span v-else class="case-status">
                <span class="status-dot" :class="`status-dot-${row.status}`"></span>
                {{ CASE_STATUS_LABELS[row.status as CaseStatus] || row.status }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="serviceUsageType" label="服務使用類型" min-width="190" align="center">
            <template #default="{ row }">
              <span>{{ SERVICE_USAGE_TYPE_LABELS[row.serviceUsageType as ServiceUsageType] || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="排班概要" min-width="320">
            <template #default="{ row }">
              <span v-if="row.activeSchedule">
                {{ TRIP_PATTERN_LABELS[row.activeSchedule.tripPattern as TripPattern] }}
                ({{ row.activeSchedule.weekdays?.map((w: number) => `週${'一二三四五六日'[w-1]}`).join('、') }})
              </span>
              <span v-else class="empty-value">尚未設定排班</span>
            </template>
          </el-table-column>
          <el-table-column prop="homeAddress" label="住家地址" min-width="190" show-overflow-tooltip />
          <el-table-column prop="remarks" label="備註" min-width="160" show-overflow-tooltip>
            <template #default="{ row }">
              <span>{{ row.remarks || '-' }}</span>
            </template>
          </el-table-column>

          <el-table-column label="操作" width="140" fixed="right" align="center">
            <template #default="{ row }">
              <TableRowActions>
                <el-button
                  link
                  type="primary"
                  size="small"
                  @click="$router.push(`/cases/${row.id}?tab=basic`)"
                >
                  編輯
                </el-button>
                <el-button
                  v-if="authStore.hasPermission('masters_cases', 'delete')"
                  link
                  type="danger"
                  size="small"
                  @click="handleDeleteCase(row as any)"
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

    <!-- 待維護：單位/去程車/回程車比對不到主檔資料的個案，供事後關聯或新增主檔；版面比照照護人員管理的待維護頁籤 -->
    <el-tab-pane label="待維護" name="unresolved">
      <div v-loading="unresolvedLoading" class="pending-panel">
        <el-empty v-if="!unresolvedLoading && unresolvedCases.length === 0" description="目前沒有待維護的個案" />
        <el-table v-else :data="unresolvedCases" border stripe table-layout="auto">
          <el-table-column prop="name" label="姓名" min-width="90" class-name="unresolved-name-col" />
          <el-table-column label="單位" min-width="220" class-name="unresolved-site-col">
            <template #default="{ row }">
              <div v-if="row.siteNameRaw" class="unresolved-slot">
                <span class="unresolved-raw-name">原始名稱：{{ row.siteNameRaw }}</span>
                <el-select
                  filterable
                  placeholder="選擇既有單位"
                  style="width: 160px"
                  @change="(val: string) => handleLinkSlot(row as CaseDTO, 'site', val)"
                >
                  <el-option v-for="site in availableSites" :key="site.id" :value="site.id" :label="site.name" />
                </el-select>
                <el-button link type="primary" size="small" @click="openQuickCreate('site', row as CaseDTO)">新增單位</el-button>
              </div>
              <span v-else class="empty-value">-</span>
            </template>
          </el-table-column>
          <el-table-column label="去程車輛" min-width="220" class-name="unresolved-outbound-col">
            <template #default="{ row }">
              <div v-if="row.outboundVehicleNameRaw" class="unresolved-slot">
                <span class="unresolved-raw-name">原始名稱：{{ row.outboundVehicleNameRaw }}</span>
                <el-select
                  filterable
                  placeholder="選擇既有車輛"
                  style="width: 160px"
                  @change="(val: string) => handleLinkSlot(row as CaseDTO, 'outboundVehicle', val)"
                >
                  <el-option v-for="vehicle in availableVehicles" :key="vehicle.id" :value="vehicle.id" :label="vehicle.displayName" />
                </el-select>
                <el-button link type="primary" size="small" @click="openQuickCreate('vehicle', row as CaseDTO, 'outboundVehicle')">新增車輛</el-button>
              </div>
              <span v-else class="empty-value">-</span>
            </template>
          </el-table-column>
          <el-table-column label="回程車輛" min-width="220" class-name="unresolved-inbound-col">
            <template #default="{ row }">
              <div v-if="row.inboundVehicleNameRaw" class="unresolved-slot">
                <span class="unresolved-raw-name">原始名稱：{{ row.inboundVehicleNameRaw }}</span>
                <el-select
                  filterable
                  placeholder="選擇既有車輛"
                  style="width: 160px"
                  @change="(val: string) => handleLinkSlot(row as CaseDTO, 'inboundVehicle', val)"
                >
                  <el-option v-for="vehicle in availableVehicles" :key="vehicle.id" :value="vehicle.id" :label="vehicle.displayName" />
                </el-select>
                <el-button link type="primary" size="small" @click="openQuickCreate('vehicle', row as CaseDTO, 'inboundVehicle')">新增車輛</el-button>
              </div>
              <span v-else class="empty-value">-</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-tab-pane>
    </el-tabs>

    <!-- 新增單位/車輛快速建立彈窗 -->
    <el-dialog v-model="quickCreateVisible" :title="quickCreateKind === 'site' ? '新增單位' : '新增車輛'" width="min(480px, calc(100vw - 32px))">
      <el-form v-if="quickCreateKind === 'site'" label-width="90px">
        <el-form-item label="單位名稱"><el-input v-model="quickCreateSiteForm.name" /></el-form-item>
        <el-form-item label="區域">
          <el-select v-model="quickCreateSiteForm.region" style="width: 100%">
            <el-option v-for="(label, key) in REGION_LABELS" :key="key" :value="key" :label="label" />
          </el-select>
        </el-form-item>
        <el-form-item label="地址"><el-input v-model="quickCreateSiteForm.address" /></el-form-item>
      </el-form>
      <el-form
        v-else
        ref="quickCreateVehicleFormRef"
        :model="quickCreateVehicleForm"
        :rules="vehicleFormRules"
        label-width="150px"
      >
        <VehicleFormFields :form="quickCreateVehicleForm" :sites="availableSites" />
      </el-form>
      <template #footer>
        <DialogFooter
          confirm-text="建立並關聯"
          :loading="quickCreateSaving"
          @confirm="handleQuickCreateAndLink"
          @cancel="quickCreateVisible = false"
        />
      </template>
    </el-dialog>

    <!-- 批次匯入彈窗 -->
    <ImportPreviewDialog
      ref="importDialogRef"
      title="批次匯入個案 (個案新增資料.xlsx)"
      :on-dry-run="dryRunImportCases"
      :on-commit="handleCommitImport"
      :on-download-template="handleDownloadTemplate"
      @success="handleImportSuccess"
    >
      <template #columns="{ checkedDuplicateRows, toggleDuplicateRow }">
        <el-table-column prop="name" label="姓名" width="100" />
        <el-table-column prop="householdType" label="戶別" width="90" />
        <el-table-column prop="nationalId" label="身分證字號" width="120" />
        <el-table-column prop="gender" label="性別" width="60" />
        <el-table-column prop="birthDate" label="生日" width="100" :formatter="(row: any) => formatRocBirthDate(row.birthDate)" />
        <el-table-column prop="siteName" label="單位" width="110" />
        <el-table-column prop="outboundVehicle" label="去程車" width="100" />
        <el-table-column prop="inboundVehicle" label="回程車" width="100" />
        <el-table-column prop="careContactRole" label="個管or照專" width="100" />
        <el-table-column prop="careContactName" label="個管姓名" width="100" />
        <el-table-column prop="registeredAddress" label="戶籍" min-width="140" show-overflow-tooltip />
        <el-table-column prop="homeAddress" label="居住地" min-width="140" show-overflow-tooltip />
        <el-table-column prop="remarks" label="備註" min-width="140" show-overflow-tooltip />
        <el-table-column label="重複個案" width="150" align="center">
          <template #default="{ row, $index }">
            <template v-if="row.isDuplicate">
              <el-tooltip
                :content="`與既有個案「${row.duplicateOf?.name ?? '未知'}」(${row.duplicateOf?.code ?? '未知'}) 疑似重複`"
                placement="top"
              >
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

    <!-- 新增個案彈窗 -->
    <el-dialog v-model="createDialogVisible" title="新增個案基本資料" width="min(600px, calc(100vw - 32px))">
      <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="110px" class="dialog-scroll-form">
        <el-form-item label="個案姓名" prop="name">
          <el-input v-model="createForm.name" placeholder="請輸入姓名（含罕用字）" />
        </el-form-item>
        <el-form-item label="身分證字號" prop="nationalId">
          <el-input v-model="createForm.nationalId" placeholder="1 碼英文字母 + 9 碼數字" />
        </el-form-item>
        <el-form-item label="申報區域" prop="region">
          <el-select
            v-model="createForm.region"
            placeholder="請選擇區域"
            filterable
            style="width: 100%"
          >
            <el-option
              v-for="(label, key) in REGION_LABELS"
              :key="key"
              :label="label"
              :value="key"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="住家地址" prop="homeAddress">
          <el-input v-model="createForm.homeAddress" placeholder="請輸入住家地址" />
        </el-form-item>
        <el-form-item label="服務類別" prop="serviceCategory">
          <el-radio-group v-model="createForm.serviceCategory">
            <el-radio :value="1">1. 補助</el-radio>
            <el-radio :value="2">2. 自費</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="服務使用類型" prop="serviceUsageType">
          <el-select v-model="createForm.serviceUsageType" placeholder="未選擇" clearable style="width: 100%">
            <el-option :value="1" label="1. 社區式長照機構" />
            <el-option :value="2" label="2. 社區服務據點(不含身障類)" />
            <el-option :value="3" label="3. 輔具中心" />
            <el-option :value="4" label="4. 身障日間照顧服務" />
          </el-select>
        </el-form-item>
        <el-form-item label="備註" prop="remarks">
          <el-input v-model="createForm.remarks" type="textarea" :rows="2" placeholder="選填" />
        </el-form-item>
      </el-form>
      <template #footer>
        <DialogFooter :loading="saving" @confirm="handleCreateCase" @cancel="createDialogVisible = false" />
      </template>
    </el-dialog>

    <!-- 匯出個案資料：勾選欲匯出的個案，欄位維持固定申報格式 -->
    <CaseSelectDialog
      v-model="exportDialogVisible"
      title="匯出個案資料"
      confirm-text="確認匯出"
      :confirm-loading="exporting"
      @confirm="handleConfirmExport"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { Plus, ArrowDown } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance, type TableInstance } from 'element-plus'
import { resolveErrorMessage } from '@/api/errorCodes'
import DataTablePage from '@/components/DataTablePage.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import ImportPreviewDialog from '@/components/ImportPreviewDialog.vue'
import DialogFooter from '@/components/DialogFooter.vue'
import CaseSelectDialog from '@/components/CaseSelectDialog.vue'
import VehicleFormFields from '@/components/VehicleFormFields.vue'
import {
  listCases,
  createCase,
  updateCase,
  deleteCase,
  downloadCaseImportTemplate,
  exportCaseProfileWorkbook,
  dryRunImportCases,
  commitImportCases,
  updateCaseTransportPreference
} from '@/api/cases'
import { listSites, listVehicles, createSite, createVehicle } from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import { downloadBlob } from '@/utils/download'
import { emptyVehicleForm, vehicleFormRules } from '@/utils/vehicleForm'
import {
  REGION_LABELS,
  CASE_STATUS_LABELS,
  TRIP_PATTERN_LABELS,
  SERVICE_USAGE_TYPE_LABELS,
  type Region,
  type CaseStatus,
  type TripPattern,
  type ServiceUsageType
} from '@/types/domain'
import type { CaseDTO, CreateCaseRequest, CreateVehicleRequest, SiteDTO, VehicleDTO } from '@/types/api'

// 匯入預覽的生日僅供人工核對，改用民國年顯示；後端仍以西元 ISO 日期解析與儲存
function formatRocBirthDate(birthDate?: string): string {
  if (!birthDate) return ''
  const d = new Date(birthDate)
  if (Number.isNaN(d.getTime())) return birthDate
  const rocYear = d.getFullYear() - 1911
  return `${String(rocYear).padStart(3, '0')}/${String(d.getMonth() + 1).padStart(2, '0')}/${String(d.getDate()).padStart(2, '0')}`
}

const authStore = useAuthStore()
const activeTab = ref<'list' | 'unresolved'>('list')
const cases = ref<CaseDTO[]>([])
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
  defaultFilters: {
    q: '',
    region: '',
    status: ''
  },
  onFetch: async () => {
    const res = await listCases({
      page: page.value,
      pageSize: pageSize.value,
      q: filters.q,
      region: filters.region,
      status: filters.status,
      excludePending: true
    })
    cases.value = res.data
    total.value = res.meta.total
  }
})

// 快速行內修改區域
async function handleQuickUpdateRegion(row: CaseDTO, newRegion: Region) {
  if (row.region === newRegion) return
  try {
    await updateCase(row.id, { region: newRegion })
    row.region = newRegion
    ElMessage.success(`已將個案「${row.name}」申報區域修改為 ${REGION_LABELS[newRegion]}`)
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '更新區域失敗'))
  }
}

// 快速行內修改狀態
async function handleQuickUpdateStatus(row: CaseDTO, newStatus: CaseStatus) {
  if (row.status === newStatus) return
  try {
    await updateCase(row.id, { status: newStatus })
    row.status = newStatus
    ElMessage.success(`已將個案「${row.name}」狀態變更為 ${CASE_STATUS_LABELS[newStatus]}`)
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '更新狀態失敗'))
  }
}

// 刪除個案
async function handleDeleteCase(row: CaseDTO) {
  try {
    await ElMessageBox.confirm(
      `確定要刪除個案「${row.name}」？此操作將一併移除其關聯排班資料，且無法復原。`,
      '刪除確認',
      {
        confirmButtonText: '刪除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )
    await deleteCase(row.id)
    ElMessage.success(`個案「${row.name}」已成功刪除`)
    executeFetch()
  } catch (err: any) {
    if (err !== 'cancel') {
      ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '刪除個案失敗'))
    }
  }
}

// 下載匯入範本
async function handleDownloadTemplate() {
  try {
    const blob = await downloadCaseImportTemplate()
    downloadBlob(blob, '個案批次匯入範本.xlsx')
    ElMessage.success('個案匯入範本 (.xlsx) 下載成功')
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '下載範本失敗'))
  }
}

// 匯出個案資料：勾選交由共用的 CaseSelectDialog 處理，本頁只負責產檔與下載
const exportDialogVisible = ref(false)
const exporting = ref(false)

function openExportDialog() {
  exportDialogVisible.value = true
}

async function handleConfirmExport(cases: CaseDTO[]) {
  exporting.value = true
  try {
    const blob = await exportCaseProfileWorkbook(cases.map((row) => row.id))
    downloadBlob(blob, '個案資料彙整.xlsx')
    ElMessage.success(`已匯出 ${cases.length} 筆個案資料`)
    exportDialogVisible.value = false
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '匯出個案資料失敗'))
  } finally {
    exporting.value = false
  }
}

function openImportDialog() {
  importDialogRef.value?.open()
}

async function handleCommitImport(file: File, includeDuplicateRows: number[]) {
  return commitImportCases(file, includeDuplicateRows)
}

// 匯入完成後，若有單位/車輛待補建關聯，導引使用者前往「待維護」頁籤處理；
// 無論點選哪個按鈕都視為使用者已確認匯入結果，一併關閉匯入視窗
function handleImportSuccess() {
  executeFetch()
  ElMessageBox.confirm(
    '本次匯入若有單位或去回程車輛未比對到既有主檔，已建立資料並列入「待維護」頁籤，是否立即前往查看？',
    '匯入完成',
    { confirmButtonText: '前往待維護', cancelButtonText: '稍後再說', type: 'info' }
  )
    .then(() => {
      activeTab.value = 'unresolved'
      unresolvedLoaded = true
      fetchUnresolvedCases()
      loadSitesAndVehicles()
    })
    .catch(() => {})
    .finally(() => {
      importDialogRef.value?.close()
    })
}

// 新增個案表單
const createDialogVisible = ref(false)
const saving = ref(false)
const createFormRef = ref<FormInstance>()
const createForm = reactive<CreateCaseRequest>({
  name: '',
  nationalId: '',
  region: 'miaoli',
  homeAddress: '',
  serviceCategory: undefined,
  serviceUsageType: undefined,
  status: 'active',
  remarks: ''
})

// 除姓名外全部欄位選填：身分證字號、居住地、區域不再是硬性阻擋條件
const createRules = {
  name: [{ required: true, message: '請輸入個案姓名', trigger: 'blur' }]
}

function openCreateDialog() {
  createForm.name = ''
  createForm.nationalId = ''
  createForm.homeAddress = ''
  createForm.region = 'miaoli'
  createForm.serviceCategory = undefined
  createForm.serviceUsageType = undefined
  createForm.remarks = ''
  createDialogVisible.value = true
}

async function handleCreateCase() {
  if (!createFormRef.value) return
  await createFormRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      await createCase(createForm)
      ElMessage.success('個案建立成功')
      createDialogVisible.value = false
      executeFetch()
    } finally {
      saving.value = false
    }
  })
}

// 待維護頁籤
const unresolvedLoading = ref(false)
const unresolvedCases = ref<CaseDTO[]>([])
const availableSites = ref<SiteDTO[]>([])
const availableVehicles = ref<VehicleDTO[]>([])

async function fetchUnresolvedCases() {
  unresolvedLoading.value = true
  try {
    const res = await listCases({ unresolvedLink: true, pageSize: 100 })
    unresolvedCases.value = res.data
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '載入待維護清單失敗'))
  } finally {
    unresolvedLoading.value = false
  }
}

async function loadSitesAndVehicles() {
  const [sitesRes, vehiclesRes] = await Promise.all([
    listSites({ status: 'active', pageSize: 100 }),
    listVehicles({ status: 'active', pageSize: 100 })
  ])
  availableSites.value = sitesRes.data
  availableVehicles.value = vehiclesRes.data
}

type UnresolvedSlot = 'site' | 'outboundVehicle' | 'inboundVehicle'

const SLOT_ID_FIELD: Record<UnresolvedSlot, 'siteId' | 'outboundVehicleId' | 'inboundVehicleId'> = {
  site: 'siteId',
  outboundVehicle: 'outboundVehicleId',
  inboundVehicle: 'inboundVehicleId'
}
const SLOT_RAW_FIELD: Record<UnresolvedSlot, 'siteNameRaw' | 'outboundVehicleNameRaw' | 'inboundVehicleNameRaw'> = {
  site: 'siteNameRaw',
  outboundVehicle: 'outboundVehicleNameRaw',
  inboundVehicle: 'inboundVehicleNameRaw'
}

// 只送出被關聯的那一個欄位 ID，其餘欄位省略以維持既有關聯不變（後端契約：未帶入=不變更）
async function handleLinkSlot(row: CaseDTO, slot: UnresolvedSlot, entityId: string) {
  if (!entityId) return
  try {
    await updateCaseTransportPreference(row.id, { [SLOT_ID_FIELD[slot]]: entityId })
    ;(row as any)[SLOT_RAW_FIELD[slot]] = undefined
    if (!row.siteNameRaw && !row.outboundVehicleNameRaw && !row.inboundVehicleNameRaw) {
      unresolvedCases.value = unresolvedCases.value.filter((c) => c.id !== row.id)
    }
    ElMessage.success(`個案「${row.name}」已完成關聯`)
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '更新關聯失敗'))
  }
}

// 新增單位/車輛快速建立彈窗
const quickCreateVisible = ref(false)
const quickCreateKind = ref<'site' | 'vehicle'>('site')
const quickCreateSaving = ref(false)
const quickCreateTargetCase = ref<CaseDTO | null>(null)
const quickCreateSlot = ref<UnresolvedSlot>('site')
const quickCreateSiteForm = reactive({ name: '', region: 'miaoli' as Region, address: '', openDays: [1, 2, 3, 4, 5] })
const quickCreateVehicleForm = reactive<CreateVehicleRequest>(emptyVehicleForm())
const quickCreateVehicleFormRef = ref<FormInstance>()

// 單位/車輛名稱預先帶入匯入時的原始名稱，使用者只需確認其餘欄位即可送出，不必重打一次名稱
function openQuickCreate(kind: 'site' | 'vehicle', row: CaseDTO, slot: UnresolvedSlot = 'site') {
  quickCreateKind.value = kind
  quickCreateTargetCase.value = row
  quickCreateSlot.value = kind === 'site' ? 'site' : slot
  quickCreateSiteForm.name = kind === 'site' ? row.siteNameRaw || '' : ''
  quickCreateSiteForm.region = 'miaoli'
  quickCreateSiteForm.address = ''
  Object.assign(quickCreateVehicleForm, emptyVehicleForm(), {
    displayName: kind === 'vehicle' ? row[SLOT_RAW_FIELD[slot]] || '' : ''
  })
  quickCreateVehicleFormRef.value?.clearValidate()
  quickCreateVisible.value = true
}

async function handleQuickCreateAndLink() {
  if (!quickCreateTargetCase.value) return
  quickCreateSaving.value = true
  try {
    if (quickCreateKind.value === 'site') {
      const site = await createSite(quickCreateSiteForm)
      availableSites.value.push(site)
      await handleLinkSlot(quickCreateTargetCase.value, 'site', site.id)
    } else {
      if (!(await quickCreateVehicleFormRef.value?.validate().catch(() => false))) return
      const vehicle = await createVehicle({ ...quickCreateVehicleForm })
      availableVehicles.value.push(vehicle)
      await handleLinkSlot(quickCreateTargetCase.value, quickCreateSlot.value, vehicle.id)
    }
    quickCreateVisible.value = false
  } catch (err: any) {
    if (!err.response?.data?.error?.details?.length) {
      ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '新增並關聯失敗'))
    }
  } finally {
    quickCreateSaving.value = false
  }
}

// 待維護頁籤首次切入時才拉取清單，避免一般個案清單頁多打一次 API
let unresolvedLoaded = false
async function handleTabChange(name: string | number) {
  if (name === 'unresolved' && !unresolvedLoaded) {
    unresolvedLoaded = true
    await Promise.all([fetchUnresolvedCases(), loadSitesAndVehicles()])
  }
}

// 初始載入
executeFetch()
</script>

<style scoped>
.case-list-view {
  display: flex;
  flex-direction: column;
}

.case-tabs {
  border-radius: var(--app-radius-md);
}

/* overflow-x: auto 讓表格加總寬度超過版面時把捲軸包在面板內，不外溢到整個頁面
   （這個面板不像 DataTablePage 的 .table-container 內建這條規則，要自己補）。 */
.pending-panel {
  min-height: 120px;
  overflow-x: auto;
}

/* table-layout="auto" 底下 el-table 本體內建 width: 100%，即使拿掉 inline style
   仍會撐滿容器；要顯式蓋成 max-content 才會縮到「各欄寬度加總」，視窗夠寬時
   不再需要橫向卷軸（見 ltc-dashboard-visual-language skill 表格欄位一節）。 */
.pending-panel :deep(.el-table) {
  width: max-content;
}

/* 不用 flex-wrap: wrap，理由同照護人員管理待維護頁籤：欄寬不夠時會把
   「選擇既有單位/車輛」跟「新增」按鈕擠成第二行，即使頁面還有空間也一樣。
   改成 nowrap + table-layout="auto"，讓欄位依內容自然撐寬、有空間就單行顯示。 */
.unresolved-slot {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: nowrap;
}

.unresolved-slot > * {
  flex-shrink: 0;
}

/* flex-wrap: nowrap 只防止元素被擠到下一行，這個 span 沒有固定寬度時
   文字本身還是會自己換行，要另外鎖 white-space: nowrap 才能維持單行。 */
.unresolved-raw-name {
  color: var(--app-status-warning-fg);
  font-size: 13px;
  white-space: nowrap;
}

/* el-table-column 的 min-width prop 在 table-layout="auto" 底下只會拿去算
   DataTablePage 外層表格總寬度的預算，不會真的變成該欄的 CSS min-width；
   欄位當筆若沒有原始名稱（顯示「-」）就會被壓到只剩幾 px，跟其他有內容
   的欄位比例明顯不一致。要另外用 class-name 補一條 :deep() min-width
   才是真的鎖住下限（見 ltc-dashboard-visual-language skill 表格欄位一節）。 */
:deep(.unresolved-name-col .cell) {
  white-space: nowrap;
  min-width: 90px;
}

:deep(.unresolved-site-col .cell),
:deep(.unresolved-outbound-col .cell),
:deep(.unresolved-inbound-col .cell) {
  min-width: 220px;
}

.inline-value,
.case-status {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--app-text-regular);
}

.inline-value-clickable { cursor: pointer; }
.inline-value-clickable:hover { color: var(--app-primary); }

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--app-text-muted);
}

.status-dot-active { background: var(--app-status-success-fg); }
.status-dot-suspended { background: var(--app-status-warning-fg); }
.status-dot-closed { background: var(--app-text-muted); }
.empty-value { color: var(--app-text-muted); }

.export-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.export-toolbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.export-selected-count {
  color: var(--app-text-secondary);
  font-size: 13px;
}
</style>
