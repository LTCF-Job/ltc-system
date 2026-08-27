<template>
  <el-drawer
    v-model="visible"
    title="✨ LTC 長照交通接送系統 — Demo 展示控制中心"
    size="520px"
    direction="rtl"
    :destroy-on-close="false"
  >
    <div class="demo-drawer-content">
      <!-- 頂部提示卡片 -->
      <div class="demo-alert-box">
        <el-icon class="alert-icon"><InfoFilled /></el-icon>
        <div class="alert-text">
          <strong>全功能展示模式已啟用</strong>
          <p>此系統已載入完整業務資料集，涵蓋所有身分角色、機構類型、排班趟次、異常裁決與四類報表。</p>
        </div>
      </div>

      <!-- 1. 角色快速切換 -->
      <el-card shadow="never" class="section-card">
        <template #header>
          <div class="card-title">
            <el-icon><UserFilled /></el-icon>
            <span>切換使用者角色體驗權限</span>
          </div>
        </template>
        <div class="role-switch-grid">
          <div
            v-for="r in roles"
            :key="r.key"
            class="role-card"
            :class="{ active: authStore.currentRole === r.key }"
            @click="switchRole(r.key)"
          >
            <div class="role-header">
              <el-tag :type="r.tagType" size="small" effect="dark">{{ r.name }}</el-tag>
              <el-icon v-if="authStore.currentRole === r.key" class="check-icon"><Select /></el-icon>
            </div>
            <p class="role-desc">{{ r.desc }}</p>
          </div>
        </div>
      </el-card>

      <!-- 2. 全類型與選項功能展示導覽 -->
      <el-card shadow="never" class="section-card">
        <template #header>
          <div class="card-title">
            <el-icon><Guide /></el-icon>
            <span>功能模組與全選項場景導覽</span>
          </div>
        </template>

        <div class="module-list">
          <div
            v-for="item in demoModules"
            :key="item.path"
            class="module-item"
          >
            <div class="module-info">
              <div class="module-title-row">
                <el-icon :class="item.iconClass"><component :is="item.icon" /></el-icon>
                <span class="module-name">{{ item.title }}</span>
              </div>
              <p class="module-desc">{{ item.description }}</p>
              <div class="tags-row">
                <el-tag
                  v-for="tag in item.tags"
                  :key="tag"
                  size="small"
                  type="info"
                  class="feature-tag"
                >
                  {{ tag }}
                </el-tag>
              </div>
            </div>
            <el-button
              type="primary"
              size="small"
              plain
              class="go-btn"
              @click="navigate(item.path)"
            >
              前往
            </el-button>
          </div>
        </div>
      </el-card>

      <!-- 3. Demo 資料管理操作 -->
      <el-card shadow="never" class="section-card">
        <template #header>
          <div class="card-title">
            <el-icon><Refresh /></el-icon>
            <span>Demo 資料操作</span>
          </div>
        </template>
        <div class="action-row">
          <el-button type="warning" plain @click="handleResetData">
            <el-icon><RefreshRight /></el-icon>
            重新載入展示資料
          </el-button>
          <el-button type="info" plain @click="visible = false">
            關閉控制中心
          </el-button>
        </div>
      </el-card>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  InfoFilled,
  UserFilled,
  Select,
  Guide,
  Refresh,
  RefreshRight,
  Odometer,
  User,
  Calendar,
  Warning,
  Bell,
  DocumentCopy,
  Connection,
  DataAnalysis,
  Download,
  Management
} from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import type { UserRole } from '@/types/domain'

const visible = ref(false)
const router = useRouter()
const authStore = useAuthStore()

