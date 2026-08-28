package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/repository"
)

// HolidayStore 定義假日同步所需的資料存取邊界。
type HolidayStore interface {
	List(context.Context, time.Time, time.Time, string) ([]repository.HolidayEntity, error)
	Upsert(context.Context, *repository.HolidayEntity) error
	BatchUpsert(context.Context, []repository.HolidayEntity) error
	Delete(context.Context, time.Time) error
}

// GovernmentHolidayProvider 代表政府行事曆來源。
type GovernmentHolidayProvider interface {
	Fetch(context.Context, int) ([]repository.HolidayEntity, error)
}

type HolidayService struct {
	repo      HolidayStore
	auditRepo *repository.AuditRepository
	provider  GovernmentHolidayProvider
}

func NewHolidayService(repo *repository.HolidayRepository, auditRepo *repository.AuditRepository) *HolidayService {
	return &HolidayService{repo: repo, auditRepo: auditRepo}
}

func NewHolidaySyncService(repo HolidayStore, auditRepo *repository.AuditRepository, provider GovernmentHolidayProvider) *HolidayService {
	return &HolidayService{repo: repo, auditRepo: auditRepo, provider: provider}
}

func (s *HolidayService) ListHolidays(ctx context.Context, startDate, endDate time.Time, region string) ([]repository.HolidayEntity, error) {
	return s.repo.List(ctx, startDate, endDate, region)
}

func (s *HolidayService) UpsertHoliday(ctx context.Context, h *repository.HolidayEntity, actorID uuid.UUID, actorRole string) error {
	if h.Source == "" {
		h.Source = "manual"
	}
	if err := s.repo.Upsert(ctx, h); err != nil {
		return err
	}
	if s.auditRepo != nil {
		dateStr := h.HolidayDate.Format("2006-01-02")
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{ActorID: &actorID, ActorRole: &actorRole, Action: "create", EntityType: "holiday", EntityID: &dateStr, AfterData: h})
	}
	return nil
}

// ImportTaiwanGovHolidays 取得並冪等儲存指定年度的政府行事曆。
func (s *HolidayService) ImportTaiwanGovHolidays(ctx context.Context, year int, actorID uuid.UUID, actorRole string) (int, error) {
	if year < 2000 || year > 2100 {
		return 0, fmt.Errorf("invalid holiday year %d", year)
	}
	if s.provider == nil {
		return 0, fmt.Errorf("government holiday provider is not configured")
	}
	holidays, err := s.provider.Fetch(ctx, year)
	if err != nil {
		return 0, fmt.Errorf("fetch government holidays for %d: %w", year, err)
	}
	for i := range holidays {
		if holidays[i].HolidayDate.Year() != year {
			return 0, fmt.Errorf("government holiday date %s is outside year %d", holidays[i].HolidayDate.Format("2006-01-02"), year)
		}
		holidays[i].Source = "gov_calendar"
	}
	if err := s.repo.BatchUpsert(ctx, holidays); err != nil {
		return 0, err
	}
	if s.auditRepo != nil {
		yearID := fmt.Sprintf("%d", year)
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{ActorID: &actorID, ActorRole: &actorRole, Action: "import", EntityType: "holiday_calendar", EntityID: &yearID, AfterData: map[string]interface{}{"count": len(holidays), "year": year}})
	}
	return len(holidays), nil
}

func (s *HolidayService) DeleteHoliday(ctx context.Context, date time.Time, actorID uuid.UUID, actorRole string) error {
	if err := s.repo.Delete(ctx, date); err != nil {
		return err
	}
	if s.auditRepo != nil {
		dateStr := date.Format("2006-01-02")
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{ActorID: &actorID, ActorRole: &actorRole, Action: "delete", EntityType: "holiday", EntityID: &dateStr})
	}
	return nil
}
