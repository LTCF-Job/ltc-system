package app

import (
	"context"
	"time"
)

// DatasetVersion 對應目前唯一支援的種子檔版本（apps/api/seed/demo/0001_baseline.up.sql）。
const DatasetVersion = "0001_baseline"

// ResetResult 為重置完成後回傳給呼叫端的結果。
type ResetResult struct {
	DatasetVersion string
	ResetAt        time.Time
}

// ResetService 協調 Demo 資料集重置：獨佔鎖與持久層清空／重新載入的執行順序。
type ResetService struct {
	repo  ResetRepository
	guard *ConcurrencyGuard
}

// NewResetService 建立 ResetService 實例。
func NewResetService(repo ResetRepository, guard *ConcurrencyGuard) *ResetService {
	return &ResetService{repo: repo, guard: guard}
}

// Reset 等待既有請求完成、於單一交易內清空並重新載入 Demo 資料集，失敗則整筆回滾。
func (s *ResetService) Reset(ctx context.Context) (ResetResult, error) {
	release := s.guard.BeginReset()
	defer release()

	if err := s.repo.Reset(ctx); err != nil {
		return ResetResult{}, err
	}
	return ResetResult{DatasetVersion: DatasetVersion, ResetAt: time.Now().UTC()}, nil
}
