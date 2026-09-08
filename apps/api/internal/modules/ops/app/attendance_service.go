package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/rocdate"
	"ltc-system/apps/api/internal/platform/clock"
)

// ErrAttendanceConflictNotFound 代表查無指定的出勤待維護衝突。
var ErrAttendanceConflictNotFound = errors.New("attendance import conflict not found")

// ErrInvalidAttendanceMonth 代表出勤月報的期別格式或月份不合法。
var ErrInvalidAttendanceMonth = errors.New("invalid attendance month")

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
	txRunner       TxRunner
	businessClock  clock.Clock
}

// AttendanceOption 調整出勤服務的交易與業務時間來源。
type AttendanceOption func(*AttendanceService)

// WithAttendanceTxRunner 將需要跨多次寫入的出勤操作包進同一筆交易。
func WithAttendanceTxRunner(runner TxRunner) AttendanceOption {
	return func(s *AttendanceService) { s.txRunner = runner }
}

// WithAttendanceClock 注入臺灣業務時間，讓跨日與月曆預設狀態可穩定測試。
func WithAttendanceClock(c clock.Clock) AttendanceOption {
	return func(s *AttendanceService) { s.businessClock = c }
}

// NewAttendanceService 建立 AttendanceService 實例。
func NewAttendanceService(
	attendanceRepo AttendanceStore,
	driverRepo DriverLister,
	auditRepo AuditWriter,
	holidayRepo HolidayReader,
	options ...AttendanceOption,
) *AttendanceService {
	service := &AttendanceService{
		attendanceRepo: attendanceRepo,
		driverRepo:     driverRepo,
		auditRepo:      auditRepo,
		holidayRepo:    holidayRepo,
	}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *AttendanceService) today() time.Time {
	if s.businessClock != nil {
		return s.businessClock.Today()
	}
	return clock.Today()
}

// GetMonthAttendance 查詢指定月份司機月曆出勤矩陣；queries[0] 為司機姓名搜尋字串。
func (s *AttendanceService) GetMonthAttendance(ctx context.Context, periodYm string, driverID *uuid.UUID, queries ...string) (*MonthAttendanceReportDTO, error) {
	startDate, endDate, daysInMonth, err := rocdate.MonthRangeStrict(periodYm)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidAttendanceMonth, err)
	}

	if s.driverRepo == nil {
		return nil, errors.New("driver repository is not configured")
	}
	keyword := ""
	if len(queries) > 0 {
		keyword = strings.TrimSpace(strings.ToLower(queries[0]))
	}
	var drivers []DriverRef
	if queryLister, ok := s.driverRepo.(interface {
		ListAllActiveByQuery(context.Context, string) ([]DriverRef, error)
	}); ok {
		drivers, err = queryLister.ListAllActiveByQuery(ctx, keyword)
	} else {
		drivers, err = s.driverRepo.ListAllActive(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list active drivers: %w", err)
	}
	if keyword != "" {
		filtered := make([]DriverRef, 0, len(drivers))
		for _, d := range drivers {
			if strings.Contains(strings.ToLower(d.Name), keyword) {
				filtered = append(filtered, d)
			}
		}
		drivers = filtered
	}

	records, err := s.attendanceRepo.GetMonthRecords(ctx, startDate, endDate, driverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get month attendance records: %w", err)
	}

	if s.holidayRepo == nil {
		return nil, errors.New("holiday repository is not configured")
	}
	holidayMap, err := s.holidayRepo.GetHolidayMap(ctx, startDate.Year(), int(startDate.Month()))
	if err != nil {
		return nil, fmt.Errorf("failed to get holiday map: %w", err)
	}
	today := s.today()

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
			Days:       make(map[string]DriverDayAttendanceDTO),
		}

		dRecords := recordMap[d.ID.String()]
		for day := 1; day <= daysInMonth; day++ {
			// 日期判斷必須與 business clock 使用相同時區；若以 UTC 午夜比較，
			// 臺灣凌晨的「今天」會被誤判為尚未到來。
			dayDate := time.Date(startDate.Year(), startDate.Month(), day, 0, 0, 0, 0, today.Location())
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
func (s *AttendanceService) Upsert(ctx context.Context, driverID uuid.UUID, recordDate time.Time, status string, note *string, actorID *uuid.UUID, actorRole *string, auditContexts ...AuditContext) (*AttendanceRecord, error) {
	var before interface{}
	if s.auditRepo != nil {
		existing, err := s.attendanceRepo.GetOne(ctx, driverID, recordDate)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			before = existing.AuditSnapshot()
		}
	}
	item, err := s.attendanceRepo.Upsert(ctx, driverID, recordDate, status, note, "manual")
	if err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    actorID,
			ActorRole:  actorRole,
			Action:     "update",
			EntityType: "attendance_records",
			EntityID:   strPtr(item.ID.String()),
			BeforeData: before,
			AfterData:  item.AuditSnapshot(),
			IPAddress:  auditContextOrEmpty(auditContexts).IPAddress,
			UserAgent:  auditContextOrEmpty(auditContexts).UserAgent,
		}); err != nil {
			slog.Warn("attendance_audit_write_failed", slog.String("action", "update"), slog.String("record_id", item.ID.String()), slog.Any("error", err))
		}
	}

	return item, nil
}

