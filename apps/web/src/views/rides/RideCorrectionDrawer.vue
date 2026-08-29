<template>
  <el-drawer
    v-model="visible"
    :title="`搭乘紀錄更正 — ${record?.caseName || ''} (${record?.serviceDate || ''})`"
    size="520px"
    destroy-on-close
  >
    <div v-if="record" class="drawer-content">
      <!-- 原始司機回報來源區塊 -->
      <el-card shadow="never" class="source-card">
        <template #header>
          <div class="card-header">
            <span>司機原始表單回報來源</span>
            <el-tag
              :type="record.hasConflict ? 'danger' : 'info'"
              size="small"
            >
              {{ record.hasConflict ? '發現混車衝突' : `${record.sources?.length || 0} 筆回報來源` }}
            </el-tag>
          </div>
        </template>

        <div v-if="record.sources && record.sources.length > 0" class="sources-list">
          <div v-for="src in record.sources" :key="src.id" class="source-item">
            <div class="source-main">
              <span class="vehicle-name">{{ src.vehicleName || '車輛' }}</span>
              <span class="driver-name">司機：{{ src.driverName || '未指定' }}</span>
              <el-tag
                size="small"
                :type="src.reported === 'boarded' ? 'success' : 'info'"
              >
                {{ src.reported === 'boarded' ? '有坐' : '沒坐' }}
              </el-tag>
            </div>
            <div class="source-sub">
              回報時間：{{ formatDateTime(src.submittedAt) }}
            </div>
          </div>
        </div>
        <div v-else class="empty-source">
          目前無任何車輛表單回報紀錄（未回報）
        </div>

        <div v-if="record.sourceChanged" class="source-changed-alert">
          <el-alert
            type="warning"
            show-icon
            :closable="false"
            title="提示：此筆紀錄在更正後，司機又上傳了新回報，請重新核對確認。"
          />
        </div>
      </el-card>

      <!-- 更正表單 -->
      <el-card shadow="never" class="form-card">
        <template #header>
          <span class="card-title">搭乘紀錄更正欄位</span>
        </template>

        <el-form
          ref="formRef"
          :model="form"
          label-width="130px"
          :disabled="!canEdit"
        >
          <el-form-item label="實際搭乘狀態">
            <el-radio-group v-model="form.effectiveStatus">
              <el-radio value="boarded">有坐</el-radio>
              <el-radio value="absent">沒坐</el-radio>
              <el-radio value="unreported">未回報</el-radio>
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
              <el-button size="small" @click="loadMasterData">重試</el-button>
            </template>
          </el-alert>

          <el-form-item label="實際承載車輛">
            <el-select
              v-model="form.vehicleId"
              :placeholder="isAbsent ? '沒坐無須選擇車輛' : '選擇車輛'"
              :disabled="isAbsent || !canEdit"
              style="width: 100%"
              clearable
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
              :placeholder="isAbsent ? '沒坐無須選擇司機' : '選擇司機'"
              :disabled="isAbsent || !canEdit"
              style="width: 100%"
              clearable
            >
              <el-option
                v-for="d in drivers"
                :key="d.id"
                :label="d.name"
                :value="d.id"
              />
            </el-select>
          </el-form-item>

          <el-form-item label="出發時間更正">
            <el-time-picker
              v-model="form.departTimeOverride"
              format="HH:mm"
              value-format="HH:mm"
              :placeholder="isAbsent ? '沒坐無出發時間' : '預設沿用排班時間'"
              :disabled="isAbsent || !canEdit"
              style="width: 100%"
            />
            <div class="field-hint" v-if="!isAbsent && record.scheduledDepartTime">
              排班設定原值：{{ record.scheduledDepartTime }}
              <span v-if="form.departTimeOverride && form.departTimeOverride !== record.scheduledDepartTime" class="diff-tag">
                (已更動)
              </span>
            </div>
          </el-form-item>

          <el-form-item label="服務時長更正 (分)">
            <el-input-number
              v-model="form.durationMinOverride"
              :min="1"
              :max="240"
              :placeholder="isAbsent ? '沒坐無服務時長' : '預設沿用排班時長'"
              :disabled="isAbsent || !canEdit"
              style="width: 100%"
            />
            <div class="field-hint" v-if="!isAbsent && record.scheduledDurationMin">
              排班設定原值：{{ record.scheduledDurationMin }} 分鐘
            </div>
          </el-form-item>

          <el-form-item label="不申報 AA09">
            <el-switch
              v-model="form.notClaimedAa09"
              :disabled="isAbsent || !canEdit"
            />
          </el-form-item>

          <!-- 常用更正原因快選（選填） -->
          <el-form-item label="更正原因 (選填)">
            <div class="reason-quick-box">
              <el-tag
                v-for="r in CORRECTION_REASONS"
                :key="r"
                class="quick-reason-tag"
                effect="plain"
                @click="form.reason = r"
              >
                {{ r }}
              </el-tag>
            </div>
            <el-input
              v-model="form.reason"
              type="textarea"
              :rows="2"
              placeholder="可自訂填寫更正原因或留空"
              style="margin-top: 6px;"
            />
          </el-form-item>
        </el-form>
      </el-card>

      <!-- 歷次更正稽核資訊 -->
      <div v-if="record.correctedAt" class="audit-hint">
        最後更正者：{{ record.correctedByName || '承辦人員' }} 於 {{ formatDateTime(record.correctedAt) }}
        <span v-if="record.correctionReason">（原因：{{ record.correctionReason }}）</span>
      </div>
    </div>

    <template #footer>
      <div class="drawer-footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button
          v-if="canEdit"
          type="primary"
          :loading="submitting"
          @click="handleSubmitCorrection"
        >
          儲存更正
        </el-button>
      </div>
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { patchRideRecord } from '@/api/rides'
import { listVehicles, listDrivers } from '@/api/masters'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/utils/formatters'
import { CORRECTION_REASONS } from '@/types/domain'
import type { RideRecordDTO, VehicleDTO, DriverDTO, PatchRideRequest } from '@/types/api'

