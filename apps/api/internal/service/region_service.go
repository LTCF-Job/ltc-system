package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/repository"
)

var (
	ErrRegionNameRequired  = errors.New("region name is required")
	ErrDuplicateRegionName = errors.New("region name already exists")
	ErrRegionNotFound      = errors.New("region not found")
)

// RegionService 提供區域主檔之維護業務邏輯。
type RegionService struct {
	repo      *repository.RegionRepository
	auditRepo *repository.AuditRepository
}

// NewRegionService 建立 RegionService 實例。
func NewRegionService(repo *repository.RegionRepository, auditRepo *repository.AuditRepository) *RegionService {
	return &RegionService{
		repo:      repo,
		auditRepo: auditRepo,
	}
}

// CreateRegionRequest 代表新增區域之請求結構。
type CreateRegionRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Status      string `json:"status"`
	SortOrder   int    `json:"sortOrder"`
}

// UpdateRegionRequest 代表修改區域之請求結構。
type UpdateRegionRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	SortOrder   *int    `json:"sortOrder"`
}

// ListRegions 取得區域分頁清單。
func (s *RegionService) ListRegions(ctx context.Context, q, status string, page, pageSize int) ([]repository.RegionEntity, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(ctx, strings.TrimSpace(q), strings.TrimSpace(status), page, pageSize)
}

// ListAllRegions 取得全部區域清單。
func (s *RegionService) ListAllRegions(ctx context.Context) ([]repository.RegionEntity, error) {
	return s.repo.ListAll(ctx)
}

// GetRegion 依 ID 取得單一區域。
func (s *RegionService) GetRegion(ctx context.Context, id uuid.UUID) (*repository.RegionEntity, error) {
	return s.repo.GetByID(ctx, id)
}

// CreateRegion 建立新區域主檔並留存稽核紀錄。
func (s *RegionService) CreateRegion(ctx context.Context, req CreateRegionRequest, actorID uuid.UUID, actorRole, ip, ua string) (*repository.RegionEntity, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, ErrRegionNameRequired
	}

	if req.Status == "" {
		req.Status = "active"
	} else if req.Status != "active" && req.Status != "inactive" {
		req.Status = "active"
	}

	existing, _ := s.repo.GetByName(ctx, req.Name)
	if existing != nil {
		return nil, ErrDuplicateRegionName
	}

	entity := repository.RegionEntity{
		Name:        req.Name,
		Description: strings.TrimSpace(req.Description),
		Status:      req.Status,
		SortOrder:   req.SortOrder,
	}

	if err := s.repo.Create(ctx, &entity); err != nil {
		return nil, fmt.Errorf("failed to create region: %w", err)
	}

	if s.auditRepo != nil {
		idStr := entity.ID.String()
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "create",
			EntityType: "regions",
			EntityID:   &idStr,
			AfterData:  entity,
			IPAddress:  &ip,
			UserAgent:  &ua,
		})
	}

	return &entity, nil
}

// UpdateRegion 更新現有區域並留存稽核紀錄。
func (s *RegionService) UpdateRegion(ctx context.Context, id uuid.UUID, req UpdateRegionRequest, actorID uuid.UUID, actorRole, ip, ua string) (*repository.RegionEntity, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrRegionNotFound
	}

	before := *existing

	if strings.TrimSpace(req.Name) != "" {
		existing.Name = strings.TrimSpace(req.Name)
	}
	if req.Description != nil {
		existing.Description = strings.TrimSpace(*req.Description)
	}
	if req.Status != nil && (*req.Status == "active" || *req.Status == "inactive") {
		existing.Status = *req.Status
	}
	if req.SortOrder != nil {
		existing.SortOrder = *req.SortOrder
	}

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update region: %w", err)
	}

	if s.auditRepo != nil {
		idStr := id.String()
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "update",
			EntityType: "regions",
			EntityID:   &idStr,
			BeforeData: before,
			AfterData:  *existing,
			IPAddress:  &ip,
			UserAgent:  &ua,
		})
	}

	return existing, nil
}

// DeleteRegion 刪除指定區域並留存稽核紀錄。
func (s *RegionService) DeleteRegion(ctx context.Context, id uuid.UUID, actorID uuid.UUID, actorRole, ip, ua string) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrRegionNotFound
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete region: %w", err)
	}

	if s.auditRepo != nil {
		idStr := id.String()
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "delete",
			EntityType: "regions",
			EntityID:   &idStr,
			BeforeData: *existing,
			IPAddress:  &ip,
			UserAgent:  &ua,
		})
	}

	return nil
}
