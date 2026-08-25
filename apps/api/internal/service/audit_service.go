package service

import (
	"context"

	"ltc-system/apps/api/internal/repository"
)

// AuditService 提供稽核日誌之查詢服務。
type AuditService struct {
	repo *repository.AuditRepository
}

// NewAuditService 建立 AuditService 實例。
func NewAuditService(repo *repository.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

// ListAuditLogs 依條件篩選並分頁取得稽核紀錄。
func (s *AuditService) ListAuditLogs(ctx context.Context, f repository.AuditFilter) ([]repository.AuditLogEntity, int64, error) {
	return s.repo.List(ctx, f)
}
