package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/rocdate"
)

// ErrAttendanceConflictNotFound 代表查無指定的出勤待維護衝突。
var ErrAttendanceConflictNotFound = errors.New("attendance import conflict not found")

// DriverDayAttendanceDTO 代表司機單日出勤紀錄。
type DriverDayAttendanceDTO struct {
	Date   string  `json:"date"` // YYYY-MM-DD
	Status string  `json:"status"`
	Note   *string `json:"note,omitempty"`
}

// DriverMonthAttendanceDTO 代表單一司機當月出勤彙整。
type DriverMonthAttendanceDTO struct {
	DriverID   string                            `json:"driverId"`
	DriverName string                            `json:"driverName"`
	Region     string                            `json:"region"`
	Days       map[string]DriverDayAttendanceDTO `json:"days"`
	WorkDays   int                               `json:"workDays"`
	LeaveDays  int                               `json:"leaveDays"`
	SickDays   int                               `json:"sickDays"`
	OffDays    int                               `json:"offDays"`
	AbsentDays int                               `json:"absentDays"`
}

// MonthAttendanceReportDTO 代表全體司機月度出勤矩陣。
type MonthAttendanceReportDTO struct {
	PeriodYM    string                     `json:"periodYm"`
	DaysInMonth int                        `json:"daysInMonth"`
	Drivers     []DriverMonthAttendanceDTO `json:"drivers"`
}

// AttendanceConflictDTO 代表一筆匯入與人工登記不一致的待維護衝突。
type AttendanceConflictDTO struct {
	ID             string  `json:"id"`
	DriverID       string  `json:"driverId"`
	DriverName     string  `json:"driverName"`
	RecordDate     string  `json:"recordDate"` // YYYY-MM-DD
	ExistingStatus string  `json:"existingStatus"`
	ImportedStatus string  `json:"importedStatus"`
	Status         string  `json:"status"`
	ResolvedChoice *string `json:"resolvedChoice,omitempty"`
}

// AttendanceRecordInput 代表批次更新單筆輸入。
type AttendanceRecordInput struct {
	DriverID   uuid.UUID `json:"driverId"`
	RecordDate string    `json:"recordDate"` // YYYY-MM-DD
	Status     string    `json:"status"`     // work, leave, sick, off
	Note       *string   `json:"note,omitempty"`
}

// AttendanceService 提供司機出勤與請假登記服務。
type AttendanceService struct {
	attendanceRepo AttendanceStore
	driverRepo     DriverLister
	auditRepo      AuditWriter
	holidayRepo    HolidayReader
}

// NewAttendanceService 建立 AttendanceService 實例。
func NewAttendanceService(
	attendanceRepo AttendanceStore,
	driverRepo DriverLister,
	auditRepo AuditWriter,
	holidayRepo HolidayReader,
) *AttendanceService {
	return &AttendanceService{
		attendanceRepo: attendanceRepo,
		driverRepo:     driverRepo,
		auditRepo:      auditRepo,
		holidayRepo:    holidayRepo,
	}
}

// GetMonthAttendance 查詢指定月份司機月曆出勤矩陣。
func (s *AttendanceService) GetMonthAttendance(ctx context.Context, periodYm string, driverID *uuid.UUID) (*MonthAttendanceReportDTO, error) {
	startDate, endDate, daysInMonth := rocdate.MonthRange(periodYm)

	var drivers []DriverRef
	if s.driverRepo != nil {
		dList, _, _ := s.driverRepo.List(ctx, "", "", 1, 100)
		drivers = dList
	}
	if len(drivers) == 0 {
		drivers = []DriverRef{
			{ID: uuid.New(), Name: "郭澤威", Region: "hsinchu"},
			{ID: uuid.New(), Name: "林大慶", Region: "hsinchu"},
			{ID: uuid.New(), Name: "陳志豪", Region: "miaoli"},
		}
	}

	records, err := s.attendanceRepo.GetMonthRecords(ctx, startDate, endDate, driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get month attendance records: %w", err)
	}

	holidayMap := map[string]bool{}
	if s.holidayRepo != nil {
		if hm, err := s.holidayRepo.GetHolidayMap(ctx, startDate.Year(), int(startDate.Month()), ""); err == nil {
			holidayMap = hm
		}
	}
	today := time.Now().UTC()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

	recordMap := make(map[string]map[string]AttendanceRecord)
	for _, rec := range records {
		dID := rec.DriverID.String()
		if _, ok := recordMap[dID]; !ok {
			recordMap[dID] = make(map[string]AttendanceRecord)
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
				// 週末或國定假日視為休假 (off)；平日無紀錄時，已過去的日期視為
				// 應出勤卻漏報 (absent)，尚未到來的日期維持預定出勤 (work)。
				isWeekend := dayDate.Weekday() == time.Saturday || dayDate.Weekday() == time.Sunday
				isRestDay := isWeekend || holidayMap[dateKey]

				var defaultStatus string
				switch {
				case isRestDay:
					defaultStatus = "off"
				case dayDate.After(today):
					defaultStatus = "work"
				default:
					defaultStatus = "absent"
				}

				dDTO.Days[dateKey] = DriverDayAttendanceDTO{
					Date:   dateKey,
					Status: defaultStatus,
				}
				switch defaultStatus {
				case "work":
					dDTO.WorkDays++
				case "absent":
					dDTO.AbsentDays++
				default:
					dDTO.OffDays++
				}
			}
		}

		report.Drivers = append(report.Drivers, dDTO)
	}

	return report, nil
}

