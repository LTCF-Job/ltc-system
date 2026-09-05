package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// UserService 封裝使用者帳號管理業務邏輯，底層由 Supabase Auth Admin API 支撐。
type UserService struct {
	admin           AdminIdentityProvider
	roleStore       RoleStore
	auditRepo       AuditWriter
	permissionCache PermissionCacheInvalidator
}

// userAuditSnapshot 是帳號異動稽核的安全快照；不直接保存 AuthUser，避免把電子郵件、電話、
// 顯示名稱或外部 Auth 回傳的其他個人資料寫入長期稽核資料。
type userAuditSnapshot struct {
	ID          uuid.UUID `json:"id"`
	EmailMasked string    `json:"emailMasked,omitempty"`
	RoleKey     string    `json:"roleKey"`
	Status      string    `json:"status"`
}

func newUserAuditSnapshot(user *AuthUser) userAuditSnapshot {
	if user == nil {
		return userAuditSnapshot{}
	}
	emailMasked := ""
	if strings.TrimSpace(user.Email) != "" {
		emailMasked = "[REDACTED]"
	}
	return userAuditSnapshot{
		ID:          user.ID,
		EmailMasked: emailMasked,
		RoleKey:     user.RoleKey,
		Status:      user.Status,
	}
}

// SetPermissionCacheInvalidator 設定使用者異動後的即時權限快取失效器。
func (s *UserService) SetPermissionCacheInvalidator(invalidator PermissionCacheInvalidator) {
	s.permissionCache = invalidator
}

// NewUserService 建立 UserService 實例。
func NewUserService(admin AdminIdentityProvider, roleStore RoleStore, auditRepo AuditWriter) *UserService {
	return &UserService{admin: admin, roleStore: roleStore, auditRepo: auditRepo}
}

func (s *UserService) requireConfigured() error {
	if s.admin == nil || !s.admin.Configured() {
		return ErrIdentityProviderUnconfigured
	}
	return nil
}

// List 取得使用者清單，支援關鍵字與角色篩選。
func (s *UserService) List(ctx context.Context, keyword, roleKey string) ([]AuthUser, error) {
	if err := s.requireConfigured(); err != nil {
		return nil, err
	}
	users, err := s.admin.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]AuthUser, 0, len(users))
	for _, u := range users {
		if roleKey != "" && u.RoleKey != roleKey {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(u.Email), strings.ToLower(keyword)) &&
			!strings.Contains(strings.ToLower(u.DisplayName), strings.ToLower(keyword)) {
			continue
		}
		filtered = append(filtered, u)
	}
	return filtered, nil
}

// Get 取得單一使用者。
func (s *UserService) Get(ctx context.Context, id uuid.UUID) (*AuthUser, error) {
	if err := s.requireConfigured(); err != nil {
		return nil, err
	}
	u, err := s.admin.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrUserNotFound
	}
	return u, nil
}

// Create 建立新使用者；role 須存在於 roles 表。
func (s *UserService) Create(ctx context.Context, in CreateAuthUserInput, actorID uuid.UUID, actorRole string) (*AuthUser, error) {
	if err := s.requireConfigured(); err != nil {
		return nil, err
	}
	if err := s.checkRoleExists(ctx, in.RoleKey); err != nil {
		return nil, err
	}
	if err := s.ensureAuditConfigured(); err != nil {
		return nil, err
	}

	u, err := s.admin.CreateUser(ctx, in)
	if err != nil {
		return nil, err
	}
	if err := s.ensureAuditConfigured(); err != nil {
		return nil, err
	}

	// Admin API 是外部系統，沒有交易可言：呼叫成功後才寫稽核，稽核失敗只記 log 不讓已建立的
	// 使用者回報失敗。
	if s.auditRepo != nil {
		entityIDStr := u.ID.String()
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "create",
			EntityType: "users",
			EntityID:   &entityIDStr,
			AfterData:  newUserAuditSnapshot(u),
		}); err != nil {
			return nil, fmt.Errorf("failed to write user audit: %w", err)
		}
	}
	if s.permissionCache != nil {
		s.permissionCache.InvalidateUser(u.ID)
	}
	return u, nil
}

// Update 更新使用者基本資料與角色。
func (s *UserService) Update(ctx context.Context, id uuid.UUID, in UpdateAuthUserInput, actorID uuid.UUID, actorRole string) (*AuthUser, error) {
	if err := s.requireConfigured(); err != nil {
		return nil, err
	}
	if in.RoleKey != nil {
		if err := s.checkRoleExists(ctx, *in.RoleKey); err != nil {
			return nil, err
		}
	}

	before, err := s.admin.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	if before == nil {
		return nil, ErrUserNotFound
	}
	if err := s.ensureAuditConfigured(); err != nil {
		return nil, err
	}

	u, err := s.admin.UpdateUser(ctx, id, in)
	if err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		entityIDStr := id.String()
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "update",
			EntityType: "users",
			EntityID:   &entityIDStr,
			BeforeData: newUserAuditSnapshot(before),
			AfterData:  newUserAuditSnapshot(u),
		}); err != nil {
			return nil, fmt.Errorf("failed to write user audit: %w", err)
		}
	}
	if s.permissionCache != nil {
		s.permissionCache.InvalidateUser(id)
		if before != nil && in.RoleKey != nil {
			s.permissionCache.InvalidateRole(before.RoleKey)
			s.permissionCache.InvalidateRole(*in.RoleKey)
		}
	}
	return u, nil
}

