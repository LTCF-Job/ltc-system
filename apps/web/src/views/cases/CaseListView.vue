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

    <!-- 待維護：單位/去回程車輛比對不到主檔、生日或身分證字號格式錯誤、疑似重複個案，
         合併成一列一實體＋彙總問題欄的呈現方式（版面比照照護人員管理待維護頁籤） -->
    <el-tab-pane label="待維護" name="unresolved">
      <div v-loading="unresolvedLoading" class="pending-panel">
        <el-empty v-if="!unresolvedLoading && pendingRows.length === 0" description="目前沒有待維護的個案" />
        <el-table v-else :data="pendingRows" border stripe table-layout="auto" row-key="key">
          <el-table-column prop="name" label="姓名" min-width="90" class-name="unresolved-name-col" />
          <el-table-column label="問題" min-width="260" class-name="unresolved-issue-col">
            <template #default="{ row }">
              <div class="issue-tags">
                <el-tag v-for="issue in row.issues" :key="issue" type="warning" size="small" effect="light">
                  {{ issue }}
                </el-tag>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="120" fixed="right" align="center" class-name="unresolved-action-col">
            <template #default="{ row }">
              <TableRowActions>
                <el-button v-if="row.kind === 'case'" link type="primary" size="small" @click="openPendingCaseEdit(row as PendingCaseRow)">
                  編輯
                </el-button>
                <el-button v-else link type="primary" size="small" @click="openDuplicateResolve(row as PendingDuplicateRow)">
                  人工裁決
                </el-button>
              </TableRowActions>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-tab-pane>
    </el-tabs>

    <!-- 待維護個案編輯彈窗：只顯示該列實際缺漏的欄位（單位/車輛關聯、生日、身分證字號） -->
    <el-dialog v-model="pendingEditVisible" title="補齊個案資料" width="min(600px, calc(100vw - 32px))">
      <template v-if="pendingEditTarget">
        <p class="pending-edit-hint">
          匯入「{{ pendingEditTarget.name }}」時，以下欄位無法對應到既有主檔或格式不正確。補齊並儲存後，這筆個案就會離開待維護清單。
        </p>
        <el-form label-width="110px">
          <el-form-item v-if="pendingEditTarget.siteNameRaw" label="單位">
            <div class="pending-edit-field">
              <span class="pending-edit-raw">原始名稱：{{ pendingEditTarget.siteNameRaw }}</span>
              <div class="pending-edit-control">
                <el-select v-model="pendingEditForm.siteId" filterable placeholder="選擇既有單位" class="pending-edit-input">
                  <el-option v-for="site in availableSites" :key="site.id" :value="site.id" :label="site.name" />
                </el-select>
                <el-button link type="primary" size="small" @click="openQuickCreate('site', pendingEditTarget)">新增單位</el-button>
              </div>
            </div>
          </el-form-item>
          <el-form-item v-if="pendingEditTarget.outboundVehicleNameRaw" label="去程車輛">
            <div class="pending-edit-field">
              <span class="pending-edit-raw">原始名稱：{{ pendingEditTarget.outboundVehicleNameRaw }}</span>
              <div class="pending-edit-control">
                <el-select v-model="pendingEditForm.outboundVehicleId" filterable placeholder="選擇既有車輛" class="pending-edit-input">
                  <el-option v-for="vehicle in availableVehicles" :key="vehicle.id" :value="vehicle.id" :label="vehicle.displayName" />
                </el-select>
                <el-button link type="primary" size="small" @click="openQuickCreate('vehicle', pendingEditTarget, 'outboundVehicle')">新增車輛</el-button>
              </div>
            </div>
          </el-form-item>
          <el-form-item v-if="pendingEditTarget.inboundVehicleNameRaw" label="回程車輛">
            <div class="pending-edit-field">
              <span class="pending-edit-raw">原始名稱：{{ pendingEditTarget.inboundVehicleNameRaw }}</span>
              <div class="pending-edit-control">
                <el-select v-model="pendingEditForm.inboundVehicleId" filterable placeholder="選擇既有車輛" class="pending-edit-input">
                  <el-option v-for="vehicle in availableVehicles" :key="vehicle.id" :value="vehicle.id" :label="vehicle.displayName" />
                </el-select>
                <el-button link type="primary" size="small" @click="openQuickCreate('vehicle', pendingEditTarget, 'inboundVehicle')">新增車輛</el-button>
              </div>
            </div>
          </el-form-item>
          <el-form-item v-if="pendingEditTarget.birthDateRaw" label="生日">
            <div class="pending-edit-field">
              <span class="pending-edit-raw">原始字串：{{ pendingEditTarget.birthDateRaw }}</span>
              <div class="pending-edit-control">
                <el-date-picker v-model="pendingEditForm.birthDate" type="date" placeholder="選擇正確生日" value-format="YYYY-MM-DD" class="pending-edit-input" />
              </div>
            </div>
          </el-form-item>
          <el-form-item v-if="pendingEditTarget.nationalIdInvalid" label="身分證字號">
            <div class="pending-edit-field">
              <div class="pending-edit-control">
                <el-input v-model="pendingEditForm.nationalId" placeholder="請重新輸入完整身分證字號" class="pending-edit-input" />
              </div>
            </div>
          </el-form-item>
        </el-form>
      </template>
      <template #footer>
        <DialogFooter
          confirm-text="儲存"
          :loading="pendingEditSaving"
          @confirm="handlePendingEditSubmit"
          @cancel="pendingEditVisible = false"
        />
      </template>
    </el-dialog>

    <!-- 疑似重複個案人工裁決彈窗：先解密明文供比對，再讓使用者確認是否為新個案 -->
    <el-dialog v-model="duplicateResolveVisible" title="疑似重複個案裁決" width="min(520px, calc(100vw - 32px))">
      <el-descriptions v-if="duplicateResolveTarget" :column="1" border size="small">
        <el-descriptions-item label="匯入姓名">{{ duplicateResolveTarget.name }}</el-descriptions-item>
        <el-descriptions-item label="身分證字號">
          {{ duplicateResolveNationalId || duplicateResolveTarget.nationalIdMasked || '未提供' }}
        </el-descriptions-item>
        <el-descriptions-item label="疑似重複之既有個案">{{ duplicateResolveTarget.duplicateCaseName }}</el-descriptions-item>
        <el-descriptions-item label="單位">{{ duplicateResolveTarget.siteNameRaw || duplicateResolveTarget.siteName || '-' }}</el-descriptions-item>
        <el-descriptions-item label="去程車輛">{{ duplicateResolveTarget.outboundVehicleNameRaw || duplicateResolveTarget.outboundVehicle || '-' }}</el-descriptions-item>
        <el-descriptions-item label="回程車輛">{{ duplicateResolveTarget.inboundVehicleNameRaw || duplicateResolveTarget.inboundVehicle || '-' }}</el-descriptions-item>
        <el-descriptions-item label="備註">{{ duplicateResolveTarget.remarks || '-' }}</el-descriptions-item>
      </el-descriptions>
      <el-checkbox v-model="duplicateMergeRemarks" style="margin-top: 12px" label="若視為既有個案，一併把備註併入既有個案" />
      <template #footer>
        <div class="duplicate-resolve-actions">
          <el-button @click="duplicateResolveVisible = false">稍後再說</el-button>
          <el-button type="warning" :loading="duplicateResolveSaving" @click="handleDuplicateResolve('merged_existing')">
            視為既有個案
          </el-button>
          <el-button type="primary" :loading="duplicateResolveSaving" @click="handleDuplicateResolve('confirmed_new')">
            確認為新個案
          </el-button>
        </div>
      </template>
    </el-dialog>

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
      <template #columns>
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
        <el-table-column label="重複個案" width="170" align="center">
          <template #default="{ row }">
            <el-tooltip
              v-if="row.isDuplicate"
              :content="`與既有個案「${row.duplicateOf?.name ?? '未知'}」(${row.duplicateOf?.code ?? '未知'}) 疑似重複，正式匯入時將建立為待裁決項目`"
              placement="top"
            >
              <el-tag type="warning" size="small">待裁決</el-tag>
            </el-tooltip>
            <span v-else class="empty-value">-</span>
          </template>
        </el-table-column>
      </template>
    </ImportPreviewDialog>

    <!-- 新增個案彈窗：跟待維護資料頁籤的「新增個案並綁定」共用同一個元件與 API -->
    <CaseCreateDialog v-model="createDialogVisible" @created="handleCaseCreated" />

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
import DataTablePage from '@/components/DataTablePage.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import ImportPreviewDialog from '@/components/ImportPreviewDialog.vue'
import DialogFooter from '@/components/DialogFooter.vue'
import CaseSelectDialog from '@/components/CaseSelectDialog.vue'
import VehicleFormFields from '@/components/VehicleFormFields.vue'
import CaseCreateDialog from '@/components/cases/CaseCreateDialog.vue'
import {
  listCases,
  listAllCases,
  updateCase,
  deleteCase,
  downloadCaseImportTemplate,
  exportCaseProfileWorkbook,
  dryRunImportCases,
  commitImportCases,
  updateCaseTransportPreference,
  listCaseDuplicateCandidates,
  revealCaseDuplicateCandidateNationalId,
  resolveCaseDuplicateCandidate
} from '@/api/cases'
import { listAllSites, listAllVehicles, listSites, listVehicles, createSite, createVehicle } from '@/api/masters'
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
import type {
  CaseDTO,
  CaseDuplicateCandidateDTO,
  CreateVehicleRequest,
  SiteDTO,
  UpdateCaseTransportPreferenceRequest,
  VehicleDTO
} from '@/types/api'

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
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  }
}