// Upsert 登記單筆出勤（使用者在司機月曆手動登記，一律視為人工來源）。
func (s *AttendanceService) Upsert(ctx context.Context, driverID uuid.UUID, recordDate time.Time, status string, note *string, actorID *uuid.UUID, actorRole *string) (*AttendanceRecord, error) {
	item, err := s.attendanceRepo.Upsert(ctx, driverID, recordDate, status, note, "manual")
	if err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    actorID,
			ActorRole:  actorRole,
			Action:     "update",
			EntityType: "attendance_records",
			EntityID:   strPtr(item.ID.String()),
			AfterData:  item,
		})
	}

	return item, nil
}

// importedAttendanceStatus 是司機接送匯報比對到司機時，該日視為的出勤狀態。
const importedAttendanceStatus = "work"

// SyncFromImport 依司機接送匯報比對到的司機與服務日期同步出勤登記：
//   - 當天尚無出勤紀錄：直接登記為出勤(work)，來源標記為 import。
//   - 當天既有紀錄本身就是上次匯入同步寫入的（source=import）：直接刷新，不算衝突。
//   - 當天既有人工登記且狀態剛好也是出勤：與匯入結果一致，不需處理。
//   - 當天既有人工登記且狀態不同：不覆蓋人工判斷，改記一筆待維護衝突讓使用者決定。
func (s *AttendanceService) SyncFromImport(ctx context.Context, driverID uuid.UUID, serviceDate time.Time) error {
	existing, err := s.attendanceRepo.GetOne(ctx, driverID, serviceDate)
	if err != nil {
		return fmt.Errorf("failed to check existing attendance record: %w", err)
	}

	if existing == nil || existing.Source == "import" {
		_, err := s.attendanceRepo.Upsert(ctx, driverID, serviceDate, importedAttendanceStatus, nil, "import")
		return err
	}
	if existing.Status == importedAttendanceStatus {
		return nil
	}
	return s.attendanceRepo.UpsertConflict(ctx, driverID, serviceDate, existing.Status, importedAttendanceStatus)
}

// ListConflicts 查詢目前待處理的出勤待維護衝突。
func (s *AttendanceService) ListConflicts(ctx context.Context) ([]AttendanceConflictDTO, error) {
	conflicts, err := s.attendanceRepo.ListConflicts(ctx, "pending")
	if err != nil {
		return nil, fmt.Errorf("failed to list attendance import conflicts: %w", err)
	}

	out := make([]AttendanceConflictDTO, 0, len(conflicts))
	for _, c := range conflicts {
		out = append(out, toAttendanceConflictDTO(c))
	}
	return out, nil
}

// ResolveConflict 依使用者選擇解決一筆出勤待維護衝突：keep_manual 保留原本人工登記，
// use_import 改採匯入判斷的出勤(work) 覆蓋人工登記。
func (s *AttendanceService) ResolveConflict(ctx context.Context, id uuid.UUID, choice string, actorID *uuid.UUID, actorRole *string) (*AttendanceConflictDTO, error) {
	if choice != "keep_manual" && choice != "use_import" {
		return nil, fmt.Errorf("invalid resolve choice: %s", choice)
	}

	conflict, err := s.attendanceRepo.GetConflict(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get attendance import conflict: %w", err)
	}
	if conflict == nil {
		return nil, ErrAttendanceConflictNotFound
	}

	if choice == "use_import" {
		if _, err := s.attendanceRepo.Upsert(ctx, conflict.DriverID, conflict.RecordDate, conflict.ImportedStatus, nil, "import"); err != nil {
			return nil, err
		}
	}

	if err := s.attendanceRepo.ResolveConflict(ctx, id, choice, actorID); err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    actorID,
			ActorRole:  actorRole,
			Action:     "resolve",
			EntityType: "attendance_import_conflicts",
			EntityID:   strPtr(id.String()),
			AfterData:  map[string]string{"choice": choice},
		})
	}

	conflict.Status = "resolved"
	conflict.ResolvedChoice = &choice
	dto := toAttendanceConflictDTO(*conflict)
	return &dto, nil
}

func toAttendanceConflictDTO(c AttendanceImportConflict) AttendanceConflictDTO {
	return AttendanceConflictDTO{
		ID:             c.ID.String(),
		DriverID:       c.DriverID.String(),
		DriverName:     c.DriverName,
		RecordDate:     c.RecordDate.Format("2006-01-02"),
		ExistingStatus: c.ExistingStatus,
		ImportedStatus: c.ImportedStatus,
		Status:         c.Status,
		ResolvedChoice: c.ResolvedChoice,
	}
}
