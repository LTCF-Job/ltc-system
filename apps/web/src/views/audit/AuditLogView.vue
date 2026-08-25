<template>
  <div class="audit-log-view">
    <DataTablePage
      :loading="loading"
      :total="total"
      :page="page"
      :page-size="pageSize"
      @page-change="onPageChange"
      @size-change="onSizeChange"
    >
      <template #filter>
        <el-select
          v-model="queryAction"
          placeholder="動作類型"
          clearable
          style="width: 170px;"
          @change="fetchAuditLogs"
        >
          <el-option
            v-for="(label, key) in AUDIT_ACTION_LABELS"
            :key="key"
            :label="label"
            :value="key"
          />
        </el-select>

        <el-select
          v-model="queryEntityType"
          placeholder="異動實體"
          clearable
          style="width: 170px;"
          @change="fetchAuditLogs"
        >
          <el-option
            v-for="(label, key) in AUDIT_ENTITY_LABELS"
            :key="key"
            :label="label"
            :value="key"
          />
        </el-select>

        <el-button type="primary" icon="Search" @click="fetchAuditLogs">
          查詢
        </el-button>
      </template>

      <template #table>
        <el-table :data="auditList" stripe border style="width: 100%;">
          <el-table-column prop="createdAt" label="操作時間" width="170" sortable />
          <el-table-column label="操作者" width="160">
            <template #default="{ row }">
              <span>{{ (row as any).actorName || '系統' }}</span>
              <el-tag
                v-if="(row as any).actorRole"
                size="small"
                :type="(row as any).actorRole === 'admin' ? 'danger' : 'info'"
                style="margin-left: 6px;"
              >
                {{ (ROLE_LABELS as any)[(row as any).actorRole] || (row as any).actorRole }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="動作" width="150">
            <template #default="{ row }">
              <el-tag :type="getActionTagType((row as any).action)">
                {{ (AUDIT_ACTION_LABELS as any)[(row as any).action] || (row as any).action }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="實體種類" width="140">
            <template #default="{ row }">
              <el-tag effect="plain" type="info">
                {{ (AUDIT_ENTITY_LABELS as any)[(row as any).entityType] || (row as any).entityType }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="entityName" label="操作對象" min-width="180" show-overflow-tooltip>
            <template #default="{ row }">
              <span>{{ (row as any).entityName || (row as any).entityId || '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="ipAddress" label="IP 位址" width="140" />
          <el-table-column label="操作" width="120" fixed="right" align="center">
            <template #default="{ row }">
              <el-button
                v-if="(row as any).beforeData || (row as any).afterData"
                type="primary"
                link
                icon="View"
                @click="openDetail(row as any)"
              >
                異動前後
              </el-button>
              <span v-else class="text-muted">-</span>
            </template>
          </el-table-column>

        </el-table>
      </template>
    </DataTablePage>

    <!-- 異動前後比較彈窗 -->
    <el-dialog
      v-model="detailVisible"
      title="稽核日誌異動詳情"
      width="720px"
      destroy-on-close
    >
      <div v-if="selectedLog" class="dialog-content">
        <el-descriptions :column="2" border style="margin-bottom: 16px;">
          <el-descriptions-item label="操作時間">{{ selectedLog.createdAt }}</el-descriptions-item>
          <el-descriptions-item label="操作人員">{{ selectedLog.actorName }} ({{ selectedLog.actorRole }})</el-descriptions-item>
          <el-descriptions-item label="動作類型">{{ AUDIT_ACTION_LABELS[selectedLog.action] }}</el-descriptions-item>
          <el-descriptions-item label="目標實體">{{ AUDIT_ENTITY_LABELS[selectedLog.entityType] }} ({{ selectedLog.entityName || selectedLog.entityId }})</el-descriptions-item>
          <el-descriptions-item label="來源 IP" :span="2">{{ selectedLog.ipAddress || '未知' }}</el-descriptions-item>
        </el-descriptions>

        <el-row :gutter="16">
          <el-col :span="12">
            <el-card shadow="never" class="json-card before-card">
              <template #header>
                <span class="card-title text-danger">異動前內容 (Before)</span>
              </template>
              <pre class="json-pre">{{ selectedLog.beforeData ? JSON.stringify(selectedLog.beforeData, null, 2) : '無（新增或初次建立）' }}</pre>
            </el-card>
          </el-col>
          <el-col :span="12">
            <el-card shadow="never" class="json-card after-card">
              <template #header>
                <span class="card-title text-success">異動後內容 (After)</span>
              </template>
              <pre class="json-pre">{{ selectedLog.afterData ? JSON.stringify(selectedLog.afterData, null, 2) : '無（已刪除）' }}</pre>
            </el-card>
          </el-col>
        </el-row>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">關閉</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import DataTablePage from '@/components/DataTablePage.vue'
import { listAuditLogs } from '@/api/audit'
import type { AuditLogDTO } from '@/types/api'
import {
  AUDIT_ACTION_LABELS,
  AUDIT_ENTITY_LABELS,
  ROLE_LABELS,
  type AuditAction,
  type AuditEntityType
} from '@/types/domain'

const auditList = ref<AuditLogDTO[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const queryAction = ref<AuditAction | undefined>(undefined)
const queryEntityType = ref<AuditEntityType | undefined>(undefined)

const detailVisible = ref(false)
const selectedLog = ref<AuditLogDTO | null>(null)

function getActionTagType(action: AuditAction): 'primary' | 'success' | 'warning' | 'info' | 'danger' {
  switch (action) {
    case 'create':
    case 'import':
      return 'success'
    case 'update':
    case 'correct':
    case 'resolve_conflict':
      return 'warning'
    case 'delete':
    case 'reveal_pii':
      return 'danger'
    case 'export':
      return 'primary'
    default:
      return 'info'
  }
}


async function fetchAuditLogs() {
  loading.value = true
  try {
    const res = await listAuditLogs({
      page: page.value,
      pageSize: pageSize.value,
      action: queryAction.value,
      entityType: queryEntityType.value
    })
    auditList.value = res.data
    total.value = res.meta?.total || res.data.length
  } finally {
    loading.value = false
  }
}

function onPageChange(p: number) {
  page.value = p
  fetchAuditLogs()
}

function onSizeChange(size: number) {
  pageSize.value = size
  page.value = 1
  fetchAuditLogs()
}

function openDetail(log: AuditLogDTO) {
  selectedLog.value = log
  detailVisible.value = true
}

onMounted(() => {
  fetchAuditLogs()
})
</script>

<style scoped>
.audit-log-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.text-muted {
  color: var(--el-text-color-placeholder);
}

.dialog-content {
  display: flex;
  flex-direction: column;
}

.json-card {
  border-radius: 6px;
  background-color: #fafafa;

  .card-title {
    font-size: 14px;
    font-weight: bold;
  }

  .text-danger {
    color: var(--el-color-danger);
  }

  .text-success {
    color: var(--el-color-success);
  }

  .json-pre {
    margin: 0;
    font-family: 'Consolas', 'Courier New', monospace;
    font-size: 12px;
    line-height: 1.5;
    max-height: 260px;
    overflow: auto;
    white-space: pre-wrap;
    word-break: break-all;
    color: #333;
  }
}
</style>
