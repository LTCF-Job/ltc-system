<template>
  <div class="case-list-view">
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
          placeholder="搜尋個案姓名／編號／身分證"
          clearable
          style="width: 240px"
          @keyup.enter="handleSearch"
        />

        <el-select
          v-model="filters.region"
          placeholder="全部區域"
          clearable
          style="width: 130px"
          @change="handleSearch"
        >
          <el-option label="全部區域" value="" />
          <el-option label="苗栗" value="miaoli" />
          <el-option label="新竹" value="hsinchu" />
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
        <el-button
          v-if="authStore.can('staff')"
          type="info"
          plain
          @click="handleDownloadTemplate"
        >
          <el-icon><Download /></el-icon>
          下載匯入範本
        </el-button>

        <el-button
          v-if="authStore.can('staff')"
          type="success"
          plain
          @click="openImportDialog"
        >
          <el-icon><Upload /></el-icon>
          批次匯入個案
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
              <span class="font-mono">{{ row.nationalId || row.nationalIdMasked || '-' }}</span>
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
                  <el-tag
                    size="small"
                    :type="row.region === 'miaoli' ? 'warning' : 'primary'"
                    effect="light"
                    style="cursor: pointer;"
                  >
                    {{ REGION_LABELS[row.region as Region] || row.region }}
                    <el-icon class="el-icon--right"><ArrowDown /></el-icon>
                  </el-tag>
                </span>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="miaoli">
                      <el-tag size="small" type="warning">苗栗</el-tag>
                    </el-dropdown-item>
                    <el-dropdown-item command="hsinchu">
                      <el-tag size="small" type="primary">新竹</el-tag>
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
              <el-tag v-else size="small" :type="row.region === 'miaoli' ? 'warning' : 'primary'">
                {{ REGION_LABELS[row.region as Region] || row.region }}
              </el-tag>
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
                  <el-tag
                    size="small"
                    :type="row.status === 'active' ? 'success' : (row.status === 'suspended' ? 'info' : 'danger')"
                    effect="light"
                    style="cursor: pointer;"
                  >
                    {{ CASE_STATUS_LABELS[row.status as CaseStatus] || row.status }}
                    <el-icon class="el-icon--right"><ArrowDown /></el-icon>
                  </el-tag>
                </span>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="active">
                      <el-tag size="small" type="success">在案</el-tag>
                    </el-dropdown-item>
                    <el-dropdown-item command="suspended">
                      <el-tag size="small" type="info">暫停</el-tag>
                    </el-dropdown-item>
                    <el-dropdown-item command="closed">
                      <el-tag size="small" type="danger">停案</el-tag>
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
              <el-tag
                v-else
                size="small"
                :type="row.status === 'active' ? 'success' : (row.status === 'suspended' ? 'info' : 'danger')"
              >
                {{ CASE_STATUS_LABELS[row.status as CaseStatus] || row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="claimStartDate" label="起聘申報日" width="115" align="center" />
          <el-table-column label="排班概要" min-width="160">
            <template #default="{ row }">
              <span v-if="row.activeSchedule">
                {{ TRIP_PATTERN_LABELS[row.activeSchedule.tripPattern as TripPattern] }}
                ({{ row.activeSchedule.weekdays?.map((w: number) => `週${'一二三四五六日'[w-1]}`).join('、') }})
              </span>
              <el-tag v-else size="small" type="info">尚未設定排班</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="homeAddress" label="住家地址" min-width="190" show-overflow-tooltip />

          <el-table-column label="操作" width="220" fixed="right" align="center">
            <template #default="{ row }">
              <el-button
                link
                type="primary"
                size="small"
                @click="$router.push(`/cases/${row.id}?tab=basic`)"
              >
                編輯明細
              </el-button>
              <el-button
                link
                type="success"
                size="small"
                @click="$router.push(`/cases/${row.id}?tab=schedule`)"
              >
                排班設定
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

    <!-- 批次匯入彈窗 -->
    <ImportPreviewDialog
      ref="importDialogRef"
      title="批次匯入個案 (個案新增資料.xlsx / .csv)"
      :on-dry-run="dryRunImportCases"
      :on-commit="handleCommitImport"
      :on-download-template="handleDownloadTemplate"
      @success="executeFetch"
    >
      <template #columns>
        <el-table-column prop="name" label="姓名" width="120" />
        <el-table-column prop="region" label="申報地區" width="90" />
        <el-table-column prop="claimStartDate" label="開始申報日" width="120" />
        <el-table-column prop="siteName" label="據點" width="120" />
        <el-table-column prop="weekdays" label="開放時間" width="130" />
        <el-table-column prop="departTime" label="去程時間" width="100" />
        <el-table-column prop="returnTime" label="回程時間" width="100" />
        <el-table-column prop="tripPattern" label="趟數" width="80" />
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
        <el-form-item label="申報區域" prop="region">
          <el-radio-group v-model="createForm.region">
            <el-radio value="miaoli">苗栗</el-radio>
            <el-radio value="hsinchu">新竹</el-radio>
          </el-radio-group>
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
import { Plus, Upload, Download, ArrowDown } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import DataTablePage from '@/components/DataTablePage.vue'
import ImportPreviewDialog from '@/components/ImportPreviewDialog.vue'
import {
  listCases,
  createCase,
  updateCase,
  deleteCase,
  downloadCaseImportTemplate,
  dryRunImportCases,
  commitImportCases
} from '@/api/cases'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import {
  REGION_LABELS,
  CASE_STATUS_LABELS,
  TRIP_PATTERN_LABELS,
  type Region,
  type CaseStatus,
  type TripPattern
} from '@/types/domain'
import type { CaseDTO, CreateCaseRequest } from '@/types/api'

const authStore = useAuthStore()
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
    ElMessage.error(err.message || '更新區域失敗')
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
    ElMessage.error(err.message || '更新狀態失敗')
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
      ElMessage.error(err.message || '刪除個案失敗')
    }
  }
}

// 下載匯入範本
async function handleDownloadTemplate() {
  try {
    const blob = await downloadCaseImportTemplate()
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = '個案批次匯入範本.csv'
    a.click()
    window.URL.revokeObjectURL(url)
    ElMessage.success('個案匯入範本下載成功')
  } catch (err: any) {
    ElMessage.error(err.message || '下載範本失敗')
  }
}

function openImportDialog() {
  importDialogRef.value?.open()
}

async function handleCommitImport(file: File) {
  await commitImportCases(file)
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
  claimStartDate: new Date().toISOString().split('T')[0],
  serviceCategory: 1,
  serviceUsageType: 2,
  status: 'active'
})

const createRules = {
  name: [{ required: true, message: '請輸入個案姓名', trigger: 'blur' }],
  nationalId: [{ required: true, message: '請輸入身分證字號', trigger: 'blur' }],
  region: [{ required: true, message: '請選擇區域', trigger: 'change' }],
  homeAddress: [{ required: true, message: '請輸入住家地址', trigger: 'blur' }],
  claimStartDate: [{ required: true, message: '請選擇開始申報日', trigger: 'change' }]
}

function openCreateDialog() {
  createForm.name = ''
  createForm.nationalId = ''
  createForm.homeAddress = ''
  createForm.region = 'miaoli'
  createForm.claimStartDate = new Date().toISOString().split('T')[0]
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

// 初始載入
executeFetch()
</script>

<style scoped>
.case-list-view {
  display: flex;
  flex-direction: column;
}
</style>
