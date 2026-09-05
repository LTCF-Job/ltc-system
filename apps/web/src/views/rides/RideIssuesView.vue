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
      <!-- 說明欄與涉及車輛欄長度不定，不設 max-width 讓表格隨頁面寬度伸展，避免版面有空間時仍卡在固定寬度卡片裡出現卷軸 -->
      <el-tab-pane label="混車衝突待裁決" name="conflict">
        <DataTablePage :loading="loading">
        <template #table>
        <el-table :data="issueList" border stripe table-layout="auto">
          <el-table-column prop="serviceDate" label="服務日期" width="110" class-name="service-date-col" />
          <el-table-column prop="caseName" label="個案姓名" min-width="90" class-name="case-name-col" />
          <el-table-column label="趟次" width="80" align="center" class-name="leg-seq-col">
            <template #default="{ row }">
              第 {{ row.legSeq }} 趟
            </template>
          </el-table-column>
          <el-table-column prop="description" label="衝突說明" min-width="260" show-overflow-tooltip class-name="conflict-desc-col" />
          <el-table-column label="涉及車輛" min-width="200" class-name="vehicle-col">
            <template #default="{ row }">
              <span v-for="(v, idx) in row.vehicles" :key="idx" class="vehicle-name">
                {{ v }}<span v-if="Number(idx) < row.vehicles.length - 1" class="vehicle-separator">、</span>
              </span>
            </template>
          </el-table-column>

          <el-table-column
            v-if="authStore.hasPermission('rides_issues', 'edit')"
            label="操作"
            width="100"
            fixed="right"
            align="center"
            class-name="action-col"
          >
            <template #default="{ row }">
              <TableRowActions>
                <el-button link type="primary" size="small" @click="openResolveDialog(row as any)">
                  人工裁決
                </el-button>
              </TableRowActions>
            </template>
          </el-table-column>
        </el-table>
        </template>
        </DataTablePage>
      </el-tab-pane>

      <!-- 分頁 2：未回報清單 -->
      <!-- 說明欄長度不定，不設 max-width 讓表格隨頁面寬度伸展，避免版面有空間時仍卡在固定寬度卡片裡出現卷軸或裁行 -->
      <el-tab-pane label="應搭未回報清單" name="unreported">
        <DataTablePage :loading="loading">
        <template #table>
        <el-table :data="issueList" border stripe table-layout="auto">
          <el-table-column prop="serviceDate" label="服務日期" width="110" class-name="service-date-col" />
          <el-table-column prop="caseName" label="個案姓名" min-width="90" class-name="case-name-col" />
          <el-table-column label="趟次" width="80" align="center" class-name="leg-seq-col">
            <template #default="{ row }">
              第 {{ row.legSeq }} 趟
            </template>
          </el-table-column>
          <el-table-column prop="description" label="說明" min-width="260" class-name="description-col" />
          <!-- 「前往回報」「查看排班」為 4 字操作文案，比一般 2 字按鈕（編輯／刪除）長，
               標準 2 顆按鈕寬度 140px 會被裁切，故加寬到 190px -->
          <el-table-column label="操作" width="190" fixed="right" align="center" class-name="action-col-190">
            <template #default="{ row }">
              <TableRowActions>
                <el-button link type="info" size="small" @click="$router.push('/rides/missing')">
                  前往回報
                </el-button>
                <el-button link type="info" size="small" @click="$router.push(`/cases/${row.caseId}`)">
                  查看排班
                </el-button>
              </TableRowActions>
            </template>
          </el-table-column>
        </el-table>
        </template>
        </DataTablePage>
      </el-tab-pane>

      <!-- 分頁 3：表單匯入錯誤 -->
      <!-- 錯誤訊息與原始 Payload 欄長度不定，不設 max-width 讓表格隨頁面寬度伸展，避免版面有空間時仍卡在固定寬度卡片裡出現卷軸 -->
      <el-tab-pane label="表單匯入異常" name="import_error">
        <DataTablePage :loading="loading">
        <template #table>
        <el-table :data="issueList" border stripe table-layout="auto">
          <el-table-column prop="serviceDate" label="服務日期" width="110" class-name="service-date-col" />
          <el-table-column prop="caseName" label="回報文字/欄位" min-width="140" class-name="report-field-col" />
          <el-table-column prop="description" label="錯誤訊息與原始 Payload" min-width="300" class-name="error-desc-col" />
          <el-table-column label="操作" width="100" fixed="right" align="center" class-name="action-col">
            <template #default="{ row }">
              <TableRowActions>
                <el-button link type="info" size="small" @click="openErrorDetail(row as any)">
                  查看
                </el-button>
              </TableRowActions>
            </template>
          </el-table-column>
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

    <!-- 表單匯入異常詳情彈窗 -->
    <el-dialog
      v-model="errorDetailVisible"
      title="表單匯入異常詳情"
      width="min(640px, calc(100vw - 32px))"
      destroy-on-close
    >
      <el-descriptions v-if="selectedError" :column="1" border>
        <el-descriptions-item label="服務日期">{{ selectedError.serviceDate }}</el-descriptions-item>
        <el-descriptions-item label="回報文字/欄位">{{ selectedError.caseName }}</el-descriptions-item>
        <el-descriptions-item label="錯誤訊息">{{ selectedError.description }}</el-descriptions-item>
        <el-descriptions-item label="原始 Payload">
          <pre class="raw-payload">{{ selectedError.rawPayload || '（無原始 Payload 紀錄）' }}</pre>
        </el-descriptions-item>
      </el-descriptions>

      <template #footer>
        <el-button @click="errorDetailVisible = false">關閉</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import PageHeader from '@/components/PageHeader.vue'
