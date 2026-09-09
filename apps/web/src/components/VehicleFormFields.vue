<template>
  <el-form-item label="車號" prop="plateNo">
    <el-input v-model="form.plateNo" placeholder="如：BZG-7915" />
  </el-form-item>
  <el-form-item label="車別" prop="displayName">
    <el-input v-model="form.displayName" placeholder="請輸入車別" />
  </el-form-item>
  <el-form-item label="據點" prop="siteName">
    <el-input v-model="form.siteName" placeholder="請輸入所屬據點（選填）" />
  </el-form-item>
  <el-form-item label="廠牌" prop="brand">
    <el-input v-model="form.brand" placeholder="如：中華" />
  </el-form-item>
  <el-form-item label="車型" prop="model">
    <el-input v-model="form.model" placeholder="如：DE241L8" />
  </el-form-item>
  <el-form-item label="出廠年月" prop="manufactureYm">
    <el-date-picker
      v-model="form.manufactureYm"
      type="month"
      placeholder="請選擇年月"
      value-format="YYYY-MM"
      style="width: 100%"
    />
  </el-form-item>
  <el-form-item
    v-for="field in VEHICLE_DATE_FIELDS"
    :key="field.prop"
    :label="field.label"
    :prop="field.prop"
  >
    <el-date-picker
      v-model="form[field.prop]"
      type="date"
      placeholder="請選擇日期"
      value-format="YYYY-MM-DD"
      style="width: 100%"
    />
    <!-- 清冊以民國年呈現，這裡即時回饋換算結果，避免選錯年份 -->
    <span v-if="form[field.prop]" class="roc-date-hint">
      民國 {{ formatRocDate(form[field.prop]) }}
    </span>
  </el-form-item>
  <el-form-item label="符合輪椅載運規定" prop="wheelchairAccessible">
    <el-radio-group v-model="form.wheelchairAccessible">
      <el-radio-button :value="true">是</el-radio-button>
      <el-radio-button :value="false">否</el-radio-button>
    </el-radio-group>
  </el-form-item>
  <el-form-item label="證件資料">
    <el-checkbox v-model="form.hasVehicleLicense">行照</el-checkbox>
    <el-checkbox v-model="form.hasPurchaseContract">汽車買賣合約書</el-checkbox>
    <el-checkbox v-model="form.hasPlateRegistration">領牌登記書</el-checkbox>
    <el-checkbox v-model="form.hasTransferRegistration">異動登記書</el-checkbox>
  </el-form-item>
  <el-form-item v-if="showStatus" label="狀態" prop="status">
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
</template>

<script setup lang="ts">
import { formatRocDate } from '@/utils/formatters'
import { VEHICLE_DATE_FIELDS } from '@/utils/vehicleForm'
import type { CreateVehicleRequest } from '@/types/api'

// form 是父層 el-form 綁定的同一個 reactive 物件，直接就地修改，不再另做一層 v-model 轉發
const { form, showStatus = false } = defineProps<{
  form: CreateVehicleRequest
  showStatus?: boolean
}>()
</script>

<style scoped>
.roc-date-hint {
  margin-left: 10px;
  color: var(--el-text-color-secondary);
  font-size: var(--app-font-xs);
  white-space: nowrap;
}
</style>