// importedAttendanceStatus 是司機接送匯報比對到司機時，該日視為的出勤狀態。
const importedAttendanceStatus = "work"

// SyncFromImport 依司機接送匯報比對結果同步該司機當日出勤登記，人工登記優先於匯入結果。
func (s *AttendanceService) SyncFromImport(ctx context.Context, driverID uuid.UUID, serviceDate time.Time) error {
	existing, err := s.attendanceRepo.GetOne(ctx, driverID, serviceDate)
	if err != nil {
		return fmt.Errorf("failed to check existing attendance record: %w", err)
	}

	// 尚無紀錄或前次同樣來自匯入時，直接以匯入結果登記／刷新
	if existing == nil || existing.Source == "import" {
		_, err := s.attendanceRepo.Upsert(ctx, driverID, serviceDate, importedAttendanceStatus, nil, "import")
		return err
	}
	// 人工登記與匯入結果一致時不需處理
	if existing.Status == importedAttendanceStatus {
		return nil
	}
	// 人工登記與匯入結果不同時不覆蓋人工判斷，改記一筆待維護衝突
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

// IgnoreConflict 忽略一筆出勤待維護衝突：直接刪除衝突列，出勤紀錄維持原本的人工登記值。
// 下次匯入若仍判斷出同一組差異會重新產生，屬預期行為。
func (s *AttendanceService) IgnoreConflict(ctx context.Context, id uuid.UUID, actorID *uuid.UUID, actorRole *string, auditContexts ...AuditContext) error {
	return s.runInTx(ctx, func(txCtx context.Context) error {
		conflict, err := s.attendanceRepo.GetConflict(txCtx, id)
		if err != nil {
			return fmt.Errorf("failed to get attendance import conflict: %w", err)
		}
		if conflict == nil {
			return ErrAttendanceConflictNotFound
		}

		if err := s.attendanceRepo.DeleteConflict(txCtx, id); err != nil {
			return err
		}

		if s.auditRepo != nil {
			if err := s.auditRepo.Write(txCtx, AuditEntry{
				ActorID:    actorID,
				ActorRole:  actorRole,
				Action:     "ignore",
				EntityType: "attendance_import_conflicts",
				EntityID:   strPtr(id.String()),
				BeforeData: conflict.AuditSnapshot(),
				IPAddress:  auditContextOrEmpty(auditContexts).IPAddress,
				UserAgent:  auditContextOrEmpty(auditContexts).UserAgent,
			}); err != nil {
				return fmt.Errorf("failed to write attendance conflict ignore audit: %w", err)
			}
		}
		return nil
	})
}

// ResolveConflict 依使用者選擇解決一筆出勤待維護衝突：keep_manual 保留原本人工登記，
// use_import 改採匯入判斷的出勤(work) 覆蓋人工登記。
func (s *AttendanceService) ResolveConflict(ctx context.Context, id uuid.UUID, choice string, actorID *uuid.UUID, actorRole *string, auditContexts ...AuditContext) (*AttendanceConflictDTO, error) {
	if choice != "keep_manual" && choice != "use_import" {
		return nil, fmt.Errorf("invalid resolve choice: %s", choice)
	}

	var conflict *AttendanceImportConflict
	txErr := s.runInTx(ctx, func(txCtx context.Context) error {
		var err error
		conflict, err = s.attendanceRepo.GetConflict(txCtx, id)
		if err != nil {
			return fmt.Errorf("failed to get attendance import conflict: %w", err)
		}
		if conflict == nil {
			return ErrAttendanceConflictNotFound
		}

		if choice == "use_import" {
			if _, err := s.attendanceRepo.Upsert(txCtx, conflict.DriverID, conflict.RecordDate, conflict.ImportedStatus, nil, "import"); err != nil {
				return err
			}
		}

		if err := s.attendanceRepo.ResolveConflict(txCtx, id, choice, actorID); err != nil {
			return err
		}

		if s.auditRepo != nil {
			if err := s.auditRepo.Write(txCtx, AuditEntry{
				ActorID:    actorID,
				ActorRole:  actorRole,
				Action:     "resolve",
				EntityType: "attendance_import_conflicts",
				EntityID:   strPtr(id.String()),
				BeforeData: conflict.AuditSnapshot(),
				AfterData:  AttendanceConflictResolutionAuditSnapshot{ConflictID: id, Choice: choice},
				IPAddress:  auditContextOrEmpty(auditContexts).IPAddress,
				UserAgent:  auditContextOrEmpty(auditContexts).UserAgent,
			}); err != nil {
				return fmt.Errorf("failed to write attendance conflict audit: %w", err)
			}
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	conflict.Status = "resolved"
	conflict.ResolvedChoice = &choice
	dto := toAttendanceConflictDTO(*conflict)
	return &dto, nil
}

func (s *AttendanceService) runInTx(ctx context.Context, fn func(context.Context) error) error {
	if s.txRunner == nil {
		return fn(ctx)
	}
	return s.txRunner.WithTx(ctx, fn)
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
