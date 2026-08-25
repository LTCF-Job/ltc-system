<template>
  <el-dialog
    v-model="visible"
    :title="title"
    width="900px"
    destroy-on-close
    :before-close="handleClose"
  >
    <!-- 第一步：上傳檔案與檢核 -->
    <div v-if="!dryRunResult" class="upload-section">
      <el-upload
        drag
        action="#"
        :auto-upload="false"
        :limit="1"
        :on-change="handleFileChange"
        accept=".xlsx,.xls"
      >
        <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
        <div class="el-upload__text">
          拖曳 Excel 檔案至此，或 <em>點擊上傳</em>
        </div>
        <template #tip>
          <div class="el-upload__tip">
            僅支援 .xlsx 與 .xls 格式之批次匯入檔案
          </div>
        </template>
      </el-upload>

      <div class="dialog-footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button
          type="primary"
          :disabled="!selectedFile"
          :loading="analyzing"
          @click="startDryRun"
        >
          開始解析與預覽
        </el-button>
      </div>
    </div>

    <!-- 第二步：預覽檢核結果與確認寫入 -->
    <div v-else class="preview-section">
      <!-- 統計摘要與提示 -->
      <div class="stats-bar">
        <el-tag type="info">總筆數：{{ dryRunResult.totalRows }}</el-tag>
        <el-tag type="success">合法筆數：{{ dryRunResult.validRows }}</el-tag>
        <el-tag v-if="dryRunResult.errorRows > 0" type="danger">
          錯誤筆數：{{ dryRunResult.errorRows }}
        </el-tag>
        <el-tag v-if="dryRunResult.warningRows > 0" type="warning">
          警告/預設值提醒：{{ dryRunResult.warningRows }}
        </el-tag>
      </div>

      <el-alert
        v-if="dryRunResult.errorRows > 0"
        type="error"
        show-icon
        :closable="false"
        title="檔案中含有格式錯誤或缺漏必填欄位，錯誤列無法寫入，請修正後重新上傳。"
        style="margin-bottom: 12px;"
      />

      <el-alert
        v-else-if="dryRunResult.warningRows > 0"
        type="warning"
        show-icon
        :closable="false"
        title="部分資料將套用系統預設值或需人工確認，請核對無誤後再執行正式匯入。"
        style="margin-bottom: 12px;"
      />

      <!-- 預覽資料表 -->
      <el-table
        :data="dryRunResult.previewRows"
        max-height="400"
        border
        size="small"
        :row-class-name="getRowClassName"
      >
        <el-table-column type="index" label="#" width="50" />
        <slot name="columns" />
      </el-table>

      <!-- 錯誤明細展開清單 -->
      <div v-if="dryRunResult.errors.length > 0" class="error-list">
        <h4>錯誤明細：</h4>
        <ul>
          <li v-for="(err, idx) in dryRunResult.errors" :key="idx">
            第 {{ err.rowIndex }} 列 {{ err.caseName ? `(${err.caseName})` : '' }}: {{ err.message }}
          </li>
        </ul>
      </div>

      <div class="dialog-footer">
        <el-button @click="resetToUpload">重新選擇檔案</el-button>
        <el-button
          type="primary"
          :disabled="dryRunResult.errorRows > 0 || dryRunResult.validRows === 0"
          :loading="submitting"
          @click="confirmImport"
        >
          確認寫入 ({{ dryRunResult.validRows }} 筆)
        </el-button>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { UploadFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { DryRunImportResultDTO } from '@/types/api'

const props = defineProps<{
  title: string
  onDryRun: (file: File) => Promise<DryRunImportResultDTO>
  onCommit: (file: File) => Promise<void>
}>()

const emit = defineEmits<{
  (e: 'success'): void
}>()

const visible = ref(false)
const selectedFile = ref<File | null>(null)
const analyzing = ref(false)
const submitting = ref(false)
const dryRunResult = ref<DryRunImportResultDTO | null>(null)

function open() {
  selectedFile.value = null
  dryRunResult.value = null
  visible.value = true
}

function handleFileChange(uploadFile: any) {
  selectedFile.value = uploadFile.raw
}

async function startDryRun() {
  if (!selectedFile.value) return
  analyzing.value = true
  try {
    const res = await props.onDryRun(selectedFile.value)
    dryRunResult.value = res
  } finally {
    analyzing.value = false
  }
}

function resetToUpload() {
  dryRunResult.value = null
  selectedFile.value = null
}

function getRowClassName({ row }: { row: any }) {
  if (row.__hasError) return 'row-error'
  if (row.__hasWarning) return 'row-warning'
  return ''
}

async function confirmImport() {
  if (!selectedFile.value) return
  submitting.value = true
  try {
    await props.onCommit(selectedFile.value)
    ElMessage.success('批次匯入成功！')
    visible.value = false
    emit('success')
  } finally {
    submitting.value = false
  }
}

function handleClose(done: () => void) {
  resetToUpload()
  done()
}

defineExpose({
  open
})
</script>

<style scoped>
.upload-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 0;
}

.stats-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.error-list {
  margin-top: 12px;
  padding: 8px 12px;
  background-color: var(--el-color-danger-light-9);
  border-radius: 4px;
  font-size: 13px;
  color: var(--el-color-danger);

  h4 {
    margin-bottom: 4px;
  }

  ul {
    padding-left: 20px;
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 20px;
}

:deep(.row-error) {
  background-color: var(--el-color-danger-light-9) !important;
}

:deep(.row-warning) {
  background-color: var(--el-color-warning-light-9) !important;
}
</style>
