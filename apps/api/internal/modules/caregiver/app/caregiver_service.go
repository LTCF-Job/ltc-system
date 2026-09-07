package app

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
)

// CaregiverService 封裝照護人員主檔的 CRUD 業務邏輯與批次匯入流程。
type CaregiverService struct {
	store     CaregiverStore
	sites     SiteLookup
	reader    SpreadsheetReader
	renderer  TemplateRenderer
	auditRepo AuditWriter
}

// NewCaregiverService 建立 CaregiverService 實例。
func NewCaregiverService(store CaregiverStore, sites SiteLookup, reader SpreadsheetReader, renderer TemplateRenderer, audits ...AuditWriter) *CaregiverService {
	var auditRepo AuditWriter
	if len(audits) > 0 {
		auditRepo = audits[0]
	}
	return &CaregiverService{store: store, sites: sites, reader: reader, renderer: renderer, auditRepo: auditRepo}
}

// List 查詢照護人員清單。pending 只取姓名或類型未填寫、待人工補齊的資料列，
// excludePending 反之排除這些資料列。
func (s *CaregiverService) List(ctx context.Context, q, status string, pending, excludePending bool, page, pageSize int) ([]Caregiver, int64, error) {
	return s.store.List(ctx, q, status, pending, excludePending, page, pageSize)
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
	Status  string
}

// Create 新增照護人員。
func (s *CaregiverService) Create(ctx context.Context, in CreateCaregiverInput, actors ...ActorContext) (*Caregiver, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, ErrCaregiverNameRequired
	}
	if !IsValidCaregiverType(in.Type) {
		return nil, ErrCaregiverTypeInvalid
	}

	// 空值採用預設狀態；明確傳入的非法狀態必須拒絕，避免 typo 被靜默升權為 active。
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "active"
	} else if status != "active" && status != "inactive" {
		return nil, ErrCaregiverStatusInvalid
	}

	c := Caregiver{SiteID: in.SiteID, Name: in.Name, Type: in.Type, Contact: in.Contact, Notes: in.Notes, Status: status}
	if err := s.store.Create(ctx, &c); err != nil {
		return nil, err
	}
	s.writeAudit(ctx, "create", c.ID, actorOrEmpty(actors), nil, c.AuditSnapshot())
	return &c, nil
}

// UpdateCaregiverInput 代表更新照護人員所需之輸入，欄位為 nil 表示不變更。
type UpdateCaregiverInput struct {
	SiteID  *uuid.UUID
	Name    *string
	Type    *string
	Contact *string
	Notes   *string
	Status  *string
}

// Update 更新照護人員。
func (s *CaregiverService) Update(ctx context.Context, id uuid.UUID, in UpdateCaregiverInput, actors ...ActorContext) (*Caregiver, error) {
	existing, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrCaregiverNotFound) {
			return nil, ErrCaregiverNotFound
		}
		return nil, err
	}
	if existing == nil {
		return nil, ErrCaregiverNotFound
	}
	before := existing.AuditSnapshot()

	if in.SiteID != nil {
		existing.SiteID = in.SiteID
		// 手動選定單位即視為完成關聯，清空匯入時保留的原始單位名稱。
		existing.SiteNameRaw = ""
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, ErrCaregiverNameRequired
		}
		existing.Name = name
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
	if in.Status != nil {
		status := strings.TrimSpace(*in.Status)
		if status != "active" && status != "inactive" {
			return nil, ErrCaregiverStatusInvalid
		}
		existing.Status = status
	}

	if err := s.store.Update(ctx, existing); err != nil {
		return nil, err
	}
	s.writeAudit(ctx, "update", id, actorOrEmpty(actors), before, existing.AuditSnapshot())
	return existing, nil
}

// Delete 刪除照護人員。
func (s *CaregiverService) Delete(ctx context.Context, id uuid.UUID, actors ...ActorContext) error {
	var before interface{}
	if s.auditRepo != nil {
		existing, err := s.store.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if existing == nil {
			return ErrCaregiverNotFound
		}
		before = existing.AuditSnapshot()
	}
	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}
	s.writeAudit(ctx, "delete", id, actorOrEmpty(actors), before, nil)
	return nil
}

// LinkSite 將待關聯的照護人員連結至單位主檔，並清空原始單位名稱。
func (s *CaregiverService) LinkSite(ctx context.Context, id, siteID uuid.UUID, actors ...ActorContext) (*Caregiver, error) {
	return s.Update(ctx, id, UpdateCaregiverInput{SiteID: &siteID}, actors...)
}

func actorOrEmpty(actors []ActorContext) ActorContext {
	if len(actors) == 0 {
		return ActorContext{}
	}
	return actors[0]
}

func (s *CaregiverService) writeAudit(ctx context.Context, action string, id uuid.UUID, actor ActorContext, before, after interface{}) {
	if s.auditRepo == nil {
		return
	}
	entityID := id.String()
	if err := s.auditRepo.Write(ctx, AuditEntry{
		ActorID:    &actor.ActorID,
		ActorRole:  &actor.ActorRole,
		Action:     action,
		EntityType: "caregivers",
		EntityID:   &entityID,
		BeforeData: before,
		AfterData:  after,
		IPAddress:  &actor.IPAddress,
		UserAgent:  &actor.UserAgent,
	}); err != nil {
		slog.Error("caregiver audit write failed", "action", action, "entity_id", entityID, "error", err)
	}
}