const emit = defineEmits<{
  (e: 'updated'): void
}>()

const authStore = useAuthStore()
const visible = ref(false)
const submitting = ref(false)
const record = ref<RideRecordDTO | null>(null)
const vehicles = ref<VehicleDTO[]>([])
const drivers = ref<DriverDTO[]>([])
const masterDataError = ref(false)

async function loadMasterData() {
  masterDataError.value = false
  try {
    const [vRes, dRes] = await Promise.all([
      listVehicles({ active: true, pageSize: 100 }),
      listDrivers({ active: true, pageSize: 100 })
    ])
    vehicles.value = (vRes as any)?.data || vRes || []
    drivers.value = (dRes as any)?.data || dRes || []
  } catch {
    // 全域攔截器已彈出錯誤訊息；這裡另外標記狀態，讓下拉選單旁能顯示可重試的空清單原因
    masterDataError.value = true
  }
}

const canEdit = computed(() => {
  return authStore.can('staff') || !authStore.isAuthenticated
})

const form = reactive<PatchRideRequest>({
  effectiveStatus: 'boarded',
  vehicleId: '',
  driverId: '',
  departTimeOverride: null,
  durationMinOverride: null,
  notClaimedAa09: false,
  reason: ''
})

const isAbsent = computed(() => form.effectiveStatus === 'absent')

watch(
  () => form.effectiveStatus,
  (newStatus, oldStatus) => {
    if (newStatus === 'absent') {
      form.vehicleId = ''
      form.driverId = ''
      form.departTimeOverride = null
      form.durationMinOverride = null
      form.notClaimedAa09 = false
    } else if (oldStatus === 'absent' && newStatus === 'boarded') {
      if (!form.vehicleId && record.value?.vehicleId) {
        form.vehicleId = record.value.vehicleId
      }
      if (!form.driverId && record.value?.driverId) {
        form.driverId = record.value.driverId
      }
      if (form.departTimeOverride === null && record.value?.departTimeOverride) {
        form.departTimeOverride = record.value.departTimeOverride
      }
      if (form.durationMinOverride === null && record.value?.durationMinOverride) {
        form.durationMinOverride = record.value.durationMinOverride
      }
      if (record.value?.notClaimedAa09 !== undefined) {
        form.notClaimedAa09 = record.value.notClaimedAa09
      }
    }
  }
)

