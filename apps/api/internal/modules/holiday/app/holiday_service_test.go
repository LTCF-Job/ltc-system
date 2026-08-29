package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"ltc-system/apps/api/internal/repository"
)

type holidayProviderStub struct{}

func (holidayProviderStub) Fetch(context.Context, int) ([]HolidayRecord, error) {
	return []HolidayRecord{
		{HolidayDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Name: "元旦", IsDayOff: true},
		{HolidayDate: time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC), Name: "和平紀念日", IsDayOff: true},
		{HolidayDate: time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC), Name: "兒童節", IsDayOff: true},
		{HolidayDate: time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC), Name: "清明節", IsDayOff: true},
		{HolidayDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Name: "勞動節", IsDayOff: true},
		{HolidayDate: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), Name: "國慶日", IsDayOff: true},
	}, nil
}

func TestHolidayService_ImportTaiwanGovHolidays(t *testing.T) {
	repo := repository.NewHolidayRepository(nil)
	svc := NewHolidaySyncService(repo, nil, holidayProviderStub{})

	ctx := context.Background()
	count, err := svc.ImportTaiwanGovHolidays(ctx, 2026, uuid.New(), "admin")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, 6)
}
