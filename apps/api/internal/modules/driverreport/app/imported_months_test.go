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

func newImportedMonthsService(ingestor *fakeIngestor) *DriverReportService {
	return NewDriverReportService(&stubStore{}, stubExcel{}, nil, nil, nil, ingestor, nil, nil, nil)
}

func TestListImportedMonths_ReturnsEachFormMonthStat(t *testing.T) {
	formA, formB := uuid.New(), uuid.New()
	lastImported := time.Date(2026, 3, 31, 10, 20, 30, 0, time.UTC)
	svc := newImportedMonthsService(&fakeIngestor{importedMonths: []ImportedMonth{
		{FormID: formA, YearMonth: "2026-03", SubmissionCount: 21, LastImportedAt: lastImported},
		{FormID: formA, YearMonth: "2026-02", SubmissionCount: 19, LastImportedAt: lastImported},
		{FormID: formB, YearMonth: "2026-03", SubmissionCount: 5, LastImportedAt: lastImported},
	}})

	months, err := svc.ListImportedMonths(context.Background())

	require.NoError(t, err)
	require.Len(t, months, 3)
	assert.Equal(t, formA, months[0].FormID)
	assert.Equal(t, "2026-03", months[0].YearMonth)
	assert.Equal(t, 21, months[0].SubmissionCount)
	assert.Equal(t, lastImported, months[0].LastImportedAt)
}

// 前端以陣列長度判斷有無匯入紀錄，nil 與空陣列在 JSON 上是 null 與 []，不能混用。
func TestListImportedMonths_NoDataReturnsEmptySlice(t *testing.T) {
	svc := newImportedMonthsService(&fakeIngestor{})

	months, err := svc.ListImportedMonths(context.Background())

	require.NoError(t, err)
	assert.NotNil(t, months)
	assert.Empty(t, months)
}

func TestListImportedMonths_PropagatesStoreError(t *testing.T) {
	svc := newImportedMonthsService(&fakeIngestor{importedErr: errors.New("db down")})

	months, err := svc.ListImportedMonths(context.Background())

	require.Error(t, err)
	assert.Nil(t, months)
}