// 快速行內修改狀態
async function handleQuickUpdateStatus(row: CaseDTO, newStatus: CaseStatus) {
  if (row.status === newStatus) return
  try {
    await updateCase(row.id, { status: newStatus })
    row.status = newStatus
    ElMessage.success(`已將個案「${row.name}」狀態變更為 ${CASE_STATUS_LABELS[newStatus]}`)
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
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
  } catch {
    // 使用者取消或 API 錯誤皆不在此重複顯示。
  }
}

// 下載匯入範本
async function handleDownloadTemplate() {
  try {
    const blob = await downloadCaseImportTemplate()
    downloadBlob(blob, '個案批次匯入範本.xlsx')
    ElMessage.success('個案匯入範本 (.xlsx) 下載成功')
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
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
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  } finally {
    exporting.value = false
  }
}

function openImportDialog() {
  importDialogRef.value?.open()
}

async function handleCommitImport(file: File) {
  return commitImportCases(file)
}

// 匯入完成後，若有列進入待維護（單位/車輛待關聯、生日/身分證字號待補正、疑似重複個案
// 待裁決），導引使用者前往「待維護」頁籤處理；無論點選哪個按鈕都視為使用者已確認匯入
// 結果，一併關閉匯入視窗
function handleImportSuccess() {
  executeFetch()
  ElMessageBox.confirm(
    '本次匯入若有資料待補齊關聯、補正格式或裁決疑似重複個案，已列入「待維護」頁籤，是否立即前往查看？',
    '匯入完成',
    { confirmButtonText: '前往待維護', cancelButtonText: '稍後再說', type: 'info' }
  )
    .then(() => {
      activeTab.value = 'unresolved'
      unresolvedLoaded = true
      fetchPendingData()
      loadSitesAndVehicles()
    })
    .catch(() => {})
    .finally(() => {
      importDialogRef.value?.close()
    })
}

