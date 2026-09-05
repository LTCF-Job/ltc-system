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
	assert.Equal(t, app.SeverityError, report.Issues[1].Severity)
	assert.Equal(t, "UNRESOLVED_CONFLICT", report.Issues[1].Code)
}
