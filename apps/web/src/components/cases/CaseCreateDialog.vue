<template>
  <el-dialog
    :model-value="modelValue"
    title="新增個案基本資料"
    width="min(600px, calc(100vw - 32px))"
    @update:model-value="(val: boolean) => emit('update:modelValue', val)"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="110px" class="dialog-scroll-form">
      <el-form-item label="個案姓名" prop="name">
        <el-input v-model="form.name" placeholder="請輸入姓名（含罕用字）" />
      </el-form-item>
      <el-form-item label="身分證字號" prop="nationalId">
        <el-input v-model="form.nationalId" placeholder="1 碼英文字母 + 9 碼數字" />
      </el-form-item>
      <el-form-item label="所屬據點" prop="siteId">
        <el-select v-model="form.siteId" placeholder="請選擇據點" filterable style="width: 100%">
          <el-option v-for="site in availableSites" :key="site.id" :label="site.name" :value="site.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="照護人員" prop="caregiverId">
        <el-select v-model="form.caregiverId" placeholder="請選擇照護人員" filterable style="width: 100%">
          <el-option
            v-for="caregiver in availableCaregivers"
            :key="caregiver.id"
            :label="caregiver.name + (caregiver.type ? `（${CAREGIVER_TYPE_LABELS[caregiver.type as CaregiverType] || caregiver.type}）` : '')"
            :value="caregiver.id"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="住家地址" prop="homeAddress">
        <el-input v-model="form.homeAddress" placeholder="請輸入住家地址" />
      </el-form-item>
      <el-form-item label="服務類別" prop="serviceCategory">
        <el-radio-group v-model="form.serviceCategory">
          <el-radio :value="1">1. 補助</el-radio>
          <el-radio :value="2">2. 自費</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="服務使用類型" prop="serviceUsageType">
        <el-select v-model="form.serviceUsageType" placeholder="未選擇" clearable style="width: 100%">
          <el-option :value="1" label="1. 社區式長照機構" />
          <el-option :value="2" label="2. 社區服務據點(不含身障類)" />
          <el-option :value="3" label="3. 輔具中心" />
          <el-option :value="4" label="4. 身障日間照顧服務" />
        </el-select>
      </el-form-item>
      <el-form-item label="備註" prop="remarks">
        <el-input v-model="form.remarks" type="textarea" :rows="2" placeholder="選填" />
      </el-form-item>
    </el-form>
    <template #footer>
      <DialogFooter :loading="saving" @confirm="handleConfirm" @cancel="emit('update:modelValue', false)" />
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch, onMounted } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import DialogFooter from '@/components/DialogFooter.vue'
import { createCase } from '@/api/cases'
import { listAllSites } from '@/api/masters'
import { listAllCaregivers } from '@/api/caregivers'
import { CAREGIVER_TYPE_LABELS, type CaregiverType } from '@/types/domain'
import type { CaseDTO, CreateCaseRequest, SiteDTO, CaregiverDTO } from '@/types/api'

const availableSites = ref<SiteDTO[]>([])
const availableCaregivers = ref<CaregiverDTO[]>([])
onMounted(async () => {
  const [sites, caregivers] = await Promise.all([
    listAllSites({ status: 'active' }),
    listAllCaregivers({ status: 'active' })
  ])
  availableSites.value = sites
  availableCaregivers.value = caregivers
})

// 跟個案清單頁「新增個案基本資料」共用同一份欄位與 API，避免兩邊各自維護造成落差；
// 呼叫端只在成功後拿到新建立的個案，趟次等匯報表專屬綁定資訊由呼叫端自行處理。
const props = defineProps<{
  modelValue: boolean
  prefillName?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  created: [caseDto: CaseDTO]
}>()

const formRef = ref<FormInstance>()
const saving = ref(false)
const form = reactive<CreateCaseRequest>({
  name: '',
  siteId: '',
  caregiverId: '',
  nationalId: '',
  homeAddress: '',
  serviceCategory: undefined,
  serviceUsageType: undefined,
  status: 'active',
  remarks: ''
})

// 姓名、所屬據點與照護人員為必填；身分證字號與居住地仍為選填
const rules = {
  name: [{ required: true, message: '請輸入個案姓名', trigger: 'blur' }],
  siteId: [{ required: true, message: '請選擇所屬據點', trigger: 'change' }],
  caregiverId: [{ required: true, message: '請選擇照護人員', trigger: 'change' }]
}

watch(
  () => props.modelValue,
  (visible) => {
    if (!visible) return
    form.name = props.prefillName || ''
    form.siteId = ''
    form.caregiverId = ''
    form.nationalId = ''
    form.homeAddress = ''
    form.serviceCategory = undefined
    form.serviceUsageType = undefined
    form.remarks = ''
    formRef.value?.clearValidate()
  }
)

async function handleConfirm() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    saving.value = true
    try {
      const created = await createCase(form)
      ElMessage.success(`個案「${created.name}」建立成功`)
      emit('update:modelValue', false)
      emit('created', created)
    } finally {
      saving.value = false
    }
  })
}
</script>
