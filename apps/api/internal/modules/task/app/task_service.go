package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/domain/calendar"
	"ltc-system/apps/api/internal/domain/rocdate"
)

// MissingRideItem 代表未回報之搭乘趟次。
type MissingRideItem struct {
	CaseID      uuid.UUID  `json:"caseId"`
	CaseName    string     `json:"caseName"`
	ServiceDate string     `json:"serviceDate"` // YYYY-MM-DD
	LegSeq      int16      `json:"legSeq"`
	Direction   string     `json:"direction"`
	DepartTime  string     `json:"departTime"`
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
	taskRepo        TaskStore
	caseRepo        MonthScheduleReader
	holidayRepo     HolidayMapReader
	notificationSvc Notifier
}

// NewTaskService 建立 TaskService 實例。
func NewTaskService(
	taskRepo TaskStore,
	caseRepo MonthScheduleReader,
	holidayRepo HolidayMapReader,
	notificationSvc Notifier,
) *TaskService {
	return &TaskService{
		taskRepo:        taskRepo,
		caseRepo:        caseRepo,
		holidayRepo:     holidayRepo,
		notificationSvc: notificationSvc,
	}
}

// listMissingReports 是比對邏輯的純查詢版本：onlyDate 給定時只回傳該日，否則回傳整月。
func (s *TaskService) listMissingReports(ctx context.Context, year, month int, onlyDate *time.Time) ([]MissingRideItem, error) {
	holidayMap, err := s.holidayRepo.GetHolidayMap(ctx, year, month)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch holidays: %w", err)
	}

	schedules, err := s.caseRepo.GetActiveSchedulesForMonth(ctx, year, month)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch monthly schedules: %w", err)
	}

	var onlyDateStr string
	if onlyDate != nil {
		onlyDateStr = onlyDate.Format("2006-01-02")
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
			CaseID:        sch.CaseID,
			ClaimEndDate:  sch.ClaimEndDate,
			EffectiveFrom: sch.EffectiveFrom,
			EffectiveTo:   sch.EffectiveTo,
			Weekdays:      sch.Weekdays,
			Holidays:      holidayMap,
			Legs:          legs,
		}

		expectedRides, err := calendar.CalculateExpectedRides(year, month, calInput)
		if err != nil {
			return nil, fmt.Errorf("calculate expected rides for case %s: %w", sch.CaseID, err)
		}
		for _, er := range expectedRides {
			dateStr := er.ServiceDate.Format("2006-01-02")
			if onlyDateStr != "" && dateStr != onlyDateStr {
				continue
			}
			expectedList = append(expectedList, MissingRideItem{
				CaseID:      sch.CaseID,
				CaseName:    sch.CaseName,
				ServiceDate: dateStr,
				LegSeq:      er.LegSeq,
				Direction:   er.Direction,
				DepartTime:  er.DepartTime,
				VehicleID:   legVehicleMap[er.LegSeq],
			})
		}
	}

	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)
	reportedSlots, err := s.taskRepo.GetReportedRideSlotsInRange(ctx, firstDay, lastDay)
	if err != nil {
		return nil, fmt.Errorf("failed to query reported rides: %w", err)
	}

	type key struct {
		caseID uuid.UUID
		date   string
		legSeq int16
	}
	reportedSet := make(map[key]bool)
	for _, slot := range reportedSlots {
		reportedSet[key{caseID: slot.CaseID, date: slot.ServiceDate.Format("2006-01-02"), legSeq: slot.LegSeq}] = true
	}

	var missingList []MissingRideItem
	for _, item := range expectedList {
		if !reportedSet[key{caseID: item.CaseID, date: item.ServiceDate, legSeq: item.LegSeq}] {
			missingList = append(missingList, item)
		}
	}
	return missingList, nil
}

// MissingReportCheckResult 代表未回報檢核結果，包含通知是否成功送出。檢核本身（比對搭乘日曆與
// 實際紀錄）與催報通知寄送是兩件獨立的事：通知寄送失敗不代表檢核失敗，已算出的未回報清單仍應
// 完整回傳，不能因為通知這個 best-effort 動作出錯就把整批檢核結果一起丟掉。
type MissingReportCheckResult struct {
	Items []MissingRideItem
	// NotificationAttempted 代表本次是否嘗試過寄送催報通知（有未回報項目且已設定通知服務）。
	NotificationAttempted bool
	// NotificationSent 只有在 NotificationAttempted 為 true 時才有意義，代表通知是否成功送出。
	NotificationSent bool
}

