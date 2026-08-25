import { http, HttpResponse } from 'msw'
import { mockPrecheckResult, mockExportJobs } from '../data/mockData'

export const exportsHandlers = [
  http.post('/api/v1/exports/precheck', () => {
    return HttpResponse.json(mockPrecheckResult)
  }),

  http.post('/api/v1/exports', async ({ request }) => {
    const body = (await request.json()) as any
    const newJob = {
      id: `job_${Date.now()}`,
      jobType: body.jobType,
      periodYm: body.periodYm,
      region: body.region,
      mode: body.mode,
      status: 'running' as const,
      totalCases: 42,
      totalRows: 380,
      createdAt: new Date().toISOString()
    }
    mockExportJobs.unshift(newJob as any)
    return HttpResponse.json(newJob)
  }),

  http.get('/api/v1/exports/:id', ({ params }) => {
    const job = mockExportJobs.find((j) => j.id === params.id) || {
      id: params.id,
      jobType: 'gov_claim',
      periodYm: '11507',
      status: 'succeeded',
      totalCases: 42,
      totalRows: 380,
      fileName: 'gov-claim-11507.xlsx',
      downloadUrl: 'https://placeholder-download.supabase.co/gov-claim-11507.xlsx'
    }
    return HttpResponse.json(job)
  }),

  http.get('/api/v1/exports', () => {
    return HttpResponse.json({ data: mockExportJobs, meta: { total: mockExportJobs.length } })
  })
]
