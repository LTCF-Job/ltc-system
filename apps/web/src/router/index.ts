import { createRouter, createWebHistory } from 'vue-router'
import DefaultLayout from '@/layouts/DefaultLayout.vue'
import { setupRouterGuards } from './guards'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/auth/LoginView.vue'),
      meta: { title: '登入系統', public: true }
    },
    {
      path: '/',
      component: DefaultLayout,
      children: [
        {
          path: '',
          name: 'Dashboard',
          component: () => import('@/views/dashboard/DashboardView.vue'),
          meta: { title: '總覽儀表板', roles: ['admin', 'staff', 'viewer'] }
        },
        {
          path: 'cases',
          name: 'CaseList',
          component: () => import('@/views/cases/CaseListView.vue'),
          meta: { title: '個案管理', roles: ['admin', 'staff', 'viewer'] }
        },
        {
          path: 'cases/:id',
          name: 'CaseDetail',
          component: () => import('@/views/cases/CaseDetailView.vue'),
          meta: { title: '個案明細與排班', roles: ['admin', 'staff', 'viewer'] }
        },
        {
          path: 'masters/sites',
          name: 'SiteList',
          component: () => import('@/views/masters/SiteListView.vue'),
          meta: { title: '據點管理', roles: ['admin', 'staff', 'viewer'] }
        },
        {
          path: 'masters/vehicles',
          name: 'VehicleList',
          component: () => import('@/views/masters/VehicleListView.vue'),
          meta: { title: '車輛管理', roles: ['admin', 'staff', 'viewer'] }
        },
        {
          path: 'masters/drivers',
          name: 'DriverList',
          component: () => import('@/views/masters/DriverListView.vue'),
          meta: { title: '司機管理', roles: ['admin', 'staff', 'viewer'] }
        },
        {
          path: 'forms',
          name: 'FormList',
          component: () => import('@/views/forms/FormListView.vue'),
          meta: { title: '表單同步管理', roles: ['admin', 'staff', 'viewer'] }
        },
        {
          path: 'forms/mappings',
          name: 'FieldMapping',
          component: () => import('@/views/forms/FieldMappingView.vue'),
          meta: { title: '欄位對應設定', roles: ['admin', 'staff', 'viewer'] }
        },
        {
          path: 'rides',
          name: 'RideCalendar',
          component: () => import('@/views/rides/RideCalendarView.vue'),
          meta: { title: '搭乘月曆矩陣', roles: ['admin', 'staff', 'viewer'] }
        },
        {
          path: 'rides/issues',
          name: 'RideIssues',
          component: () => import('@/views/rides/RideIssuesView.vue'),
          meta: { title: '異常集中處理', roles: ['admin', 'staff', 'viewer'] }
        },
        {
          path: 'rides/missing',
          name: 'MissingRides',
          component: () => import('@/views/rides/MissingRidesView.vue'),
          meta: { title: '未回報清單與催報歷史', roles: ['admin', 'staff', 'viewer'] }
        },
        {
          path: 'reports/trip-summary',
          name: 'TripSummary',
          component: () => import('@/views/reports/TripSummaryView.vue'),
          meta: { title: '車輛趟數表', roles: ['admin', 'staff', 'viewer'] }
        },
        {
          path: 'audit',
          name: 'AuditLog',
          component: () => import('@/views/audit/AuditLogView.vue'),
          meta: { title: '系統稽核紀錄', roles: ['admin'] }
        },
        {
          path: 'settings/notifications',
          name: 'NotificationSettings',
          component: () => import('@/views/settings/NotificationSettingsView.vue'),
          meta: { title: '通知收件人管理', roles: ['admin', 'staff', 'viewer'] }
        },
        {
          path: 'exports',
          name: 'GovExport',
          component: () => import('@/views/exports/ExportView.vue'),
          meta: { title: '政府申報匯出', roles: ['admin', 'staff'] }
        }
      ]
    },

    {
      path: '/:pathMatch(.*)*',
      redirect: '/'
    }
  ]
})

setupRouterGuards(router)

export default router
