<template>
  <div class="site-list-view">
    <DataTablePage
      title="據點管理"
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
          placeholder="搜尋據點名稱／地址"
          clearable
          style="width: 240px"
          @keyup.enter="handleSearch"
        />

        <el-select
          v-model="filters.region"
          placeholder="搜尋區域"
          clearable
          filterable
          style="width: 140px"
          @change="handleSearch"
          @clear="handleSearch"
        >
          <el-option v-for="option in regionOptions" :key="option" :label="option" :value="option" />
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
          新增據點
        </el-button>
      </template>

      <!-- 表格 -->
      <template #table>
        <el-table :data="sites" border stripe style="width: 100%">
          <el-table-column prop="name" label="據點名稱" min-width="140" class-name="site-nowrap-col site-name-col" />
          <el-table-column prop="region" label="區域" min-width="120" align="center" class-name="site-nowrap-col site-region-col">
            <template #default="{ row }">
              <span>{{ row.region || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="address" label="據點地址" min-width="180" class-name="site-address-col" show-overflow-tooltip>
            <template #default="{ row }">
              <span>{{ row.address || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="remarks" label="備註" min-width="140" show-overflow-tooltip class-name="site-remarks-col">
            <template #default="{ row }">
              <span>{{ row.remarks || '-' }}</span>
            </template>
          </el-table-column>

          <el-table-column prop="status" label="狀態" min-width="130" align="center" class-name="site-nowrap-col site-status-col">
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
            min-width="140"
            fixed="right"
            align="center"
            class-name="site-nowrap-col site-actions-col"
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
      :title="editingId ? '編輯據點資料' : '新增據點'"
      width="min(600px, calc(100vw - 32px))"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px" class="dialog-scroll-form">
        <el-form-item label="據點名稱" prop="name">
          <el-input v-model="form.name" placeholder="如：竹北日照中心" />
        </el-form-item>
        <el-form-item label="區域" prop="region">
          <el-select
            v-model="form.region"
            placeholder="請選擇或輸入區域，例如：新竹、苗栗、竹南頭份"
            filterable
            allow-create
            default-first-option
            clearable
            style="width: 100%"
          >
            <el-option v-for="option in regionOptions" :key="option" :label="option" :value="option" />
          </el-select>
        </el-form-item>
        <el-form-item label="據點地址" prop="address">
          <el-input v-model="form.address" placeholder="請輸入完整地址" clearable />
        </el-form-item>
        <el-form-item label="備註" prop="remarks">
          <el-input v-model="form.remarks" type="textarea" :rows="2" placeholder="選填備註" clearable />
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
import DataTablePage from '@/components/DataTablePage.vue'
import DialogFooter from '@/components/DialogFooter.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import { listSites, listAllSites, createSite, updateSite, deleteSite } from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import { useListQuery } from '@/composables/useListQuery'
import type { SiteDTO, CreateSiteRequest } from '@/types/api'

const authStore = useAuthStore()
const sites = ref<SiteDTO[]>([])
const regionOptions = ref<string[]>([])
const dialogVisible = ref(false)
const editingId = ref<string | null>(null)
const submitting = ref(false)
const formRef = ref<FormInstance>()

const form = reactive<CreateSiteRequest>({
  name: '',
  region: '',
  address: '',
  remarks: '',
  status: 'active'
})

async function refreshRegionOptions() {
  const allSites = await listAllSites()
  const distinctRegions = new Set(allSites.map((site) => site.region).filter((region): region is string => !!region))
  regionOptions.value = Array.from(distinctRegions).sort((a, b) => a.localeCompare(b, 'zh-Hant'))
}

async function handleToggleStatus(row: SiteDTO, newActive: boolean) {
  // 快速切換狀態仍送出完整據點內容：更新 API 是整筆覆寫，只送 status 會清掉其餘欄位
  const newStatus = newActive ? 'active' : 'inactive'
  try {
    await updateSite(row.id, {
      name: row.name,
      region: row.region,
      address: row.address,
      remarks: row.remarks || '',
      status: newStatus
    })
    row.status = newStatus
    ElMessage.success(`已將據點「${row.name}」切換為 ${newActive ? '啟用' : '停用'}`)
  } catch {
    // 全域攔截器負責顯示 API 錯誤。
  }
}

const rules = {
  name: [{ required: true, message: '請輸入據點名稱', trigger: 'blur' }],
  region: [{ required: true, message: '請輸入區域', trigger: 'blur' }]
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
  form.region = ''
  form.address = ''
  form.remarks = ''
  form.status = 'active'
  dialogVisible.value = true
}

function openEditDialog(row: any) {
  editingId.value = row.id
  form.name = row.name
  form.region = row.region || ''
  form.address = row.address || ''
  form.remarks = row.remarks || ''
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
        ElMessage.success('據點資料已更新')
      } else {
        await createSite(form)
        ElMessage.success('據點新增成功')
      }
      dialogVisible.value = false
      executeFetch()
      refreshRegionOptions()
    } finally {
      submitting.value = false
    }
  })
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`確定刪除據點「${row.name}」？此操作無法還原。`, '確認刪除', {
    confirmButtonText: '刪除',
    cancelButtonText: '取消',
    type: 'warning',
    confirmButtonClass: 'el-button--danger'
  })

  await deleteSite(row.id)
  ElMessage.success('據點已刪除')
  executeFetch()
  refreshRegionOptions()
}

executeFetch()
refreshRegionOptions()
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

:deep(.site-nowrap-col .cell) {
  white-space: nowrap;
}

:deep(.site-name-col .cell) {
  min-width: 140px;
}

:deep(.site-region-col .cell) {
  min-width: 120px;
}

:deep(.site-address-col .cell) {
  min-width: 180px;
}

:deep(.site-remarks-col .cell) {
  min-width: 140px;
}

:deep(.site-status-col .cell) {
  min-width: 130px;
}

:deep(.site-actions-col .cell) {
  min-width: 140px;
}
</style>
