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
          :disabled="!authStore.can('staff')"
        >
          <el-form-item label="實際搭乘狀態">
            <el-radio-group v-model="form.effectiveStatus">
              <el-radio value="boarded">有坐</el-radio>
              <el-radio value="absent">沒坐</el-radio>
              <el-radio value="unreported">未回報</el-radio>
            </el-radio-group>
          </el-form-item>

          <el-form-item label="實際承載車輛">
            <el-select v-model="form.vehicleId" placeholder="選擇車輛" style="width: 100%" clearable>
              <el-option
                v-for="v in vehicles"
                :key="v.id"
                :label="`${v.displayName} (${v.plateNo})`"
                :value="v.id"
              />
            </el-select>
          </el-form-item>

          <el-form-item label="實際駕駛司機">
            <el-select v-model="form.driverId" placeholder="選擇司機" style="width: 100%" clearable>
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
              placeholder="預設沿用排班時間"
              style="width: 100%"
            />
            <div class="field-hint" v-if="record.scheduledDepartTime">
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
              placeholder="預設沿用排班時長"
              style="width: 100%"
            />
            <div class="field-hint" v-if="record.scheduledDurationMin">
              排班設定原值：{{ record.scheduledDurationMin }} 分鐘
            </div>
          </el-form-item>

          <el-form-item label="不申報 AA09">
            <el-switch v-model="form.notClaimedAa09" />
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
        最後更正者：{{ record.correctedByName || '承辦人員' }} 於 {{ record.correctedAt }}
        <span v-if="record.correctionReason">（原因：{{ record.correctionReason }}）</span>
      </div>
    </div>

    <template #footer>
      <div class="drawer-footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button
          v-if="authStore.can('staff')"
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
import { ref, reactive } from 'vue'
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

const form = reactive<PatchRideRequest>({
  effectiveStatus: 'boarded',
  vehicleId: '',
  driverId: '',
  departTimeOverride: null,
  durationMinOverride: null,
  notClaimedAa09: false,
  reason: ''
})

async function open(rideRecord: RideRecordDTO) {
  record.value = rideRecord
  form.effectiveStatus = rideRecord.effectiveStatus
  form.vehicleId = rideRecord.vehicleId || ''
  form.driverId = rideRecord.driverId || ''
  form.departTimeOverride = rideRecord.departTimeOverride || null
  form.durationMinOverride = rideRecord.durationMinOverride || null
  form.notClaimedAa09 = rideRecord.notClaimedAa09 || false
  form.reason = rideRecord.correctionReason || ''

  visible.value = true

  if (vehicles.value.length === 0) {
    const [vRes, dRes] = await Promise.all([
      listVehicles({ active: true, pageSize: 100 }),
      listDrivers({ active: true, pageSize: 100 })
    ])
    vehicles.value = vRes.data
    drivers.value = dRes.data
  }
}

async function handleSubmitCorrection() {
  if (!record.value) return

  // 二次確認
  await ElMessageBox.confirm(
    `確定更正 ${record.value.serviceDate} 第 ${record.value.legSeq} 趟搭乘紀錄？此操作將記錄於稽核紀錄。`,
    '確認更正紀錄',
    {
      confirmButtonText: '確認',
      cancelButtonText: '取消',
      type: 'warning'
    }
  )

  submitting.value = true
  try {
    await patchRideRecord(record.value.id, form)
    ElMessage.success('搭乘紀錄已成功更正')
    visible.value = false
    emit('updated')
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
