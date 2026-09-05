<template>
  <el-dialog
    :model-value="modelValue"
    title="新增司機"
    width="min(480px, calc(100vw - 32px))"
    @update:model-value="(val: boolean) => emit('update:modelValue', val)"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
      <el-form-item label="司機姓名" prop="name">
        <el-input v-model="form.name" placeholder="請輸入姓名" />
      </el-form-item>
      <el-form-item label="身分證字號" prop="nationalId">
        <el-input v-model="form.nationalId" placeholder="1 碼英文 + 9 碼數字" />
      </el-form-item>
      <el-form-item label="所屬區域" prop="region">
        <el-select v-model="form.region" placeholder="請選擇區域" filterable style="width: 100%">
          <el-option v-for="(label, key) in REGION_LABELS" :key="key" :label="label" :value="key" />
        </el-select>
      </el-form-item>
      <el-form-item label="電子信箱" prop="email">
        <el-input v-model="form.email" placeholder="通知寄送用信箱" />
      </el-form-item>
      <el-form-item label="駕照類別" prop="licenseClass">
        <el-select v-model="form.licenseClass" placeholder="請選擇駕照類別" clearable style="width: 100%">
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
          placeholder="請選擇駕照有效日期"
          style="width: 100%"
        />
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
import { DRIVER_LICENSE_CLASS_LABELS, REGION_LABELS } from '@/types/domain'
import type { CreateDriverRequest, DriverDTO } from '@/types/api'
import { isValidNationalID } from '@/utils/nationalId'

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
  region: 'miaoli',
    email: '',
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
  ],
  region: [{ required: true, message: '請選擇所屬區域', trigger: 'change' }]
}

watch(
  () => props.modelValue,
  (visible) => {
    if (!visible) return
    form.name = props.prefillName || ''
    form.nationalId = ''
    form.region = 'miaoli'
    form.email = ''
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
