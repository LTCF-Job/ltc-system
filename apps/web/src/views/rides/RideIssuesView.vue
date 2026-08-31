<template>
  <div class="ride-issues-view">
    <PageHeader title="異常集中處理" />
    <!-- 篩選列 -->
    <el-card shadow="never" class="filter-card mb-3" style="margin-bottom: 12px;">
      <el-row :gutter="16" align="middle">
        <el-col :xs="24" :lg="18" class="issue-filter-controls">
          <el-input
            v-model="issueQuery"
            placeholder="搜尋個案姓名／涉及車輛／說明"
            clearable
            style="width: 240px;"
            @keyup.enter="fetchIssues"
          />
          <el-button type="primary" @click="fetchIssues">
            查詢
          </el-button>
          <el-button @click="handleReset">
            重設
          </el-button>
        </el-col>
      </el-row>
    </el-card>

    <el-tabs v-model="activeTab" type="border-card" class="issues-tabs" @tab-change="fetchIssues">
      <!-- 分頁 1：混車衝突 -->
      <el-tab-pane label="混車衝突待裁決" name="conflict">
        <DataTablePage :loading="loading">
        <template #table>
        <el-table :data="issueList" border stripe>
          <el-table-column prop="serviceDate" label="服務日期" width="110" />
          <el-table-column prop="caseName" label="個案姓名" width="120" />
          <el-table-column label="趟次" width="80" align="center">
            <template #default="{ row }">
              第 {{ row.legSeq }} 趟
            </template>
          </el-table-column>
          <el-table-column prop="description" label="衝突說明" min-width="260" show-overflow-tooltip />
          <el-table-column label="涉及車輛" width="200">
            <template #default="{ row }">
              <span v-for="(v, idx) in row.vehicles" :key="idx" class="vehicle-name">
                {{ v }}<span v-if="Number(idx) < row.vehicles.length - 1" class="vehicle-separator">、</span>
              </span>
            </template>
          </el-table-column>

          <el-table-column
            v-if="authStore.can('staff')"
            label="操作"
            width="120"
            align="center"
          >
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="openResolveDialog(row as any)">
                人工裁決
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        </template>
        </DataTablePage>
      </el-tab-pane>

      <!-- 分頁 2：未回報清單 -->
      <el-tab-pane label="應搭未回報清單" name="unreported">
        <DataTablePage :loading="loading">
        <template #table>
        <el-table :data="issueList" border stripe>
          <el-table-column prop="serviceDate" label="服務日期" width="110" />
          <el-table-column prop="caseName" label="個案姓名" width="120" />
          <el-table-column label="趟次" width="80" align="center">
            <template #default="{ row }">
              第 {{ row.legSeq }} 趟
            </template>
          </el-table-column>
          <el-table-column prop="description" label="說明" min-width="260" />
          <el-table-column label="操作" width="160" align="center">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="$router.push('/rides/missing')">
                前往回報
              </el-button>
              <el-button link type="info" size="small" @click="$router.push(`/cases/${row.caseId}`)">
                查看排班
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        </template>
        </DataTablePage>
      </el-tab-pane>

      <!-- 分頁 3：表單匯入錯誤 -->
      <el-tab-pane label="表單匯入異常" name="import_error">
        <DataTablePage :loading="loading">
        <template #table>
        <el-table :data="issueList" border stripe>
          <el-table-column prop="serviceDate" label="服務日期" width="110" />
          <el-table-column prop="caseName" label="回報文字/欄位" width="180" />
          <el-table-column prop="description" label="錯誤訊息與原始 Payload" min-width="300" show-overflow-tooltip />
        </el-table>
        </template>
        </DataTablePage>
      </el-tab-pane>
    </el-tabs>

    <!-- 混車衝突人工裁決彈窗 -->
    <el-dialog v-model="resolveDialogVisible" title="混車衝突裁決" width="min(480px, calc(100vw - 32px))">
      <el-form ref="resolveFormRef" :model="resolveForm" label-width="110px">
        <el-form-item label="裁決實際承載車輛">
          <el-select v-model="resolveForm.vehicleId" placeholder="請指定正確認定之車輛" style="width: 100%">
            <el-option
              v-for="v in allVehicles"
              :key="v.id"
              :label="`${v.displayName} (${v.plateNo})`"
              :value="v.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="指定實際駕駛司機">
          <el-select v-model="resolveForm.driverId" placeholder="請指定正確認定之司機" style="width: 100%">
            <el-option
              v-for="d in allDrivers"
              :key="d.id"
              :label="d.name"
              :value="d.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="裁決備註">
          <el-input v-model="resolveForm.reason" placeholder="如：與司機確認後由竹北一車承載" />
        </el-form-item>
      </el-form>

      <template #footer>
        <DialogFooter
          confirm-text="確認送出"
          :loading="submitting"
          @confirm="handleResolveSubmit"
          @cancel="resolveDialogVisible = false"
        />
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import PageHeader from '@/components/PageHeader.vue'
import DialogFooter from '@/components/DialogFooter.vue'
import DataTablePage from '@/components/DataTablePage.vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listIssueRides, resolveConflict } from '@/api/rides'
import { listVehicles, listDrivers } from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import type { IssueRideDTO, VehicleDTO, DriverDTO } from '@/types/api'

