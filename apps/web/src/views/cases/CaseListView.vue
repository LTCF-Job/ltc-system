<template>
  <div class="case-list-view">
    <el-tabs v-model="activeTab" type="border-card" class="case-tabs" @tab-change="handleTabChange">
    <el-tab-pane label="個案清單" name="list">
    <DataTablePage
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
          placeholder="搜尋姓名／編號／身分證／電話／地址"
          clearable
          style="width: 270px"
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
        <el-dropdown
          v-if="authStore.can('staff')"
          trigger="click"
          @command="(val: 'xlsx' | 'csv') => handleDownloadTemplate(val)"
        >
          <el-button type="info" plain>
            <el-icon><Download /></el-icon>
            下載匯入範本
            <el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="xlsx">標準 Excel 範本 (.xlsx)</el-dropdown-item>
              <el-dropdown-item command="csv">標準 CSV 範本 (.csv)</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>

        <el-button
          v-if="authStore.can('staff')"
          type="success"
          plain
          @click="openImportDialog"
        >
          <el-icon><Upload /></el-icon>
          批次匯入個案
        </el-button>

        <el-button v-if="authStore.can('staff')" plain @click="handleExportProfile">
          <el-icon><Download /></el-icon>
          匯出個案資料
        </el-button>

        <el-button
          v-if="authStore.can('staff')"
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
          <el-table-column prop="code" label="個案編號" width="95" align="center" />
          <el-table-column prop="name" label="姓名" width="110" />
          <el-table-column prop="nationalId" label="身分證字號" width="130" align="center">
            <template #default="{ row }">
              <span class="font-mono">{{ row.nationalId || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="phone" label="聯絡電話" width="130" align="center">
            <template #default="{ row }">
              <span>{{ row.phone || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="region" label="區域" width="115" align="center">
            <template #default="{ row }">
              <el-dropdown
                v-if="authStore.can('staff')"
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
                v-if="authStore.can('staff')"
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
          <el-table-column prop="claimStartDate" label="起聘申報日" width="115" align="center" />
          <el-table-column label="排班概要" min-width="160">
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
              <el-button
                link
                type="success"
                size="small"
                :icon="Edit"
                @click="$router.push(`/cases/${row.id}?tab=basic`)"
              >
                編輯
              </el-button>
              <el-button
                v-if="authStore.can('staff')"
                link
                type="danger"
                size="small"
                @click="handleDeleteCase(row as any)"
              >
                刪除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>
    </DataTablePage>
    </el-tab-pane>

    <!-- 待補建關聯：據點/去程車/回程車比對不到主檔資料的個案，供事後關聯或新增主檔 -->
    <el-tab-pane label="待補建關聯" name="unresolved">
      <div v-loading="unresolvedLoading" class="unresolved-panel">
        <el-empty v-if="!unresolvedLoading && unresolvedCases.length === 0" description="目前沒有待補建關聯的個案" />
        <el-table v-else :data="unresolvedCases" border stripe style="width: 100%">
          <el-table-column prop="code" label="個案編號" width="95" align="center" />
          <el-table-column prop="name" label="姓名" width="110" />
          <el-table-column label="據點" min-width="220">
            <template #default="{ row }">
              <div v-if="row.siteNameRaw" class="unresolved-slot">
                <span class="unresolved-raw-name">原始名稱：{{ row.siteNameRaw }}</span>
                <el-select
                  filterable
                  placeholder="選擇既有據點"
                  style="width: 160px"
                  @change="(val: string) => handleLinkSlot(row as CaseDTO, 'site', val)"
                >
                  <el-option v-for="site in availableSites" :key="site.id" :value="site.id" :label="site.name" />
                </el-select>
                <el-button link type="primary" size="small" @click="openQuickCreate('site', row as CaseDTO)">新增據點</el-button>
              </div>
              <span v-else class="empty-value">-</span>
            </template>
          </el-table-column>
          <el-table-column label="去程車輛" min-width="220">
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
          <el-table-column label="回程車輛" min-width="220">
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

    <!-- 新增據點/車輛快速建立彈窗 -->
    <el-dialog v-model="quickCreateVisible" :title="quickCreateKind === 'site' ? '新增據點' : '新增車輛'" width="480px">
      <el-form v-if="quickCreateKind === 'site'" label-width="90px">
        <el-form-item label="據點名稱"><el-input v-model="quickCreateSiteForm.name" /></el-form-item>
        <el-form-item label="區域">
          <el-select v-model="quickCreateSiteForm.region" style="width: 100%">
            <el-option v-for="(label, key) in REGION_LABELS" :key="key" :value="key" :label="label" />
          </el-select>
        </el-form-item>
        <el-form-item label="地址"><el-input v-model="quickCreateSiteForm.address" /></el-form-item>
      </el-form>
      <el-form v-else label-width="90px">
        <el-form-item label="車牌號碼"><el-input v-model="quickCreateVehicleForm.plateNo" /></el-form-item>
        <el-form-item label="顯示名稱"><el-input v-model="quickCreateVehicleForm.displayName" /></el-form-item>
        <el-form-item label="區域">
          <el-select v-model="quickCreateVehicleForm.region" style="width: 100%">
            <el-option v-for="(label, key) in REGION_LABELS" :key="key" :value="key" :label="label" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="quickCreateVisible = false">取消</el-button>
        <el-button type="primary" :loading="quickCreateSaving" @click="handleQuickCreateAndLink">建立並關聯</el-button>
      </template>
    </el-dialog>

    <!-- 批次匯入彈窗 -->
    <ImportPreviewDialog
      ref="importDialogRef"
      title="批次匯入個案 (個案新增資料.xlsx / .csv)"
      :on-dry-run="dryRunImportCases"
      :on-commit="handleCommitImport"
      :on-download-template="handleDownloadTemplate"
      @success="executeFetch"
    >
      <template #columns="{ checkedDuplicateRows, toggleDuplicateRow }">
        <el-table-column prop="name" label="姓名" width="100" />
        <el-table-column prop="householdType" label="戶別" width="90" />
        <el-table-column prop="nationalId" label="身分證字號" width="120" />
        <el-table-column prop="gender" label="性別" width="60" />
        <el-table-column prop="birthDate" label="生日" width="100" />
        <el-table-column prop="siteName" label="據點" width="110" />
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
    <el-dialog v-model="createDialogVisible" title="新增個案基本資料" width="600px">
      <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="120px">
        <el-form-item label="個案姓名" prop="name">
          <el-input v-model="createForm.name" placeholder="請輸入姓名（含罕用字）" />
        </el-form-item>
        <el-form-item label="身分證字號" prop="nationalId">
          <el-input v-model="createForm.nationalId" placeholder="1 碼英文字母 + 9 碼數字" />
        </el-form-item>
        <el-form-item label="聯絡電話" prop="phone">
          <el-input v-model="createForm.phone" placeholder="如：0912345678" />
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
        <el-form-item label="開始申報日" prop="claimStartDate">
          <el-date-picker
            v-model="createForm.claimStartDate"
            type="date"
            placeholder="選擇申報日期"
            value-format="YYYY-MM-DD"
          />
        </el-form-item>
        <el-form-item label="服務類別" prop="serviceCategory">
          <el-radio-group v-model="createForm.serviceCategory">
            <el-radio :value="1">1. 補助</el-radio>
            <el-radio :value="2">2. 自費</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="服務使用類型" prop="serviceUsageType">
          <el-select v-model="createForm.serviceUsageType" style="width: 100%">
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
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleCreateCase">確認新增</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { Plus, Upload, Download, ArrowDown, Edit } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { resolveErrorMessage } from '@/api/errorCodes'
import DataTablePage from '@/components/DataTablePage.vue'
import ImportPreviewDialog from '@/components/ImportPreviewDialog.vue'
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
import {
  REGION_LABELS,
  CASE_STATUS_LABELS,
  TRIP_PATTERN_LABELS,
  type Region,
  type CaseStatus,
  type TripPattern
} from '@/types/domain'
import type { CaseDTO, CreateCaseRequest, SiteDTO, VehicleDTO } from '@/types/api'

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
      status: filters.status
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
      `確定要刪除個案「${row.name} (${row.code})」？此操作將一併移除其關聯排班資料，且無法復原。`,
      '刪除確認',
      {
        confirmButtonText: '確定刪除',
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
async function handleDownloadTemplate(format: any = 'xlsx') {
  try {
    const safeFormat: 'xlsx' | 'csv' = typeof format === 'string' && format.toLowerCase() === 'csv' ? 'csv' : 'xlsx'
    const blob = await downloadCaseImportTemplate(safeFormat)
    downloadBlob(blob, `個案批次匯入範本.${safeFormat}`)
    ElMessage.success(`個案匯入範本 (.${safeFormat}) 下載成功`)
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '下載範本失敗'))
  }
}

async function handleExportProfile() {
  try {
    const blob = await exportCaseProfileWorkbook()
    downloadBlob(blob, '個案資料彙整.xlsx')
    ElMessage.success('個案資料匯出完成')
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '匯出個案資料失敗'))
  }
}

function openImportDialog() {
  importDialogRef.value?.open()
}

async function handleCommitImport(file: File, includeDuplicateRows: number[]) {
  return commitImportCases(file, includeDuplicateRows)
}

// 新增個案表單
const createDialogVisible = ref(false)
const saving = ref(false)
const createFormRef = ref<FormInstance>()
const createForm = reactive<CreateCaseRequest>({
  name: '',
  nationalId: '',
  phone: '',
  region: 'miaoli',
  homeAddress: '',
  claimStartDate: new Date().toISOString().split('T')[0],
  serviceCategory: 1,
  serviceUsageType: 2,
  status: 'active',
  remarks: ''
})

// 除姓名外全部欄位選填：身分證字號、居住地、區域、起聘申報日不再是硬性阻擋條件
const createRules = {
  name: [{ required: true, message: '請輸入個案姓名', trigger: 'blur' }]
}

function openCreateDialog() {
  createForm.name = ''
  createForm.nationalId = ''
  createForm.phone = ''
  createForm.homeAddress = ''
  createForm.region = 'miaoli'
  createForm.claimStartDate = new Date().toISOString().split('T')[0]
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

// 待補建關聯頁籤
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
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '載入待補建關聯清單失敗'))
  } finally {
    unresolvedLoading.value = false
  }
}

