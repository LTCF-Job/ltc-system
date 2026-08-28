import { http, HttpResponse } from 'msw'
import { mockPrecheckResult, mockExportJobs } from '../data/mockData'
import { createMockExcelBlob } from '../utils/mockExcel'

export const exportsHandlers = [
  http.post('/api/v1/exports/precheck', () => {
    return HttpResponse.json(mockPrecheckResult)
  }),

  http.post('/api/v1/exports', async ({ request }) => {
    const body = (await request.json()) as any
    const jobId = `job_${Date.now()}`
    const jobType = body.jobType || 'gov_claim'
    const periodYm = body.periodYm || '115-07'
    const region = body.region || 'hsinchu'
    const fileName = jobType === 'trip_summary'
      ? `trip-summary-${periodYm}.xlsx`
      : jobType === 'hsinchu_schedule'
        ? 'hsinchu-schedule.xlsx'
        : `gov-claim-${region}-${periodYm.replaceAll('-', '')}.xlsx`
    const newJob = {
      id: jobId,
      jobType,
      periodYm,
      region,
      mode: body.mode || 'single_multi_case',
      status: 'succeeded' as const,
      totalCases: 42,
      totalRows: 380,
      fileName,
      downloadUrl: `/api/v1/exports/${jobId}/download?jobType=${jobType}&periodYm=${periodYm}&region=${region}`,
      createdAt: new Date().toISOString()
    }
    mockExportJobs.unshift(newJob as any)
    return HttpResponse.json(newJob)
  }),

  http.get('/api/v1/exports/:id', ({ params }) => {
    const job = mockExportJobs.find((j) => j.id === params.id) || {
      id: params.id,
      jobType: 'gov_claim',
      periodYm: '115-07',
      status: 'succeeded',
      totalCases: 42,
      totalRows: 380,
      fileName: 'gov-claim-115-07.xlsx',
      downloadUrl: `/api/v1/exports/${params.id}/download?periodYm=115-07&region=hsinchu`
    }
    return HttpResponse.json(job)
  }),

  http.get('/api/v1/exports/:id/download', ({ request }) => {
    const url = new URL(request.url)
    const jobType = url.searchParams.get('jobType') || 'gov_claim'
    const periodYm = url.searchParams.get('periodYm') || '115-07'
    const region = url.searchParams.get('region') || 'hsinchu'
    const fileName = jobType === 'trip_summary'
      ? `trip-summary-${periodYm}.xlsx`
      : jobType === 'hsinchu_schedule'
        ? 'hsinchu-schedule.xlsx'
        : `gov-claim-${region}-${periodYm.replaceAll('-', '')}.xlsx`
    const excelBlob = createMockExcelBlob()
    return new HttpResponse(excelBlob, {
      headers: {
        'Content-Type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'Content-Disposition': `attachment; filename="${fileName}"`
      }
    })
  }),

  http.get('/api/v1/exports', () => {
    const data = mockExportJobs.map((job) => ({
      ...job,
      downloadUrl: job.downloadUrl?.startsWith('/api/v1/')
        ? job.downloadUrl
        : `/api/v1/exports/${job.id}/download?jobType=${job.jobType}&periodYm=${job.periodYm}&region=${job.region || 'hsinchu'}`
    }))
    return HttpResponse.json({ data, meta: { total: data.length } })
  })
]
