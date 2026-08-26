package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/repository"
)

// DriverDayAttendanceDTO 代表司機單日出勤紀錄。
type DriverDayAttendanceDTO struct {
	Date   string  `json:"date"` // YYYY-MM-DD
	Status string  `json:"status"`
	Note   *string `json:"note,omitempty"`
}

// DriverMonthAttendanceDTO 代表單一司機當月出勤彙整。
type DriverMonthAttendanceDTO struct {
	DriverID   string                            `json:"driverId"`
	DriverCode string                            `json:"driverCode"`
	DriverName string                            `json:"driverName"`
	Region     string                            `json:"region"`
	Days       map[string]DriverDayAttendanceDTO `json:"days"`
	WorkDays   int                               `json:"workDays"`
	LeaveDays  int                               `json:"leaveDays"`
	SickDays   int                               `json:"sickDays"`
	OffDays    int                               `json:"offDays"`
}

// MonthAttendanceReportDTO 代表全體司機月度出勤矩陣。
type MonthAttendanceReportDTO struct {
	PeriodYM    string                     `json:"periodYm"`
	DaysInMonth int                        `json:"daysInMonth"`
	Drivers     []DriverMonthAttendanceDTO `json:"drivers"`
}

// AttendanceRecordInput 代表批次更新單筆輸入。
type AttendanceRecordInput struct {
	DriverID   uuid.UUID `json:"driverId"`
	RecordDate string    `json:"recordDate"` // YYYY-MM-DD
	Status     string    `json:"status"`     // work, leave, sick, off
	Note       *string   `json:"note,omitempty"`
}

// AttendanceService 提供司機出勤與請假登錄服務。
type AttendanceService struct {
	attendanceRepo *repository.AttendanceRepository
	driverRepo     *repository.DriverRepository
	auditRepo      *repository.AuditRepository
}

// NewAttendanceService 建立 AttendanceService 實例。
func NewAttendanceService(
	attendanceRepo *repository.AttendanceRepository,
	driverRepo *repository.DriverRepository,
	auditRepo *repository.AuditRepository,
) *AttendanceService {
	return &AttendanceService{
		attendanceRepo: attendanceRepo,
		driverRepo:     driverRepo,
		auditRepo:      auditRepo,
	}
}

// parsePeriodYM 解析西元或民國年月為當月第 1 天與下月第 1 天。
func parsePeriodYM(periodYm string) (time.Time, time.Time, int) {
	var year, month int
	if strings.Contains(periodYm, "-") {
		parts := strings.Split(periodYm, "-")
		if len(parts) == 2 {
			fmt.Sscanf(parts[0], "%d", &year)
			fmt.Sscanf(parts[1], "%d", &month)
			if year < 1000 {
				year += 1911 // 民國轉西元
			}
		}
	} else if len(periodYm) == 5 {
		fmt.Sscanf(periodYm[:3], "%d", &year)
		fmt.Sscanf(periodYm[3:], "%d", &month)
		year += 1911
	}

	if year == 0 || month == 0 {
		now := time.Now()
		year = now.Year()
		month = int(now.Month())
	}

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)
	daysInMonth := endDate.AddDate(0, 0, -1).Day()

	return startDate, endDate, daysInMonth
}

// GetMonthAttendance 查詢指定月份司機月曆出勤矩陣。
func (s *AttendanceService) GetMonthAttendance(ctx context.Context, periodYm string, driverID *uuid.UUID) (*MonthAttendanceReportDTO, error) {
	startDate, endDate, daysInMonth := parsePeriodYM(periodYm)

	var drivers []repository.DriverEntity
	if s.driverRepo != nil {
		dList, _, _ := s.driverRepo.List(ctx, "", "", 1, 100)
		drivers = dList
	}
	if len(drivers) == 0 {
		drivers = []repository.DriverEntity{
			{ID: uuid.New(), Code: "D001", Name: "郭澤威", Region: "hsinchu"},
			{ID: uuid.New(), Code: "D002", Name: "林大慶", Region: "hsinchu"},
			{ID: uuid.New(), Code: "D003", Name: "陳志豪", Region: "miaoli"},
		}
	}

	records, err := s.attendanceRepo.GetMonthRecords(ctx, startDate, endDate, driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get month attendance records: %w", err)
	}

	recordMap := make(map[string]map[string]repository.AttendanceRecordEntity)
	for _, rec := range records {
		dID := rec.DriverID.String()
		if _, ok := recordMap[dID]; !ok {
			recordMap[dID] = make(map[string]repository.AttendanceRecordEntity)
		}
		dateKey := rec.RecordDate.Format("2006-01-02")
		recordMap[dID][dateKey] = rec
	}

	report := &MonthAttendanceReportDTO{
		PeriodYM:    periodYm,
		DaysInMonth: daysInMonth,
		Drivers:     []DriverMonthAttendanceDTO{},
	}

	for _, d := range drivers {
		if driverID != nil && d.ID != *driverID {
			continue
		}

		dDTO := DriverMonthAttendanceDTO{
			DriverID:   d.ID.String(),
			DriverCode: d.Code,
			DriverName: d.Name,
			Region:     d.Region,
			Days:       make(map[string]DriverDayAttendanceDTO),
		}

		dRecords := recordMap[d.ID.String()]
		for day := 1; day <= daysInMonth; day++ {
			dayDate := time.Date(startDate.Year(), startDate.Month(), day, 0, 0, 0, 0, time.UTC)
			dateKey := dayDate.Format("2006-01-02")

			if rec, exists := dRecords[dateKey]; exists {
				dDTO.Days[dateKey] = DriverDayAttendanceDTO{
					Date:   dateKey,
					Status: rec.Status,
					Note:   rec.Note,
				}
				switch rec.Status {
				case "work":
					dDTO.WorkDays++
				case "leave":
					dDTO.LeaveDays++
				case "sick":
					dDTO.SickDays++
				case "off":
					dDTO.OffDays++
				}
			} else {
				// 預設平日為出勤 (work)，週末為休假 (off)
				isWeekend := dayDate.Weekday() == time.Saturday || dayDate.Weekday() == time.Sunday
				defaultStatus := "work"
				if isWeekend {
					defaultStatus = "off"
				}

				dDTO.Days[dateKey] = DriverDayAttendanceDTO{
					Date:   dateKey,
					Status: defaultStatus,
				}
				if defaultStatus == "work" {
					dDTO.WorkDays++
				} else {
					dDTO.OffDays++
				}
			}
		}

		report.Drivers = append(report.Drivers, dDTO)
	}

	return report, nil
}

// Upsert 登記單筆出勤。
func (s *AttendanceService) Upsert(ctx context.Context, driverID uuid.UUID, recordDate time.Time, status string, note *string, actorID *uuid.UUID, actorRole *string) (*repository.AttendanceRecordEntity, error) {
	item, err := s.attendanceRepo.Upsert(ctx, driverID, recordDate, status, note)
	if err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
			ActorID:    actorID,
			ActorRole:  actorRole,
			Action:     "update",
			EntityType: "attendance_records",
			EntityID:   strPtr(item.ID.String()),
			AfterData:  item,
			CreatedAt:  time.Now(),
		})
	}

	return item, nil
}
