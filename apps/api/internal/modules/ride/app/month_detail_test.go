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

func TestRideService_ListSubmissionsForFormMonth_DelegatesToStore(t *testing.T) {
	formID := uuid.New()
	monthStart := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	monthEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	store := newFakeRecordStore(nil)
	store.monthSubmissions = []MonthSubmissionDetail{
		{ServiceDate: monthStart, DriverNameRaw: "王大明", Remark: "備註", Answers: map[string]string{"欄位A": "V"}},
	}
	svc := NewRideService(store, nil, nil, nil, nil)

	got, err := svc.ListSubmissionsForFormMonth(context.Background(), formID, monthStart, monthEnd)

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "王大明", got[0].DriverNameRaw)
	assert.Equal(t, "V", got[0].Answers["欄位A"])
}

func TestRideService_ListSubmissionsForFormMonth_PropagatesStoreError(t *testing.T) {
	store := newFakeRecordStore(nil)
	store.monthSubmissionsErr = errors.New("db down")
	svc := NewRideService(store, nil, nil, nil, nil)

	_, err := svc.ListSubmissionsForFormMonth(context.Background(), uuid.New(), time.Now(), time.Now())

	assert.Error(t, err)
}

func TestRideService_ListRideEntriesForFormMonth_DelegatesToStore(t *testing.T) {
	formID := uuid.New()
	caseID := uuid.New()
	monthStart := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	monthEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	store := newFakeRecordStore(nil)
	store.monthRideEntries = []MonthRideEntry{
		{CaseID: caseID, CaseName: "陳小華", ServiceDate: monthStart, LegSeq: 1, Reported: "boarded"},
	}
	svc := NewRideService(store, nil, nil, nil, nil)

	got, err := svc.ListRideEntriesForFormMonth(context.Background(), formID, monthStart, monthEnd)

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "陳小華", got[0].CaseName)
	assert.Equal(t, "boarded", got[0].Reported)
}

func TestRideService_ListRideEntriesForFormMonth_PropagatesStoreError(t *testing.T) {
	store := newFakeRecordStore(nil)
	store.monthRideEntriesErr = errors.New("db down")
	svc := NewRideService(store, nil, nil, nil, nil)

	_, err := svc.ListRideEntriesForFormMonth(context.Background(), uuid.New(), time.Now(), time.Now())

	assert.Error(t, err)
}