import DialogFooter from '@/components/DialogFooter.vue'
import DataTablePage from '@/components/DataTablePage.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listIssueRides, resolveConflict } from '@/api/rides'
import { listAllVehicles, listAllDrivers } from '@/api/masters'
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

const errorDetailVisible = ref(false)
const selectedError = ref<IssueRideDTO | null>(null)

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
  resolveForm.vehicleId = ''
  resolveForm.driverId = ''
  resolveForm.reason = '混車確認'
  resolveDialogVisible.value = true
}

function openErrorDetail(row: IssueRideDTO) {
  selectedError.value = row
  errorDetailVisible.value = true
}

async function handleResolveSubmit() {
  if (!selectedIssue.value) return
  if (!resolveForm.vehicleId) {
    ElMessage.warning('請先指定正確認定的車輛')
    return
  }
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
    listAllVehicles({ status: 'active' }),
    listAllDrivers({ status: 'active' })
  ])
  allVehicles.value = vRes
  allDrivers.value = dRes

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

/* table-layout="auto" 底下 el-table 本體內建 width: 100%，即使拿掉 inline style
   仍會撐滿容器；要顯式蓋成 max-content 才會縮到「各欄寬度加總」，內容短時不再
   被拉滿版面，內容真的超版面寬時交給 DataTablePage 既有的橫向捲軸接手
   （見 ltc-dashboard-visual-language skill 表格欄位一節）。 */
.issues-tabs :deep(.el-table) {
  width: max-content;
}

.vehicle-name { color: var(--app-text-regular); white-space: nowrap; }
.vehicle-separator { color: var(--app-text-muted); }

:deep(.case-name-col .cell),
:deep(.description-col .cell),
:deep(.report-field-col .cell),
:deep(.error-desc-col .cell),
:deep(.vehicle-col .cell) {
  white-space: nowrap;
}

/* el-table-column 的 min-width prop 在 table-layout="auto" 底下只會拿去算表格
   總寬度的預算，不會變成該欄真正的 CSS min-width——欄位當筆內容比 min-width
   短時（例如較短的個案姓名、單一涉及車輛）欄寬會被壓到只剩內容本身，跟其他
   撐開的欄位比例不一致。要另外補一條 :deep() min-width 才是真的鎖住下限
   （見 ltc-dashboard-visual-language skill 表格欄位一節）。 */
:deep(.case-name-col .cell) { min-width: 90px; }
:deep(.description-col .cell) { min-width: 260px; }
:deep(.report-field-col .cell) { min-width: 140px; }
:deep(.error-desc-col .cell) { min-width: 300px; }
:deep(.vehicle-col .cell) { min-width: 200px; }

/* 操作欄同樣受這個問題影響：fixed width 在這個 pattern 下一樣不是真正下限，
   按鈕欄可能被壓得比規範表要求的寬度還窄（見上方 component-contract.md
   「操作欄一律 fixed="right"」那條的按鈕數對照寬度）。 */
:deep(.action-col .cell) { min-width: 100px; }
:deep(.action-col-190 .cell) { min-width: 190px; }
:deep(.service-date-col .cell) { min-width: 110px; white-space: nowrap; }
:deep(.leg-seq-col .cell) { min-width: 80px; white-space: nowrap; }
:deep(.conflict-desc-col .cell) { min-width: 260px; }

.raw-payload {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: var(--el-font-family-mono, monospace);
  font-size: 13px;
}

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
