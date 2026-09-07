package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFuelService_ListRejectsInvalidPaginationBeforeRepository(t *testing.T) {
	svc := NewFuelService(nil, nil)

	_, _, err := svc.List(context.Background(), 1, 0, nil, nil, nil, nil, "")

	require.ErrorIs(t, err, ErrInvalidFuelPagination)
}