// UpdatePermissions 覆寫使用者個人自訂權限。
func (s *UserService) UpdatePermissions(ctx context.Context, id uuid.UUID, perms map[string]ModulePermission, actorID uuid.UUID, actorRole string) error {
	if err := s.requireConfigured(); err != nil {
		return err
	}
	if err := validateModuleKeys(perms); err != nil {
		return err
	}
	if err := s.ensureAuditConfigured(); err != nil {
		return err
	}
	if err := s.admin.SetCustomPermissions(ctx, id, perms); err != nil {
		return err
	}

	if s.auditRepo != nil {
		entityIDStr := id.String()
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "update_permissions",
			EntityType: "users",
			EntityID:   &entityIDStr,
			AfterData:  perms,
		}); err != nil {
			return fmt.Errorf("failed to write user audit: %w", err)
		}
	}
	if s.permissionCache != nil {
		s.permissionCache.InvalidateUser(id)
	}
	return nil
}

// Delete 刪除使用者；不可刪除自己的帳號。
func (s *UserService) Delete(ctx context.Context, id, actorID uuid.UUID, actorRole string) error {
	if err := s.requireConfigured(); err != nil {
		return err
	}
	if id == actorID {
		return ErrCannotDeleteSelf
	}
	if err := s.ensureAuditConfigured(); err != nil {
		return err
	}

	before, err := s.admin.GetUser(ctx, id)
	if err != nil {
		return err
	}

	if err := s.admin.DeleteUser(ctx, id); err != nil {
		return err
	}

	if s.auditRepo != nil {
		entityIDStr := id.String()
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "delete",
			EntityType: "users",
			EntityID:   &entityIDStr,
			BeforeData: newUserAuditSnapshot(before),
		}); err != nil {
			return fmt.Errorf("failed to write user audit: %w", err)
		}
	}
	if s.permissionCache != nil {
		s.permissionCache.InvalidateUser(id)
	}
	return nil
}

// ChangeSelfPassword 讓已登入使用者變更自己的密碼；先以舊密碼向 Supabase 驗證身分，
// 通過後才呼叫 Admin API 設定新密碼，未通過驗證不會呼叫 Admin API。
func (s *UserService) ChangeSelfPassword(ctx context.Context, actorID uuid.UUID, email, oldPassword, newPassword string) error {
	if err := s.requireConfigured(); err != nil {
		return err
	}

	if err := s.admin.VerifyPassword(ctx, email, oldPassword); err != nil {
		return ErrInvalidCredentials
	}
	if err := s.ensureAuditConfigured(); err != nil {
		return err
	}

	if err := s.admin.SetPassword(ctx, actorID, newPassword); err != nil {
		return fmt.Errorf("failed to set new password: %w", err)
	}

	if s.auditRepo != nil {
		entityIDStr := actorID.String()
		actorRole := ""
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "change_password",
			EntityType: "users",
			EntityID:   &entityIDStr,
		}); err != nil {
			return fmt.Errorf("failed to write user audit: %w", err)
		}
	}
	return nil
}

// ResetPassword 讓管理員直接設定他人密碼，不需驗證舊密碼；不可用來重設自己的帳號。
func (s *UserService) ResetPassword(ctx context.Context, id, actorID uuid.UUID, actorRole, newPassword string) error {
	if err := s.requireConfigured(); err != nil {
		return err
	}
	if id == actorID {
		return ErrCannotResetOwnPassword
	}
	if err := s.ensureAuditConfigured(); err != nil {
		return err
	}

	if err := s.admin.SetPassword(ctx, id, newPassword); err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	if s.auditRepo != nil {
		entityIDStr := id.String()
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "reset_password",
			EntityType: "users",
			EntityID:   &entityIDStr,
		}); err != nil {
			return fmt.Errorf("failed to write user audit: %w", err)
		}
	}
	return nil
}

func (s *UserService) checkRoleExists(ctx context.Context, roleKey string) error {
	if roleKey == "" {
		return nil
	}
	role, err := s.roleStore.GetByKey(ctx, roleKey)
	if err != nil {
		return err
	}
	if role == nil {
		return ErrUnknownRole
	}
	return nil
}

func (s *UserService) ensureAuditConfigured() error {
	if s.auditRepo == nil {
		return ErrAuditUnavailable
	}
	return nil
}
