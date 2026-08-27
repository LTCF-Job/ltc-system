import { http, HttpResponse } from 'msw'
import { mockTripSummaryReport, mockHsinchuScheduleReport } from '../data/mockData'
import { createMockExcelBlob } from '../utils/mockExcel'

export const reportsHandlers = [
  // 車輛趟數表
  http.get('/api/v1/reports/trip-summary', ({ request }) => {
    const url = new URL(request.url)
    const vehicleId = url.searchParams.get('vehicleId')
    const q = url.searchParams.get('q')?.trim().toLowerCase()

    const report = { ...mockTripSummaryReport }
    if (vehicleId) {
      report.vehicles = report.vehicles.filter((v) => v.vehicleId === vehicleId)
    }
    if (q) {
      report.vehicles = report.vehicles.filter(
        (v) =>
          v.vehicleName.toLowerCase().includes(q) ||
          (v.plateNo && v.plateNo.toLowerCase().includes(q)) ||
          (v.driverName && v.driverName.toLowerCase().includes(q))
      )
    }
    return HttpResponse.json(report)
  }),

  http.get('/api/v1/reports/trip-summary/export', () => {
    const excelBlob = createMockExcelBlob()
    return new HttpResponse(excelBlob, {
      headers: {
        'Content-Type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'Content-Disposition': 'attachment; filename="trip-summary.xlsx"'
      }
    })
  }),

  // 新竹接送時刻表
  http.get('/api/v1/reports/hsinchu-schedule', ({ request }) => {
    const url = new URL(request.url)
    const q = url.searchParams.get('q')?.trim().toLowerCase()

    if (!q) {
      return HttpResponse.json(mockHsinchuScheduleReport)
    }

    const matchItem = (item: any) =>
      item.caseName.toLowerCase().includes(q) ||
      item.caseCode.toLowerCase().includes(q) ||
      item.origin.toLowerCase().includes(q) ||
      item.destination.toLowerCase().includes(q) ||
      item.vehicleName.toLowerCase().includes(q) ||
      item.siteName.toLowerCase().includes(q) ||
      (item.note && item.note.toLowerCase().includes(q))

    return HttpResponse.json({
      ...mockHsinchuScheduleReport,
      outbound: mockHsinchuScheduleReport.outbound.filter(matchItem),
      inbound: mockHsinchuScheduleReport.inbound.filter(matchItem)
    })
  }),

  http.get('/api/v1/reports/hsinchu-schedule/export', () => {
    const excelBlob = createMockExcelBlob()
    return new HttpResponse(excelBlob, {
      headers: {
        'Content-Type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'Content-Disposition': 'attachment; filename="hsinchu-schedule.xlsx"'
      }
    })
  })
]
