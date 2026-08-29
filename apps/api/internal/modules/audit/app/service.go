package app

import "context"

// Service 提供稽核日誌之寫入與查詢服務。
type Service struct {
	store Store
}

// NewService 建立 Service 實例。
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Write 寫入一筆稽核紀錄。
func (s *Service) Write(ctx context.Context, e Entry) error {
	return s.store.Insert(ctx, e)
}

// List 依條件篩選並分頁取得稽核紀錄。
func (s *Service) List(ctx context.Context, f Filter) ([]Record, int64, error) {
	return s.store.List(ctx, f)
}
