package app

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeResetRepository struct {
	err   error
	calls int
}

func (f *fakeResetRepository) Reset(ctx context.Context) error {
	f.calls++
	return f.err
}

func TestResetService_Reset_Success(t *testing.T) {
	repo := &fakeResetRepository{}
	guard := NewConcurrencyGuard()
	svc := NewResetService(repo, guard)

	before := time.Now().UTC()
	result, err := svc.Reset(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.DatasetVersion != DatasetVersion {
		t.Fatalf("expected dataset version %q, got %q", DatasetVersion, result.DatasetVersion)
	}
	if result.ResetAt.Before(before) {
		t.Fatalf("expected ResetAt >= %v, got %v", before, result.ResetAt)
	}
	if repo.calls != 1 {
		t.Fatalf("expected repo.Reset called once, got %d", repo.calls)
	}
}

func TestResetService_Reset_RepositoryErrorPropagatesAndReleasesGuard(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &fakeResetRepository{err: wantErr}
	guard := NewConcurrencyGuard()
	svc := NewResetService(repo, guard)

	_, err := svc.Reset(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}

	// 失敗後獨佔鎖必須釋放，否則之後所有一般請求會永久卡住。
	done := make(chan struct{})
	go func() {
		release := guard.BeginRequest()
		release()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("guard was not released after Reset returned an error")
	}
}

func TestConcurrencyGuard_ResetWaitsForInFlightRequest(t *testing.T) {
	guard := NewConcurrencyGuard()
	var events []string

	requestReleased := make(chan struct{})
	resetStarted := make(chan struct{})
	resetDone := make(chan struct{})

	releaseRequest := guard.BeginRequest()
	events = append(events, "request_started")

	go func() {
		close(resetStarted)
		release := guard.BeginReset()
		events = append(events, "reset_acquired")
		release()
		close(resetDone)
	}()

	<-resetStarted
	// 給 BeginReset 一點時間嘗試取得鎖並確認它會被卡住，而不是立刻取得。
	select {
	case <-resetDone:
		t.Fatal("BeginReset acquired the lock before the in-flight request released it")
	case <-time.After(50 * time.Millisecond):
	}

	events = append(events, "request_released")
	releaseRequest()
	close(requestReleased)

	select {
	case <-resetDone:
	case <-time.After(time.Second):
		t.Fatal("BeginReset never acquired the lock after the request released it")
	}

	want := []string{"request_started", "request_released", "reset_acquired"}
	if len(events) != len(want) {
		t.Fatalf("expected event sequence %v, got %v", want, events)
	}
	for i, e := range want {
		if events[i] != e {
			t.Fatalf("expected event sequence %v, got %v", want, events)
		}
	}
}