// 新增個案彈窗：欄位、驗證與送出邏輯都在 CaseCreateDialog 元件內，這裡只管開關與成功後的收尾
const createDialogVisible = ref(false)

function openCreateDialog() {
  createDialogVisible.value = true
}

function handleCaseCreated() {
  executeFetch()
}

// 待維護頁籤：A/B 類（單位/車輛待關聯、生日/身分證字號待補正）與 C 類（疑似重複個案待裁決）
// 分別來自不同 API，合併成一列一實體＋彙總問題欄呈現（比照照護人員管理待維護頁籤）。
const unresolvedLoading = ref(false)
const unresolvedCases = ref<CaseDTO[]>([])
const duplicateCandidates = ref<CaseDuplicateCandidateDTO[]>([])
const availableSites = ref<SiteDTO[]>([])
const availableVehicles = ref<VehicleDTO[]>([])

interface PendingCaseRow extends CaseDTO {
  kind: 'case'
  key: string
  issues: string[]
}

interface PendingDuplicateRow extends CaseDuplicateCandidateDTO {
  kind: 'duplicate'
  key: string
  issues: string[]
}

type PendingRow = PendingCaseRow | PendingDuplicateRow

function caseIssues(row: CaseDTO): string[] {
  const issues: string[] = []
  if (row.siteNameRaw) issues.push('單位待關聯')
  if (row.outboundVehicleNameRaw) issues.push('去程車輛待關聯')
  if (row.inboundVehicleNameRaw) issues.push('回程車輛待關聯')
  if (row.birthDateRaw) issues.push('生日待補正')
  if (row.nationalIdInvalid) issues.push('身分證字號待補打')
  return issues
}

