package app

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/domain/govform"
	"ltc-system/apps/api/internal/domain/rocdate"
)

func TestNewExportLine_ReturnsROCDateConversionError(t *testing.T) {
	_, err := newExportLine(
		1,
		caseGroup{caseID: uuid.New()},
		govform.ClaimRow{ServiceDate: time.Date(1911, 12, 31, 0, 0, 0, 0, time.UTC)},
		nil,
	)

	require.ErrorIs(t, err, rocdate.ErrBeforeROCYear)
}
