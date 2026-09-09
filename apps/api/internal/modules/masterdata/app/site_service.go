package app

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// SiteService 封裝據點主檔業務邏輯。
type SiteService struct {
	store     SiteStore
	auditRepo AuditWriter
}

// NewSiteService 建立 SiteService 實例。
func NewSiteService(store SiteStore, audits ...AuditWriter) *SiteService {
	var auditRepo AuditWriter
	if len(audits) > 0 {
		auditRepo = audits[0]
	}
	return &SiteService{store: store, auditRepo: auditRepo}
}

// List 查詢據點清單。
func (s *SiteService) List(ctx context.Context, region, q, status string, page, pageSize int) ([]Site, int64, error) {
	return s.store.List(ctx, region, q, status, page, pageSize)
}

// CreateSiteInput 代表新增據點所需之輸入。
type CreateSiteInput struct {
	Name    string
	Address string
	Region  string
	Status  string
}

// Create 新增據點主檔。
func (s *SiteService) Create(ctx context.Context, in CreateSiteInput, actors ...ActorContext) (*Site, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrSiteNameRequired
	}
	address := strings.TrimSpace(in.Address)
	region := strings.TrimSpace(in.Region)

	// 未提供狀態時預設 active；非法值不可靜默改寫。
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "active"
	} else if status != "active" && status != "inactive" {
		return nil, ErrInvalidStatus
	}

	site := Site{
		Name:    name,
		Address: address,
		Region:  region,
		Status:  status,
	}
	if err := s.store.Create(ctx, &site); err != nil {
		return nil, err
	}
	writeAuditBestEffort(ctx, s.auditRepo, actorOrEmpty(actors), "create", "sites", site.ID, nil, site.AuditSnapshot())
	return &site, nil
}

// UpdateSiteInput 代表更新據點所需之輸入。
type UpdateSiteInput struct {
	Name    string
	Address string
	Region  string
	Status  string
}

// Update 更新據點主檔。
func (s *SiteService) Update(ctx context.Context, id uuid.UUID, in UpdateSiteInput, actors ...ActorContext) (*Site, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrSiteNameRequired
	}
	address := strings.TrimSpace(in.Address)
	region := strings.TrimSpace(in.Region)

	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "active"
	} else if status != "active" && status != "inactive" {
		return nil, ErrInvalidStatus
	}

	var before interface{}
	if s.auditRepo != nil {
		existing, err := s.store.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, ErrSiteNotFound
		}
		before = existing.AuditSnapshot()
	}

	site := Site{
		ID:      id,
		Name:    name,
		Address: address,
		Region:  region,
		Status:  status,
	}
	if err := s.store.Update(ctx, &site); err != nil {
		return nil, err
	}
	writeAuditBestEffort(ctx, s.auditRepo, actorOrEmpty(actors), "update", "sites", id, before, site.AuditSnapshot())
	return &site, nil
}

// Delete 刪除據點。若該據點仍被個案排班參照，資料庫外鍵限制會回傳錯誤。
func (s *SiteService) Delete(ctx context.Context, id uuid.UUID, actors ...ActorContext) error {
	var before interface{}
	if s.auditRepo != nil {
		existing, err := s.store.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrSiteNotFound
		}
		before = existing.AuditSnapshot()
	}
	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}
	writeAuditBestEffort(ctx, s.auditRepo, actorOrEmpty(actors), "delete", "sites", id, before, nil)
	return nil
}
