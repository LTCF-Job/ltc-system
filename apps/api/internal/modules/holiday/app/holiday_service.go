package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Holiday 代表一個國定假日或停駛日。
type Holiday struct {
	HolidayDate time.Time
	Name        string
	Source      string
	IsDayOff    bool
	CreatedAt   time.Time
}

// HolidayAuditSnapshot 是假日異動的明確稽核快照。
type HolidayAuditSnapshot struct {
	HolidayDate time.Time `json:"holidayDate"`
	Name        string    `json:"name"`
	Source      string    `json:"source"`
	IsDayOff    bool      `json:"isDayOff"`
}

// HolidayImportAuditSnapshot 是政府行事曆批次匯入的摘要，不直接保存來源資料列。
type HolidayImportAuditSnapshot struct {
	Year  int `json:"year"`
	Count int `json:"count"`
}

// AuditSnapshot 產生假日的明確稽核快照。
func (h Holiday) AuditSnapshot() HolidayAuditSnapshot {
	return HolidayAuditSnapshot{HolidayDate: h.HolidayDate, Name: h.Name, Source: h.Source, IsDayOff: h.IsDayOff}
}

// AuditEntry 是本模組寫入稽核日誌的內容。
type AuditEntry struct {
	ActorID    *uuid.UUID
	ActorRole  *string
	Action     string
	EntityType string
	EntityID   *string
	BeforeData interface{}
	AfterData  interface{}
}

// AuditWriter 定義假日異動留痕的寫入邊界。
type AuditWriter interface {
	Write(ctx context.Context, e AuditEntry) error
}

// HolidayStore 定義假日同步所需的資料存取邊界。
type HolidayStore interface {
	List(context.Context, time.Time, time.Time) ([]Holiday, error)
	Upsert(context.Context, *Holiday) error
	BatchUpsert(context.Context, []Holiday) error
	Delete(context.Context, time.Time) error
}

// HolidayReader 是可選的既有假日讀取 port，供 delete／upsert 稽核保存 before snapshot。
type HolidayReader interface {
	GetByDate(context.Context, time.Time) (*Holiday, error)
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
	auditRepo AuditWriter
	provider  GovernmentHolidayProvider
}

func NewHolidayService(repo HolidayStore, auditRepo AuditWriter) *HolidayService {
	return &HolidayService{repo: repo, auditRepo: auditRepo}
}

func NewHolidaySyncService(repo HolidayStore, auditRepo AuditWriter, provider GovernmentHolidayProvider) *HolidayService {
	return &HolidayService{repo: repo, auditRepo: auditRepo, provider: provider}
}

func (s *HolidayService) ListHolidays(ctx context.Context, startDate, endDate time.Time) ([]Holiday, error) {
	return s.repo.List(ctx, startDate, endDate)
}

// UpsertHolidayInput 代表新增或更新單一國定假日所需之輸入。
type UpsertHolidayInput struct {
	HolidayDate time.Time
	Name        string
	Source      string
	IsDayOff    bool
}

func (s *HolidayService) UpsertHoliday(ctx context.Context, in UpsertHolidayInput, actorID uuid.UUID, actorRole string) (*Holiday, error) {
	var before *Holiday
	if s.auditRepo != nil {
		if reader, ok := s.repo.(HolidayReader); ok {
			var err error
			before, err = reader.GetByDate(ctx, in.HolidayDate)
			if err != nil {
				return nil, err
			}
		}
	}
	source := in.Source
	if source == "" {
		source = "manual"
	}
	h := &Holiday{
		HolidayDate: in.HolidayDate,
		Name:        in.Name,
		Source:      source,
		IsDayOff:    in.IsDayOff,
	}
	if err := s.repo.Upsert(ctx, h); err != nil {
		return nil, err
	}
	if s.auditRepo != nil {
		dateStr := h.HolidayDate.Format("2006-01-02")
		action := "create"
		var beforeData interface{}
		if before != nil {
			action = "update"
			beforeData = before.AuditSnapshot()
		}
		if err := s.auditRepo.Write(ctx, AuditEntry{ActorID: &actorID, ActorRole: &actorRole, Action: action, EntityType: "holiday", EntityID: &dateStr, BeforeData: beforeData, AfterData: h.AuditSnapshot()}); err != nil {
			slog.Error("holiday audit write failed", "action", action, "entity_id", dateStr, "error", err)
		}
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
	holidays := make([]Holiday, 0, len(records))
	for _, rec := range records {
		if rec.HolidayDate.Year() != year {
			return 0, fmt.Errorf("government holiday date %s is outside year %d", rec.HolidayDate.Format("2006-01-02"), year)
		}
		holidays = append(holidays, Holiday{
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
		if err := s.auditRepo.Write(ctx, AuditEntry{ActorID: &actorID, ActorRole: &actorRole, Action: "import", EntityType: "holiday_calendar", EntityID: &yearID, AfterData: HolidayImportAuditSnapshot{Year: year, Count: len(holidays)}}); err != nil {
			slog.Error("holiday import audit write failed", "year", year, "error", err)
		}
	}
	return len(holidays), nil
}

func (s *HolidayService) DeleteHoliday(ctx context.Context, date time.Time, actorID uuid.UUID, actorRole string) error {
	var before *Holiday
	if s.auditRepo != nil {
		if reader, ok := s.repo.(HolidayReader); ok {
			var err error
			before, err = reader.GetByDate(ctx, date)
			if err != nil {
				return err
			}
		}
	}
	if err := s.repo.Delete(ctx, date); err != nil {
		return err
	}
	if s.auditRepo != nil {
		dateStr := date.Format("2006-01-02")
		var beforeData interface{}
		if before != nil {
			beforeData = before.AuditSnapshot()
		}
		if err := s.auditRepo.Write(ctx, AuditEntry{ActorID: &actorID, ActorRole: &actorRole, Action: "delete", EntityType: "holiday", EntityID: &dateStr, BeforeData: beforeData}); err != nil {
			slog.Error("holiday audit write failed", "action", "delete", "entity_id", dateStr, "error", err)
		}
	}
	return nil
}
