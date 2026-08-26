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
          meta: { title: '總覽儀表板', module: 'dashboard', roles: ['admin', 'staff', 'dispatcher', 'driver', 'viewer'] }
        },
        {
          path: 'cases',
          name: 'CaseList',
          component: () => import('@/views/cases/CaseListView.vue'),
          meta: { title: '個案管理', module: 'masters_cases', roles: ['admin', 'staff', 'dispatcher', 'viewer'] }
        },
        {
          path: 'cases/:id',
          name: 'CaseDetail',
          component: () => import('@/views/cases/CaseDetailView.vue'),
          meta: { title: '個案編輯', module: 'masters_cases', roles: ['admin', 'staff', 'dispatcher', 'viewer'] }
        },
        {
          path: 'masters/regions',
          name: 'RegionList',
          component: () => import('@/views/masters/RegionListView.vue'),
          meta: { title: '地區管理', module: 'masters_regions', roles: ['admin', 'staff', 'dispatcher', 'viewer'] }
        },
        {
          path: 'masters/sites',
          name: 'SiteList',
          component: () => import('@/views/masters/SiteListView.vue'),
          meta: { title: '據點管理', module: 'masters_sites', roles: ['admin', 'staff', 'dispatcher', 'viewer'] }
        },
        {
          path: 'masters/vehicles',
          name: 'VehicleList',
          component: () => import('@/views/masters/VehicleListView.vue'),
          meta: { title: '車輛管理', module: 'masters_vehicles', roles: ['admin', 'staff', 'dispatcher', 'driver', 'viewer'] }
        },
        {
          path: 'masters/drivers',
          name: 'DriverList',
          component: () => import('@/views/masters/DriverListView.vue'),
          meta: { title: '司機管理', module: 'masters_drivers', roles: ['admin', 'staff', 'dispatcher', 'driver', 'viewer'] }
        },
        {
          path: 'forms',
          name: 'FormList',
          component: () => import('@/views/forms/FormListView.vue'),
          meta: { title: '表單同步管理', module: 'forms_sync', roles: ['admin', 'staff', 'dispatcher', 'viewer'] }
        },
        {
          path: 'forms/mappings',
          name: 'FieldMapping',
          component: () => import('@/views/forms/FieldMappingView.vue'),
          meta: { title: '欄位對應設定', module: 'forms_mappings', roles: ['admin', 'staff', 'dispatcher', 'viewer'] }
        },
        {
          path: 'rides',
          name: 'RideCalendar',
          component: () => import('@/views/rides/RideCalendarView.vue'),
          meta: { title: '搭乘月曆矩陣', module: 'rides_calendar', roles: ['admin', 'staff', 'dispatcher', 'driver', 'viewer'] }
        },
        {
          path: 'rides/issues',
          name: 'RideIssues',
          component: () => import('@/views/rides/RideIssuesView.vue'),
          meta: { title: '異常集中處理', module: 'rides_issues', roles: ['admin', 'staff', 'dispatcher', 'viewer'] }
        },
        {
          path: 'rides/missing',
          name: 'MissingRides',
          component: () => import('@/views/rides/MissingRidesView.vue'),
          meta: { title: '未回報清單與催報歷史', module: 'rides_missing', roles: ['admin', 'staff', 'dispatcher', 'viewer'] }
        },
        {
          path: 'reports/trip-summary',
          name: 'TripSummary',
          component: () => import('@/views/reports/TripSummaryView.vue'),
          meta: { title: '車輛趟數表', module: 'reports_trip_summary', roles: ['admin', 'staff', 'dispatcher', 'viewer'] }
        },
        {
          path: 'reports/hsinchu-schedule',
          name: 'HsinchuSchedule',
          component: () => import('@/views/reports/HsinchuScheduleView.vue'),
          meta: { title: '新竹接送時刻表', module: 'reports_hsinchu_schedule', roles: ['admin', 'staff', 'dispatcher', 'viewer'] }
        },
        {
          path: 'vehicles/maintenance',
          name: 'VehicleMaintenance',
          component: () => import('@/views/vehicles/MaintenanceView.vue'),
          meta: { title: '車輛維修保養', module: 'vehicles_maintenance', roles: ['admin', 'staff', 'dispatcher', 'driver', 'viewer'] }
        },
        {
          path: 'attendance',
          name: 'AttendanceFuel',
          component: () => import('@/views/attendance/AttendanceFuelView.vue'),
          meta: { title: '出勤與油資登錄', module: 'attendance_fuel', roles: ['admin', 'staff', 'dispatcher', 'driver', 'viewer'] }
        },
        {
          path: 'audit',
          name: 'AuditLog',
          component: () => import('@/views/audit/AuditLogView.vue'),
          meta: { title: '系統操作紀錄', module: 'audit_logs', roles: ['admin'] }
        },
        {
          path: 'settings/users',
          name: 'UserManagement',
          component: () => import('@/views/settings/UserManagementView.vue'),
          meta: { title: '使用者管理', module: 'settings_users', roles: ['admin'] }
        },
        {
          path: 'settings/roles',
          name: 'RoleManagement',
          component: () => import('@/views/settings/RoleManagementView.vue'),
          meta: { title: '角色身分管理', module: 'settings_roles', roles: ['admin'] }
        },
        {
          path: 'settings/notifications',
          name: 'NotificationSettings',
          component: () => import('@/views/settings/NotificationSettingsView.vue'),
          meta: { title: '通知收件人管理', module: 'settings_notifications', roles: ['admin', 'staff', 'dispatcher', 'viewer'] }
        },
        {
          path: 'exports',
          name: 'GovExport',
          component: () => import('@/views/exports/ExportView.vue'),
          meta: { title: '政府申報匯出', module: 'exports', roles: ['admin', 'staff', 'dispatcher'] }
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
