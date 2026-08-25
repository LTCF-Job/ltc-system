<template>
  <div class="site-list-view">
    <DataTablePage
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
          placeholder="全部區域"
          clearable
          style="width: 130px"
          @change="handleSearch"
        >
          <el-option label="全部區域" value="" />
          <el-option label="苗栗" value="miaoli" />
          <el-option label="新竹" value="hsinchu" />
        </el-select>

        <el-button type="primary" @click="handleSearch">查詢</el-button>
        <el-button @click="handleReset">重設</el-button>
      </template>

      <!-- 操作按鈕 -->
      <template #actions>
        <el-button
          v-if="authStore.can('staff')"
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
          <el-table-column prop="name" label="據點名稱" width="160" />
          <el-table-column prop="region" label="區域" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="row.region === 'miaoli' ? 'warning' : 'primary'">
                {{ REGION_LABELS[row.region as Region] }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="address" label="據點地址" min-width="240" />
          <el-table-column label="開放時間" width="180">
            <template #default="{ row }">
              {{ row.openDays?.map((d: number) => `週${'一二三四五六日'[d-1]}`).join('、') || '未設定' }}
            </template>
          </el-table-column>

          <el-table-column
            v-if="authStore.can('staff')"
            label="操作"
            width="140"
            fixed="right"
            align="center"
          >
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="openEditDialog(row)">
                編輯
              </el-button>
              <el-button link type="danger" size="small" @click="handleDelete(row)">
                刪除
              </el-button>
            </template>

          </el-table-column>

        </el-table>
      </template>
    </DataTablePage>

    <!-- 新增/編輯彈窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '編輯據點資料' : '新增據點'"
      width="540px"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="據點名稱" prop="name">
          <el-input v-model="form.name" placeholder="如：竹北日照中心" />
        </el-form-item>
        <el-form-item label="所屬區域" prop="region">
          <el-radio-group v-model="form.region">
            <el-radio value="miaoli">苗栗</el-radio>
            <el-radio value="hsinchu">新竹</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="據點地址" prop="address">
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
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          確認儲存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import DataTablePage from '@/components/DataTablePage.vue'
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
  openDays: [1, 2, 3, 4, 5]
})

const rules = {
  name: [{ required: true, message: '請輸入據點名稱', trigger: 'blur' }],
  region: [{ required: true, message: '請選擇區域', trigger: 'change' }],
  address: [{ required: true, message: '請輸入據點地址', trigger: 'blur' }]
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
    region: ''
  },
  onFetch: async () => {
    const res = await listSites({
      page: page.value,
      pageSize: pageSize.value,
      q: filters.q,
      region: filters.region
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
  dialogVisible.value = true
}

function openEditDialog(row: any) {
  editingId.value = row.id
  form.name = row.name
  form.region = row.region
  form.address = row.address
  form.openDays = [...row.openDays]
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
    } finally {
      submitting.value = false
    }
  })
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm(`確定刪除據點「${row.name}」？此操作無法還原。`, '確認刪除', {
    confirmButtonText: '確定',
    cancelButtonText: '取消',
    type: 'warning'
  })

  await deleteSite(row.id)
  ElMessage.success('據點已刪除')
  executeFetch()
}

executeFetch()
</script>
