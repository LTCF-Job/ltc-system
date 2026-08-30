package app

import (
	"context"

	"github.com/google/uuid"
)

// CaregiverService 封裝照護人員主檔的 CRUD 業務邏輯與批次匯入流程。
type CaregiverService struct {
	store    CaregiverStore
	sites    SiteLookup
	reader   SpreadsheetReader
	renderer TemplateRenderer
}

// NewCaregiverService 建立 CaregiverService 實例。
func NewCaregiverService(store CaregiverStore, sites SiteLookup, reader SpreadsheetReader, renderer TemplateRenderer) *CaregiverService {
	return &CaregiverService{store: store, sites: sites, reader: reader, renderer: renderer}
}

// List 查詢照護人員清單。unresolvedLink 篩選單位待關聯既有據點的資料列，
// incomplete 篩選聯絡方式或備註缺漏待補齊的資料列。
func (s *CaregiverService) List(ctx context.Context, q string, unresolvedLink, incomplete bool, page, pageSize int) ([]Caregiver, int64, error) {
	return s.store.List(ctx, q, unresolvedLink, incomplete, page, pageSize)
}

// GetByID 依 UUID 取得照護人員。
func (s *CaregiverService) GetByID(ctx context.Context, id uuid.UUID) (*Caregiver, error) {
	return s.store.GetByID(ctx, id)
}

// CreateCaregiverInput 代表新增照護人員所需之輸入。
type CreateCaregiverInput struct {
	SiteID  *uuid.UUID
	Name    string
	Type    string
	Contact string
	Notes   string
}

// Create 新增照護人員。
func (s *CaregiverService) Create(ctx context.Context, in CreateCaregiverInput) (*Caregiver, error) {
	if in.Name == "" {
		return nil, ErrCaregiverNameRequired
	}
	if !IsValidCaregiverType(in.Type) {
		return nil, ErrCaregiverTypeInvalid
	}
	c := Caregiver{SiteID: in.SiteID, Name: in.Name, Type: in.Type, Contact: in.Contact, Notes: in.Notes}
	if err := s.store.Create(ctx, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// UpdateCaregiverInput 代表更新照護人員所需之輸入，欄位為 nil 表示不變更。
type UpdateCaregiverInput struct {
	SiteID  *uuid.UUID
	Name    *string
	Type    *string
	Contact *string
	Notes   *string
}

// Update 更新照護人員。
func (s *CaregiverService) Update(ctx context.Context, id uuid.UUID, in UpdateCaregiverInput) (*Caregiver, error) {
	existing, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrCaregiverNotFound
	}

	if in.SiteID != nil {
		existing.SiteID = in.SiteID
		// 手動選定單位即視為完成關聯，清空匯入時保留的原始單位名稱。
		existing.SiteNameRaw = ""
	}
	if in.Name != nil {
		if *in.Name == "" {
			return nil, ErrCaregiverNameRequired
		}
		existing.Name = *in.Name
	}
	if in.Type != nil {
		if !IsValidCaregiverType(*in.Type) {
			return nil, ErrCaregiverTypeInvalid
		}
		existing.Type = *in.Type
	}
	if in.Contact != nil {
		existing.Contact = *in.Contact
	}
	if in.Notes != nil {
		existing.Notes = *in.Notes
	}

	if err := s.store.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// Delete 刪除照護人員。
func (s *CaregiverService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.store.Delete(ctx, id)
}

// LinkSite 將單位待關聯的照護人員連結至既有據點，並清空原始單位名稱。
func (s *CaregiverService) LinkSite(ctx context.Context, id, siteID uuid.UUID) (*Caregiver, error) {
	return s.Update(ctx, id, UpdateCaregiverInput{SiteID: &siteID})
}
