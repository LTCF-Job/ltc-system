<template>
  <el-dialog
    :model-value="modelValue"
    title="新增司機"
    width="min(560px, calc(100vw - 32px))"
    @update:model-value="(val: boolean) => emit('update:modelValue', val)"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
      <el-form-item label="司機姓名" prop="name">
        <el-input v-model="form.name" placeholder="請輸入姓名" />
      </el-form-item>
      <el-form-item label="身分證字號" prop="nationalId">
        <el-input v-model="form.nationalId" placeholder="1 碼英文 + 9 碼數字" />
      </el-form-item>
      <el-form-item label="性別" prop="gender">
        <el-select v-model="form.gender" placeholder="請選擇性別（選填）" clearable style="width: 100%">
          <el-option value="男" label="男" />
          <el-option value="女" label="女" />
        </el-select>
      </el-form-item>
      <el-form-item label="生日" prop="birthDate">
        <el-date-picker
          v-model="form.birthDate"
          type="date"
          value-format="YYYY-MM-DD"
          placeholder="請選擇出生日期（選填）"
          clearable
          style="width: 100%"
        />
      </el-form-item>
      <el-form-item label="所屬區域" prop="region">
        <el-select v-model="form.region" placeholder="請選擇區域（選填）" clearable filterable style="width: 100%">
          <el-option v-for="opt in regionOptions" :key="opt.code" :label="opt.name" :value="opt.code" />
        </el-select>
      </el-form-item>
      <el-form-item label="電子信箱" prop="email">
        <el-input v-model="form.email" placeholder="通知寄送用信箱（選填）" clearable />
      </el-form-item>
      <el-form-item label="到職日" prop="employmentDate">
        <el-date-picker
          v-model="form.employmentDate"
          type="date"
          value-format="YYYY-MM-DD"
          placeholder="請選擇到職日（選填）"
          clearable
          style="width: 100%"
        />
      </el-form-item>
      <el-form-item label="駕照類別" prop="licenseClass">
        <el-select v-model="form.licenseClass" placeholder="請選擇駕照類別（選填）" clearable style="width: 100%">
          <el-option
            v-for="(label, value) in DRIVER_LICENSE_CLASS_LABELS"
            :key="value"
            :label="label"
            :value="value"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="駕照有效日期" prop="licenseExpiryDate">
        <el-date-picker
          v-model="form.licenseExpiryDate"
          type="date"
          value-format="YYYY-MM-DD"
          placeholder="請選擇駕照有效日期（選填）"
          clearable
          style="width: 100%"
        />
      </el-form-item>
      <el-form-item label="證照證明">
        <el-checkbox v-model="form.hasProfessionalLicense">職業駕照</el-checkbox>
        <el-checkbox v-model="form.hasTransferCert">異動登記書</el-checkbox>
      </el-form-item>
      <el-form-item label="驗車日" prop="inspectionDate">
        <el-date-picker
          v-model="form.inspectionDate"
          type="date"
          value-format="YYYY-MM-DD"
          placeholder="請選擇驗車日（選填）"
          clearable
          style="width: 100%"
        />
      </el-form-item>
      <el-form-item label="備註" prop="remarks">
        <el-input v-model="form.remarks" type="textarea" :rows="2" placeholder="選填備註" clearable />
      </el-form-item>
    </el-form>
    <template #footer>
      <DialogFooter :loading="saving" @confirm="handleConfirm" @cancel="emit('update:modelValue', false)" />
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import DialogFooter from '@/components/DialogFooter.vue'
import { createDriver } from '@/api/masters'
import { DRIVER_LICENSE_CLASS_LABELS } from '@/types/domain'
import type { CreateDriverRequest, DriverDTO } from '@/types/api'
import { isValidNationalID } from '@/utils/nationalId'

import { fetchRegionOptions, type RegionOption } from '@/api/regionOptions'

// 區域選項一律來自地區主檔，避免前端寫死清單與主檔、DB 值域三方不一致。
const regionOptions = ref<RegionOption[]>([])
onMounted(async () => {
  regionOptions.value = await fetchRegionOptions()
})

// 跟司機管理頁「新增司機」共用同一份欄位與 API，避免兩邊各自維護造成落差；
// 編輯流程不在本元件範圍，維持在司機管理頁自行處理。
const props = defineProps<{
  modelValue: boolean
  prefillName?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  created: [driver: DriverDTO]
}>()

const formRef = ref<FormInstance>()
const saving = ref(false)
const form = reactive<CreateDriverRequest>({
  name: '',
  nationalId: '',
  region: '',
  email: '',
  gender: '',
  birthDate: null,
  hasProfessionalLicense: false,
  employmentDate: null,
  hasTransferCert: false,
  inspectionDate: null,
  remarks: '',
  licenseClass: null,
  licenseExpiryDate: null
})

const rules = {
  name: [{ required: true, message: '請輸入司機姓名', trigger: 'blur' }],
  nationalId: [
    { required: true, message: '請輸入身分證字號', trigger: 'blur' },
    {
      validator: (_rule: unknown, value: string, callback: (error?: Error) => void) => {
        if (value && !isValidNationalID(value)) {
          callback(new Error('身分證字號格式錯誤，請確認後再試'))
          return
        }
        callback()
      },
      trigger: 'blur'
    }
  ]
}

watch(
  () => props.modelValue,
  (visible) => {
    if (!visible) return
    form.name = props.prefillName || ''
    form.nationalId = ''
    form.region = ''
    form.email = ''
    form.gender = ''
    form.birthDate = null
    form.hasProfessionalLicense = false
    form.employmentDate = null
    form.hasTransferCert = false
    form.inspectionDate = null
    form.remarks = ''
    form.licenseClass = null
    form.licenseExpiryDate = null
    formRef.value?.clearValidate()
  }
)

async function handleConfirm() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      const created = await createDriver(form)
      ElMessage.success(`司機「${created.name}」建立成功`)
      emit('update:modelValue', false)
      emit('created', created)
    } finally {
      saving.value = false
    }
  })
}
</script>
