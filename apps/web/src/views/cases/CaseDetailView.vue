<template>
  <div v-loading="loading" class="case-detail-view">
    <!-- 頂部標題與操作 -->
    <div class="page-header">
      <div class="title-group">
        <el-button link @click="$router.push('/cases')">
          <el-icon><ArrowLeft /></el-icon>
          返回清單
        </el-button>
        <h2>{{ caseData?.name || '個案明細' }}</h2>
        <el-tag v-if="caseData" :type="caseData.status === 'active' ? 'success' : 'info'">
          {{ CASE_STATUS_LABELS[caseData.status] }}
        </el-tag>
        <el-tag v-if="caseData" :type="caseData.region === 'miaoli' ? 'warning' : 'primary'">
          {{ REGION_LABELS[caseData.region] }}
        </el-tag>
      </div>

      <div class="action-group">
        <el-button
          v-if="authStore.can('staff') && caseData?.status === 'active'"
          type="danger"
          plain
          @click="handleToggleStatus('suspended')"
        >
          暫停服務
        </el-button>
        <el-button
          v-if="authStore.can('staff') && caseData?.status === 'suspended'"
          type="success"
          plain
          @click="handleToggleStatus('active')"
        >
          恢復在案
        </el-button>
      </div>
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
            <el-descriptions-item label="個案編號">{{ caseData?.code }}</el-descriptions-item>
            <el-descriptions-item label="身分證字號">
              <MaskedId
                v-if="caseData"
                :masked-value="caseData.nationalId"
                :on-reveal="fetchPlainId"
              />
            </el-descriptions-item>
            <el-descriptions-item label="建立時間">{{ caseData?.createdAt }}</el-descriptions-item>
            <el-descriptions-item label="最後更新">{{ caseData?.updatedAt }}</el-descriptions-item>
          </el-descriptions>

          <el-row :gutter="20">
            <el-col :span="12">
              <el-form-item label="個案姓名" prop="name">
                <el-input v-model="editForm.name" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="申報地區" prop="region">
                <el-select v-model="editForm.region" style="width: 100%">
                  <el-option value="miaoli" label="苗栗" />
                  <el-option value="hsinchu" label="新竹" />
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
            <el-col :span="12">
              <el-form-item label="服務類別" prop="serviceCategory">
                <el-radio-group v-model="editForm.serviceCategory">
                  <el-radio :value="1">1. 補助</el-radio>
                  <el-radio :value="2">2. 自費</el-radio>
                </el-radio-group>
              </el-form-item>
            </el-col>
            <el-col :span="12">
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
            <el-col :span="12">
              <el-form-item label="開始申報日" prop="claimStartDate">
                <el-date-picker
                  v-model="editForm.claimStartDate"
                  type="date"
                  value-format="YYYY-MM-DD"
                  style="width: 100%"
                />
              </el-form-item>
            </el-col>
            <el-col :span="12">
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
          @saved="fetchDetail"
        />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowLeft } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import MaskedId from '@/components/MaskedId.vue'
import ScheduleEditor from './ScheduleEditor.vue'
import { getCase, updateCase, revealCaseId } from '@/api/cases'
import { useAuthStore } from '@/stores/auth'
import { CASE_STATUS_LABELS, REGION_LABELS } from '@/types/domain'
import type { CaseDTO, UpdateCaseRequest } from '@/types/api'

const route = useRoute()
const authStore = useAuthStore()
const caseId = route.params.id as string

const loading = ref(false)
const saving = ref(false)
const activeTab = ref('basic')
const caseData = ref<CaseDTO | null>(null)

const editForm = reactive<UpdateCaseRequest>({
  name: '',
  region: 'miaoli',
  homeAddress: '',
  serviceCategory: 1,
  serviceUsageType: 2,
  claimStartDate: '',
  claimEndDate: ''
})

async function fetchDetail() {
  loading.value = true
  try {
    const res = await getCase(caseId)
    caseData.value = res
    editForm.name = res.name
    editForm.region = res.region
    editForm.homeAddress = res.homeAddress
    editForm.serviceCategory = res.serviceCategory
    editForm.serviceUsageType = res.serviceUsageType
    editForm.claimStartDate = res.claimStartDate
    editForm.claimEndDate = res.claimEndDate
  } finally {
    loading.value = false
  }
}

async function fetchPlainId(): Promise<string> {
  const res = await revealCaseId(caseId)
  return res.nationalId
}

async function handleUpdateCase() {
  saving.value = true
  try {
    await updateCase(caseId, editForm)
    ElMessage.success('個案基本資料已更新')
    fetchDetail()
  } finally {
    saving.value = false
  }
}

async function handleToggleStatus(newStatus: 'active' | 'suspended') {
  const actionText = newStatus === 'suspended' ? '暫停服務' : '恢復在案'
  await ElMessageBox.confirm(
    `確定將個案「${caseData.value?.name}」變更為「${actionText}」？`,
    '確認操作',
    {
      confirmButtonText: '確定',
      cancelButtonText: '取消',
      type: 'warning'
    }
  )

  await updateCase(caseId, { status: newStatus })
  ElMessage.success(`個案已${actionText}`)
  fetchDetail()
}

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
  justify-content: space-between;
  align-items: center;
  background-color: #ffffff;
  padding: 16px 20px;
  border-radius: 8px;

  .title-group {
    display: flex;
    align-items: center;
    gap: 12px;

    h2 {
      margin: 0;
      font-size: 20px;
      color: var(--el-color-primary);
    }
  }

  .action-group {
    display: flex;
    gap: 10px;
  }
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
