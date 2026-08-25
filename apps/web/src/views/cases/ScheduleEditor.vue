<template>
  <div class="schedule-editor">
    <el-form
      ref="formRef"
      :model="formData"
      :rules="rules"
      label-width="130px"
      :disabled="!authStore.can('staff')"
    >
      <!-- 排班基本條件 -->
      <el-card shadow="never" class="section-card">
        <template #header>
          <span class="card-title">排班條件設定</span>
        </template>

        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="所屬據點" prop="siteId">
              <el-select
                v-model="formData.siteId"
                placeholder="請選擇據點"
                style="width: 100%"
                filterable
              >
                <el-option
                  v-for="site in availableSites"
                  :key="site.id"
                  :label="site.name"
                  :value="site.id"
                />
              </el-select>
            </el-form-item>
          </el-col>

          <el-col :span="12">
            <el-form-item label="有效起始日" prop="effectiveFrom">
              <el-date-picker
                v-model="formData.effectiveFrom"
                type="date"
                placeholder="選擇生效日期"
                value-format="YYYY-MM-DD"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="趟數型態" prop="tripPattern">
              <el-radio-group v-model="formData.tripPattern" @change="handlePatternChange">
                <el-radio-button :value="1">單向 1 趟</el-radio-button>
                <el-radio-button :value="2">一般 2 趟</el-radio-button>
                <el-radio-button :value="4">四趟 (早/午去回)</el-radio-button>
              </el-radio-group>
            </el-form-item>
          </el-col>

          <el-col :span="12">
            <el-form-item label="每週搭乘日" prop="weekdays">
              <el-checkbox-group v-model="formData.weekdays">
                <el-checkbox :value="1">週一</el-checkbox>
                <el-checkbox :value="2">週二</el-checkbox>
                <el-checkbox :value="3">週三</el-checkbox>
                <el-checkbox :value="4">週四</el-checkbox>
                <el-checkbox :value="5">週五</el-checkbox>
                <el-checkbox :value="6">週六</el-checkbox>
                <el-checkbox :value="7">週日</el-checkbox>
              </el-checkbox-group>
            </el-form-item>
          </el-col>
        </el-row>

        <!-- 費用與時長欄位（規格書明訂一律可編輯，不得唯讀或隱藏） -->
        <el-row :gutter="20">
          <el-col :span="8">
            <el-form-item label="申報單價 (元)" prop="unitPrice">
              <el-input-number
                v-model="formData.unitPrice"
                :min="1"
                :max="9999"
                :precision="2"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>

          <el-col :span="8">
            <el-form-item label="單趟里程 (公里)" prop="distanceKm">
              <el-input-number
                v-model="formData.distanceKm"
                :min="0.1"
                :max="999"
                :precision="2"
                placeholder="無預設值，必填"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>

          <el-col :span="8">
            <el-form-item label="服務時長 (分鐘)" prop="serviceDurationMin">
              <el-input-number
                v-model="formData.serviceDurationMin"
                :min="1"
                :max="240"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-card>

      <!-- 各時段 (Legs) 車輛與時間配置 -->
      <el-card shadow="never" class="section-card">
        <template #header>
          <span class="card-title">時段與車輛配置 (共 {{ formData.legs.length }} 趟)</span>
        </template>

        <div v-for="(leg, idx) in formData.legs" :key="idx" class="leg-row-box">
          <div class="leg-header">
            <el-tag effect="dark" size="small">第 {{ leg.legSeq }} 趟</el-tag>
            <el-tag :type="leg.direction === 'outbound' ? 'success' : 'info'" size="small">
              {{ leg.direction === 'outbound' ? '去程 (住家 -> 據點)' : '回程 (據點 -> 住家)' }}
            </el-tag>
          </div>

          <el-row :gutter="16" class="leg-inputs">
            <el-col :span="6">
              <el-form-item
                :label="`出發時間`"
                :prop="`legs.${idx}.departTime`"
                :rules="[{ required: true, message: '請選擇出發時間', trigger: 'change' }]"
              >
                <el-time-picker
                  v-model="leg.departTime"
                  format="HH:mm"
                  value-format="HH:mm"
                  placeholder="出發 HH:mm"
                  style="width: 100%"
                />
              </el-form-item>
            </el-col>

            <el-col :span="6">
              <el-form-item label="抵達時間 (選填)">
                <el-time-picker
                  v-model="leg.arriveTime"
                  format="HH:mm"
                  value-format="HH:mm"
                  placeholder="抵達 HH:mm"
                  style="width: 100%"
                />
              </el-form-item>
            </el-col>

            <el-col :span="6">
              <el-form-item
                label="預設指派車輛"
                :prop="`legs.${idx}.vehicleId`"
                :rules="[{ required: true, message: '請選擇車輛', trigger: 'change' }]"
              >
                <el-select
                  v-model="leg.vehicleId"
                  placeholder="選擇車輛"
                  style="width: 100%"
                  filterable
                >
                  <el-option
                    v-for="v in availableVehicles"
                    :key="v.id"
                    :label="`${v.displayName} (${v.plateNo})`"
                    :value="v.id"
                  />
                </el-select>
              </el-form-item>
            </el-col>

            <el-col :span="6">
              <el-form-item label="車次序號 (RunNo)">
                <el-input-number v-model="leg.runNo" :min="1" :max="20" style="width: 100%" />
              </el-form-item>
            </el-col>
          </el-row>
        </div>
      </el-card>

      <!-- 儲存按鈕 -->
      <div v-if="authStore.can('staff')" class="form-actions">
        <el-button type="primary" size="large" :loading="saving" @click="handleSave">
          儲存排班設定
        </el-button>
      </div>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, onMounted } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { listSites, listVehicles } from '@/api/masters'
