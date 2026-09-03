<template>
  <el-dialog
    v-model="visible"
    :title="`人工填寫搭乘記錄 — ${caseInfo.caseName} (${caseInfo.serviceDate})`"
    width="min(600px, calc(100vw - 32px))"
    destroy-on-close
    class="manual-entry-dialog"
  >
    <div class="dialog-body">
      <!-- 個案與日期摘要 -->
      <el-descriptions :column="2" border size="small" class="info-descriptions">
        <el-descriptions-item label="個案姓名">
          <strong>{{ caseInfo.caseName }}</strong>
          <span v-if="caseInfo.caseCode" class="text-secondary ml-1">({{ caseInfo.caseCode }})</span>
        </el-descriptions-item>
        <el-descriptions-item label="服務日期">
          {{ caseInfo.serviceDate }}
        </el-descriptions-item>
      </el-descriptions>

      <el-form
        ref="formRef"
        :model="form"
        label-width="110px"
        label-position="right"
        style="margin-top: 16px;"
      >
        <el-form-item label="趟次 (時段)" required>
          <el-select
            v-model="form.legSeq"
            placeholder="請選擇趟次"
            style="width: 100%;"
          >
            <el-option
              v-for="leg in legOptions"
              :key="leg.value"
              :label="leg.label"
              :value="leg.value"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="實際搭乘狀態" required>
          <el-radio-group v-model="form.effectiveStatus">
            <el-radio value="boarded">有坐</el-radio>
            <el-radio value="absent">沒坐</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-alert
          v-if="masterDataError"
          type="error"
          show-icon
          :closable="false"
          title="車輛與司機主檔載入失敗，下方清單可能不完整"
          style="margin-bottom: 12px"
        >
          <template #default>
            <el-button size="small" @click="fetchMasterData">重試</el-button>
          </template>
        </el-alert>

        <el-form-item label="實際承載車輛">
          <el-select
            v-model="form.vehicleId"
            :placeholder="isAbsent ? '沒坐無須選擇車輛' : '請選擇承載車輛'"
            :disabled="isAbsent"
            filterable
            clearable
            style="width: 100%;"
          >
            <el-option
              v-for="v in vehicles"
              :key="v.id"
              :label="`${v.displayName} (${v.plateNo})`"
              :value="v.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="實際駕駛司機">
          <el-select
            v-model="form.driverId"
            :placeholder="isAbsent ? '沒坐無須選擇司機' : '請選擇駕駛司機'"
            :disabled="isAbsent"
            filterable
            clearable
            style="width: 100%;"
          >
            <el-option
              v-for="d in drivers"
              :key="d.id"
              :label="d.name"
              :value="d.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="出發時間">
          <el-time-picker
            v-model="form.departTimeOverride"
            format="HH:mm"
            value-format="HH:mm"
            :placeholder="isAbsent ? '沒坐無出發時間' : '如 09:00 (選填)'"
            :disabled="isAbsent"
            style="width: 100%;"
          />
        </el-form-item>

        <el-form-item label="服務時長 (分)">
          <el-input-number
            v-model="form.durationMinOverride"
            :min="1"
            :max="240"
            :placeholder="isAbsent ? '沒坐無服務時長' : '預設 10 分鐘'"
            :disabled="isAbsent"
            style="width: 100%;"
          />
        </el-form-item>

        <el-form-item label="不申報 AA09">
          <el-switch
            v-model="form.notClaimedAa09"
            :disabled="isAbsent"
          />
        </el-form-item>

        <el-form-item label="填寫原因 / 備註">
          <div class="reason-tags-wrapper">
            <el-tag
              v-for="r in QUICK_REASONS"
              :key="r"
              class="quick-reason-tag"
              effect="plain"
              size="small"
              @click="form.reason = r"
            >
              {{ r }}
            </el-tag>
          </div>
          <el-input
            v-model="form.reason"
            type="textarea"
            :rows="2"
            placeholder="請輸入人工填寫原因或備註 (選填)"
            style="margin-top: 6px;"
          />
        </el-form-item>
      </el-form>
    </div>

    <template #footer>
      <DialogFooter :loading="saving" @confirm="handleSubmit" @cancel="visible = false" />
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { submitManualRideReport } from '@/api/rides'
import DialogFooter from '@/components/DialogFooter.vue'
import { listVehicles, listDrivers } from '@/api/masters'
import type { VehicleDTO, DriverDTO, ManualReportRideRequest } from '@/types/api'

interface OpenOptions {
  caseId: string
  caseName: string
  caseCode?: string
  serviceDate: string
  tripPattern?: number
  targetLegSeq?: number
  existingLegs?: number[]
}

const emit = defineEmits<{
  (e: 'saved'): void
}>()

const visible = ref(false)
const saving = ref(false)

const caseInfo = reactive({
  caseId: '',
  caseName: '',
  caseCode: '',
  serviceDate: '',
  tripPattern: 2,
  existingLegs: [] as number[]
})

const QUICK_REASONS = [
  '非排定日臨時搭乘',
  '電話確認已搭乘',
  '司機口頭回報',
  '代班司機回報',
  '事後補登',
  '請假標記沒坐'
]

const form = reactive<ManualReportRideRequest>({
  caseId: '',
  serviceDate: '',
  legSeq: 1,
  effectiveStatus: 'boarded',
  vehicleId: '',
  driverId: '',
  departTimeOverride: null,
  durationMinOverride: 10,
  notClaimedAa09: false,
  reason: '非排定日臨時搭乘'
})

