package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"ltc-system/apps/api/internal/repository"
)

func TestHolidayService_ImportTaiwanGovHolidays(t *testing.T) {
	repo := repository.NewHolidayRepository(nil)
	svc := NewHolidayService(repo, nil)

	ctx := context.Background()
	count, err := svc.ImportTaiwanGovHolidays(ctx, 2026, uuid.New(), "admin")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count, 6)
}
