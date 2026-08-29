package app

import "context"

// Store 定義稽核日誌的讀寫邊界。
type Store interface {
	Insert(ctx context.Context, e Entry) error
	List(ctx context.Context, f Filter) ([]Record, int64, error)
}
