package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type retryAuditStore struct {
	remainingFailures int
	calls             int
}

func (s *retryAuditStore) Insert(context.Context, Entry) error {
	s.calls++
	if s.remainingFailures > 0 {
		s.remainingFailures--
		return errors.New("temporary audit failure")
	}
	return nil
}

func (s *retryAuditStore) List(context.Context, Filter) ([]Record, int64, error) {
	return nil, 0, nil
}

func TestServiceWrite_RetriesTransientFailure(t *testing.T) {
	store := &retryAuditStore{remainingFailures: 2}
	service := NewService(store)

	require.NoError(t, service.Write(context.Background(), Entry{Action: "update", EntityType: "test"}))
	require.Equal(t, 3, store.calls)
	require.Zero(t, service.FailedWriteCount())
}

func TestServiceWrite_RecordsPermanentFailure(t *testing.T) {
	store := &retryAuditStore{remainingFailures: 3}
	service := NewService(store)

	err := service.Write(context.Background(), Entry{Action: "delete", EntityType: "test"})
	require.Error(t, err)
	require.Equal(t, 3, store.calls)
	require.Equal(t, uint64(1), service.FailedWriteCount())
}
