import { http, HttpResponse } from 'msw'
import {
  mockPrecheckResult,
  mockExportJobs,
  mockCases,
  mockSites,
  mockVehicles,
  mockDrivers,
  mockAuditLogs
} from '../data/mockData'
import { createGovClaimExcelBlob, createGovClaimZipBlob } from '../utils/mockExcel'
import { listDemoBoardedRides } from '../utils/demoRides'
import type { CaseDTO, ExportJobDTO, ExportJobFileDTO } from '@/types/api'

// 政府申報一律一個個案一個月一份工作簿，逐案下載與壓縮檔都由這批資料衍生。
export const exportsHandlers = [
  http.post('/api/v1/exports/precheck', () => {
    return HttpResponse.json(mockPrecheckResult)
  }),

  http.post('/api/v1/exports', async ({ request }) => {
    const body = (await request.json()) as any
    const jobId = `job_${Date.now()}`
    const periodYm: string = body.periodYm || '11507'
    const mode: 'direct' | 'zip' = body.mode === 'zip' ? 'zip' : 'direct'
    const caseIds: string[] = Array.isArray(body.caseIds) ? body.caseIds : []

    const files: ExportJobFileDTO[] = caseIds
      .map((caseId) => mockCases.find((c) => c.id === caseId))
      .filter((c): c is CaseDTO => Boolean(c))
      .map((c) => ({
        caseId: c.id,
        caseCode: c.code,
        caseName: c.name,
        region: c.region,
        rowCount: buildClaimRows(c, periodYm).length,
        fileName: `${c.name}${periodYm}.xlsx`,
        downloadUrl: `/api/v1/exports/${jobId}/files/${c.id}/download`
      }))

    if (files.length === 0) {
      return HttpResponse.json(
        { error: { code: 'NO_EXPORT_DATA', message: '指定條件下沒有可申報的資料' } },
        { status: 422 }
      )
    }

    const now = new Date().toISOString()
    const totalCases = files.length
    const totalRows = files.reduce((sum, f) => sum + f.rowCount, 0)
    const newJob: ExportJobDTO = {
      id: jobId,
      jobType: 'gov_claim',
      periodYm,
      region: body.region || undefined,
      mode,
      status: 'succeeded',
      totalCases,
      totalRows,
      files,
      zipFileName: mode === 'zip' ? `gov-claim-${body.region || 'all'}-${periodYm}.zip` : undefined,
      downloadUrl: mode === 'zip' ? `/api/v1/exports/${jobId}/download` : undefined,
      createdByName: '當前使用者',
      createdAt: now,
      completedAt: now
    }
    mockExportJobs.unshift(newJob)

    mockAuditLogs.unshift({
      id: `audit_${Date.now()}`,
      actorId: 'usr_staff',
      actorName: '當前使用者',
      actorRole: 'staff',
      action: 'export',
      entityType: 'export_jobs',
      entityId: jobId,
      entityName: `政府申報匯出 (${periodYm})`,
      beforeData: undefined,
      afterData: {
        periodYm,
        region: body.region || '',
        mode,
        totalCases,
        totalRows,
        cases: files.map((f) => ({
          caseCode: f.caseCode,
          caseName: f.caseName,
          region: f.region,
          fileName: f.fileName,
          rowCount: f.rowCount
        }))
      },
      ipAddress: '127.0.0.1',
      userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
      createdAt: now
    })

    return HttpResponse.json(newJob, { status: 202 })
  }),

  http.get('/api/v1/exports/:id/files/:caseId/download', ({ params }) => {
    const job = mockExportJobs.find((j) => j.id === params.id)
    const file = job?.files?.find((f) => f.caseId === params.caseId)
    const target = mockCases.find((c) => c.id === params.caseId)
    if (!job || !file || !target) {
      return HttpResponse.json({ error: { code: 'NOT_FOUND', message: '查無資料' } }, { status: 404 })
    }

    return new HttpResponse(createGovClaimExcelBlob(buildClaimRows(target, job.periodYm)), {
      headers: {
        'Content-Type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'Content-Disposition': `attachment; filename*=UTF-8''${encodeURIComponent(file.fileName)}`
      }
    })
  }),

  http.get('/api/v1/exports/:id/download', async ({ params }) => {
    const job = mockExportJobs.find((j) => j.id === params.id)
    if (!job) {
      return HttpResponse.json({ error: { code: 'NOT_FOUND', message: '查無資料' } }, { status: 404 })
    }
    // 逐案下載模式沒有整包檔案，與後端 ErrNotZipJob 的回應一致
    if (job.mode !== 'zip') {
      return HttpResponse.json(
        { error: { code: 'VALIDATION_FAILED', message: '輸入資料不符合規則，請確認後再試' } },
        { status: 400 }
      )
    }

    const entries = (job.files || [])
      .map((file) => {
        const target = mockCases.find((c) => c.id === file.caseId)
        if (!target) return null
        return { name: file.fileName, content: createGovClaimExcelBlob(buildClaimRows(target, job.periodYm)) }
      })
      .filter((entry): entry is { name: string; content: Blob } => entry !== null)

    const archive = await createGovClaimZipBlob(entries)
    return new HttpResponse(archive, {
      headers: {
        'Content-Type': 'application/zip',
        'Content-Disposition': `attachment; filename="${job.zipFileName || 'gov-claim.zip'}"`
      }
    })
  }),

  http.get('/api/v1/exports/:id', ({ params }) => {
    const job = mockExportJobs.find((j) => j.id === params.id)
    if (!job) {
      return HttpResponse.json({ error: { code: 'NOT_FOUND', message: '查無資料' } }, { status: 404 })
    }
    return HttpResponse.json(job)
  }),

  http.get('/api/v1/exports', ({ request }) => {
    const url = new URL(request.url)
    const page = Number(url.searchParams.get('page') || 1)
    const pageSize = Number(url.searchParams.get('pageSize') || 10)
    const data = mockExportJobs.slice((page - 1) * pageSize, page * pageSize).map((job) => ({
      ...job,
      // 歷史列表不帶檔案明細與下載連結，與後端 List 的回應一致
      files: undefined,
      downloadUrl: undefined,
      zipFileName: undefined
    }))
    return HttpResponse.json({
      data,
      meta: {
        page,
        pageSize,
        total: mockExportJobs.length,
        totalPages: Math.ceil(mockExportJobs.length / pageSize)
      }
    })
  })
]

