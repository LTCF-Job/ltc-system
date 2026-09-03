<template>
  <div class="site-list-view">
    <DataTablePage
      title="單位管理"
      :max-width="1020"
      v-model:page="page"
      v-model:pageSize="pageSize"
      :total="total"
      :loading="loading"
      @page-change="handlePageChange"
      @size-change="handleSizeChange"
    >
      <!-- 篩選列 -->
      <template #filter>
        <el-input
          v-model="filters.q"
          placeholder="搜尋單位名稱／地址"
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
        <el-button
          v-if="authStore.hasPermission('masters_sites', 'edit')"
          type="primary"
          @click="openCreateDialog"
        >
          <el-icon><Plus /></el-icon>
          新增單位
        </el-button>
      </template>

      <!-- 表格 -->
      <template #table>
        <el-table :data="sites" border stripe table-layout="auto" style="width: 100%">
          <el-table-column prop="name" label="單位名稱" min-width="140" class-name="site-name-col" />
          <el-table-column prop="region" label="區域" width="120" align="center">
            <template #default="{ row }">
              <span>{{ REGION_LABELS[row.region as Region] || row.region }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="address" label="單位地址" min-width="180" class-name="site-address-col" show-overflow-tooltip />
          <el-table-column label="開放時間" width="260" class-name="open-days-column">
            <template #default="{ row }">
              {{ row.openDays?.map((d: number) => `週${'一二三四五六日'[d-1]}`).join('、') || '未設定' }}
            </template>
          </el-table-column>

          <el-table-column prop="status" label="狀態" width="130" align="center">
            <template #default="{ row }">
              <el-tooltip
                v-if="authStore.hasPermission('masters_sites', 'edit')"
                :content="row.status === 'active' ? '目前為啟用，點選切換為停用' : '目前為停用，點選切換為啟用'"
                placement="top"
                :show-after="300"
              >
                <button
                  type="button"
                  class="status-toggle-pill"
                  :class="row.status === 'active' ? 'is-active' : 'is-inactive'"
                  @click="handleToggleStatus(row as any, row.status !== 'active')"
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

          <el-table-column
            v-if="authStore.hasPermission('masters_sites', 'edit') || authStore.hasPermission('masters_sites', 'delete')"
            label="操作"
            width="140"
            fixed="right"
            align="center"
          >
            <template #default="{ row }">
              <TableRowActions>
                <el-button
                  v-if="authStore.hasPermission('masters_sites', 'edit')"
                  link
                  type="primary"
                  size="small"
                  @click="openEditDialog(row)"
                >
                  編輯
                </el-button>
                <el-button
                  v-if="authStore.hasPermission('masters_sites', 'delete')"
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

    <!-- 新增/編輯對話框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '編輯單位資料' : '新增單位'"
      width="min(600px, calc(100vw - 32px))"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px" class="dialog-scroll-form">
        <el-form-item label="單位名稱" prop="name">
          <el-input v-model="form.name" placeholder="如：竹北日照中心" />
        </el-form-item>
        <el-form-item label="所屬區域" prop="region">
          <el-select
            v-model="form.region"
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
        <el-form-item label="單位地址" prop="address">
          <el-input v-model="form.address" placeholder="請輸入完整地址" />
        </el-form-item>
        <el-form-item label="開放星期" prop="openDays">
          <el-checkbox-group v-model="form.openDays">
            <el-checkbox :value="1">週一</el-checkbox>
            <el-checkbox :value="2">週二</el-checkbox>
            <el-checkbox :value="3">週三</el-checkbox>
            <el-checkbox :value="4">週四</el-checkbox>
            <el-checkbox :value="5">週五</el-checkbox>
            <el-checkbox :value="6">週六</el-checkbox>
            <el-checkbox :value="7">週日</el-checkbox>
          </el-checkbox-group>
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
          :loading="submitting"
          @confirm="handleSubmit"
          @cancel="dialogVisible = false"
        />
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { Plus, Edit } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { resolveErrorMessage } from '@/api/errorCodes'
import DataTablePage from '@/components/DataTablePage.vue'
import DialogFooter from '@/components/DialogFooter.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import { listSites, createSite, updateSite, deleteSite } from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import { REGION_LABELS, type Region } from '@/types/domain'
import type { SiteDTO, CreateSiteRequest } from '@/types/api'

const authStore = useAuthStore()
const sites = ref<SiteDTO[]>([])
const dialogVisible = ref(false)
const editingId = ref<string | null>(null)
const submitting = ref(false)
const formRef = ref<FormInstance>()

const form = reactive<CreateSiteRequest>({
  name: '',
  region: 'miaoli',
  address: '',
  openDays: [1, 2, 3, 4, 5],
  status: 'active'
})

async function handleToggleStatus(row: SiteDTO, newActive: boolean) {
  // 快速切換狀態仍送出完整單位內容：更新 API 是整筆覆寫，只送 status 會清掉其餘欄位
  const newStatus = newActive ? 'active' : 'inactive'
  try {
    await updateSite(row.id, {
      name: row.name,
      region: row.region,
      address: row.address,
      openDays: row.openDays,
      status: newStatus
    })
    row.status = newStatus
    ElMessage.success(`已將單位「${row.name}」切換為 ${newActive ? '啟用' : '停用'}`)
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '更新狀態失敗'))
  }
}

const rules = {
  name: [{ required: true, message: '請輸入單位名稱', trigger: 'blur' }],
  region: [{ required: true, message: '請選擇區域', trigger: 'change' }],
  address: [{ required: true, message: '請輸入單位地址', trigger: 'blur' }]
}

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
    const res = await listSites({
      page: page.value,
      pageSize: pageSize.value,
      q: filters.q,
      region: filters.region,
      status: filters.status || undefined
    })
    sites.value = res.data
    total.value = res.meta.total
  }
})

function openCreateDialog() {
  editingId.value = null
  form.name = ''
  form.region = 'miaoli'
  form.address = ''
  form.openDays = [1, 2, 3, 4, 5]
  form.status = 'active'
  dialogVisible.value = true
}

function openEditDialog(row: any) {
  editingId.value = row.id
  form.name = row.name
  form.region = row.region
  form.address = row.address
  form.openDays = [...row.openDays]
  form.status = row.status || 'active'
  dialogVisible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (editingId.value) {
        await updateSite(editingId.value, form)
        ElMessage.success('單位資料已更新')
      } else {
        await createSite(form)
        ElMessage.success('單位新增成功')
      }
      dialogVisible.value = false
      executeFetch()
    } finally {
      submitting.value = false
    }
  })
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`確定刪除單位「${row.name}」？此操作無法還原。`, '確認刪除', {
    confirmButtonText: '刪除',
    cancelButtonText: '取消',
    type: 'warning',
    confirmButtonClass: 'el-button--danger'
  })

  await deleteSite(row.id)
  ElMessage.success('單位已刪除')
  executeFetch()
}

executeFetch()
</script>

<style scoped>
.region-label {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-weight: 600;
  color: var(--app-text-primary);
  white-space: nowrap;
}

.region-label::before {
  content: '';
  width: 7px;
  height: 7px;
  border: 2px solid var(--app-border-color);
  border-radius: 50%;
}

.region-label.region-miaoli::before { border-color: var(--app-status-warning-fg); }
.region-label.region-hsinchu::before { border-color: var(--app-primary); }

:deep(.open-days-column .cell) {
  white-space: nowrap;
}

:deep(.site-name-col .cell) {
  white-space: nowrap;
  min-width: 140px;
}

:deep(.site-address-col .cell) {
  min-width: 180px;
}
</style>
