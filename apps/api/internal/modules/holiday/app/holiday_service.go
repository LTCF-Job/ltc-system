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

// HolidayRecord 代表政府行事曆來源回傳的單筆假日資料，供 provider port
// 使用；由 service 負責轉換為 repository 的 persistence row。
type HolidayRecord struct {
	HolidayDate time.Time
	Name        string
	Source      string
	IsDayOff    bool
}

// GovernmentHolidayProvider 代表政府行事曆來源。
type GovernmentHolidayProvider interface {
	Fetch(context.Context, int) ([]HolidayRecord, error)
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

// UpsertHolidayInput 代表新增或更新單一國定假日所需之輸入。
type UpsertHolidayInput struct {
	HolidayDate time.Time
	Name        string
	Region      *string
	Source      string
	IsDayOff    bool
}

func (s *HolidayService) UpsertHoliday(ctx context.Context, in UpsertHolidayInput, actorID uuid.UUID, actorRole string) (*repository.HolidayEntity, error) {
	source := in.Source
	if source == "" {
		source = "manual"
	}
	h := &repository.HolidayEntity{
		HolidayDate: in.HolidayDate,
		Name:        in.Name,
		Region:      in.Region,
		Source:      source,
		IsDayOff:    in.IsDayOff,
	}
	if err := s.repo.Upsert(ctx, h); err != nil {
		return nil, err
	}
	if s.auditRepo != nil {
		dateStr := h.HolidayDate.Format("2006-01-02")
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{ActorID: &actorID, ActorRole: &actorRole, Action: "create", EntityType: "holiday", EntityID: &dateStr, AfterData: h})
	}
	return h, nil
}

// ImportTaiwanGovHolidays 取得並冪等儲存指定年度的政府行事曆。
func (s *HolidayService) ImportTaiwanGovHolidays(ctx context.Context, year int, actorID uuid.UUID, actorRole string) (int, error) {
	if year < 2000 || year > 2100 {
		return 0, fmt.Errorf("invalid holiday year %d", year)
	}
	if s.provider == nil {
		return 0, fmt.Errorf("government holiday provider is not configured")
	}
	records, err := s.provider.Fetch(ctx, year)
	if err != nil {
		return 0, fmt.Errorf("fetch government holidays for %d: %w", year, err)
	}
	holidays := make([]repository.HolidayEntity, 0, len(records))
	for _, rec := range records {
		if rec.HolidayDate.Year() != year {
			return 0, fmt.Errorf("government holiday date %s is outside year %d", rec.HolidayDate.Format("2006-01-02"), year)
		}
		holidays = append(holidays, repository.HolidayEntity{
			HolidayDate: rec.HolidayDate,
			Name:        rec.Name,
			Source:      "gov_calendar",
			IsDayOff:    rec.IsDayOff,
		})
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
