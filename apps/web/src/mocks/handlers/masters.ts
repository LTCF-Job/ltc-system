import { http, HttpResponse } from 'msw'
import { mockRegions, mockSites, mockVehicles, mockDrivers } from '../data/mockData'

export const mastersHandlers = [
  // 區域主檔
  http.get('/api/v1/regions', ({ request }) => {
    const url = new URL(request.url)
    const isAll = url.searchParams.get('all') === 'true'
    const q = url.searchParams.get('q')?.trim().toLowerCase()
    const status = url.searchParams.get('status')

    let filtered = [...mockRegions]
    if (q) {
      filtered = filtered.filter(
        (r) =>
          r.code.toLowerCase().includes(q) ||
          r.name.toLowerCase().includes(q) ||
          (r.description && r.description.toLowerCase().includes(q))
      )
    }
    if (status) {
      filtered = filtered.filter((r) => r.status === status)
    }

    // 依 sortOrder 與 code 排序
    filtered.sort((a, b) => (a.sortOrder - b.sortOrder) || a.code.localeCompare(b.code))

    if (isAll) {
      return HttpResponse.json({ data: filtered })
    }

    return HttpResponse.json({
      data: filtered,
      meta: {
        total: filtered.length,
        page: 1,
        pageSize: 100,
        totalPages: 1
      }
    })
  }),

  http.post('/api/v1/regions', async ({ request }) => {
    const body = (await request.json()) as any
    const existing = mockRegions.find(r => r.code.toLowerCase() === body.code?.toLowerCase())
    if (existing) {
      return HttpResponse.json({ error: { code: 'VALIDATION_FAILED', message: '區域代碼已存在' } }, { status: 409 })
    }
    const newReg = {
      id: `reg_${Date.now()}`,
      code: body.code.toLowerCase(),
      name: body.name,
      description: body.description || '',
      status: body.status || 'active',
      sortOrder: Number(body.sortOrder) || mockRegions.length + 1,
      createdAt: new Date().toISOString().slice(0, 10),
      updatedAt: new Date().toISOString().slice(0, 10)
    }
    mockRegions.push(newReg)
    return HttpResponse.json({ data: newReg }, { status: 201 })
  }),

  http.patch('/api/v1/regions/:id', async ({ params, request }) => {
    const reg = mockRegions.find((item) => item.id === params.id)
    if (!reg) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    if (body.name !== undefined) reg.name = body.name
    if (body.description !== undefined) reg.description = body.description
    if (body.status !== undefined) reg.status = body.status
    if (body.sortOrder !== undefined) reg.sortOrder = Number(body.sortOrder)
    reg.updatedAt = new Date().toISOString().slice(0, 10)
    return HttpResponse.json({ data: reg })
  }),

  http.delete('/api/v1/regions/:id', ({ params }) => {
    const idx = mockRegions.findIndex((item) => item.id === params.id)
    if (idx !== -1) mockRegions.splice(idx, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  // 據點主檔
  http.get('/api/v1/sites', ({ request }) => {
    const url = new URL(request.url)
    const q = url.searchParams.get('q')?.trim().toLowerCase()
    const region = url.searchParams.get('region')

    let filtered = [...mockSites]
    if (q) {
      filtered = filtered.filter(
        (s) =>
          s.name.toLowerCase().includes(q) ||
          s.address.toLowerCase().includes(q)
      )
    }
    if (region) {
      filtered = filtered.filter((s) => s.region === region)
    }

    return HttpResponse.json({ data: filtered, meta: { total: filtered.length } })
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
  http.get('/api/v1/vehicles', ({ request }) => {
    const url = new URL(request.url)
    const q = url.searchParams.get('q')?.trim().toLowerCase()
    const region = url.searchParams.get('region')
    const activeStr = url.searchParams.get('active')

    let filtered = [...mockVehicles]
    if (q) {
      filtered = filtered.filter(
        (v) =>
          v.plateNo.toLowerCase().includes(q) ||
          v.displayName.toLowerCase().includes(q)
      )
    }
    if (region) {
      filtered = filtered.filter((v) => v.region === region)
    }
    if (activeStr !== null && activeStr !== '') {
      const active = activeStr === 'true'
      filtered = filtered.filter((v) => v.active === active)
    }

    return HttpResponse.json({ data: filtered, meta: { total: filtered.length } })
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
  http.get('/api/v1/drivers', ({ request }) => {
    const url = new URL(request.url)
    const q = url.searchParams.get('q')?.trim().toLowerCase()
    const activeStr = url.searchParams.get('active')

    let filtered = [...mockDrivers]
    if (q) {
      filtered = filtered.filter(
        (d) =>
          d.name.toLowerCase().includes(q) ||
          (d.code && d.code.toLowerCase().includes(q)) ||
          (d.nationalId && d.nationalId.toLowerCase().includes(q)) ||
          (d.phone && d.phone.includes(q)) ||
          (d.email && d.email.toLowerCase().includes(q))
      )
    }
    if (activeStr !== null && activeStr !== '') {
      const active = activeStr === 'true'
      filtered = filtered.filter((d) => d.active === active)
    }

    return HttpResponse.json({ data: filtered, meta: { total: filtered.length } })
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
        vehiclePlateNo: veh?.plateNo,
        plateNo: veh?.plateNo,
        startDate: body.startDate,
        endDate: body.endDate,
        isPrimary: body.isPrimary
      }
    ]
    return HttpResponse.json({ success: true })
  })
]
