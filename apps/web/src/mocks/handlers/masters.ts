import { http, HttpResponse } from 'msw'
import { mockRegions, mockSites, mockVehicles, mockDrivers } from '../data/mockData'

// 車輛的司機由 driver_assignments 反查，與後端 GET /vehicles 帶出 drivers 的契約一致
function driversOfVehicle(vehicleId: string) {
  return mockDrivers
    .filter((d) => (d.assignments || []).some((a) => a.vehicleId === vehicleId))
    .map((d) => ({ id: d.id, code: d.code, name: d.name }))
}

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
          r.name.toLowerCase().includes(q) ||
          (r.description && r.description.toLowerCase().includes(q))
      )
    }
    if (status) {
      filtered = filtered.filter((r) => r.status === status)
    }

    // 依 sortOrder 與名稱排序
    filtered.sort((a, b) => (a.sortOrder - b.sortOrder) || a.name.localeCompare(b.name))

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
    const existing = mockRegions.find(r => r.name.toLowerCase() === body.name?.trim().toLowerCase())
    if (existing) {
      return HttpResponse.json({ error: { code: 'VALIDATION_FAILED', message: '區域名稱已存在' } }, { status: 409 })
    }
    const newReg = {
      id: `reg_${Date.now()}`,
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

    const withDrivers = filtered.map((v) => ({ ...v, drivers: driversOfVehicle(v.id) }))
    return HttpResponse.json({ data: withDrivers, meta: { total: withDrivers.length } })
  }),

  http.put('/api/v1/vehicles/:id/drivers', async ({ params, request }) => {
    const vehicleId = params.id as string
    const vehicle = mockVehicles.find((v) => v.id === vehicleId)
    if (!vehicle) return new HttpResponse(null, { status: 404 })

    const body = (await request.json()) as { driverIds?: string[]; effectiveFrom?: string }
    const driverIds = body.driverIds || []
    const startDate = body.effectiveFrom || new Date().toISOString().split('T')[0]

    mockDrivers.forEach((d) => {
      const keptElsewhere = (d.assignments || []).filter((a) => a.vehicleId !== vehicleId)
      if (driverIds.includes(d.id)) {
        // 一位司機同期只有一台車：掛到本車時，其他車的指派一併收掉
        d.assignments = [
          {
            id: `asgn_${vehicleId}_${d.id}`,
            driverId: d.id,
            vehicleId,
            vehicleName: vehicle.displayName,
            vehiclePlateNo: vehicle.plateNo,
            plateNo: vehicle.plateNo,
            startDate
          }
        ]
      } else {
        d.assignments = keptElsewhere
      }
    })

    return HttpResponse.json({ vehicleId, driverIds })
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
    // 一位司機同期只有一台車，指派新車即取代原本的指派
    d.assignments = [
      {
        id: `asgn_${Date.now()}`,
        driverId: d.id,
        vehicleId: body.vehicleId,
        vehicleName: veh?.displayName,
        vehiclePlateNo: veh?.plateNo,
        plateNo: veh?.plateNo,
        startDate: body.startDate,
        endDate: body.endDate
      }
    ]
    return HttpResponse.json({ success: true })
  }),

  // 主檔批次匯入（車輛／司機／班表），預覽契約比照 /cases/import
  http.post('/api/v1/masters/import', ({ request }) => {
    const url = new URL(request.url)
    const isDryRun = url.searchParams.get('dryRun') === 'true'

    if (isDryRun) {
      return HttpResponse.json({
        totalRows: 6,
        validRows: 5,
        errorRows: 1,
        warningRows: 0,
        previewRows: [
          { rowIndex: 2, name: '林小明', code: 'DRV004', vehiclePlateNo: 'BZG-7915', phone: '0912345678' },
          { rowIndex: 3, name: '陳大同', code: 'DRV005', vehiclePlateNo: 'BZH-2201', phone: '0923456789' }
        ],
        errors: [
          { rowIndex: 6, caseName: '未指定車牌', field: '車牌號碼', message: '車牌號碼未填寫或格式錯誤' }
        ],
        warnings: []
      })
    }

    return HttpResponse.json({ count: 5 })
  })
]
