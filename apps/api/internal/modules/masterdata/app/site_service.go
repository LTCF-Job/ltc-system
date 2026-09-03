package app

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// SiteService 封裝單位主檔業務邏輯。
type SiteService struct {
	store SiteStore
}

// NewSiteService 建立 SiteService 實例。
func NewSiteService(store SiteStore) *SiteService {
	return &SiteService{store: store}
}

// List 查詢單位清單。
func (s *SiteService) List(ctx context.Context, region, q, status string, page, pageSize int) ([]Site, int64, error) {
	return s.store.List(ctx, region, q, status, page, pageSize)
}

// CreateSiteInput 代表新增單位所需之輸入。
type CreateSiteInput struct {
	Name     string
	Address  string
	Region   string
	OpenDays []int16
	Status   string
}

// Create 新增單位主檔。
func (s *SiteService) Create(ctx context.Context, in CreateSiteInput) (*Site, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrSiteNameRequired
	}
	address := strings.TrimSpace(in.Address)
	if address == "" {
		return nil, ErrSiteAddressRequired
	}
	region := strings.TrimSpace(in.Region)
	if region == "" {
		return nil, ErrSiteRegionRequired
	}

	// 確保 status 符合資料庫 check constraint，空值或非法值預設 active
	status := strings.TrimSpace(in.Status)
	if status != "active" && status != "inactive" {
		status = "active"
	}

	openDays := in.OpenDays
	if len(openDays) == 0 {
		openDays = []int16{1, 2, 3, 4, 5}
	}

	site := Site{
		Name:     name,
		Address:  address,
		Region:   region,
		OpenDays: openDays,
		Status:   status,
	}
	if err := s.store.Create(ctx, &site); err != nil {
		return nil, err
	}
	return &site, nil
}

// UpdateSiteInput 代表更新單位所需之輸入。
type UpdateSiteInput struct {
	Name     string
	Address  string
	Region   string
	OpenDays []int16
	Status   string
}

// Update 更新單位主檔。
func (s *SiteService) Update(ctx context.Context, id uuid.UUID, in UpdateSiteInput) (*Site, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrSiteNameRequired
	}
	address := strings.TrimSpace(in.Address)
	if address == "" {
		return nil, ErrSiteAddressRequired
	}
	region := strings.TrimSpace(in.Region)
	if region == "" {
		return nil, ErrSiteRegionRequired
	}

	status := strings.TrimSpace(in.Status)
	if status != "active" && status != "inactive" {
		status = "active"
	}

	openDays := in.OpenDays
	if len(openDays) == 0 {
		openDays = []int16{1, 2, 3, 4, 5}
	}

	site := Site{
		ID:       id,
		Name:     name,
		Address:  address,
		Region:   region,
		OpenDays: openDays,
		Status:   status,
	}
	if err := s.store.Update(ctx, &site); err != nil {
		return nil, err
	}
	return &site, nil
}

// Delete 刪除單位。若該單位仍被個案排班參照，資料庫外鍵限制會回傳錯誤。
func (s *SiteService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.store.Delete(ctx, id)
}
