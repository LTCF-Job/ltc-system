package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchPendingColumnsByName_ReturnsPendingColumnsWithSameOrSimilarName(t *testing.T) {
	store := &stubStore{existing: []ColumnMapping{
		{ID: "col-1", FormID: testFormID, ColumnHeader: "1.吳桂 [去程]", CleanedName: "吳桂", MappingStatus: "pending"},
		{ID: "col-2", FormID: testFormID, ColumnHeader: "2.吳貴 [去程]", CleanedName: "吳貴", MappingStatus: "pending"},
		{ID: "col-3", FormID: testFormID, ColumnHeader: "3.陳大明 [去程]", CleanedName: "陳大明", MappingStatus: "pending"},
		{ID: "col-4", FormID: testFormID, ColumnHeader: "4.吳桂 [回程]", CleanedName: "吳桂", MappingStatus: "mapped"},
	}}
	svc := NewDriverReportService(store, stubExcel{}, nil, nil, nil, &fakeIngestor{}, nil, nil, directTxRunner{})

	matches, err := svc.MatchPendingColumnsByName(context.Background(), "吳桂")

	require.NoError(t, err)
	ids := make([]string, 0, len(matches))
	for _, m := range matches {
		ids = append(ids, m.ID)
	}
	assert.ElementsMatch(t, []string{"col-1", "col-2"}, ids, "精確相符與近似姓名都要回傳，已對應的欄位不算")
}

func TestBindPendingDriver_DelegatesToRideIngestor(t *testing.T) {
	ingestor := &fakeIngestor{backfillDriverResult: 3}
	svc := NewDriverReportService(&stubStore{}, stubExcel{}, nil, nil, nil, ingestor, nil, nil, directTxRunner{})
	driverID := uuid.New()

	affected, err := svc.BindPendingDriver(context.Background(), "林彥衡", driverID.String())

	require.NoError(t, err)
	assert.Equal(t, 3, affected)
	require.Len(t, ingestor.backfillDriverCalls, 1)
	assert.Equal(t, "林彥衡", ingestor.backfillDriverCalls[0].driverNameRaw)
	assert.Equal(t, driverID, ingestor.backfillDriverCalls[0].driverID)
}

func TestBindPendingDriver_SyncsAttendanceForEachAffectedDate(t *testing.T) {
	d1 := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	ingestor := &fakeIngestor{backfillDriverResult: 2, backfillDriverDates: []time.Time{d1, d2}}
	registrar := &fakeAttendanceRegistrar{}
	svc := NewDriverReportService(&stubStore{}, stubExcel{}, nil, nil, nil, ingestor, registrar, nil, directTxRunner{})
	driverID := uuid.New()

	affected, err := svc.BindPendingDriver(context.Background(), "林彥衡", driverID.String())

	require.NoError(t, err)
	assert.Equal(t, 2, affected)
	require.Len(t, registrar.calls, 2, "補綁定要跟初次匯入一樣同步這些日期的出勤，不用使用者再手動登記")
	assert.Equal(t, driverID, registrar.calls[0].driverID)
	assert.ElementsMatch(t, []time.Time{d1, d2}, []time.Time{registrar.calls[0].serviceDate, registrar.calls[1].serviceDate})
}

func TestBindPendingDriver_RejectsInvalidDriverID(t *testing.T) {
	ingestor := &fakeIngestor{}
	svc := NewDriverReportService(&stubStore{}, stubExcel{}, nil, nil, nil, ingestor, nil, nil, directTxRunner{})

	_, err := svc.BindPendingDriver(context.Background(), "林彥衡", "not-a-uuid")

	require.Error(t, err)
	assert.Empty(t, ingestor.backfillDriverCalls)
}

func TestListSubmissionReview_CombinesCaseAndDriverIssuesOnTheSameRow(t *testing.T) {
	formID := uuid.MustParse(testFormID)
	submissionWithBoth := uuid.New()
	submissionCaseOnly := uuid.New()
	submissionDriverOnly := uuid.New()
	serviceDate := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)

	store := &stubStore{existing: []ColumnMapping{
		{ID: "col-1", FormID: testFormID, ColumnHeader: "1.吳桂 [去程]", CleanedName: "吳桂", MappingStatus: "pending"},
	}}
	ingestor := &fakeIngestor{
		submissionsForForms: []SubmissionAnswerRow{
			{
				SubmissionID: submissionWithBoth, FormID: formID, FormTitle: "竹南2車匯報表", VehicleName: "竹南2車",
				ServiceDate: serviceDate, Answers: map[string]string{"1.吳桂 [去程]": "有坐"},
			},
			{
				SubmissionID: submissionCaseOnly, FormID: formID, FormTitle: "竹南2車匯報表", VehicleName: "竹南2車",
				ServiceDate: serviceDate, Answers: map[string]string{"1.吳桂 [去程]": "有坐"},
			},
			{
				// 這一列雖然在 pending 欄位下有儲存格，但值代表「沒有回報」，不算問題
				SubmissionID: uuid.New(), FormID: formID, FormTitle: "竹南2車匯報表", VehicleName: "竹南2車",
				ServiceDate: serviceDate, Answers: map[string]string{"1.吳桂 [去程]": ""},
			},
		},
		unmatchedDrivers: []UnmatchedDriverSubmission{
			{SubmissionID: submissionWithBoth, FormID: formID, FormTitle: "竹南2車匯報表", VehicleName: "竹南2車", ServiceDate: serviceDate, DriverNameRaw: "林彥衡"},
			{SubmissionID: submissionDriverOnly, FormID: formID, FormTitle: "竹南2車匯報表", VehicleName: "竹南2車", ServiceDate: serviceDate, DriverNameRaw: "陳大文"},
		},
	}
	svc := NewDriverReportService(store, stubExcel{}, nil, nil, nil, ingestor, nil, nil, directTxRunner{})

	reviews, err := svc.ListSubmissionReview(context.Background())
	require.NoError(t, err)
	require.Len(t, reviews, 3)

	byID := map[string]SubmissionReview{}
	for _, r := range reviews {
		byID[r.SubmissionID] = r
	}

	both := byID[submissionWithBoth.String()]
	assert.Len(t, both.CaseIssues, 1, "同一列同時有個案與司機問題要合併在同一筆")
	require.NotNil(t, both.DriverIssue)
	assert.Equal(t, "林彥衡", both.DriverIssue.DriverNameRaw)
	assert.Equal(t, "2026-03-02", both.ServiceDate)
	assert.Equal(t, "竹南2車匯報表", both.FormTitle)

	caseOnly := byID[submissionCaseOnly.String()]
	assert.Len(t, caseOnly.CaseIssues, 1)
	assert.Nil(t, caseOnly.DriverIssue)

	driverOnly := byID[submissionDriverOnly.String()]
	assert.Empty(t, driverOnly.CaseIssues)
	require.NotNil(t, driverOnly.DriverIssue)
	assert.Equal(t, "陳大文", driverOnly.DriverIssue.DriverNameRaw)
}