const pendingRows = computed<PendingRow[]>(() => [
  ...unresolvedCases.value.map((c) => ({ ...c, kind: 'case' as const, key: `case:${c.id}`, issues: caseIssues(c) })),
  ...duplicateCandidates.value.map((d) => ({
    ...d,
    kind: 'duplicate' as const,
    key: `dup:${d.id}`,
    issues: [`疑似重複個案（既有：${d.duplicateCaseName}）`]
  }))
])

async function fetchUnresolvedCases() {
  unresolvedCases.value = await listAllCases({ unresolvedLink: true })
}

async function fetchDuplicateCandidates() {
  duplicateCandidates.value = await listCaseDuplicateCandidates()
}

async function fetchPendingData() {
  unresolvedLoading.value = true
  try {
    await Promise.all([fetchUnresolvedCases(), fetchDuplicateCandidates()])
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  } finally {
    unresolvedLoading.value = false
  }
}

async function loadSitesAndVehicles() {
  const [sitesRes, vehiclesRes] = await Promise.all([
    listAllSites({ status: 'active' }),
    listAllVehicles({ status: 'active' })
  ])
  availableSites.value = sitesRes
  availableVehicles.value = vehiclesRes
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

// 完整替換交通偏好，保留同一列中尚未處理的既有關聯與原始名稱。
async function handleLinkSlot(row: CaseDTO, slot: UnresolvedSlot, entityId: string) {
  if (!entityId) return
  try {
    // 這支 API 是完整替換：其他兩欄還沒關聯的原始名稱要原樣回送，否則會被清成 NULL，
    // 該列就無聲離開待維護清單，匯入時填的名稱也再也找不回來
    const payload: UpdateCaseTransportPreferenceRequest = {
      siteId: row.siteId || null,
      outboundVehicleId: row.outboundVehicleId || null,
      inboundVehicleId: row.inboundVehicleId || null,
      siteNameRaw: row.siteNameRaw || '',
      outboundVehicleNameRaw: row.outboundVehicleNameRaw || '',
      inboundVehicleNameRaw: row.inboundVehicleNameRaw || ''
    }
    payload[SLOT_ID_FIELD[slot]] = entityId
    payload[SLOT_RAW_FIELD[slot]] = ''
    await updateCaseTransportPreference(row.id, payload)
    ;(row as any)[SLOT_ID_FIELD[slot]] = entityId
    ;(row as any)[SLOT_RAW_FIELD[slot]] = undefined
    if (!row.siteNameRaw && !row.outboundVehicleNameRaw && !row.inboundVehicleNameRaw) {
      unresolvedCases.value = unresolvedCases.value.filter((c) => c.id !== row.id)
    }
    ElMessage.success(`個案「${row.name}」已完成關聯`)
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  }
}

// 待維護個案編輯彈窗：一次處理該列所有缺漏欄位，成功後就地移除已解決的項目，
// 全部解決才把整列從待維護清單移除。
const pendingEditVisible = ref(false)
const pendingEditTarget = ref<CaseDTO | null>(null)
const pendingEditSaving = ref(false)
const pendingEditForm = reactive<{
  siteId: string
  outboundVehicleId: string
  inboundVehicleId: string
  birthDate: string
  nationalId: string
}>({ siteId: '', outboundVehicleId: '', inboundVehicleId: '', birthDate: '', nationalId: '' })

function openPendingCaseEdit(row: PendingCaseRow) {
  // pendingRows 是 computed 展開出來的副本，改副本不會反映到清單上；一律取回原始列物件
  pendingEditTarget.value = unresolvedCases.value.find((c) => c.id === row.id) ?? row
  pendingEditForm.siteId = ''
  pendingEditForm.outboundVehicleId = ''
  pendingEditForm.inboundVehicleId = ''
  pendingEditForm.birthDate = ''
  pendingEditForm.nationalId = ''
  pendingEditVisible.value = true
}

async function handlePendingEditSubmit() {
  const row = pendingEditTarget.value
  if (!row) return
  pendingEditSaving.value = true
  try {
    // 沒有挑任何關聯就不要送出：這支 API 是完整替換，空送一次會把三欄尚未處理的
    // 原始名稱一併清成 NULL。有挑的那幾欄才清掉自己的原始名稱，其餘原樣回送。
    if (pendingEditForm.siteId || pendingEditForm.outboundVehicleId || pendingEditForm.inboundVehicleId) {
      const payload: UpdateCaseTransportPreferenceRequest = {
        siteId: pendingEditForm.siteId || row.siteId || null,
        outboundVehicleId: pendingEditForm.outboundVehicleId || row.outboundVehicleId || null,
        inboundVehicleId: pendingEditForm.inboundVehicleId || row.inboundVehicleId || null,
        siteNameRaw: pendingEditForm.siteId ? '' : row.siteNameRaw || '',
        outboundVehicleNameRaw: pendingEditForm.outboundVehicleId ? '' : row.outboundVehicleNameRaw || '',
        inboundVehicleNameRaw: pendingEditForm.inboundVehicleId ? '' : row.inboundVehicleNameRaw || ''
      }
      await updateCaseTransportPreference(row.id, payload)
      if (pendingEditForm.siteId) {
        row.siteId = pendingEditForm.siteId
        row.siteNameRaw = undefined
      }
      if (pendingEditForm.outboundVehicleId) {
        row.outboundVehicleId = pendingEditForm.outboundVehicleId
        row.outboundVehicleNameRaw = undefined
      }
      if (pendingEditForm.inboundVehicleId) {
        row.inboundVehicleId = pendingEditForm.inboundVehicleId
        row.inboundVehicleNameRaw = undefined
      }
    }
    if (row.birthDateRaw && pendingEditForm.birthDate) {
      await updateCase(row.id, { birthDate: pendingEditForm.birthDate })
      row.birthDateRaw = undefined
    }
    if (row.nationalIdInvalid && pendingEditForm.nationalId) {
      await updateCase(row.id, { nationalId: pendingEditForm.nationalId })
      row.nationalIdInvalid = false
    }
    if (!row.siteNameRaw && !row.outboundVehicleNameRaw && !row.inboundVehicleNameRaw && !row.birthDateRaw && !row.nationalIdInvalid) {
      unresolvedCases.value = unresolvedCases.value.filter((c) => c.id !== row.id)
    }
    ElMessage.success(`個案「${row.name}」資料已更新`)
    pendingEditVisible.value = false
  } catch {
    // 全域攔截器負責顯示 API 錯誤（含身分證字號格式錯誤 400、與既有個案衝突 409）。
  } finally {
    pendingEditSaving.value = false
  }
}

// 疑似重複個案人工裁決：開啟時先解密明文供比對（若有身分證字號），裁決前用
// ElMessageBox 二次確認，避免誤觸「確認為新個案」或「視為既有個案」。
const duplicateResolveVisible = ref(false)
const duplicateResolveTarget = ref<CaseDuplicateCandidateDTO | null>(null)
const duplicateResolveNationalId = ref('')
const duplicateResolveSaving = ref(false)
const duplicateMergeRemarks = ref(true)

async function openDuplicateResolve(row: PendingDuplicateRow) {
  duplicateResolveTarget.value = row
  duplicateResolveNationalId.value = ''
  duplicateMergeRemarks.value = true
  duplicateResolveVisible.value = true
  if (row.nationalIdMasked) {
    try {
      const { nationalId } = await revealCaseDuplicateCandidateNationalId(row.id)
      duplicateResolveNationalId.value = nationalId
    } catch {
      // 全域攔截器負責顯示 API 錯誤；解密失敗時維持顯示遮罩值。
    }
  }
}

async function handleDuplicateResolve(decision: 'confirmed_new' | 'merged_existing') {
  const row = duplicateResolveTarget.value
  if (!row) return
  try {
    await ElMessageBox.confirm(
      decision === 'confirmed_new'
        ? `確認個案「${row.name}」為獨立新個案，將正式建立個案主檔？`
        : `確認將「${row.name}」視為既有個案「${row.duplicateCaseName}」，${duplicateMergeRemarks.value ? '並把備註併入既有個案' : '不合併任何資料'}？`,
      '裁決確認',
      { confirmButtonText: '確認', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return
  }
  duplicateResolveSaving.value = true
  try {
    await resolveCaseDuplicateCandidate(row.id, { decision, mergeRemarks: decision === 'merged_existing' ? duplicateMergeRemarks.value : undefined })
    duplicateCandidates.value = duplicateCandidates.value.filter((d) => d.id !== row.id)
    ElMessage.success(`個案「${row.name}」已完成裁決`)
    duplicateResolveVisible.value = false
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  } finally {
    duplicateResolveSaving.value = false
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
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  } finally {
    quickCreateSaving.value = false
  }
}

// 待維護頁籤首次切入時才拉取清單，避免一般個案清單頁多打一次 API
let unresolvedLoaded = false
async function handleTabChange(name: string | number) {
  if (name === 'unresolved' && !unresolvedLoaded) {
    unresolvedLoaded = true
    await Promise.all([fetchPendingData(), loadSitesAndVehicles()])
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

.pending-edit-hint {
  margin: 0 0 var(--app-space-4);
  color: var(--app-text-secondary);
  font-size: var(--app-font-sm);
  line-height: 1.6;
}

/* 對話框寬度固定，不像表格可以依內容撐寬，所以這裡不套用 component-contract
   表格欄位那組「原始名稱＋選單＋按鈕」單行 nowrap 寫法——那組靠 flex-shrink: 0
   維持單行，在固定寬度容器裡會直接溢出對話框、把「新增」按鈕推到看不見。 */
.pending-edit-field {
  display: flex;
  flex-direction: column;
  gap: var(--app-space-1);
  width: 100%;
  min-width: 0;
}

.pending-edit-control {
  display: flex;
  align-items: center;
  gap: var(--app-space-2);
  width: 100%;
  min-width: 0;
}

/* min-width: 0 解除 flex item 預設的 min-width: auto，否則選單會被內容撐寬
   而不肯收縮，「新增」按鈕一樣會被擠出容器。 */
.pending-edit-input {
  flex: 1;
  min-width: 0;
}

/* el-date-picker 的根元素吃 Element Plus 的 .el-date-editor { width: 220px }，
   優先權高於上面那條單一 class 規則，不另外蓋寬度的話只有生日欄會短一截。 */
.pending-edit-control :deep(.el-date-editor.el-input) {
  flex: 1;
  width: auto;
  min-width: 0;
}

.pending-edit-raw {
  color: var(--app-text-secondary);
  font-size: var(--app-font-sm);
  line-height: 1.4;
}

/* el-table-column 的 min-width prop 在 table-layout="auto" 底下只會拿去算
   DataTablePage 外層表格總寬度的預算，不會真的變成該欄的 CSS min-width；
   欄位當筆內容較短就會被壓到只剩幾 px，跟其他有內容的欄位比例明顯不一致。
   要另外用 class-name 補一條 :deep() min-width 才是真的鎖住下限
   （見 ltc-dashboard-visual-language skill 表格欄位一節）。 */
:deep(.unresolved-name-col .cell) {
  white-space: nowrap;
  min-width: 90px;
}

:deep(.unresolved-issue-col .cell) {
  min-width: 260px;
}

:deep(.unresolved-action-col .cell) {
  min-width: 120px;
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

/* nowrap 才能讓 width: max-content 的表格依標籤總長撐寬；用 wrap 的話一列
   五個問題標籤在版面還有空間時就先折成三行，欄位反而擠在左側。 */
.issue-tags {
  display: flex;
  flex-wrap: nowrap;
  gap: var(--app-space-2);
}

.issue-tags .el-tag {
  flex-shrink: 0;
}

.duplicate-resolve-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
