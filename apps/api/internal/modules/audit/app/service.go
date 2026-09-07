package app

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"
)

// Service 提供稽核日誌之寫入與查詢服務。
type Service struct {
	store        Store
	failedWrites atomic.Uint64
}

// NewService 建立 Service 實例。
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Write 寫入一筆稽核紀錄。
func (s *Service) Write(ctx context.Context, e Entry) error {
	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := s.store.Insert(ctx, e); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if attempt == maxAttempts {
			break
		}
		slog.Warn("audit_write_retry", "action", e.Action, "entity_type", e.EntityType, "entity_id", e.EntityID, "attempt", attempt+1, "error", lastErr)
		timer := time.NewTimer(time.Duration(attempt) * 25 * time.Millisecond)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
	s.failedWrites.Add(1)
	slog.Error("audit_write_failed", "metric_name", "ltc_audit_write_failures_total", "metric_increment", 1, "action", e.Action, "entity_type", e.EntityType, "entity_id", e.EntityID, "attempts", maxAttempts, "error", lastErr)
	return fmt.Errorf("audit write failed after %d attempts: %w", maxAttempts, lastErr)
}

// FailedWriteCount 回傳目前 process 內稽核寫入最終失敗的累計次數。
// 這個計數器讓沒有 metrics backend 的部署仍能觀測稽核失敗，正式環境可由 adapter 轉接至監控系統。
func (s *Service) FailedWriteCount() uint64 {
	return s.failedWrites.Load()
}

// List 依條件篩選並分頁取得稽核紀錄。
func (s *Service) List(ctx context.Context, f Filter) ([]Record, int64, error) {
	return s.store.List(ctx, f)
}
