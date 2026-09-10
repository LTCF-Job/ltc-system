package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
	svc := NewHolidaySyncService(discardHolidayStore{}, nil, holidayProviderStub{})

	ctx := context.Background()
	count, err := svc.ImportTaiwanGovHolidays(ctx, 2026, uuid.New(), "admin")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, 6)
}

// discardHolidayStore 接受任何寫入且回傳空清單，供不驗證持久化的 use case 測試使用。
type discardHolidayStore struct{}

func (discardHolidayStore) List(context.Context, time.Time, time.Time) ([]Holiday, error) {
	return nil, nil
}
func (discardHolidayStore) Upsert(context.Context, *Holiday) error       { return nil }
func (discardHolidayStore) BatchUpsert(context.Context, []Holiday) error { return nil }
func (discardHolidayStore) Delete(context.Context, time.Time) error      { return nil }

// fakeHolidayStore 實作 HolidayReader，供測試手動新增假日的日期衝突檢查。
type fakeHolidayStore struct {
	byDate    map[string]*Holiday
	upsertErr error
	lastWrite *Holiday
}

func newFakeHolidayStore() *fakeHolidayStore {
	return &fakeHolidayStore{byDate: map[string]*Holiday{}}
}

func (f *fakeHolidayStore) List(context.Context, time.Time, time.Time) ([]Holiday, error) {
	return nil, nil
}
func (f *fakeHolidayStore) Upsert(_ context.Context, h *Holiday) error {
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.lastWrite = h
	f.byDate[h.HolidayDate.Format("2006-01-02")] = h
	return nil
}
func (f *fakeHolidayStore) BatchUpsert(context.Context, []Holiday) error { return nil }
func (f *fakeHolidayStore) Delete(context.Context, time.Time) error     { return nil }
func (f *fakeHolidayStore) GetByDate(_ context.Context, date time.Time) (*Holiday, error) {
	return f.byDate[date.Format("2006-01-02")], nil
}

func TestHolidayService_UpsertHoliday_RejectsDateConflict(t *testing.T) {
	store := newFakeHolidayStore()
	svc := NewHolidayService(store, nil)
	date := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	actorID := uuid.New()

	first, err := svc.UpsertHoliday(context.Background(), UpsertHolidayInput{
		HolidayDate: date, Name: "測試假日A", IsDayOff: true,
	}, actorID, "admin")
	assert.NoError(t, err)
	assert.Equal(t, "測試假日A", first.Name)

	_, err = svc.UpsertHoliday(context.Background(), UpsertHolidayInput{
		HolidayDate: date, Name: "測試假日B", IsDayOff: true,
	}, actorID, "admin")

	assert.ErrorIs(t, err, ErrHolidayDateConflict)
	// 原本那筆不能被覆蓋掉。
	assert.Equal(t, "測試假日A", store.byDate[date.Format("2006-01-02")].Name)
}

func TestHolidayService_UpsertHoliday_AllowsNewDate(t *testing.T) {
	store := newFakeHolidayStore()
	svc := NewHolidayService(store, nil)

	h, err := svc.UpsertHoliday(context.Background(), UpsertHolidayInput{
		HolidayDate: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), Name: "國慶日", IsDayOff: true,
	}, uuid.New(), "admin")

	assert.NoError(t, err)
	assert.Equal(t, "國慶日", h.Name)
	assert.NotNil(t, store.lastWrite)
}
