package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/repository"
)

// AuditFilter 代表稽核紀錄查詢條件，由 transport 層依查詢參數組裝。
type AuditFilter struct {
	ActorID    *uuid.UUID
	Action     string
	EntityType string
	EntityID   string
	StartDate  *time.Time
	EndDate    *time.Time
	Q          string
	Page       int
	PageSize   int
}

// AuditStore 定義稽核紀錄查詢邊界。
type AuditStore interface {
	List(ctx context.Context, f repository.AuditFilter) ([]repository.AuditLogEntity, int64, error)
}

// AuditService 提供稽核日誌之查詢服務。
type AuditService struct {
	store AuditStore
}

// NewAuditService 建立 AuditService 實例。
func NewAuditService(store AuditStore) *AuditService {
	return &AuditService{store: store}
}

// ListAuditLogs 依條件篩選並分頁取得稽核紀錄。
func (s *AuditService) ListAuditLogs(ctx context.Context, f AuditFilter) ([]repository.AuditLogEntity, int64, error) {
	return s.store.List(ctx, repository.AuditFilter{
		ActorID:    f.ActorID,
		Action:     f.Action,
		EntityType: f.EntityType,
		EntityID:   f.EntityID,
		StartDate:  f.StartDate,
		EndDate:    f.EndDate,
		Q:          f.Q,
		Page:       f.Page,
		PageSize:   f.PageSize,
	})
}
