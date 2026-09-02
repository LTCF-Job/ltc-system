<template>
  <div class="audit-log-view">
    <DataTablePage
      title="系統操作紀錄"
      :loading="loading"
      :total="total"
      :page="page"
      :page-size="pageSize"
      @page-change="onPageChange"
      @size-change="onSizeChange"
    >
      <template #filter>
        <el-input
          v-model="queryKeyword"
          placeholder="搜尋操作者／實體編號／動作"
          clearable
          style="width: 240px;"
          @keyup.enter="fetchAuditLogs"
        />

        <el-select
          v-model="queryAction"
          placeholder="動作類型"
          clearable
          style="width: 150px;"
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
          style="width: 150px;"
          @change="fetchAuditLogs"
        >
          <el-option
            v-for="(label, key) in AUDIT_ENTITY_LABELS"
            :key="key"
            :label="label"
            :value="key"
          />
        </el-select>

        <el-button type="primary" @click="fetchAuditLogs">
          查詢
        </el-button>
        <el-button @click="handleReset">
          重設
        </el-button>
      </template>

      <template #table>
        <el-table :data="auditList" stripe border table-layout="auto">
          <el-table-column
            prop="createdAt"
            label="操作時間"
            min-width="170"
            sortable
            align="center"
            class-name="op-time-col"
          >
            <template #default="{ row }">
              <span>{{ formatDateTime(row.createdAt) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作者" min-width="110" align="center" class-name="op-actor-col">
            <template #default="{ row }">
              <span>{{ getActorDisplayName(row) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="動作" width="150" align="center">
            <template #default="{ row }">
              <StatusTag :status="(row as any).action" preset="auditAction" variant="chip" />
            </template>
          </el-table-column>
          <el-table-column label="實體種類" min-width="110" align="center" class-name="entity-type-col">
            <template #default="{ row }">
              <span>{{ (AUDIT_ENTITY_LABELS as any)[(row as any).entityType] || (row as any).entityType }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="entityName" label="操作對象" class-name="entity-col">
            <template #default="{ row }">
              <span :title="(row as any).entityId ? `實體編號：${(row as any).entityId}` : undefined">
                {{ getEntityDisplayName(row as any) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="ipAddress" label="IP 位址" width="120" align="center" />
          <el-table-column label="操作" width="120" fixed="right" align="center">
            <template #default="{ row }">
              <TableRowActions v-if="(row as any).beforeData || (row as any).afterData">
                <el-button link type="primary" size="small" @click="openDetail(row as any)">
                  異動前後
                </el-button>
              </TableRowActions>
              <span v-else class="text-muted">-</span>
            </template>
          </el-table-column>
        </el-table>
      </template>
    </DataTablePage>

    <!-- 異動前後比較對話框 -->
    <el-dialog
      v-model="detailVisible"
      title="系統操作紀錄異動詳情"
      width="min(820px, calc(100vw - 32px))"
      destroy-on-close
    >
      <div v-if="selectedLog" class="dialog-content">
        <el-descriptions :column="2" border style="margin-bottom: 16px;">
          <el-descriptions-item label="操作時間">{{ formatDateTime(selectedLog.createdAt) }}</el-descriptions-item>
          <el-descriptions-item label="操作者">
            {{ getActorDisplayName(selectedLog) }}
          </el-descriptions-item>
          <el-descriptions-item label="動作類型">
            <span :class="getActionClass(selectedLog.action)">
              {{ AUDIT_ACTION_LABELS[selectedLog.action] || selectedLog.action }}
            </span>
            <span class="entity-badge">({{ getEntityDisplayName(selectedLog) }})</span>
          </el-descriptions-item>
          <el-descriptions-item label="來源 IP" :span="2">{{ selectedLog.ipAddress || '未知' }}</el-descriptions-item>
        </el-descriptions>

        <!-- 結構化中文欄位對照比較表 -->
        <div class="diff-table-container">
          <div class="diff-header">
            <span class="diff-title">
              <el-icon><DocumentCopy /></el-icon>
              區塊與欄位異動對照
            </span>
          </div>

          <el-table
            :data="computedDiffList"
            border
            stripe
            size="small"
            style="width: 100%;"
            max-height="400"
          >
            <el-table-column label="所屬區塊" width="130" align="center">
              <template #default="{ row }">
                <span>{{ row.section }}</span>
              </template>
            </el-table-column>
            <el-table-column label="欄位名稱" min-width="150">
              <template #default="{ row }">
                <span class="field-label">{{ row.label }}</span>
              </template>
            </el-table-column>
            <el-table-column label="異動前" min-width="190">
              <template #default="{ row }">
                <span :class="['diff-val', row.status === 'deleted' || row.status === 'modified' ? 'diff-old' : '']">
                  {{ row.beforeText }}
                </span>
              </template>
            </el-table-column>
            <el-table-column label="異動後" min-width="190">
              <template #default="{ row }">
                <span :class="['diff-val', row.status === 'created' || row.status === 'modified' ? 'diff-new' : '']">
                  {{ row.afterText }}
                </span>
              </template>
            </el-table-column>
            <el-table-column label="狀態" width="90" align="center">
              <template #default="{ row }">
                <span>{{ row.statusText }}</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">關閉</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import DataTablePage from '@/components/DataTablePage.vue'
import TableRowActions from '@/components/TableRowActions.vue'
import StatusTag from '@/components/StatusTag.vue'
import { listAuditLogs } from '@/api/audit'
import { formatDateTime } from '@/utils/formatters'
import type { AuditLogDTO } from '@/types/api'
import {
  AUDIT_ACTION_LABELS,
  AUDIT_ENTITY_LABELS,
  AUDIT_FIELD_LABELS,
  AUDIT_FIELD_SECTIONS,
  AUDIT_VALUE_LABELS,
  ROLE_LABELS,
  REGION_LABELS,
  SERVICE_CATEGORY_LABELS,
  SERVICE_USAGE_TYPE_LABELS,
  NOTIFICATION_TOPIC_LABELS,
  SYSTEM_MODULES,
  type AuditAction,
  type AuditEntityType
} from '@/types/domain'
import { DocumentCopy } from '@element-plus/icons-vue'

const auditList = ref<AuditLogDTO[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const queryKeyword = ref('')
const queryAction = ref<AuditAction | undefined>(undefined)
const queryEntityType = ref<AuditEntityType | undefined>(undefined)

const detailVisible = ref(false)
const selectedLog = ref<AuditLogDTO | null>(null)

// 依 CRUD 類型標示文字顏色
function getActionClass(action: AuditAction): string {
  switch (action) {
    case 'create':
    case 'import':
    case 'manual_report':
      return 'crud-create'
    case 'update':
    case 'correct':
    case 'resolve_conflict':
    case 'setting_change':
      return 'crud-update'
    case 'delete':
    case 'reveal_pii':
      return 'crud-delete'
    case 'login':
    case 'logout':
    case 'export':
    default:
      return 'crud-read'
  }
}

function getActorDisplayName(row?: { actorRole?: string; actorName?: string } | null): string {
  if (!row) return '系統'
  if (row.actorRole && (ROLE_LABELS as any)[row.actorRole]) {
    return (ROLE_LABELS as any)[row.actorRole]
  }
  return row.actorName || '系統'
}

// 智慧解析操作對象為使用者面向看得懂的親切中文標籤
function getEntityDisplayName(row?: AuditLogDTO | null): string {
  if (!row) return '-'

  // 若 entityName 存在且非純 UUID / 非 entityId 原始字串，優先採用
  const isUuid = (str?: string) =>
    !!str && /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/.test(str)

  if (row.entityName && !isUuid(row.entityName) && row.entityName !== row.entityId) {
    return row.entityName
  }

  // 從 afterData / beforeData 中萃取可讀名稱與業務識別
  const data: Record<string, any> = { ...(row.beforeData || {}), ...(row.afterData || {}) }

  switch (row.entityType) {
    case 'ride_records': {
      const caseName = data.caseName || data.case_name || (data.caseId ? '搭乘個案' : '')
      const date = data.serviceDate || data.service_date || ''
      const leg = data.legSeq || data.leg_seq
      let legText = ''
      if (leg === 1) legText = '去程'
      else if (leg === 2) legText = '回程'
      else if (leg) legText = `第 ${leg} 趟`

      const tripInfo = [date, legText].filter(Boolean).join(' ')

      if (caseName && tripInfo) {
        return `${caseName} (${tripInfo})`
      }
      if (caseName) {
        return `${caseName} (搭乘紀錄)`
      }
      if (tripInfo) {
        return `搭乘紀錄 (${tripInfo})`
      }
      if (data.reason || data.correctionReason || data.correction_reason) {
        return `搭乘紀錄 (${data.reason || data.correctionReason || data.correction_reason})`
      }
      return '搭乘紀錄'
    }

    case 'cases': {
      const name = data.name || data.caseName || data.case_name
      const code = data.code || data.caseCode || data.case_code
      if (name) return `${name}${code ? ` (${code})` : ''}`
      return '個案主檔'
    }

    case 'vehicles': {
      const plate = data.plateNo || data.plate_no
      const name = data.displayName || data.display_name || data.brandModel || data.brand_model
      if (name || plate) return `${name || '車輛'}${plate ? ` (${plate})` : ''}`
      return '車輛主檔'
    }

    case 'drivers': {
      const name = data.name || data.driverName || data.driver_name
      if (name) return `${name} (司機)`
      return '司機主檔'
    }

    case 'sites': {
      const name = data.name || data.siteName || data.site_name
      if (name) return `${name} (據點)`
      return '據點主檔'
    }

    case 'regions': {
      const name = data.name || (data.region && REGION_LABELS[data.region as keyof typeof REGION_LABELS])
      if (name) return `${name} (地區)`
      return '地區主檔'
    }

    case 'attendance_records': {
      const driver = data.driverName || data.driver_name
      const date = data.date || data.attendanceDate || data.serviceDate || ''
      return `${driver || '司機'}${date ? ` (${date} 出勤紀錄)` : ' 出勤紀錄'}`
    }

    case 'fuel_logs': {
      const veh = data.vehicleName || data.plateNo || data.plate_no
      const date = (data.fuelDate || data.fuel_date || '').slice(0, 10)
      return `${veh || '車輛'}${date ? ` (${date} 加油紀錄)` : ' 加油紀錄'}`
    }

    case 'maintenance_logs': {
      const veh = data.vehicleName || data.plateNo || data.plate_no
      const date = (data.serviceDate || data.maintenanceDate || '').slice(0, 10)
      return `${veh || '車輛'}${date ? ` (${date} 保養紀錄)` : ' 保養紀錄'}`
    }

    case 'holiday':
    case 'holiday_calendar': {
      const name = data.name || (data.year ? `${data.year} 年行事曆` : '')
      const date = data.holidayDate || data.holiday_date
      if (name && date) return `${name} (${date})`
      if (name || date) return `${name || date} (行事曆)`
      return '行政行事曆'
    }

    case 'users': {
      const name = data.displayName || data.display_name || data.name
      const email = data.email
      if (name) return `${name}${email ? ` (${email})` : ''}`
      if (email) return `使用者 (${email})`
      return '使用者帳號'
    }

    case 'roles': {
      const name = data.name || (data.role && (ROLE_LABELS as any)[data.role])
      return `${name || '角色身分'}`
    }

    case 'notification_recipients':
    case 'notification_recipient': {
      const name = data.displayName || data.display_name
      const email = data.email
      const topic = data.topic && (NOTIFICATION_TOPIC_LABELS as any)[data.topic]
      if (name && topic) return `${name} (${topic})`
      if (email && topic) return `${email} (${topic})`
      if (name || email) return `${name || email} (通知收件人)`
      return '通知收件人'
    }

    case 'export_jobs': {
      const ym = data.periodYm || data.period_ym
      const reg = data.region ? (REGION_LABELS[data.region as keyof typeof REGION_LABELS] || data.region) : '全區'
      return `${ym || ''} ${reg} 申報匯出`
    }

    case 'google_forms': {
      const title = data.title || data.formId || data.sheetTab
      return `${title || 'Google 表單'}`
    }

    case 'app_settings':
      return '系統全域設定'

    case 'auth': {
      const user = data.email || data.name || data.actorName
      return `${user ? `${user} (登入)` : '登入驗證'}`
    }

    default:
      break
  }

  // 若均無符合且 entityId 存在時，以實體種類中文加上簡短 ID 呈現
  const typeLabel = (AUDIT_ENTITY_LABELS as any)[row.entityType] || '系統資料'
  if (row.entityId) {
    if (isUuid(row.entityId)) {
      return `${typeLabel} (#${row.entityId.slice(0, 8)})`
    }
    return `${typeLabel} (${row.entityId})`
  }
  return typeLabel
}

// 格式化欄位數值為繁體中文親切文字
function formatFieldValue(val: any, key?: string): string {
  if (val === undefined || val === null) return '（無）'

  // 布林值處理
  if (typeof val === 'boolean') {
    if (key === 'active') return val ? '啟用' : '停用'
    if (key === 'hasConflict' || key === 'has_conflict') return val ? '有衝突' : '無衝突'
    return val ? '是' : '否'
  }

  // 依欄位 key 進行特定列舉或字串對照
  if (typeof val === 'string' || typeof val === 'number') {
    const strVal = String(val)

    if (key === 'role' || key === 'actorRole') {
      return (ROLE_LABELS as any)[strVal] || strVal
    }
    if (key === 'region') {
      return (REGION_LABELS as any)[strVal] || strVal
    }
    if (key === 'serviceCategory' || key === 'service_category') {
      return (SERVICE_CATEGORY_LABELS as any)[Number(strVal)] ? `類別 ${strVal} (${(SERVICE_CATEGORY_LABELS as any)[Number(strVal)]})` : strVal
    }
    if (key === 'serviceUsageType' || key === 'service_usage_type') {
      return (SERVICE_USAGE_TYPE_LABELS as any)[Number(strVal)] || strVal
    }
    if (key === 'topic') {
      return (NOTIFICATION_TOPIC_LABELS as any)[strVal] || strVal
    }

    // 通用數值常數中文化字典對照
    if (AUDIT_VALUE_LABELS[strVal]) {
      return AUDIT_VALUE_LABELS[strVal]
    }

    // 地區對照
    if (REGION_LABELS[strVal]) {
      return REGION_LABELS[strVal]
    }

    // 角色對照
    if ((ROLE_LABELS as any)[strVal]) {
      return (ROLE_LABELS as any)[strVal]
    }

    return strVal
  }

  // 陣列與物件結構化處理
  if (typeof val === 'object') {
    if (Array.isArray(val)) {
      if (val.length === 0) return '（空清單）'
      return val.map((item) => formatFieldValue(item, key)).join('、')
    }

    // 自訂模組權限物件呈現
    if (key === 'customPermissions' || key === 'custom_permissions') {
      const entries = Object.entries(val)
      const readable = entries
        .filter(([_, p]: any) => p && (p.view || p.edit))
        .map(([modId, p]: any) => {
          const mod = SYSTEM_MODULES.find((m) => m.id === modId)
          const modName = mod ? mod.name : modId
          const permText = p.edit ? '可編輯' : '僅檢視'
          return `${modName}(${permText})`
        })
      return readable.length > 0 ? readable.join('、') : '（未啟用任何模組權限）'
    }

    // 一般物件結構化為「欄位: 數值」
    const pairs = Object.entries(val).map(([subKey, subVal]) => {
      const subLabel = AUDIT_FIELD_LABELS[subKey] || subKey
      return `${subLabel}: ${formatFieldValue(subVal, subKey)}`
    })
    return pairs.length > 0 ? pairs.join('；') : '（無詳細內容）'
  }

  return String(val)
}

// 計算異動前後欄位比對清單
interface DiffRowItem {
  key: string
  section: string
  label: string
  beforeText: string
  afterText: string
  status: 'created' | 'modified' | 'deleted' | 'unchanged'
  statusText: string
  tagType: 'success' | 'warning' | 'danger' | 'info'
}

const computedDiffList = computed<DiffRowItem[]>(() => {
  if (!selectedLog.value) return []
  const before = selectedLog.value.beforeData || {}
  const after = selectedLog.value.afterData || {}

  const allKeys = Array.from(new Set([...Object.keys(before), ...Object.keys(after)]))
  const results: DiffRowItem[] = []

  for (const k of allKeys) {
    const hasBefore = Object.prototype.hasOwnProperty.call(before, k)
    const hasAfter = Object.prototype.hasOwnProperty.call(after, k)
    const beforeVal = before[k]
    const afterVal = after[k]

    let status: 'created' | 'modified' | 'deleted' | 'unchanged' = 'unchanged'
    let statusText = '未變更'
    let tagType: 'success' | 'warning' | 'danger' | 'info' = 'info'

    if (!hasBefore && hasAfter) {
      status = 'created'
      statusText = '新增'
      tagType = 'success'
    } else if (hasBefore && !hasAfter) {
      status = 'deleted'
      statusText = '刪除'
      tagType = 'danger'
    } else if (JSON.stringify(beforeVal) !== JSON.stringify(afterVal)) {
      status = 'modified'
      statusText = '已修改'
      tagType = 'warning'
    }

    const section = AUDIT_FIELD_SECTIONS[k] || '基本資料'
    const label = AUDIT_FIELD_LABELS[k] || k

    results.push({
      key: k,
      section,
      label,
      beforeText: hasBefore ? formatFieldValue(beforeVal, k) : '（未設定）',
      afterText: hasAfter ? formatFieldValue(afterVal, k) : '（已刪除）',
      status,
      statusText,
      tagType
    })
  }

  // 將有異動的欄位排在最前面
  return results.sort((a, b) => {
    const scoreA = a.status === 'modified' ? 3 : a.status === 'created' ? 2 : a.status === 'deleted' ? 1 : 0
    const scoreB = b.status === 'modified' ? 3 : b.status === 'created' ? 2 : b.status === 'deleted' ? 1 : 0
    return scoreB - scoreA
  })
})

async function fetchAuditLogs() {
  loading.value = true
  try {
    const res = await listAuditLogs({
      page: page.value,
      pageSize: pageSize.value,
      action: queryAction.value,
      entityType: queryEntityType.value,
      q: queryKeyword.value || undefined
    })
    auditList.value = res.data
    total.value = res.meta?.total || res.data.length
  } finally {
    loading.value = false
  }
}

function handleReset() {
  queryKeyword.value = ''
  queryAction.value = undefined
  queryEntityType.value = undefined
  page.value = 1
  fetchAuditLogs()
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

/* Element Plus 的 .el-table 是 div，內建 width: 100%；改成 width: auto 沒用，
   因為 div 預設就是撐滿容器寬度，不會像原生表格一樣自動縮到內容寬度。
   改用 max-content 才能讓外層容器真的縮到跟各欄實際寬度加總一致，
   搭配 table-layout="auto" 讓「操作對象」欄依內容自然伸縮、不吃掉多餘空間。
   欄位加總寬度若超過版面，交給 DataTablePage 既有的 overflow-x: auto 處理水平捲動。 */
.audit-log-view :deep(.el-table) {
  width: max-content;
}

/* 「操作對象」欄不套 show-overflow-tooltip：那個 prop 底層是 overflow: hidden，
   會讓 table-layout="auto" 在量測「這欄實際需要多寬」時把這欄當成 0 寬（反正會被裁掉），
   結果欄寬永遠縮到最小、跟內容脫鉤（連頁面還有空間時也一樣，等於本來想要的「有空間就展開」失效）。
   改成只鎖 white-space: nowrap（不換行），不加 overflow: hidden，
   讓瀏覽器照實際文字寬度分配欄寬；真的超出版面時交給外層既有的 overflow-x: auto 水平捲動，不做裁切省略。 */
/* 其餘欄位（操作時間／操作者／實體種類）同理：改用 min-width 搭配 nowrap，
   讓欄寬依內容自然撐開，不被固定 width 逼換行。 */
.audit-log-view :deep(.entity-col .cell),
.audit-log-view :deep(.entity-type-col .cell),
.audit-log-view :deep(.op-time-col .cell),
.audit-log-view :deep(.op-actor-col .cell) {
  white-space: nowrap;
}

.entity-badge {
  font-weight: normal;
  color: var(--app-text-secondary);
  margin-left: 4px;
}

/* CRUD 文字顏色標示，純文字無額外外框 */
.crud-create {
  color: var(--app-status-success-fg);
  font-weight: 600;
}

.crud-update {
  color: #d97706; /* amber */
  font-weight: 600;
}

.crud-delete {
  color: var(--app-status-danger-fg);
  font-weight: 600;
}

.crud-read {
  color: var(--app-primary);
  font-weight: 600;
}

.dialog-content {
  display: flex;
  flex-direction: column;
}

.diff-table-container {
  margin-top: 8px;

  .diff-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;

    .diff-title {
      font-size: 14px;
      font-weight: bold;
      color: var(--app-text-primary);
      display: flex;
      align-items: center;
      gap: 6px;
    }
  }
}

.field-label {
  font-weight: 600;
  color: var(--app-text-primary);
}

.diff-val {
  font-family: inherit;
  font-size: 13px;
  word-break: break-all;
}

.diff-old {
  color: var(--app-status-danger-fg);
  border-left: 2px solid var(--app-status-danger-fg);
  padding-left: 6px;
}

.diff-new {
  color: var(--app-status-success-fg);
  border-left: 2px solid var(--app-status-success-fg);
  font-weight: 600;
  padding-left: 6px;
}
</style>
