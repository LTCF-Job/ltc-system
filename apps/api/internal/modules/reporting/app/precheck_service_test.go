package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/modules/reporting/app"
)

type precheckRepoStub struct {
	incomplete    []app.IncompleteCase
	incompleteErr error
	conflicts     []app.UnresolvedConflict
	conflictsErr  error
}

func (s precheckRepoStub) FindIncompleteActiveCases(context.Context, app.ClaimScope) ([]app.IncompleteCase, error) {
	return s.incomplete, s.incompleteErr
}

func (s precheckRepoStub) FindUnresolvedConflicts(context.Context, app.ClaimScope) ([]app.UnresolvedConflict, error) {
	return s.conflicts, s.conflictsErr
}

func TestRunPrecheck_FailsWhenIncompleteCaseQueryFails(t *testing.T) {
	wantErr := errors.New("database unavailable")

	report, err := app.NewPrecheckService(precheckRepoStub{incompleteErr: wantErr}).RunPrecheck(
		context.Background(), app.ClaimScope{},
	)

	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, report)
}

func TestRunPrecheck_FailsWhenConflictQueryFails(t *testing.T) {
	wantErr := errors.New("database unavailable")

	report, err := app.NewPrecheckService(precheckRepoStub{conflictsErr: wantErr}).RunPrecheck(
		context.Background(), app.ClaimScope{},
	)

	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, report)
}

func TestRunPrecheck_UnresolvedConflictBlocksExport(t *testing.T) {
	report, err := app.NewPrecheckService(precheckRepoStub{
		conflicts: []app.UnresolvedConflict{{
			RideID:      uuid.New(),
			CaseName:    "王小明",
			ServiceDate: time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC),
		}},
	}).RunPrecheck(context.Background(), app.ClaimScope{})

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.False(t, report.Passed)
	assert.Equal(t, 1, report.TotalErrors)
	assert.Equal(t, app.SeverityError, report.Issues[0].Severity)
	assert.Equal(t, "UNRESOLVED_CONFLICT", report.Issues[0].Code)
}

func TestRunPrecheck_IncompleteCaseWarnsWithoutBlockingExport(t *testing.T) {
	report, err := app.NewPrecheckService(precheckRepoStub{
		incomplete: []app.IncompleteCase{{ID: uuid.New(), Name: "王小明"}},
	}).RunPrecheck(context.Background(), app.ClaimScope{})

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.True(t, report.Passed, "缺個案資料只留白匯出，不得擋下整批申報")
	assert.Equal(t, 0, report.TotalErrors)
	assert.Equal(t, 1, report.TotalWarnings)
	assert.Equal(t, app.SeverityWarning, report.Issues[0].Severity)
	assert.Equal(t, "MISSING_CASE_PROFILE", report.Issues[0].Code)
}

func TestRunPrecheck_DoesNotEmitQuotaSkippedInfo(t *testing.T) {
	// 曾經有一則恆常輸出的 QUOTA_CHECK_SKIPPED info，每次檢核都出現、永遠不會變，
	// 對操作者沒有資訊量，已移除。這則測試鎖定它不會被無意間加回來。
	report, err := app.NewPrecheckService(precheckRepoStub{}).RunPrecheck(
		context.Background(), app.ClaimScope{},
	)

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, 0, report.TotalInfos)
	assert.Empty(t, report.Issues, "沒有查出任何問題時，檢核報告不應憑空出現項目")
	for _, issue := range report.Issues {
		assert.NotEqual(t, "QUOTA_CHECK_SKIPPED", issue.Code)
	}
}

func TestRunPrecheckMulti(t *testing.T) {
	months := []app.ClaimMonth{
		{PeriodYM: "11505", StartDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)},
		{PeriodYM: "11506", StartDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)},
	}
	caseID := uuid.New()

	t.Run("個案資料缺漏跨月去重", func(t *testing.T) {
		// 缺欄位是個案主檔的狀態，與月份無關；不去重的話選 6 個月就會看到同一則警告 6 次。
		report, err := app.NewPrecheckService(precheckRepoStub{
			incomplete: []app.IncompleteCase{{ID: caseID, Name: "王小明"}},
		}).RunPrecheckMulti(context.Background(), months, nil)

		require.NoError(t, err)
		assert.True(t, report.Passed)
		assert.Equal(t, 1, report.TotalWarnings)
		require.Len(t, report.Issues, 1)
		assert.Equal(t, "MISSING_CASE_PROFILE", report.Issues[0].Code)
	})

	t.Run("未裁決衝突逐月保留並標示月份", func(t *testing.T) {
		// 每一筆衝突綁定特定日期的特定搭乘紀錄，都要各自裁決，不能去重。
		report, err := app.NewPrecheckService(precheckRepoStub{
			conflicts: []app.UnresolvedConflict{{
				RideID:      uuid.New(),
				CaseName:    "王小明",
				ServiceDate: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
			}},
		}).RunPrecheckMulti(context.Background(), months, nil)

		require.NoError(t, err)
		assert.False(t, report.Passed, "任一月份有未裁決衝突，整批都不得放行")
		assert.Equal(t, 2, report.TotalErrors)
		require.Len(t, report.Issues, 2)
		assert.Equal(t, "11505", report.Issues[0].Details["periodYm"])
		assert.Equal(t, "11506", report.Issues[1].Details["periodYm"])
	})

	t.Run("未指定月份時拒絕執行", func(t *testing.T) {
		_, err := app.NewPrecheckService(precheckRepoStub{}).RunPrecheckMulti(context.Background(), nil, nil)
		assert.ErrorIs(t, err, app.ErrPeriodsRequired)
	})
}

func TestParseClaimMonths(t *testing.T) {
	t.Run("去重並依月份升冪排序", func(t *testing.T) {
		months, err := app.ParseClaimMonths([]string{"11507", "11412", "11507", "115-01"})

		require.NoError(t, err)
		require.Len(t, months, 3)
		assert.Equal(t, "11412", months[0].PeriodYM)
		assert.Equal(t, "11501", months[1].PeriodYM)
		assert.Equal(t, "11507", months[2].PeriodYM)
	})

	t.Run("換算成該月西元起訖", func(t *testing.T) {
		months, err := app.ParseClaimMonths([]string{"11412"})

		require.NoError(t, err)
		require.Len(t, months, 1)
		assert.Equal(t, time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC), months[0].StartDate)
		assert.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), months[0].EndDate)
	})

	t.Run("任一筆格式錯誤即整批拒絕", func(t *testing.T) {
		// 只匯出一半月份卻沒人察覺，比整批擋下難查得多。
		_, err := app.ParseClaimMonths([]string{"11507", "abc"})
		assert.ErrorIs(t, err, app.ErrInvalidPeriodYM)
	})

	t.Run("空清單回 ErrPeriodsRequired", func(t *testing.T) {
		_, err := app.ParseClaimMonths(nil)
		assert.ErrorIs(t, err, app.ErrPeriodsRequired)
	})
}
