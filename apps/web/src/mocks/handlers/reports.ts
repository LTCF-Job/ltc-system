import { http, HttpResponse } from 'msw'
import { mockTripSummaryReport, mockHsinchuScheduleReport } from '../data/mockData'

export const reportsHandlers = [
  // 車輛趟數表
  http.get('/api/v1/reports/trip-summary', ({ request }) => {
    const url = new URL(request.url)
    const vehicleId = url.searchParams.get('vehicleId')
    const report = { ...mockTripSummaryReport }
    if (vehicleId) {
      report.vehicles = report.vehicles.filter((v) => v.vehicleId === vehicleId)
    }
    return HttpResponse.json(report)
  }),

  http.get('/api/v1/reports/trip-summary/export', () => {
    const dummyBlob = new Blob(['mock-excel-binary-data'], {
      type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    })
    return new HttpResponse(dummyBlob, {
      headers: {
        'Content-Type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'Content-Disposition': 'attachment; filename="trip-summary.xlsx"'
      }
    })
  }),

  // 新竹接送時刻表
  http.get('/api/v1/reports/hsinchu-schedule', () => {
    return HttpResponse.json(mockHsinchuScheduleReport)
  }),

  http.get('/api/v1/reports/hsinchu-schedule/export', () => {
    const dummyBlob = new Blob(['mock-hsinchu-schedule-binary'], {
      type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    })
    return new HttpResponse(dummyBlob, {
      headers: {
        'Content-Type': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'Content-Disposition': 'attachment; filename="hsinchu-schedule.xlsx"'
      }
    })
  })
]