/**
 * 依該月實際成行的搭乘紀錄組出申報列，欄位順序與型別對齊後端 govform.BuildClaimRow：
 * 去程整月在前、回程整月在後，兩者的出發地與目的地對調，系統目前沒有資料的欄位一律留空。
 * 搭乘紀錄與搭乘月曆讀同一份 listDemoBoardedRides，缺席與未回報不會出現在申報檔。
 */
function buildClaimRows(target: CaseDTO, periodYm: string): (string | number)[][] {
  const schedule = target.activeSchedule
  if (!schedule) return []

  const rocYear = Number(periodYm.slice(0, 3))
  const month = Number(periodYm.slice(3))
  const year = rocYear + 1911

  const site = mockSites.find((s) => s.id === schedule.siteId)

  return listDemoBoardedRides(target, year, month).map((ride) => {
    const day = Number(ride.serviceDate.slice(8))
    const departTime = ride.departTimeOverride || ride.scheduledDepartTime
    const [departHour, departMinute] = departTime.split(':').map(Number)
    const durationMin = ride.durationMinOverride || ride.scheduledDurationMin || 10
    const endMinutes = departHour * 60 + departMinute + durationMin

    // 車號與服務人員取這一趟實際承載的車與司機，不是個案排班的預設值
    const vehicle = mockVehicles.find((v) => v.id === ride.vehicleId)
    const driver = mockDrivers.find((d) => d.id === ride.driverId)

    const cells: (string | number)[] = new Array(33).fill('')
    cells[0] = target.nationalId || ''
    cells[1] = rocYear * 10000 + month * 100 + day
    cells[2] = schedule.serviceCode || 'BD03'
    cells[3] = target.serviceCategory || 1
    cells[4] = 1
    cells[5] = schedule.unitPrice
    cells[6] = driver?.nationalId || ''
    cells[7] = departHour
    cells[8] = departMinute
    cells[9] = Math.floor(endMinutes / 60)
    cells[10] = endMinutes % 60
    cells[16] = ride.notClaimedAa09 ? 1 : ''
    cells[24] = ride.direction === 'inbound' ? site?.address || '' : target.homeAddress || ''
    cells[25] = ride.direction === 'inbound' ? target.homeAddress || '' : site?.address || ''
    cells[30] = schedule.distanceKm
    cells[31] = vehicle?.plateNo || ''
    cells[32] = target.serviceUsageType || 2
    return cells
  })
}
