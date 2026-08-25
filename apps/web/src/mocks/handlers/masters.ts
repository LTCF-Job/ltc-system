import { http, HttpResponse } from 'msw'
import { mockSites, mockVehicles, mockDrivers } from '../data/mockData'

export const mastersHandlers = [
  // 據點主檔
  http.get('/api/v1/sites', () => {
    return HttpResponse.json({ data: mockSites, meta: { total: mockSites.length } })
  }),

  http.post('/api/v1/sites', async ({ request }) => {
    const body = (await request.json()) as any
    const newSite = { id: `site_${Date.now()}`, ...body, createdAt: '2026-08-25' }
    mockSites.push(newSite)
    return HttpResponse.json(newSite)
  }),

  http.patch('/api/v1/sites/:id', async ({ params, request }) => {
    const s = mockSites.find((item) => item.id === params.id)
    if (!s) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    Object.assign(s, body)
    return HttpResponse.json(s)
  }),

  http.delete('/api/v1/sites/:id', ({ params }) => {
    const idx = mockSites.findIndex((item) => item.id === params.id)
    if (idx !== -1) mockSites.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // 車輛主檔
  http.get('/api/v1/vehicles', () => {
    return HttpResponse.json({ data: mockVehicles, meta: { total: mockVehicles.length } })
  }),

  http.post('/api/v1/vehicles', async ({ request }) => {
    const body = (await request.json()) as any
    const newV = { id: `veh_${Date.now()}`, ...body, createdAt: '2026-08-25' }
    mockVehicles.push(newV)
    return HttpResponse.json(newV)
  }),

  http.patch('/api/v1/vehicles/:id', async ({ params, request }) => {
    const v = mockVehicles.find((item) => item.id === params.id)
    if (!v) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    Object.assign(v, body)
    return HttpResponse.json(v)
  }),

  http.delete('/api/v1/vehicles/:id', ({ params }) => {
    const idx = mockVehicles.findIndex((item) => item.id === params.id)
    if (idx !== -1) mockVehicles.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // 司機主檔
  http.get('/api/v1/drivers', () => {
    return HttpResponse.json({ data: mockDrivers, meta: { total: mockDrivers.length } })
  }),

  http.post('/api/v1/drivers', async ({ request }) => {
    const body = (await request.json()) as any
    const newD = { id: `drv_${Date.now()}`, ...body, createdAt: '2026-08-25' }
    mockDrivers.push(newD)
    return HttpResponse.json(newD)
  }),

  http.patch('/api/v1/drivers/:id', async ({ params, request }) => {
    const d = mockDrivers.find((item) => item.id === params.id)
    if (!d) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    Object.assign(d, body)
    return HttpResponse.json(d)
  }),

  http.delete('/api/v1/drivers/:id', ({ params }) => {
    const idx = mockDrivers.findIndex((item) => item.id === params.id)
    if (idx !== -1) mockDrivers.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  http.post('/api/v1/drivers/:id/reveal', ({ params }) => {
    const plainMap: Record<string, string> = {
      drv_1: 'G121806465',
      drv_2: 'J120011223',
      drv_3: 'K120098177'
    }
    return HttpResponse.json({ nationalId: plainMap[params.id as string] || 'G123456789' })
  }),

  http.post('/api/v1/drivers/:id/assignments', async ({ params, request }) => {
    const d = mockDrivers.find((item) => item.id === params.id)
    if (!d) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    const veh = mockVehicles.find((v) => v.id === body.vehicleId)
    d.assignments = [
      {
        id: `asgn_${Date.now()}`,
        driverId: d.id,
        vehicleId: body.vehicleId,
        vehicleName: veh?.displayName,
        startDate: body.startDate,
        endDate: body.endDate,
        isPrimary: body.isPrimary
      }
    ]
    return HttpResponse.json({ success: true })
  })
]