async function loadSitesAndVehicles() {
  const [sitesRes, vehiclesRes] = await Promise.all([
    listSites({ pageSize: 100 }),
    listVehicles({ active: true, pageSize: 100 })
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

// 新增據點/車輛快速建立彈窗
const quickCreateVisible = ref(false)
const quickCreateKind = ref<'site' | 'vehicle'>('site')
const quickCreateSaving = ref(false)
const quickCreateTargetCase = ref<CaseDTO | null>(null)
const quickCreateSlot = ref<UnresolvedSlot>('site')
const quickCreateSiteForm = reactive({ name: '', region: 'miaoli' as Region, address: '', openDays: [1, 2, 3, 4, 5] })
const quickCreateVehicleForm = reactive({ plateNo: '', displayName: '', region: 'miaoli' as Region })

function openQuickCreate(kind: 'site' | 'vehicle', row: CaseDTO, slot: UnresolvedSlot = 'site') {
  quickCreateKind.value = kind
  quickCreateTargetCase.value = row
  quickCreateSlot.value = kind === 'site' ? 'site' : slot
  quickCreateSiteForm.name = ''
  quickCreateSiteForm.region = 'miaoli'
  quickCreateSiteForm.address = ''
  quickCreateVehicleForm.plateNo = ''
  quickCreateVehicleForm.displayName = ''
  quickCreateVehicleForm.region = 'miaoli'
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
      const vehicle = await createVehicle(quickCreateVehicleForm)
      availableVehicles.value.push(vehicle)
      await handleLinkSlot(quickCreateTargetCase.value, quickCreateSlot.value, vehicle.id)
    }
    quickCreateVisible.value = false
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '新增並關聯失敗'))
  } finally {
    quickCreateSaving.value = false
  }
}

// 待補建關聯頁籤首次切入時才拉取清單，避免一般個案清單頁多打一次 API
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
  border-radius: 8px;
}

.unresolved-panel {
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

.inline-value,
.case-status {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--el-text-color-regular);
}

.inline-value-clickable { cursor: pointer; }
.inline-value-clickable:hover { color: var(--el-color-primary); }

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--el-text-color-placeholder);
}

.status-dot-active { background: var(--el-color-success); }
.status-dot-suspended { background: var(--el-color-warning); }
.status-dot-closed { background: var(--el-text-color-placeholder); }
.empty-value { color: var(--el-text-color-placeholder); }
</style>
