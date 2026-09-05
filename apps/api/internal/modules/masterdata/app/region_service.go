package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// RegionService 提供區域主檔之維護業務邏輯。
type RegionService struct {
	store RegionStore
	audit AuditWriter
}

// NewRegionService 建立 RegionService 實例。
func NewRegionService(store RegionStore, audit AuditWriter) *RegionService {
	return &RegionService{store: store, audit: audit}
}

// ActorContext 代表發動異動的操作者與來源資訊，供稽核留痕使用。
type ActorContext struct {
	ActorID   uuid.UUID
	ActorRole string
	IPAddress string
	UserAgent string
}

// CreateRegionInput 代表新增區域所需之輸入。
type CreateRegionInput struct {
	Name        string
	Description string
	Status      string
	SortOrder   int
}

// UpdateRegionInput 代表更新區域所需之輸入，欄位為 nil 表示不變更。
type UpdateRegionInput struct {
	Name        string
	Description *string
	Status      *string
	SortOrder   *int
}

// ListRegions 取得區域分頁清單。
func (s *RegionService) ListRegions(ctx context.Context, q, status string, page, pageSize int) ([]Region, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.store.List(ctx, strings.TrimSpace(q), strings.TrimSpace(status), page, pageSize)
}

// ListAllRegions 取得全部區域清單。
func (s *RegionService) ListAllRegions(ctx context.Context) ([]Region, error) {
	return s.store.ListAll(ctx)
}

// GetRegion 依 ID 取得單一區域。
func (s *RegionService) GetRegion(ctx context.Context, id uuid.UUID) (*Region, error) {
	return s.store.GetByID(ctx, id)
}

// CreateRegion 建立新區域主檔並留存稽核紀錄。
func (s *RegionService) CreateRegion(ctx context.Context, in CreateRegionInput, actor ActorContext) (*Region, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrRegionNameRequired
	}

	// 未提供狀態時預設 active；一旦提供非白名單值就拒絕，不猜測使用者意圖。
	status := in.Status
	if status == "" {
		status = "active"
	} else if status != "active" && status != "inactive" {
		return nil, ErrInvalidStatus
	}

	existing, err := s.store.GetByName(ctx, name)
	if err != nil && !errors.Is(err, ErrRegionNotFound) {
		return nil, fmt.Errorf("failed to check duplicate region name: %w", err)
	}
	if existing != nil {
		return nil, ErrDuplicateRegionName
	}

	region := Region{
		Name:        name,
		Description: strings.TrimSpace(in.Description),
		Status:      status,
		SortOrder:   in.SortOrder,
	}

	if err := s.store.Create(ctx, &region); err != nil {
		return nil, fmt.Errorf("failed to create region: %w", err)
	}

	s.writeAudit(ctx, "create", region.ID, actor, nil, region.Snapshot())
	return &region, nil
}

// UpdateRegion 更新現有區域並留存稽核紀錄。
func (s *RegionService) UpdateRegion(ctx context.Context, id uuid.UUID, in UpdateRegionInput, actor ActorContext) (*Region, error) {
	existing, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	before := existing.Snapshot()

	if strings.TrimSpace(in.Name) != "" {
		existing.Name = strings.TrimSpace(in.Name)
	}
	if in.Description != nil {
		existing.Description = strings.TrimSpace(*in.Description)
	}
	if in.Status != nil {
		if *in.Status != "active" && *in.Status != "inactive" {
			return nil, ErrInvalidStatus
		}
		existing.Status = *in.Status
	}
	if in.SortOrder != nil {
		existing.SortOrder = *in.SortOrder
	}

	if err := s.store.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update region: %w", err)
	}

	s.writeAudit(ctx, "update", id, actor, before, existing.Snapshot())
	return existing, nil
}

// DeleteRegion 刪除指定區域並留存稽核紀錄。
func (s *RegionService) DeleteRegion(ctx context.Context, id uuid.UUID, actor ActorContext) error {
	existing, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrRegionNotFound) {
			return ErrRegionNotFound
		}
		return fmt.Errorf("failed to load region: %w", err)
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete region: %w", err)
	}

	s.writeAudit(ctx, "delete", id, actor, existing.Snapshot(), nil)
	return nil
}

// writeAudit 寫入區域異動留痕。稽核失敗不影響已完成的主檔異動，僅在讀取路徑外
// 被忽略，與既有行為一致。
func (s *RegionService) writeAudit(ctx context.Context, action string, id uuid.UUID, actor ActorContext, before, after interface{}) {
	if s.audit == nil {
		return
	}
	idStr := id.String()
	_ = s.audit.Write(ctx, AuditEntry{
		ActorID:    &actor.ActorID,
		ActorRole:  &actor.ActorRole,
		Action:     action,
		EntityType: "regions",
		EntityID:   &idStr,
		BeforeData: before,
		AfterData:  after,
		IPAddress:  &actor.IPAddress,
		UserAgent:  &actor.UserAgent,
	})
}
