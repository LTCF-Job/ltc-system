package app

import "context"

// ResetRepository 定義重新載入 Demo 資料集所需的持久層操作。
type ResetRepository interface {
	Reset(ctx context.Context) error
}
