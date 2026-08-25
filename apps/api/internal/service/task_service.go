package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/domain/calendar"
	"ltc-system/apps/api/internal/domain/rocdate"
	"ltc-system/apps/api/internal/repository"
)

// MissingRideItem 代表未回報之搭乘趟次。
type MissingRideItem struct {
	CaseID      uuid.UUID `json:"caseId"`
	CaseCode    string    `json:"caseCode"`
	CaseName    string    `json:"caseName"`
	Region      string    `json:"region"`
	ServiceDate string    `json:"serviceDate"` // YYYY-MM-DD
	LegSeq      int16     `json:"legSeq"`
	Direction   string    `json:"direction"`
	DepartTime  string    `json:"departTime"`
	VehicleID   *uuid.UUID `json:"vehicleId,omitempty"`
}

// MonthEndSummary 代表月底申報檢核與提醒彙整。
type MonthEndSummary struct {
	YearMonth       string `json:"yearMonth"` // 例: 115-07
	TotalRides      int    `json:"totalRides"`
	BoardedRides    int    `json:"boardedRides"`
	UnreportedRides int    `json:"unreportedRides"`
	ConflictCount   int    `json:"conflictCount"`
	PrecheckErrors  int    `json:"precheckErrors"`
}

// TaskService 負責處理定期與後台非同步排程任務。
type TaskService struct {
	db              *pgxpool.Pool
	caseRepo        *repository.CaseRepository
	holidayRepo     *repository.HolidayRepository
	notificationSvc *NotificationService
}

// NewTaskService 建立 TaskService 實例。
func NewTaskService(
	db *pgxpool.Pool,
	caseRepo *repository.CaseRepository,
	holidayRepo *repository.HolidayRepository,
	notificationSvc *NotificationService,
) *TaskService {
	return &TaskService{
		db:              db,
		caseRepo:        caseRepo,
		holidayRepo:     holidayRepo,
		notificationSvc: notificationSvc,
	}
}

// CheckMissingReports 比對特定日期應搭乘日曆與實際搭乘紀錄，偵測未回報趟次並觸發告警通知。
func (s *TaskService) CheckMissingReports(ctx context.Context, targetDate time.Time, region string) ([]MissingRideItem, error) {
	year := targetDate.Year()
	month := int(targetDate.Month())
	dateStr := targetDate.Format("2006-01-02")

	holidayMap, err := s.holidayRepo.GetHolidayMap(ctx, year, month, region)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch holidays: %w", err)
	}

	schedules, err := s.caseRepo.GetActiveSchedulesForMonth(ctx, year, month, region)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch monthly schedules: %w", err)
	}

	var expectedList []MissingRideItem
	for _, sch := range schedules {
		var legs []calendar.LegInput
		legVehicleMap := make(map[int16]*uuid.UUID)
		for _, l := range sch.Legs {
			legs = append(legs, calendar.LegInput{
				LegSeq:     l.LegSeq,
				Direction:  l.Direction,
				DepartTime: l.DepartTime,
			})
			legVehicleMap[l.LegSeq] = l.VehicleID
		}

		calInput := calendar.CaseScheduleCalendarInput{
			CaseID:         sch.CaseID,
			ClaimStartDate: sch.ClaimStartDate,
			ClaimEndDate:   sch.ClaimEndDate,
			EffectiveFrom:  sch.EffectiveFrom,
			EffectiveTo:    sch.EffectiveTo,
			Weekdays:       sch.Weekdays,
			SiteOpenDays:   sch.SiteOpenDays,
			Holidays:       holidayMap,
			Legs:           legs,
		}

		expectedRides := calendar.CalculateExpectedRides(year, month, calInput)
		for _, er := range expectedRides {
			if er.ServiceDate.Format("2006-01-02") == dateStr {
				expectedList = append(expectedList, MissingRideItem{
					CaseID:      sch.CaseID,
					CaseCode:    sch.CaseCode,
					CaseName:    sch.CaseName,
					Region:      sch.Region,
					ServiceDate: dateStr,
					LegSeq:      er.LegSeq,
					Direction:   er.Direction,
					DepartTime:  er.DepartTime,
					VehicleID:   legVehicleMap[er.LegSeq],
				})
			}
		}
	}

	// 本機離線測試降級支援
	if s.db == nil {
		return expectedList, nil
	}

	reportedQuery := `
		SELECT case_id, leg_seq
		FROM ride_records
		WHERE service_date = $1 AND effective_status != 'unreported'
	`
	rows, err := s.db.Query(ctx, reportedQuery, targetDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query reported rides: %w", err)
	}
	defer rows.Close()

	type key struct {
		caseID uuid.UUID
		legSeq int16
	}
	reportedSet := make(map[key]bool)
	for rows.Next() {
		var k key
		if err := rows.Scan(&k.caseID, &k.legSeq); err == nil {
			reportedSet[k] = true
		}
	}

	var missingList []MissingRideItem
	for _, item := range expectedList {
		if !reportedSet[key{caseID: item.CaseID, legSeq: item.LegSeq}] {
			missingList = append(missingList, item)
		}
	}

	// 存在未回報趟次時主動發送通報
	if len(missingList) > 0 && s.notificationSvc != nil {
		subject := fmt.Sprintf("【長照接送未回報告警】%s 共有 %d 筆趟次尚未回報", dateStr, len(missingList))
		body := fmt.Sprintf("日期：%s\n未回報趟數：%d 筆\n請相關人員至系統「異常集中處理」或「未回報清單」確認司機填報狀況。", dateStr, len(missingList))
		_ = s.notificationSvc.SendNotification(ctx, "missing_report", subject, body)
		slog.Info("Missing report notification triggered", slog.String("date", dateStr), slog.Int("missing_count", len(missingList)))
	}

	return missingList, nil
}

// MonthEndReminder 執行每月 26 日申報提醒檢查與發信通知。
func (s *TaskService) MonthEndReminder(ctx context.Context, year, month int) (*MonthEndSummary, error) {
	rocYM := rocdate.FormatROCYearMonth(year, month)
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)

	summary := &MonthEndSummary{
		YearMonth: rocYM,
	}

	if s.db != nil {
		query := `
			SELECT 
				COUNT(*) as total,
				COUNT(CASE WHEN effective_status = 'boarded' THEN 1 END) as boarded,
				COUNT(CASE WHEN effective_status = 'unreported' THEN 1 END) as unreported,
				COUNT(CASE WHEN has_conflict = true THEN 1 END) as conflicts
			FROM ride_records
			WHERE service_date >= $1 AND service_date <= $2
		`
		_ = s.db.QueryRow(ctx, query, firstDay, lastDay).Scan(
			&summary.TotalRides, &summary.BoardedRides, &summary.UnreportedRides, &summary.ConflictCount,
		)
	}

	if s.notificationSvc != nil {
		subject := fmt.Sprintf("【長照申報月底提醒】%s 申報進度與異常檢查", rocYM)
		body := fmt.Sprintf(
			"月份：%s\n總搭乘紀錄：%d 筆\n已確認搭乘：%d 筆\n未回報筆數：%d 筆\n混車衝突筆數：%d 筆\n\n申報期限將近，請至系統檢查前置檢核項目並完成 33 欄申報檔案匯出作業。",
			rocYM, summary.TotalRides, summary.BoardedRides, summary.UnreportedRides, summary.ConflictCount,
		)
		_ = s.notificationSvc.SendNotification(ctx, "month_end", subject, body)
		slog.Info("Month-end reminder notification triggered", slog.String("month", rocYM))
	}

	return summary, nil
}
