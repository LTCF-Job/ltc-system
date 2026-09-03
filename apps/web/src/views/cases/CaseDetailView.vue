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
          :disabled="!authStore.hasPermission('masters_cases', 'edit')"
        >
          <el-descriptions title="系統與身分資訊" :column="2" border style="margin-bottom: 20px">
            <template #extra>
              <div class="descriptions-actions">
                <el-button
                  v-if="authStore.hasPermission('masters_cases', 'delete')"
                  type="danger"
                  plain
                  @click="handleDeleteCase"
                >
                  刪除個案
                </el-button>
              </div>
            </template>
            <el-descriptions-item label="身分證字號">
              <span class="font-mono font-bold">{{ caseData?.nationalId }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="目前狀態">
              <StatusTag v-if="caseData" :status="caseData.status" preset="caseStatus" />
            </el-descriptions-item>
            <el-descriptions-item label="建立時間">{{ formatDateTime(caseData?.createdAt) }}</el-descriptions-item>
            <el-descriptions-item label="最後更新" :span="2">{{ formatDateTime(caseData?.updatedAt) }}</el-descriptions-item>
          </el-descriptions>

          <el-row :gutter="16">
            <el-col :xs="24" :sm="12" :lg="6">
              <el-form-item label="個案姓名" prop="name">
                <el-input v-model="editForm.name" />
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

          <el-row :gutter="16">
            <el-col :span="24">
              <el-form-item label="住家地址" prop="homeAddress">
                <el-input v-model="editForm.homeAddress" />
              </el-form-item>
            </el-col>
          </el-row>

          <el-row :gutter="16">
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
                <el-select v-model="editForm.serviceUsageType" placeholder="未選擇" clearable style="width: 100%">
                  <el-option :value="1" label="1. 社區式長照機構" />
                  <el-option :value="2" label="2. 社區服務據點(不含身障類)" />
                  <el-option :value="3" label="3. 輔具中心" />
                  <el-option :value="4" label="4. 身障日間照顧服務" />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>

          <el-row :gutter="16">
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

          <el-divider content-position="left">個案背景與聯絡人</el-divider>

          <el-row :gutter="16">
            <el-col :xs="24" :sm="12" :lg="6">
              <el-form-item label="家戶類型" prop="householdType">
                <el-input v-model="editForm.householdType" placeholder="如：獨居、與子女同住" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :lg="6">
              <el-form-item label="性別" prop="gender">
                <el-select v-model="editForm.gender" clearable style="width: 100%">
                  <el-option value="男" label="男" />
                  <el-option value="女" label="女" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :lg="6">
              <el-form-item label="出生日期" prop="birthDate">
                <el-date-picker
                  v-model="editForm.birthDate"
                  type="date"
                  value-format="YYYY-MM-DD"
                  style="width: 100%"
                />
              </el-form-item>
            </el-col>
          </el-row>

          <el-row :gutter="16">
            <el-col :xs="24" :sm="12" :lg="8">
              <el-form-item label="照顧者聯絡人角色" prop="careContactRole">
                <el-input v-model="editForm.careContactRole" placeholder="如：個管、照專" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :lg="8">
              <el-form-item label="照顧者聯絡人姓名" prop="careContactName">
                <el-input v-model="editForm.careContactName" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="24" :lg="8">
              <el-form-item label="戶籍地址" prop="registeredAddress">
                <el-input v-model="editForm.registeredAddress" />
              </el-form-item>
            </el-col>
          </el-row>

          <el-row :gutter="16">
            <el-col :span="24">
              <el-form-item label="備註" prop="remarks">
                <el-input v-model="editForm.remarks" type="textarea" :rows="2" placeholder="選填" />
              </el-form-item>
            </el-col>
          </el-row>

          <div v-if="authStore.hasPermission('masters_cases', 'edit')" class="form-actions">
            <el-button type="primary" :loading="saving" @click="handleUpdateCase">
              儲存基本資料
            </el-button>
          </div>
        </el-form>

        <el-divider content-position="left">交通偏好</el-divider>

        <el-form label-width="140px" :disabled="!authStore.hasPermission('masters_cases', 'edit')">
          <el-row :gutter="16">
            <el-col :xs="24" :sm="12" :lg="8">
              <el-form-item label="所屬單位">
                <el-select v-model="transportForm.siteId" filterable style="width: 100%">
                  <el-option
                    v-for="site in availableSites"
                    :key="site.id"
                    :value="site.id"
                    :label="site.name"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :lg="8">
              <el-form-item label="去程車輛">
                <el-select v-model="transportForm.outboundVehicleId" filterable style="width: 100%">
                  <el-option
                    v-for="vehicle in availableVehicles"
                    :key="vehicle.id"
                    :value="vehicle.id"
                    :label="vehicle.displayName"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12" :lg="8">
              <el-form-item label="回程車輛">
                <el-select v-model="transportForm.inboundVehicleId" filterable style="width: 100%">
                  <el-option
                    v-for="vehicle in availableVehicles"
                    :key="vehicle.id"
                    :value="vehicle.id"
                    :label="vehicle.displayName"
                  />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>

          <div v-if="authStore.hasPermission('masters_cases', 'edit')" class="form-actions">
            <el-button type="primary" :loading="savingTransportPreference" @click="handleUpdateTransportPreference">
              儲存交通偏好
            </el-button>
          </div>
        </el-form>
      </el-tab-pane>

      <!-- 分頁 2：排班設定編輯器 -->
      <el-tab-pane label="排班設定" name="schedule">
        <ScheduleEditor
          v-if="caseData"
          :case-id="caseData.id"
          :region="caseData.region || 'miaoli'"
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
import { resolveErrorMessage } from '@/api/errorCodes'
import ScheduleEditor from './ScheduleEditor.vue'
import StatusTag from '@/components/StatusTag.vue'
import { getCase, updateCase, deleteCase, getCaseSchedule, updateCaseTransportPreference } from '@/api/cases'
import { listSites, listVehicles } from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/utils/formatters'
import { REGION_LABELS } from '@/types/domain'
import type { CaseDTO, UpdateCaseRequest, UpdateCaseTransportPreferenceRequest, SiteDTO, VehicleDTO } from '@/types/api'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const caseId = computed(() => route.params.id as string)

const loading = ref(false)
const saving = ref(false)
const savingTransportPreference = ref(false)
const activeTab = ref(route.query.tab === 'schedule' ? 'schedule' : 'basic')
const caseData = ref<CaseDTO | null>(null)
const availableSites = ref<SiteDTO[]>([])
const availableVehicles = ref<VehicleDTO[]>([])

const editForm = reactive<UpdateCaseRequest>({
  name: '',
  region: 'miaoli',
  homeAddress: '',
  serviceCategory: undefined,
  serviceUsageType: undefined,
  claimEndDate: '',
  status: 'active',
  householdType: '',
  gender: '',
  birthDate: '',
  careContactRole: '',
  careContactName: '',
  registeredAddress: '',
  remarks: ''
})

const transportForm = reactive<UpdateCaseTransportPreferenceRequest>({
  siteId: '',
  outboundVehicleId: '',
  inboundVehicleId: ''
})

async function fetchDetail() {
  if (!caseId.value) return
  loading.value = true
  try {
    const [rawCase, rawSchedule] = await Promise.all([
      getCase(caseId.value) as Promise<any>,
      // 404／查無排班是合法的「尚未排班」狀態；其餘錯誤（如伺服器錯誤）需往外拋出，不得一併當成無排班
      getCaseSchedule(caseId.value).catch((err: any) => {
        if (err?.response?.status === 404) return null
        throw err
      }) as Promise<any>
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
    editForm.region = res.region || 'miaoli'
    editForm.homeAddress = res.homeAddress || ''
    editForm.serviceCategory = res.serviceCategory
    editForm.serviceUsageType = res.serviceUsageType
    editForm.claimEndDate = res.claimEndDate ? String(res.claimEndDate).slice(0, 10) : ''
    editForm.status = res.status || 'active'
    editForm.householdType = res.householdType || ''
    editForm.gender = res.gender || ''
    editForm.birthDate = res.birthDate ? String(res.birthDate).slice(0, 10) : ''
    editForm.careContactRole = res.careContactRole || ''
    editForm.careContactName = res.careContactName || ''
    editForm.registeredAddress = res.registeredAddress || ''
    editForm.remarks = res.remarks || ''
    transportForm.siteId = res.siteId || ''
    transportForm.outboundVehicleId = res.outboundVehicleId || ''
    transportForm.inboundVehicleId = res.inboundVehicleId || ''
  } catch (err: any) {
    ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '載入個案明細失敗'))
  } finally {
    loading.value = false
  }
}

async function loadSitesAndVehicles() {
  const [sitesRes, vehiclesRes] = await Promise.all([
    listSites({ status: 'active', pageSize: 100 }),
    listVehicles({ status: 'active', pageSize: 100 })
  ])
  availableSites.value = sitesRes.data
  availableVehicles.value = vehiclesRes.data
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

async function handleUpdateTransportPreference() {
  savingTransportPreference.value = true
  try {
    // 三個欄位皆選填：只送出有值的欄位，避免把使用者未異動、原本為空的欄位當成「明確清空」送出
    const payload: UpdateCaseTransportPreferenceRequest = {}
    if (transportForm.siteId) payload.siteId = transportForm.siteId
    if (transportForm.outboundVehicleId) payload.outboundVehicleId = transportForm.outboundVehicleId
    if (transportForm.inboundVehicleId) payload.inboundVehicleId = transportForm.inboundVehicleId
    await updateCaseTransportPreference(caseId.value, payload)
    ElMessage.success('交通偏好已更新')
  } finally {
    savingTransportPreference.value = false
  }
}

function handleScheduleSaved() {
  router.push('/cases')
}

async function handleDeleteCase() {
  try {
    await ElMessageBox.confirm(
      `確定要刪除個案「${caseData.value?.name}」？此操作將一併移除其關聯排班資料，且無法復原。`,
      '刪除確認',
      {
        confirmButtonText: '刪除',
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
      ElMessage.error(resolveErrorMessage(err.response?.data?.error?.code, '刪除個案失敗'))
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
  loadSitesAndVehicles()
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
