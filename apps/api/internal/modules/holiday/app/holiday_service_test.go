package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestHolidayService_ImportTaiwanGovHolidays_RejectsYearOutOfRange(t *testing.T) {
	svc := NewHolidaySyncService(discardHolidayStore{}, nil, holidayProviderStub{})

	_, err := svc.ImportTaiwanGovHolidays(context.Background(), 1999, uuid.New(), "admin")

	assert.ErrorIs(t, err, ErrInvalidHolidayYear, "年份超出範圍應以獨立 sentinel error 回傳，與外部服務失敗區分")
}

type failingHolidayProvider struct{}

func (failingHolidayProvider) Fetch(context.Context, int) ([]HolidayRecord, error) {
	return nil, errors.New("connection reset by peer")
}

func TestHolidayService_ImportTaiwanGovHolidays_WrapsProviderFailure(t *testing.T) {
	svc := NewHolidaySyncService(discardHolidayStore{}, nil, failingHolidayProvider{})

	_, err := svc.ImportTaiwanGovHolidays(context.Background(), 2026, uuid.New(), "admin")

	assert.ErrorIs(t, err, ErrGovHolidayFetchFailed, "外部行事曆來源失敗應以獨立 sentinel error 回傳，供 handler 映射 503")
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
func (f *fakeHolidayStore) Delete(_ context.Context, date time.Time) error {
	delete(f.byDate, date.Format("2006-01-02"))
	return nil
}
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

func TestHolidayService_DeleteHoliday_RejectsNonexistentDate(t *testing.T) {
	store := newFakeHolidayStore()
	svc := NewHolidayService(store, nil)

	err := svc.DeleteHoliday(context.Background(), time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), uuid.New(), "admin")

	assert.ErrorIs(t, err, ErrHolidayNotFound, "刪除不存在的假日不應被當成成功，需回傳 ErrHolidayNotFound 供 handler 映射 404")
}

func TestHolidayService_DeleteHoliday_DeletesExistingDate(t *testing.T) {
	store := newFakeHolidayStore()
	svc := NewHolidayService(store, nil)
	date := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	_, err := svc.UpsertHoliday(context.Background(), UpsertHolidayInput{HolidayDate: date, Name: "國慶日", IsDayOff: true}, uuid.New(), "admin")
	require.NoError(t, err)

	err = svc.DeleteHoliday(context.Background(), date, uuid.New(), "admin")

	assert.NoError(t, err)
	assert.Nil(t, store.byDate[date.Format("2006-01-02")])
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