import { saveCaseSchedule } from '@/api/cases'
import type { CaseScheduleDTO, CreateScheduleRequest, SiteDTO, VehicleDTO } from '@/types/api'
import type { Region, TripPattern } from '@/types/domain'

const props = defineProps<{
  caseId: string
  region: Region
  schedule?: CaseScheduleDTO | null
}>()

const emit = defineEmits<{
  (e: 'saved'): void
}>()

const authStore = useAuthStore()
const formRef = ref<FormInstance>()
const saving = ref(false)
const availableSites = ref<SiteDTO[]>([])
const availableVehicles = ref<VehicleDTO[]>([])

const formData = reactive<CreateScheduleRequest>({
  siteId: '',
  effectiveFrom: new Date().toISOString().split('T')[0],
  tripPattern: 2,
  weekdays: [1, 2, 3, 4, 5],
  unitPrice: 115,
  distanceKm: 5,
  serviceDurationMin: 10,
  legs: [
    { legSeq: 1, direction: 'outbound', departTime: '09:00', runNo: 1, vehicleId: '' },
    { legSeq: 2, direction: 'inbound', departTime: '16:00', runNo: 1, vehicleId: '' }
  ]
})

const rules = {
  siteId: [{ required: true, message: '請選擇據點', trigger: 'change' }],
  effectiveFrom: [{ required: true, message: '請選擇生效日期', trigger: 'change' }],
  weekdays: [{ type: 'array', required: true, min: 1, message: '請至少選擇一個每週搭乘日', trigger: 'change' }],
  unitPrice: [{ required: true, message: '請輸入單價', trigger: 'blur' }],
  distanceKm: [{ required: true, message: '請輸入單趟里程（公里）', trigger: 'blur' }],
  serviceDurationMin: [{ required: true, message: '請輸入服務時長', trigger: 'blur' }]
}

// 趟數切換時調整 legs 陣列
function handlePatternChange(pattern: TripPattern) {
  const currentVehicle = formData.legs[0]?.vehicleId || availableVehicles.value[0]?.id || ''

  if (pattern === 1) {
    formData.legs = [
      { legSeq: 1, direction: 'outbound', departTime: '09:00', runNo: 1, vehicleId: currentVehicle }
    ]
  } else if (pattern === 2) {
    formData.legs = [
      { legSeq: 1, direction: 'outbound', departTime: '09:00', runNo: 1, vehicleId: currentVehicle },
      { legSeq: 2, direction: 'inbound', departTime: '16:00', runNo: 1, vehicleId: currentVehicle }
    ]
  } else if (pattern === 4) {
    formData.legs = [
      { legSeq: 1, direction: 'outbound', departTime: '08:30', runNo: 1, vehicleId: currentVehicle },
      { legSeq: 2, direction: 'inbound', departTime: '11:30', runNo: 1, vehicleId: currentVehicle },
      { legSeq: 3, direction: 'outbound', departTime: '13:30', runNo: 1, vehicleId: currentVehicle },
      { legSeq: 4, direction: 'inbound', departTime: '16:30', runNo: 1, vehicleId: currentVehicle }
    ]
  }
}

watch(
  () => props.schedule,
  (s) => {
    if (s) {
      formData.siteId = s.siteId
      formData.effectiveFrom = s.effectiveFrom
      formData.tripPattern = s.tripPattern
      formData.weekdays = [...s.weekdays]
      formData.unitPrice = s.unitPrice
      formData.distanceKm = s.distanceKm
      formData.serviceDurationMin = s.serviceDurationMin
      formData.legs = s.legs.map((leg) => ({
        legSeq: leg.legSeq,
        direction: leg.direction,
        departTime: leg.departTime,
        arriveTime: leg.arriveTime,
        runNo: leg.runNo,
        vehicleId: leg.vehicleId
      }))
    }
  },
  { immediate: true }
)

onMounted(async () => {
  // 載入相同區域的據點與車輛
  const [sitesRes, vehiclesRes] = await Promise.all([
    listSites({ region: props.region, pageSize: 100 }),
    listVehicles({ region: props.region, active: true, pageSize: 100 })
  ])
  availableSites.value = sitesRes.data
  availableVehicles.value = vehiclesRes.data

  if (!formData.siteId && availableSites.value.length > 0) {
    formData.siteId = availableSites.value[0].id
  }
  if (!formData.legs[0]?.vehicleId && availableVehicles.value.length > 0) {
    formData.legs.forEach((leg) => {
      if (!leg.vehicleId) leg.vehicleId = availableVehicles.value[0].id
    })
  }
})

async function handleSave() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      await saveCaseSchedule(props.caseId, formData)
      ElMessage.success('排班設定儲存成功')
      emit('saved')
    } finally {
      saving.value = false
    }
  })
}
</script>

<style scoped>
.schedule-editor {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-card {
  margin-bottom: 16px;
  border-radius: 8px;

  .card-title {
    font-size: 15px;
    font-weight: bold;
    color: var(--el-color-primary);
  }
}

.leg-row-box {
  padding: 12px 16px;
  background-color: var(--el-fill-color-light);
  border-radius: 6px;
  margin-bottom: 12px;

  .leg-header {
    display: flex;
    gap: 8px;
    margin-bottom: 12px;
  }
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 10px;
}
</style>
