package app

import (
	"context"

	"github.com/google/uuid"
)

// SiteService 封裝據點主檔業務邏輯。
type SiteService struct {
	store SiteStore
}

// NewSiteService 建立 SiteService 實例。
func NewSiteService(store SiteStore) *SiteService {
	return &SiteService{store: store}
}

// List 查詢據點清單。
func (s *SiteService) List(ctx context.Context, region, q string, page, pageSize int) ([]Site, int64, error) {
	return s.store.List(ctx, region, q, page, pageSize)
}

// CreateSiteInput 代表新增據點所需之輸入。
type CreateSiteInput struct {
	Code     string
	Name     string
	Address  string
	Region   string
	OpenDays []int16
	Status   string
}

// Create 新增據點。
func (s *SiteService) Create(ctx context.Context, in CreateSiteInput) (*Site, error) {
	site := Site{
		Code:     in.Code,
		Name:     in.Name,
		Address:  in.Address,
		Region:   in.Region,
		OpenDays: in.OpenDays,
		Status:   in.Status,
	}
	if err := s.store.Create(ctx, &site); err != nil {
		return nil, err
	}
	return &site, nil
}

// UpdateSiteInput 代表更新據點所需之輸入。
type UpdateSiteInput struct {
	Code     string
	Name     string
	Address  string
	Region   string
	OpenDays []int16
	Status   string
}

// Update 更新據點。
func (s *SiteService) Update(ctx context.Context, id uuid.UUID, in UpdateSiteInput) (*Site, error) {
	site := Site{
		ID:       id,
		Code:     in.Code,
		Name:     in.Name,
		Address:  in.Address,
		Region:   in.Region,
		OpenDays: in.OpenDays,
		Status:   in.Status,
	}
	if err := s.store.Update(ctx, &site); err != nil {
		return nil, err
	}
	return &site, nil
}

// Delete 刪除據點。若該據點仍被個案排班參照，資料庫外鍵限制會回傳錯誤。
func (s *SiteService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.store.Delete(ctx, id)
}
