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
          meta: { title: '總覽儀表板', module: 'dashboard' }
        },
        {
          path: 'cases',
          name: 'CaseList',
          component: () => import('@/views/cases/CaseListView.vue'),
          meta: { title: '個案管理', module: 'masters_cases' }
        },
        {
          path: 'cases/:id',
          name: 'CaseDetail',
          component: () => import('@/views/cases/CaseDetailView.vue'),
          meta: { title: '個案編輯', module: 'masters_cases' }
        },
        {
          path: 'masters/regions',
          name: 'RegionList',
          component: () => import('@/views/masters/RegionListView.vue'),
          meta: { title: '地區管理', module: 'masters_regions' }
        },
        {
          path: 'masters/sites',
          name: 'SiteList',
          component: () => import('@/views/masters/SiteListView.vue'),
          meta: { title: '單位管理', module: 'masters_sites' }
        },
        {
          path: 'masters/vehicles',
          name: 'VehicleList',
          component: () => import('@/views/masters/VehicleListView.vue'),
          meta: { title: '車輛管理', module: 'masters_vehicles' }
        },
        {
          path: 'masters/drivers',
          name: 'DriverList',
          component: () => import('@/views/masters/DriverListView.vue'),
          meta: { title: '司機管理', module: 'masters_drivers' }
        },
        {
          path: 'masters/caregivers',
          name: 'CaregiverList',
          component: () => import('@/views/masters/CaregiverListView.vue'),
          meta: { title: '照護人員管理', module: 'masters_caregivers' }
        },
        {
          path: 'driver-reports',
          redirect: '/driver-reports/status'
        },
        {
          path: 'driver-reports/status',
          name: 'DriverReportStatus',
          component: () => import('@/views/driverReports/DriverReportStatusView.vue'),
          meta: { title: '接送匯報總覽', module: 'driver_reports' }
        },
        {
          path: 'driver-reports/import',
          name: 'DriverReportImport',
          component: () => import('@/views/driverReports/DriverReportImportView.vue'),
          meta: { title: '批次上傳', module: 'driver_reports' }
        },
        {
          path: 'driver-reports/batch-import',
          redirect: '/driver-reports/import'
        },
        {
          path: 'driver-reports/mappings',
          redirect: '/driver-reports/import'
        },
        {
          path: 'rides',
          name: 'RideCalendar',
          component: () => import('@/views/rides/RideCalendarView.vue'),
          meta: { title: '搭乘月曆表', module: 'rides_calendar' }
        },
        {
          path: 'rides/issues',
          name: 'RideIssues',
          component: () => import('@/views/rides/RideIssuesView.vue'),
          meta: { title: '異常集中處理', module: 'rides_issues' }
        },
        {
          path: 'rides/missing',
          name: 'MissingRides',
          component: () => import('@/views/rides/MissingRidesView.vue'),
          meta: { title: '未回報清單與催報歷史', module: 'rides_missing' }
        },
        {
          path: 'reports/trip-summary',
          name: 'TripSummary',
          component: () => import('@/views/reports/TripSummaryView.vue'),
          meta: { title: '車輛趟數表', module: 'reports_trip_summary' }
        },
        {
          path: 'reports/hsinchu-schedule',
          name: 'HsinchuSchedule',
          component: () => import('@/views/reports/HsinchuScheduleView.vue'),
          meta: { title: '新竹接送時刻表', module: 'reports_hsinchu_schedule' }
        },
        {
          path: 'vehicles/maintenance',
          name: 'VehicleMaintenance',
          component: () => import('@/views/vehicles/MaintenanceView.vue'),
          meta: { title: '車輛維修保養', module: 'vehicles_maintenance' }
        },
        {
          path: 'attendance',
          name: 'AttendanceFuel',
          component: () => import('@/views/attendance/AttendanceFuelView.vue'),
          meta: { title: '出勤與油資登錄', module: 'attendance_fuel' }
        },
        {
          path: 'audit',
          name: 'AuditLog',
          component: () => import('@/views/audit/AuditLogView.vue'),
          meta: { title: '系統操作紀錄', module: 'audit_logs' }
        },
        {
          path: 'settings/users',
          name: 'UserManagement',
          component: () => import('@/views/settings/UserManagementView.vue'),
          meta: { title: '使用者管理', module: 'settings_users' }
        },
        {
          path: 'settings/roles',
          name: 'RoleManagement',
          component: () => import('@/views/settings/RoleManagementView.vue'),
          meta: { title: '角色身分管理', module: 'settings_roles' }
        },
        {
          path: 'settings/notifications',
          name: 'NotificationSettings',
          component: () => import('@/views/settings/NotificationSettingsView.vue'),
          meta: { title: '通知收件人管理', module: 'settings_notifications' }
        },
        {
          path: 'settings/holidays',
          name: 'HolidayCalendar',
          component: () => import('@/views/settings/HolidayCalendarView.vue'),
          meta: { title: '政府假日與工作日設定', module: 'settings_holidays' }
        },
        {
          path: 'exports',
          name: 'GovExport',
          component: () => import('@/views/exports/ExportView.vue'),
          meta: { title: '政府申報匯出', module: 'exports' }
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
