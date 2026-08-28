<template>
  <div v-loading="loading" class="case-detail-view">
    <!-- 頂部返回 -->
    <div class="page-header">
      <el-button link @click="$router.push('/cases')">
        <el-icon><ArrowLeft /></el-icon>
        返回清單
      </el-button>
    </div>

    <!-- 分頁導覽：基本資料 / 排班設定 / 搭乘月曆 -->
    <el-tabs v-model="activeTab" type="border-card" class="detail-tabs">
      <!-- 分頁 1：基本資料 -->
      <el-tab-pane label="基本資料" name="basic">
        <el-form
          ref="editFormRef"
          :model="editForm"
          label-width="140px"
          :disabled="!authStore.can('staff')"
        >
          <el-descriptions title="系統與身分資訊" :column="2" border style="margin-bottom: 20px">
            <template #extra>
              <div class="descriptions-actions">
                <el-button
                  v-if="authStore.can('staff')"
                  type="danger"
                  plain
                  @click="handleDeleteCase"
                >
                  刪除個案
                </el-button>
              </div>
            </template>
            <el-descriptions-item label="個案編號">{{ caseData?.code }}</el-descriptions-item>
            <el-descriptions-item label="身分證字號">
              <span class="font-mono font-bold">{{ caseData?.nationalId }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="目前狀態">
              <el-tag
                v-if="caseData"
                :type="caseData.status === 'active' ? 'success' : (caseData.status === 'suspended' ? 'warning' : 'danger')"
              >
                {{ CASE_STATUS_LABELS[caseData.status] || caseData.status }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="建立時間">{{ formatDateTime(caseData?.createdAt) }}</el-descriptions-item>
            <el-descriptions-item label="最後更新" :span="2">{{ formatDateTime(caseData?.updatedAt) }}</el-descriptions-item>
          </el-descriptions>

          <el-row :gutter="20">
            <el-col :xs="24" :sm="12" :lg="6">
              <el-form-item label="個案姓名" prop="name">
                <el-input v-model="editForm.name" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :lg="6">
              <el-form-item label="聯絡電話" prop="phone">
                <el-input v-model="editForm.phone" placeholder="如：0912345678" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :lg="6">
              <el-form-item label="申報地區" prop="region">
                <el-select v-model="editForm.region" filterable style="width: 100%">
                  <el-option
                    v-for="(label, key) in REGION_LABELS"
                    :key="key"
                    :value="key"
                    :label="label"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :lg="6">
              <el-form-item label="個案狀態" prop="status">
                <el-select v-model="editForm.status" style="width: 100%">
                  <el-option value="active" label="在案" />
                  <el-option value="suspended" label="暫停" />
                  <el-option value="closed" label="停案" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>

          <el-row :gutter="20">
            <el-col :span="24">
              <el-form-item label="住家地址" prop="homeAddress">
                <el-input v-model="editForm.homeAddress" />
              </el-form-item>
            </el-col>
          </el-row>

          <el-row :gutter="20">
            <el-col :xs="24" :lg="12">
              <el-form-item label="服務類別" prop="serviceCategory">
                <el-radio-group v-model="editForm.serviceCategory">
                  <el-radio :value="1">1. 補助</el-radio>
                  <el-radio :value="2">2. 自費</el-radio>
                </el-radio-group>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :lg="12">
              <el-form-item label="服務使用類型" prop="serviceUsageType">
                <el-select v-model="editForm.serviceUsageType" style="width: 100%">
                  <el-option :value="1" label="1. 社區式長照機構" />
                  <el-option :value="2" label="2. 社區服務據點(不含身障類)" />
                  <el-option :value="3" label="3. 輔具中心" />
                  <el-option :value="4" label="4. 身障日間照顧服務" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>

          <el-row :gutter="20">
            <el-col :xs="24" :lg="12">
              <el-form-item label="開始申報日" prop="claimStartDate">
                <el-date-picker
                  v-model="editForm.claimStartDate"
                  type="date"
                  value-format="YYYY-MM-DD"
                  style="width: 100%"
                />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :lg="12">
              <el-form-item label="結束申報日">
                <el-date-picker
                  v-model="editForm.claimEndDate"
                  type="date"
                  value-format="YYYY-MM-DD"
                  placeholder="未填代表持續有效"
                  style="width: 100%"
                />
              </el-form-item>
            </el-col>
          </el-row>

          <div v-if="authStore.can('staff')" class="form-actions">
            <el-button type="primary" :loading="saving" @click="handleUpdateCase">
              儲存基本資料
            </el-button>
          </div>
        </el-form>
      </el-tab-pane>

      <!-- 分頁 2：排班設定編輯器 -->
      <el-tab-pane label="排班設定" name="schedule">
        <ScheduleEditor
          v-if="caseData"
          :case-id="caseData.id"
          :region="caseData.region"
          :schedule="caseData.activeSchedule"
          @saved="handleScheduleSaved"
        />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import ScheduleEditor from './ScheduleEditor.vue'
import { getCase, updateCase, deleteCase, getCaseSchedule } from '@/api/cases'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/utils/formatters'
import { CASE_STATUS_LABELS, REGION_LABELS } from '@/types/domain'
import type { CaseDTO, UpdateCaseRequest } from '@/types/api'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const caseId = computed(() => route.params.id as string)

const loading = ref(false)
const saving = ref(false)
const activeTab = ref(route.query.tab === 'schedule' ? 'schedule' : 'basic')
const caseData = ref<CaseDTO | null>(null)

const editForm = reactive<UpdateCaseRequest>({
  name: '',
  phone: '',
  region: 'miaoli',
  homeAddress: '',
  serviceCategory: 1,
  serviceUsageType: 2,
  claimStartDate: '',
  claimEndDate: '',
  status: 'active'
})

async function fetchDetail() {
  if (!caseId.value) return
  loading.value = true
  try {
    const [rawCase, rawSchedule] = await Promise.all([
      getCase(caseId.value) as Promise<any>,
      getCaseSchedule(caseId.value).catch(() => null) as Promise<any>
    ])
    const res: any = rawCase?.data ?? rawCase
    const sched: any = rawSchedule?.data ?? rawSchedule

    if (!res) return

    res.nationalId = res.nationalId || res.nationalIdMasked || ''
    if (sched) {
      res.activeSchedule = sched?.data ?? sched
    }

    caseData.value = res
    editForm.name = res.name || ''
    editForm.phone = res.phone || ''
    editForm.region = res.region || 'miaoli'
    editForm.homeAddress = res.homeAddress || ''
    editForm.serviceCategory = res.serviceCategory || 1
    editForm.serviceUsageType = res.serviceUsageType || 2
    editForm.claimStartDate = res.claimStartDate ? String(res.claimStartDate).slice(0, 10) : ''
    editForm.claimEndDate = res.claimEndDate ? String(res.claimEndDate).slice(0, 10) : ''
    editForm.status = res.status || 'active'
  } catch (err: any) {
    ElMessage.error(err.message || '載入個案明細失敗')
  } finally {
    loading.value = false
  }
}

async function handleUpdateCase() {
  saving.value = true
  try {
    await updateCase(caseId.value, editForm)
    ElMessage.success('個案基本資料已更新')
    router.push('/cases')
  } finally {
    saving.value = false
  }
}

function handleScheduleSaved() {
  router.push('/cases')
}

async function handleDeleteCase() {
  try {
    await ElMessageBox.confirm(
      `確定要刪除個案「${caseData.value?.name} (${caseData.value?.code})」？此操作將一併移除其關聯排班資料，且無法復原。`,
      '刪除確認',
      {
        confirmButtonText: '確定刪除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )
    await deleteCase(caseId.value)
    ElMessage.success(`個案「${caseData.value?.name}」已成功刪除`)
    router.push('/cases')
  } catch (err: any) {
    if (err !== 'cancel') {
      ElMessage.error(err.message || '刪除個案失敗')
    }
  }
}

watch(
  () => route.params.id,
  () => {
    fetchDetail()
  }
)

watch(
  () => route.query.tab,
  (newTab) => {
    if (newTab === 'schedule' || newTab === 'basic') {
      activeTab.value = newTab
    }
  }
)

onMounted(() => {
  fetchDetail()
})
</script>

<style scoped>
.case-detail-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header {
  display: flex;
  align-items: center;
  background-color: #ffffff;
  padding: 16px 20px;
  border-radius: 8px;
}

.descriptions-actions {
  display: flex;
  gap: 8px;
}

.detail-tabs {
  border-radius: 8px;
  background-color: #ffffff;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
}
</style>