// CheckMissingReports 比對特定日期應搭乘日曆與實際搭乘紀錄，偵測未回報趟次並嘗試觸發告警通知；
// 通知寄送失敗會降級為 warning 記錄於伺服器端 log，不會讓已算出的未回報清單跟著遺失。
func (s *TaskService) CheckMissingReports(ctx context.Context, targetDate time.Time) (*MissingReportCheckResult, error) {
	year := targetDate.Year()
	month := int(targetDate.Month())
	dateStr := targetDate.Format("2006-01-02")

	missingList, err := s.listMissingReports(ctx, year, month, &targetDate)
	if err != nil {
		return nil, err
	}

	result := &MissingReportCheckResult{Items: missingList}

	// 存在未回報趟次時主動發送通報
	if len(missingList) > 0 && s.notificationSvc != nil {
		result.NotificationAttempted = true
		subject := fmt.Sprintf("【長照接送未回報告警】%s 共有 %d 筆趟次尚未回報", dateStr, len(missingList))
		body := fmt.Sprintf("日期：%s\n未回報趟數：%d 筆\n請相關人員至系統「異常集中處理」或「未回報清單」確認司機填報狀況。", dateStr, len(missingList))
		if err := s.sendDeduplicatedNotification(ctx, "missing_report", subject, body, "missing_report:"+dateStr); err != nil {
			slog.Error("failed to send missing report notification", slog.String("date", dateStr), slog.Int("missing_count", len(missingList)), slog.Any("error", err))
			result.NotificationSent = false
		} else {
			result.NotificationSent = true
			slog.Info("Missing report notification triggered", slog.String("date", dateStr), slog.Int("missing_count", len(missingList)))
		}
	}

	return result, nil
}

// ListMissingReports 只查詢指定日期的未回報資料，不觸發通知或其他副作用。
func (s *TaskService) ListMissingReports(ctx context.Context, targetDate time.Time) ([]MissingRideItem, error) {
	return s.listMissingReports(ctx, targetDate.Year(), int(targetDate.Month()), &targetDate)
}

// ListMissingReportsForMonth 回傳整月未回報趟次，不觸發告警通知（供「異常集中處理」頁面查詢用）。
func (s *TaskService) ListMissingReportsForMonth(ctx context.Context, year, month int) ([]MissingRideItem, error) {
	return s.listMissingReports(ctx, year, month, nil)
}

// MonthEndReminder 執行每月 26 日申報提醒檢查與發信通知。
func (s *TaskService) MonthEndReminder(ctx context.Context, year, month int) (*MonthEndSummary, error) {
	rocYM := rocdate.FormatROCYearMonth(year, month)
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)

	summary := &MonthEndSummary{
		YearMonth: rocYM,
	}

	stats, err := s.taskRepo.GetMonthEndRideStats(ctx, firstDay, lastDay)
	if err != nil {
		return nil, fmt.Errorf("failed to get month-end ride stats: %w", err)
	}
	summary.TotalRides = stats.TotalRides
	summary.BoardedRides = stats.BoardedRides
	summary.UnreportedRides = stats.UnreportedRides
	summary.ConflictCount = stats.ConflictCount

	if s.notificationSvc != nil {
		subject := fmt.Sprintf("【長照申報月底提醒】%s 申報進度與異常檢查", rocYM)
		body := fmt.Sprintf(
			"月份：%s\n總搭乘紀錄：%d 筆\n已確認搭乘：%d 筆\n未回報筆數：%d 筆\n混車衝突筆數：%d 筆\n\n申報期限將近，請至系統檢查前置檢核項目並完成 33 欄申報檔案匯出作業。",
			rocYM, summary.TotalRides, summary.BoardedRides, summary.UnreportedRides, summary.ConflictCount,
		)
		err := s.sendDeduplicatedNotification(ctx, "month_end", subject, body, "month_end:"+rocYM)
		if err != nil {
			return nil, fmt.Errorf("failed to send month-end notification: %w", err)
		}
		slog.Info("Month-end reminder notification triggered", slog.String("month", rocYM))
	}

	return summary, nil
}

func (s *TaskService) sendDeduplicatedNotification(ctx context.Context, topic, subject, body, dedupKey string) error {
	if dedup, ok := s.notificationSvc.(interface {
		SendNotificationDedup(context.Context, string, string, string, string) error
	}); ok {
		return dedup.SendNotificationDedup(ctx, topic, subject, body, dedupKey)
	}
	return s.notificationSvc.SendNotification(ctx, topic, subject, body)
}