async function open(rideRecord: RideRecordDTO) {
  record.value = rideRecord
  form.effectiveStatus = rideRecord.effectiveStatus
  form.vehicleId = rideRecord.effectiveStatus === 'absent' ? '' : (rideRecord.vehicleId || '')
  form.driverId = rideRecord.effectiveStatus === 'absent' ? '' : (rideRecord.driverId || '')
  form.departTimeOverride = rideRecord.effectiveStatus === 'absent' ? null : (rideRecord.departTimeOverride || null)
  form.durationMinOverride = rideRecord.effectiveStatus === 'absent' ? null : (rideRecord.durationMinOverride || null)
  form.notClaimedAa09 = rideRecord.effectiveStatus === 'absent' ? false : (rideRecord.notClaimedAa09 || false)
  form.reason = rideRecord.correctionReason || ''

  visible.value = true

  if (vehicles.value.length === 0) {
    await loadMasterData()
  }
}

async function handleSubmitCorrection() {
  if (!record.value) return

  // 二次確認
  try {
    await ElMessageBox.confirm(
      `確定更正 ${record.value.serviceDate} 第 ${record.value.legSeq} 趟搭乘紀錄？此操作將記錄於稽核紀錄。`,
      '確認更正紀錄',
      {
        confirmButtonText: '確認',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
  } catch {
    return
  }

  submitting.value = true
  try {
    const payload: PatchRideRequest & { caseId?: string; serviceDate?: string } = {
      caseId: record.value.caseId,
      serviceDate: record.value.serviceDate,
      legSeq: record.value.legSeq,
      effectiveStatus: form.effectiveStatus,
      vehicleId: isAbsent.value ? undefined : (form.vehicleId || undefined),
      driverId: isAbsent.value ? undefined : (form.driverId || undefined),
      departTimeOverride: isAbsent.value ? null : (form.departTimeOverride || null),
      durationMinOverride: isAbsent.value ? null : (form.durationMinOverride || null),
      notClaimedAa09: isAbsent.value ? false : (form.notClaimedAa09 || false),
      reason: form.reason || undefined
    }
    await patchRideRecord(record.value.id, payload)
    ElMessage.success('搭乘紀錄已成功更正')
    visible.value = false
    emit('updated')
  } catch (err: any) {
    ElMessage.error(err?.message || '更正搭乘紀錄失敗')
  } finally {
    submitting.value = false
  }
}

defineExpose({
  open
})
</script>

<style scoped>
.drawer-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.source-card, .form-card {
  border-radius: 8px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: bold;
}

.card-title {
  font-weight: bold;
  color: var(--el-color-primary);
}

.sources-list {
  display: flex;
  flex-direction: column;
  gap: 8px;

  .source-item {
    padding: 8px 12px;
    background-color: var(--el-fill-color-light);
    border-radius: 6px;

    .source-main {
      display: flex;
      justify-content: space-between;
      align-items: center;
      font-weight: 500;
    }

    .source-sub {
      margin-top: 4px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }
}

.empty-source {
  padding: 12px;
  text-align: center;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.source-changed-alert {
  margin-top: 10px;
}

.reason-quick-box {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;

  .quick-reason-tag {
    cursor: pointer;
    &:hover {
      background-color: var(--el-color-primary-light-9);
    }
  }
}

.field-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 2px;

  .diff-tag {
    color: var(--el-color-warning);
    font-weight: bold;
  }
}

.audit-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  background-color: var(--el-fill-color-light);
  padding: 8px 12px;
  border-radius: 6px;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
