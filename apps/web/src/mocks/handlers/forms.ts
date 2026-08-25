import { http, HttpResponse } from 'msw'
import { mockForms, mockFormColumns } from '../data/mockData'

export const formsHandlers = [
  http.get('/api/v1/forms', () => {
    return HttpResponse.json(mockForms)
  }),

  http.post('/api/v1/forms/:id/sync', () => {
    return HttpResponse.json({ syncedRows: 24, newColumns: 1 })
  }),

  http.get('/api/v1/forms/columns', ({ request }) => {
    const url = new URL(request.url)
    const status = url.searchParams.get('mappingStatus')
    let cols = [...mockFormColumns]
    if (status) {
      cols = cols.filter((c) => c.mappingStatus === status)
    }
    return HttpResponse.json(cols)
  }),

  http.patch('/api/v1/forms/columns/:id/mapping', async ({ params, request }) => {
    const col = mockFormColumns.find((c) => c.id === params.id)
    if (!col) return new HttpResponse(null, { status: 404 })
    const body = (await request.json()) as any
    Object.assign(col, body)
    return HttpResponse.json(col)
  }),

  http.post('/api/v1/forms/columns/batch-mapping', async ({ request }) => {
    const body = (await request.json()) as any
    return HttpResponse.json({ updatedCount: body.mappings.length })
  })
]