const isAbsent = computed(() => form.effectiveStatus === 'absent')

const legOptions = computed(() => {
  const count = Math.max(caseInfo.tripPattern || 2, 2)
  const list = []
  for (let i = 1; i <= Math.max(count, 4); i++) {
    let desc = ''
    if (i === 1) desc = ' (去程)'
    else if (i === 2) desc = ' (回程)'
    else desc = ` (第 ${i} 趟)`

    const isExisting = caseInfo.existingLegs.includes(i)
    list.push({
      value: i,
      label: `第 ${i} 趟${desc}${isExisting ? ' [已有紀錄，將覆寫]' : ''}`
    })
  }
  return list
})

const vehicles = ref<VehicleDTO[]>([])
const drivers = ref<DriverDTO[]>([])
const masterDataError = ref(false)

watch(
  () => form.effectiveStatus,
  (newStatus) => {
    if (newStatus === 'absent') {
      form.vehicleId = ''
      form.driverId = ''
      form.departTimeOverride = null
      form.durationMinOverride = null
      form.notClaimedAa09 = false
      if (!form.reason || form.reason === '非排定日臨時搭乘') {
        form.reason = '請假標記沒坐'
      }
    } else {
      form.durationMinOverride = 10
      if (form.reason === '請假標記沒坐') {
        form.reason = '非排定日臨時搭乘'
      }
    }
  }
)

async function fetchMasterData() {
  if (vehicles.value.length === 0 || drivers.value.length === 0) {
    masterDataError.value = false
    try {
      const [vRes, dRes] = await Promise.all([
        listVehicles({ active: true, pageSize: 100 }),
        listDrivers({ active: true, pageSize: 100 })
      ])
      vehicles.value = vRes.data
      drivers.value = dRes.data
    } catch {
      // 全域攔截器已彈出錯誤訊息；這裡另外標記狀態，讓下拉選單旁能顯示可重試的空清單原因
      masterDataError.value = true
    }
  }
}

function open(options: OpenOptions) {
  caseInfo.caseId = options.caseId
  caseInfo.caseName = options.caseName
  caseInfo.caseCode = options.caseCode || ''
  caseInfo.serviceDate = options.serviceDate
  caseInfo.tripPattern = options.tripPattern || 2
  caseInfo.existingLegs = options.existingLegs || []

  // 自動推算預設 legSeq：優先使用傳入之 targetLegSeq，否則挑選尚未有紀錄的最小 legSeq
  let defaultLeg = options.targetLegSeq || 1
  if (!options.targetLegSeq) {
    for (let i = 1; i <= Math.max(caseInfo.tripPattern, 2); i++) {
      if (!caseInfo.existingLegs.includes(i)) {
        defaultLeg = i
        break
      }
    }
  }

  const defaultDepartTime =
    defaultLeg === 1 ? '09:00' : defaultLeg === 2 ? '16:00' : defaultLeg === 3 ? '13:00' : '17:00'

  form.caseId = options.caseId
  form.serviceDate = options.serviceDate
  form.legSeq = defaultLeg
  form.effectiveStatus = 'boarded'
  form.vehicleId = ''
  form.driverId = ''
  form.departTimeOverride = defaultDepartTime
  form.durationMinOverride = 10
  form.notClaimedAa09 = false
  form.reason = '非排定日臨時搭乘'

  visible.value = true
  fetchMasterData()
}

async function handleSubmit() {
  if (!form.caseId || !form.serviceDate) return

  // 確認若覆寫既有紀錄
  if (caseInfo.existingLegs.includes(form.legSeq)) {
    try {
      await ElMessageBox.confirm(
        `此個案在 ${form.serviceDate} 已有第 ${form.legSeq} 趟搭乘紀錄，確認是否覆寫？`,
        '確認覆寫紀錄',
        {
          confirmButtonText: '確認覆寫',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )
    } catch {
      return
    }
  }

  saving.value = true
  try {
    const payload: ManualReportRideRequest = {
      caseId: form.caseId,
      serviceDate: form.serviceDate,
      legSeq: form.legSeq,
      effectiveStatus: form.effectiveStatus,
      vehicleId: isAbsent.value ? undefined : (form.vehicleId || undefined),
      driverId: isAbsent.value ? undefined : (form.driverId || undefined),
      departTimeOverride: isAbsent.value ? undefined : (form.departTimeOverride || undefined),
      durationMinOverride: isAbsent.value ? undefined : (form.durationMinOverride || undefined),
      notClaimedAa09: isAbsent.value ? false : form.notClaimedAa09,
      reason: form.reason || undefined
    }

    await submitManualRideReport(payload)
    ElMessage.success('搭乘紀錄已成功儲存')
    visible.value = false
    emit('saved')
  } catch (err: any) {
    ElMessage.error(err?.message || '儲存搭乘紀錄失敗')
  } finally {
    saving.value = false
  }
}

defineExpose({
  open
})
</script>

<style scoped>
.dialog-body {
  display: flex;
  flex-direction: column;
}

.info-descriptions {
  border-radius: 6px;
  overflow: hidden;
}

.reason-tags-wrapper {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;

  .quick-reason-tag {
    cursor: pointer;
    &:hover {
      background-color: var(--app-primary-light);
    }
  }
}

</style>

