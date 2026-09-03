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

func newMonthDetailService(ingestor *fakeIngestor) *DriverReportService {
	return NewDriverReportService(&stubStore{}, stubExcel{}, nil, nil, nil, ingestor, nil, nil, nil)
}

func TestGetMonthDetail_CombinesSubmissionsAndRideEntries(t *testing.T) {
	formID := uuid.New()
	caseID := uuid.New()
	serviceDate := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	svc := newMonthDetailService(&fakeIngestor{
		monthSubmissions: []MonthSubmissionDetail{
			{ServiceDate: serviceDate, DriverNameRaw: "王大明", Remark: "無", Answers: map[string]string{"欄位A": "V"}},
		},
		monthRideEntries: []MonthRideEntry{
			{CaseID: caseID, CaseName: "陳小華", ServiceDate: serviceDate, LegSeq: 1, Reported: "boarded"},
		},
	})

	detail, err := svc.GetMonthDetail(context.Background(), formID, "2026-03")

	require.NoError(t, err)
	require.Len(t, detail.Submissions, 1)
	assert.Equal(t, "王大明", detail.Submissions[0].DriverNameRaw)
	require.Len(t, detail.RideEntries, 1)
	assert.Equal(t, "陳小華", detail.RideEntries[0].CaseName)
}

// 前端以陣列長度判斷有無資料，nil 與空陣列在 JSON 上是 null 與 []，不能混用。
func TestGetMonthDetail_NoDataReturnsEmptySlices(t *testing.T) {
	svc := newMonthDetailService(&fakeIngestor{})

	detail, err := svc.GetMonthDetail(context.Background(), uuid.New(), "2026-03")

	require.NoError(t, err)
	assert.NotNil(t, detail.Submissions)
	assert.Empty(t, detail.Submissions)
	assert.NotNil(t, detail.RideEntries)
	assert.Empty(t, detail.RideEntries)
}

func TestGetMonthDetail_PropagatesSubmissionsError(t *testing.T) {
	svc := newMonthDetailService(&fakeIngestor{monthDetailErr: errors.New("db down")})

	_, err := svc.GetMonthDetail(context.Background(), uuid.New(), "2026-03")

	assert.Error(t, err)
}
