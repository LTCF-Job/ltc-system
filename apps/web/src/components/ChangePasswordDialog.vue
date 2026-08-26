<template>
  <el-dialog
    v-model="visible"
    title="修改登入密碼"
    width="460px"
    destroy-on-close
    :close-on-click-modal="false"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="110px"
      label-position="right"
      @submit.prevent="handleSubmit"
    >
      <el-form-item label="目前密碼" prop="oldPassword">
        <el-input
          v-model="form.oldPassword"
          type="password"
          placeholder="請輸入目前使用的密碼"
          show-password
          autocomplete="current-password"
        />
      </el-form-item>

      <el-form-item label="新密碼" prop="newPassword">
        <el-input
          v-model="form.newPassword"
          type="password"
          placeholder="請輸入新密碼（至少 6 碼）"
          show-password
          autocomplete="new-password"
        />
      </el-form-item>

      <el-form-item label="確認新密碼" prop="confirmPassword">
        <el-input
          v-model="form.confirmPassword"
          type="password"
          placeholder="請再次輸入新密碼"
          show-password
          autocomplete="new-password"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="loading" @click="handleSubmit">
          確認變更
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { supabase } from '@/lib/supabase'
import { changeSelfPassword } from '@/api/users'

const visible = ref(false)
const loading = ref(false)
const formRef = ref<FormInstance>()

const form = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const validateConfirmPassword = (_rule: any, value: string, callback: any) => {
  if (!value) {
    return callback(new Error('請再次輸入新密碼'))
  }
  if (value !== form.newPassword) {
    return callback(new Error('兩次輸入的新密碼不相符'))
  }
  callback()
}

const rules = {
  oldPassword: [{ required: true, message: '請輸入目前密碼', trigger: 'blur' }],
  newPassword: [
    { required: true, message: '請輸入新密碼', trigger: 'blur' },
    { min: 6, message: '密碼長度至少需為 6 個字元', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

function open() {
  form.oldPassword = ''
  form.newPassword = ''
  form.confirmPassword = ''
  visible.value = true
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    loading.value = true
    try {
      if (supabase) {
        const { error } = await supabase.auth.updateUser({
          password: form.newPassword
        })
        if (error) {
          throw new Error(error.message)
        }
      }
      // 呼叫 API 記錄密碼變更留痕與 Mock 支援
      await changeSelfPassword({
        oldPassword: form.oldPassword,
        newPassword: form.newPassword
      })

      ElMessage.success('密碼修改成功，請妥善保管新密碼')
      visible.value = false
    } catch (err: any) {
      ElMessage.error(err.message || '密碼修改失敗，請檢查目前密碼是否正確')
    } finally {
      loading.value = false
    }
  })
}

defineExpose({
  open
})
</script>

<style scoped>
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