const roles: { key: UserRole; name: string; tagType: 'danger' | 'primary' | 'info'; desc: string }[] = [
  {
    key: 'admin',
    name: '系統管理者 (Admin)',
    tagType: 'danger',
    desc: '完整全系統存取權限，含稽核日誌與系統設定'
  },
  {
    key: 'staff',
    name: '行政人員 (Staff)',
    tagType: 'primary',
    desc: '日常營運排班、搭乘登錄、異常裁決與申報匯出'
  },
  {
    key: 'viewer',
    name: '主管檢視者 (Viewer)',
    tagType: 'info',
    desc: '全模組唯讀檢視權限，適合主管審閱與對帳'
  }
]

const demoModules = [
  {
    title: '總覽營運儀表板',
    path: '/',
    icon: Odometer,
    iconClass: 'color-primary',
    description: '當月搭乘趟次統計、車輛趟數排行、司機出勤率與快速待辦異常提醒。',
    tags: ['全月指標', '趟次趨勢', '出勤佔比']
  },
  {
    title: '個案管理與排班設定',
    path: '/cases',
    icon: User,
    iconClass: 'color-success',
    description: '包含 10+ 位個案，涵蓋苗栗/新竹、在案/暫停/停案、補助/自費、4種機構類型與單向/2趟/4趟排班。',
    tags: ['在案/暫停/停案', '補助/自費', '4類機構', '1/2/4趟排班', '身分證遮罩/解密']
  },
  {
    title: '搭乘月曆表',
    path: '/rides',
    icon: Calendar,
    iconClass: 'color-warning',
    description: '7 月完整 31 天矩陣，展示有坐 (√)、請假 (／)、未回報 (?)、混車衝突 (!) 與 5 種原因人工更正。',
    tags: ['整月矩陣', '有坐/沒坐/未回報', '跨車衝突', '更正軌跡', 'AA09旗標']
  },
  {
    title: '異常集中處理中心',
    path: '/rides/issues',
    icon: Warning,
    iconClass: 'color-danger',
    description: '跨車回報混車衝突（竹北一車與二車皆回報），線上裁決承載車輛並自動修正。',
    tags: ['跨車衝突裁決', '應搭未回報', '文字解析異常']
  },
  {
    title: '未回報催報與發送歷史',
    path: '/rides/missing',
    icon: Bell,
    iconClass: 'color-warning',
    description: '逾期未回報清單追蹤（1天~10天），支援手動即時觸發與定時自動催報通知。',
    tags: ['多天逾期', '一鍵催報', 'Email發送歷史']
  },
  {
    title: '車輛趟數統計表',
    path: '/reports/trip-summary',
    icon: DataAnalysis,
    iconClass: 'color-primary',
    description: '依車輛分組彙總各個案去回程與總趟數，含小計、總計與 Excel 匯出。',
    tags: ['車輛分組', '去程/回程小計', 'Excel匯出']
  },
  {
    title: '新竹接送時刻表',
    path: '/reports/hsinchu-schedule',
    icon: DocumentCopy,
    iconClass: 'color-success',
    description: '依去程梯次、回程梯次依序排列之司機接送路線表，含起迄地址與停靠時間。',
    tags: ['去回程班次', '站點路線', '接送時間']
  },
  {
    title: '車輛維修保養管理',
    path: '/vehicles/maintenance',
    icon: Management,
    iconClass: 'color-info',
    description: '定期五萬公里大保養、機油換新、輪胎更換、保養費用與協力廠商紀錄。',
    tags: ['保養紀錄', '里程與費用', '空白保養表匯出']
  },
  {
    title: '司機出勤與油資登錄',
    path: '/attendance',
    icon: Calendar,
    iconClass: 'color-warning',
    description: '涵蓋出勤、事假、病假、休假 4 種狀態與請假備註；油資登錄支援加油公升與發票金額。',
    tags: ['4種出勤狀態', '請假備註', '加油發票登記']
  },
  {
    title: 'Google 表單同步與智慧欄位',
    path: '/forms/mappings',
    icon: Connection,
    iconClass: 'color-primary',
    description: 'Google 表單智慧辨識（系統欄/搭乘欄/問題欄/未判定），支援高低信心度智慧推薦。',
    tags: ['4種欄位種類', '3種對應狀態', 'AI推薦分數']
  },
  {
    title: '政府申報匯出與預檢',
    path: '/exports',
    icon: Download,
    iconClass: 'color-danger',
    description: '4 種報表匯出任務（政府申報、趟數表、時刻表、保養表），具備三層級前置預檢。',
    tags: ['4種匯出類型', '4種任務狀態', 'Error/Warning/Info預檢']
  }
]

