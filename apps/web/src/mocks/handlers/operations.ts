import { http, HttpResponse } from 'msw'
import {
  mockVehicles,
  mockDrivers,
  mockMaintenanceLogs,
  mockAttendanceReport,
  mockFuelLogs
} from '../data/mockData'

export const operationsHandlers = [
  // 車輛維修保養
  http.get('/api/v1/vehicles/maintenance', ({ request }) => {
    const url = new URL(request.url)
    const vehicleId = url.searchParams.get('vehicleId')
    let list = [...mockMaintenanceLogs]
    if (vehicleId) {
      list = list.filter((m) => m.vehicleId === vehicleId)
    }
    return HttpResponse.json({
      data: list,
      meta: {
        page: 1,
        pageSize: 20,
        total: list.length,
        totalPages: 1
      }
    })
  }),

  http.post('/api/v1/vehicles/maintenance', async ({ request }) => {
    const body = (await request.json()) as any
    const veh = mockVehicles.find((v) => v.id === body.vehicleId)
    const newLog = {
      id: `maint_${Date.now()}`,
      vehicleId: body.vehicleId,
      vehicleName: veh?.displayName || '未知車輛',
      plateNo: veh?.plateNo || '',
      serviceDate: body.serviceDate,
      mileage: body.mileage,
      items: body.items,
      vendor: body.vendor,
      cost: body.cost,
      note: body.note,
      createdBy: '系統使用者',
      createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
    }
    mockMaintenanceLogs.unshift(newLog)
    return HttpResponse.json(newLog)
  }),

  http.patch('/api/v1/vehicles/maintenance/:id', async ({ params, request }) => {
    const target = mockMaintenanceLogs.find((m) => m.id === params.id)
    if (!target) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    Object.assign(target, body)
    return HttpResponse.json(target)
  }),

  http.delete('/api/v1/vehicles/maintenance/:id', ({ params }) => {
    const idx = mockMaintenanceLogs.findIndex((m) => m.id === params.id)
    if (idx !== -1) mockMaintenanceLogs.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  http.get('/api/v1/vehicles/maintenance/blank-template', () => {
    const dummyBlob = new Blob(['mock-blank-maintenance-excel'], {
      type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    })
    return new HttpResponse(dummyBlob, {
      headers: {
        'Content-Type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'Content-Disposition': 'attachment; filename="blank-maintenance-template.xlsx"'
      }
    })
  }),

  // 司機出勤與請假
  http.get('/api/v1/attendance', ({ request }) => {
    const url = new URL(request.url)
    const driverId = url.searchParams.get('driverId')
    const report = { ...mockAttendanceReport }
    if (driverId) {
      report.drivers = report.drivers.filter((d) => d.driverId === driverId)
    }
    return HttpResponse.json(report)
  }),

  http.post('/api/v1/attendance', async ({ request }) => {
    const body = (await request.json()) as any
    const driver = mockAttendanceReport.drivers.find((d) => d.driverId === body.driverId)
    if (driver) {
      driver.days[body.recordDate] = {
        date: body.recordDate,
        status: body.status,
        note: body.note
      }
    }
    return HttpResponse.json({ success: true })
  }),

  // 車輛油資紀錄
  http.get('/api/v1/fuel-logs', ({ request }) => {
    const url = new URL(request.url)
    const vehicleId = url.searchParams.get('vehicleId')
    let list = [...mockFuelLogs]
    if (vehicleId) {
      list = list.filter((f) => f.vehicleId === vehicleId)
    }
    return HttpResponse.json({
      data: list,
      meta: {
        page: 1,
        pageSize: 20,
        total: list.length,
        totalPages: 1
      }
    })
  }),

  http.post('/api/v1/fuel-logs', async ({ request }) => {
    const body = (await request.json()) as any
    const veh = mockVehicles.find((v) => v.id === body.vehicleId)
    const drv = mockDrivers.find((d) => d.id === body.driverId)
    const newLog = {
      id: `fuel_${Date.now()}`,
      vehicleId: body.vehicleId,
      vehicleName: veh?.displayName || '',
      plateNo: veh?.plateNo || '',
      driverId: body.driverId,
      driverName: drv?.name || '',
      fuelDate: body.fuelDate,
      liters: body.liters,
      cost: body.cost,
      createdBy: '系統使用者',
      createdAt: new Date().toISOString().replace('T', ' ').substring(0, 19)
    }
    mockFuelLogs.unshift(newLog)
    return HttpResponse.json(newLog)
  }),

  http.patch('/api/v1/fuel-logs/:id', async ({ params, request }) => {
    const target = mockFuelLogs.find((f) => f.id === params.id)
    if (!target) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    Object.assign(target, body)
    return HttpResponse.json(target)
  }),

  http.delete('/api/v1/fuel-logs/:id', ({ params }) => {
    const idx = mockFuelLogs.findIndex((f) => f.id === params.id)
    if (idx !== -1) mockFuelLogs.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  })
]