const authStore = useAuthStore()
const activeTab = ref<'conflict' | 'unreported' | 'import_error'>('conflict')
const loading = ref(false)
const issueQuery = ref('')
const issueList = ref<IssueRideDTO[]>([])

const allVehicles = ref<VehicleDTO[]>([])
const allDrivers = ref<DriverDTO[]>([])

const resolveDialogVisible = ref(false)
const selectedIssue = ref<IssueRideDTO | null>(null)
const submitting = ref(false)

const resolveForm = reactive({
  vehicleId: '',
  driverId: '',
  reason: '混車確認'
})

async function fetchIssues() {
  loading.value = true
  try {
    const res = await listIssueRides({
      issueType: activeTab.value,
      pageSize: 50,
      q: issueQuery.value || undefined
    })
    issueList.value = res.data
  } finally {
    loading.value = false
  }
}

function handleReset() {
  issueQuery.value = ''
  fetchIssues()
}

function openResolveDialog(row: any) {
  selectedIssue.value = row
  resolveForm.vehicleId = allVehicles.value[0]?.id || ''
  resolveForm.driverId = allDrivers.value[0]?.id || ''
  resolveForm.reason = '混車確認'
  resolveDialogVisible.value = true
}

async function handleResolveSubmit() {
  if (!selectedIssue.value) return
  await ElMessageBox.confirm(
    `確定將該搭乘紀錄裁決為指定車輛與司機？`,
    '確認裁決',
    {
      confirmButtonText: '確認送出',
      cancelButtonText: '取消',
      type: 'warning'
    }
  )

  submitting.value = true
  try {
    await resolveConflict(selectedIssue.value.id, resolveForm)
    ElMessage.success('混車衝突已裁決')
    resolveDialogVisible.value = false
    fetchIssues()
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  const [vRes, dRes] = await Promise.all([
    listVehicles({ active: true, pageSize: 100 }),
    listDrivers({ active: true, pageSize: 100 })
  ])
  allVehicles.value = vRes.data
  allDrivers.value = dRes.data

  fetchIssues()
})
</script>

<style scoped>
.ride-issues-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.issues-tabs {
  border-radius: 8px;
  background-color: #ffffff;
}

.vehicle-name { color: var(--el-text-color-regular); }
.vehicle-separator { color: var(--el-text-color-placeholder); }

.issue-filter-controls {
  display: flex;
  gap: 8px;
}

@media (max-width: 640px) {
  .issue-filter-controls {
    flex-wrap: wrap;
  }

  .issue-filter-controls :deep(.el-input) {
    width: 100%;
  }
}
</style>
