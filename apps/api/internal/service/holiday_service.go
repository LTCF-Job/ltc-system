package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/repository"
)

// HolidayService 提供國定假日與行事曆維護業務邏輯。
type HolidayService struct {
	repo      *repository.HolidayRepository
	auditRepo *repository.AuditRepository
}

// NewHolidayService 建立 HolidayService 實例。
func NewHolidayService(repo *repository.HolidayRepository, auditRepo *repository.AuditRepository) *HolidayService {
	return &HolidayService{
		repo:      repo,
		auditRepo: auditRepo,
	}
}

// ListHolidays 取得特定區間之國定假日。
func (s *HolidayService) ListHolidays(ctx context.Context, startDate, endDate time.Time, region string) ([]repository.HolidayEntity, error) {
	return s.repo.List(ctx, startDate, endDate, region)
}

// UpsertHoliday 新增或更新單一假日。
func (s *HolidayService) UpsertHoliday(ctx context.Context, h *repository.HolidayEntity, actorID uuid.UUID, actorRole string) error {
	if err := s.repo.Upsert(ctx, h); err != nil {
		return err
	}

	if s.auditRepo != nil {
		dateStr := h.HolidayDate.Format("2006-01-02")
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "create",
			EntityType: "holiday",
			EntityID:   &dateStr,
			AfterData:  h,
		})
	}
	return nil
}

// ImportTaiwanGovHolidays 匯入特定年份之台灣國定假日基準資料。
func (s *HolidayService) ImportTaiwanGovHolidays(ctx context.Context, year int, actorID uuid.UUID, actorRole string) (int, error) {
	// 定義標準台灣國定假日清單 (以 2026/2027 年範例)
	holidays := []repository.HolidayEntity{
		{HolidayDate: time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC), Name: "中華民國開國紀念日", Source: "gov_calendar"},
		{HolidayDate: time.Date(year, 2, 28, 0, 0, 0, 0, time.UTC), Name: "和平紀念日", Source: "gov_calendar"},
		{HolidayDate: time.Date(year, 4, 4, 0, 0, 0, 0, time.UTC), Name: "兒童節", Source: "gov_calendar"},
		{HolidayDate: time.Date(year, 4, 5, 0, 0, 0, 0, time.UTC), Name: "民族掃墓節(清明)", Source: "gov_calendar"},
		{HolidayDate: time.Date(year, 5, 1, 0, 0, 0, 0, time.UTC), Name: "勞動節", Source: "gov_calendar"},
		{HolidayDate: time.Date(year, 10, 10, 0, 0, 0, 0, time.UTC), Name: "國慶日", Source: "gov_calendar"},
	}

	// 特殊農曆對應 (如 2026 年)
	if year == 2026 {
		holidays = append(holidays,
			repository.HolidayEntity{HolidayDate: time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC), Name: "農曆除夕", Source: "gov_calendar"},
			repository.HolidayEntity{HolidayDate: time.Date(2026, 2, 17, 0, 0, 0, 0, time.UTC), Name: "春節(初一)", Source: "gov_calendar"},
			repository.HolidayEntity{HolidayDate: time.Date(2026, 2, 18, 0, 0, 0, 0, time.UTC), Name: "春節(初二)", Source: "gov_calendar"},
			repository.HolidayEntity{HolidayDate: time.Date(2026, 2, 19, 0, 0, 0, 0, time.UTC), Name: "春節(初三)", Source: "gov_calendar"},
			repository.HolidayEntity{HolidayDate: time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC), Name: "端午節", Source: "gov_calendar"},
			repository.HolidayEntity{HolidayDate: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC), Name: "中秋節", Source: "gov_calendar"},
		)
	}

	if err := s.repo.BatchUpsert(ctx, holidays); err != nil {
		return 0, err
	}

	if s.auditRepo != nil {
		yearStr := string(rune(year))
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "import",
			EntityType: "holiday_calendar",
			EntityID:   &yearStr,
			AfterData:  map[string]interface{}{"count": len(holidays), "year": year},
		})
	}

	return len(holidays), nil
}

// DeleteHoliday 刪除特定日期之假日。
func (s *HolidayService) DeleteHoliday(ctx context.Context, date time.Time, actorID uuid.UUID, actorRole string) error {
	if err := s.repo.Delete(ctx, date); err != nil {
		return err
	}

	if s.auditRepo != nil {
		dateStr := date.Format("2006-01-02")
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "delete",
			EntityType: "holiday",
			EntityID:   &dateStr,
		})
	}
	return nil
}