function open() {
  visible.value = true
}

function switchRole(role: UserRole) {
  authStore.setSession(`mock_jwt_${role}`, {
    id: `usr_${role}`,
    email: `${role}@ltc.example.com`,
    displayName: role === 'admin' ? '系統管理員' : (role === 'staff' ? '承辦行政' : '主管檢視者'),
    role
  })
  ElMessage.success(`已切換為【${role === 'admin' ? '系統管理員' : (role === 'staff' ? '行政人員' : '檢視者')}】身分`)
}

function navigate(path: string) {
  router.push(path)
  visible.value = false
}

function handleResetData() {
  ElMessage.success('已重新整理 Demo 完整展示資料集')
  window.location.reload()
}

defineExpose({
  open
})
</script>

<style scoped>
.demo-drawer-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.demo-alert-box {
  background: linear-gradient(135deg, #e6f4ff 0%, #f0f5ff 100%);
  border: 1px solid #91caff;
  border-radius: 8px;
  padding: 12px 16px;
  display: flex;
  gap: 12px;
  align-items: flex-start;

  .alert-icon {
    font-size: 22px;
    color: #1677ff;
    margin-top: 2px;
  }

  .alert-text {
    font-size: 13px;
    color: #1d39c4;

    strong {
      display: block;
      font-size: 14px;
      margin-bottom: 4px;
    }

    p {
      margin: 0;
      line-height: 1.5;
      color: #2f54eb;
    }
  }
}

.section-card {
  border-radius: 8px;

  :deep(.el-card__header) {
    padding: 12px 16px;
    background-color: #fafafa;
    border-bottom: 1px solid #f0f0f0;
  }

  :deep(.el-card__body) {
    padding: 14px;
  }
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: bold;
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.role-switch-grid {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.role-card {
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  padding: 10px 12px;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    border-color: var(--el-color-primary);
    background-color: var(--el-color-primary-light-9);
  }

  &.active {
    border-color: var(--el-color-primary);
    background-color: var(--el-color-primary-light-9);
    box-shadow: 0 0 0 1px var(--el-color-primary);
  }

  .role-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 4px;

    .check-icon {
      color: var(--el-color-primary);
      font-weight: bold;
    }
  }

  .role-desc {
    margin: 0;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
}

.module-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 400px;
  overflow-y: auto;
  padding-right: 4px;
}

.module-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  background-color: #ffffff;
  transition: all 0.2s ease;

  &:hover {
    border-color: var(--el-color-primary-light-5);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  }

  .module-info {
    flex: 1;
    margin-right: 12px;

    .module-title-row {
      display: flex;
      align-items: center;
      gap: 6px;
      margin-bottom: 4px;

      .module-name {
        font-size: 13px;
        font-weight: 600;
        color: var(--el-text-color-primary);
      }
    }

    .module-desc {
      margin: 0 0 6px 0;
      font-size: 12px;
      color: var(--el-text-color-secondary);
      line-height: 1.4;
    }

    .tags-row {
      display: flex;
      flex-wrap: wrap;
      gap: 4px;

      .feature-tag {
        font-size: 11px;
        height: 20px;
        padding: 0 6px;
      }
    }
  }

  .go-btn {
    flex-shrink: 0;
  }
}

.action-row {
  display: flex;
  justify-content: space-between;
}

.color-primary { color: var(--el-color-primary); }
.color-success { color: var(--el-color-success); }
.color-warning { color: var(--el-color-warning); }
.color-danger { color: var(--el-color-danger); }
.color-info { color: var(--el-color-info); }
</style>
