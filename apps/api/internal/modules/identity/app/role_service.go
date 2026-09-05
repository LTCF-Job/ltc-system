package app

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// RoleService 封裝角色身分與權限矩陣之業務邏輯。
type RoleService struct {
	store           RoleStore
	userCounter     UserCounter
	auditRepo       AuditWriter
	txRunner        TxRunner
	permissionCache PermissionCacheInvalidator
}

// SetPermissionCacheInvalidator 設定角色異動後的即時權限快取失效器。
func (s *RoleService) SetPermissionCacheInvalidator(invalidator PermissionCacheInvalidator) {
	s.permissionCache = invalidator
}

// NewRoleService 建立 RoleService 實例。
func NewRoleService(store RoleStore, userCounter UserCounter, auditRepo AuditWriter, txRunner TxRunner) *RoleService {
	return &RoleService{store: store, userCounter: userCounter, auditRepo: auditRepo, txRunner: txRunner}
}

func (s *RoleService) countUsers(ctx context.Context, key string) (int, error) {
	if s.userCounter == nil {
		return 0, nil
	}
	return s.userCounter.CountUsersByRoleKey(ctx, key)
}

// List 取得角色清單，並補上各角色實際使用者數。
func (s *RoleService) List(ctx context.Context) ([]Role, error) {
	roles, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	return s.fillUserCounts(ctx, roles)
}

func (s *RoleService) fillUserCounts(ctx context.Context, roles []Role) ([]Role, error) {
	for i := range roles {
		count, err := s.countUsers(ctx, roles[i].Key)
		if err != nil {
			return nil, err
		}
		roles[i].UserCount = count
	}
	return roles, nil
}

// GetByID 取得單一角色。
func (s *RoleService) GetByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	role, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}
	count, err := s.countUsers(ctx, role.Key)
	if err != nil {
		return nil, err
	}
	role.UserCount = count
	return role, nil
}

// CreateRoleInput 是新增角色的輸入。Key 未提供時由 Name 產生 slug。
type CreateRoleInput struct {
	Key         string
	Name        string
	Description string
	TagType     string
	BaseRole    string
	Permissions map[string]ModulePermission
}

// Create 新增自訂角色。
func (s *RoleService) Create(ctx context.Context, in CreateRoleInput, actorID uuid.UUID, actorRole string) (*Role, error) {
	if err := validateModuleKeys(in.Permissions); err != nil {
		return nil, err
	}
	if err := s.ensureAuditConfigured(); err != nil {
		return nil, err
	}
	key := strings.TrimSpace(in.Key)
	if key == "" {
		key = slugify(in.Name)
	}
	key, err := s.ensureUniqueKey(ctx, key)
	if err != nil {
		return nil, err
	}

	baseRole := in.BaseRole
	if baseRole == "" {
		baseRole = "viewer"
	}

	role := &Role{
		ID:          uuid.New(),
		Key:         key,
		Name:        in.Name,
		Description: in.Description,
		TagType:     in.TagType,
		IsSystem:    false,
		BaseRole:    baseRole,
		Permissions: in.Permissions,
	}

	err = s.runInTx(ctx, func(ctx context.Context) error {
		if err := s.store.Create(ctx, role); err != nil {
			return err
		}
		return s.writeAudit(ctx, "create", role.ID, actorID, actorRole, nil, role)
	})
	if err != nil {
		return nil, err
	}
	return role, nil
}

// UpdateRoleInput 是更新角色的輸入，nil 欄位維持現況。
type UpdateRoleInput struct {
	Name        *string
	Description *string
	TagType     *string
	BaseRole    *string
	Permissions map[string]ModulePermission
}

// Update 更新自訂角色；系統角色不可修改。
func (s *RoleService) Update(ctx context.Context, id uuid.UUID, in UpdateRoleInput, actorID uuid.UUID, actorRole string) (*Role, error) {
	if err := validateModuleKeys(in.Permissions); err != nil {
		return nil, err
	}

	before, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if before == nil {
		return nil, ErrRoleNotFound
	}
	if before.IsSystem {
		return nil, ErrSystemRoleImmutable
	}
	if err := s.ensureAuditConfigured(); err != nil {
		return nil, err
	}

	after := *before
	if in.Name != nil {
		after.Name = *in.Name
	}
	if in.Description != nil {
		after.Description = *in.Description
	}
	if in.TagType != nil {
		after.TagType = *in.TagType
	}
	if in.BaseRole != nil {
		after.BaseRole = *in.BaseRole
	}
	if in.Permissions != nil {
		after.Permissions = in.Permissions
	}

	err = s.runInTx(ctx, func(ctx context.Context) error {
		if err := s.store.Update(ctx, &after); err != nil {
			return err
		}
		return s.writeAudit(ctx, "update", id, actorID, actorRole, before, &after)
	})
	if err != nil {
		return nil, err
	}
	if s.permissionCache != nil {
		s.permissionCache.InvalidateRole(after.Key)
	}
	return &after, nil
}

// Delete 刪除自訂角色；系統角色或仍有使用者的角色不可刪除。
func (s *RoleService) Delete(ctx context.Context, id, actorID uuid.UUID, actorRole string) error {
	before, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if before == nil {
		return ErrRoleNotFound
	}
	if before.IsSystem {
		return ErrSystemRoleImmutable
	}

	count, err := s.countUsers(ctx, before.Key)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrRoleInUse
	}
	if err := s.ensureAuditConfigured(); err != nil {
		return err
	}

	err = s.runInTx(ctx, func(ctx context.Context) error {
		if err := s.store.Delete(ctx, id); err != nil {
			return err
		}
		return s.writeAudit(ctx, "delete", id, actorID, actorRole, before, nil)
	})
	if err == nil && s.permissionCache != nil {
		s.permissionCache.InvalidateRole(before.Key)
	}
	return err
}

func (s *RoleService) runInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if s.txRunner == nil {
		return fn(ctx)
	}
	return s.txRunner.WithTx(ctx, fn)
}

func (s *RoleService) writeAudit(ctx context.Context, action string, id, actorID uuid.UUID, actorRole string, before, after any) error {
	if s.auditRepo == nil {
		return ErrAuditUnavailable
	}
	entityIDStr := id.String()
	return s.auditRepo.Write(ctx, AuditEntry{
		ActorID:    &actorID,
		ActorRole:  &actorRole,
		Action:     action,
		EntityType: "roles",
		EntityID:   &entityIDStr,
		BeforeData: before,
		AfterData:  after,
	})
}

func (s *RoleService) ensureAuditConfigured() error {
	if s.auditRepo == nil {
		return ErrAuditUnavailable
	}
	return nil
}

var slugSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

// slugify 把角色名稱轉成適合當 key 的 slug；全部字元皆無法轉換時退回固定值 "role"。
func slugify(name string) string {
	s := slugSanitizer.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "_")
	s = strings.Trim(s, "_")
	if s == "" {
		s = "role"
	}
	return s
}

// ensureUniqueKey 在 key 已存在時附加遞增序號，直到找到未被使用的 key。
func (s *RoleService) ensureUniqueKey(ctx context.Context, base string) (string, error) {
	key := base
	for i := 1; ; i++ {
		existing, err := s.store.GetByKey(ctx, key)
		if err != nil {
			return "", err
		}
		if existing == nil {
			return key, nil
		}
		key = fmt.Sprintf("%s_%d", base, i)
	}
}
