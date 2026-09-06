package app

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// UserService 封裝使用者帳號管理業務邏輯，底層由 Supabase Auth Admin API 支撐。
type UserService struct {
	admin           AdminIdentityProvider
	roleStore       RoleStore
	auditRepo       AuditWriter
	permissionCache PermissionCacheInvalidator
	securityState   UserSecurityStateStore
	directory       UserDirectoryStore
	directoryMu     sync.Mutex
	directoryReady  bool
}

// userAuditSnapshot 是帳號異動稽核的安全快照；不直接保存 AuthUser，避免把電子郵件、電話、
// 顯示名稱或外部 Auth 回傳的其他個人資料寫入長期稽核資料。
type userAuditSnapshot struct {
	ID          uuid.UUID `json:"id"`
	EmailMasked string    `json:"emailMasked,omitempty"`
	RoleKey     string    `json:"roleKey"`
	Status      string    `json:"status"`
}

// userPermissionsAuditSnapshot 是個人權限覆寫的明確稽核 DTO；不直接保存 request map，
// 避免呼叫端後續修改造成稽核內容漂移。
type userPermissionsAuditSnapshot struct {
	UserID      uuid.UUID                   `json:"userId"`
	Permissions map[string]ModulePermission `json:"permissions"`
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

func newUserPermissionsAuditSnapshot(userID uuid.UUID, permissions map[string]ModulePermission) userPermissionsAuditSnapshot {
	copyPermissions := make(map[string]ModulePermission, len(permissions))
	for key, permission := range permissions {
		copyPermissions[key] = permission
	}
	return userPermissionsAuditSnapshot{UserID: userID, Permissions: copyPermissions}
}

// SetPermissionCacheInvalidator 設定使用者異動後的即時權限快取失效器。
func (s *UserService) SetPermissionCacheInvalidator(invalidator PermissionCacheInvalidator) {
	s.permissionCache = invalidator
}

// SetUserSecurityStateStore 設定跨 replica 共用的授權狀態投影寫入器。
func (s *UserService) SetUserSecurityStateStore(store UserSecurityStateStore) {
	s.securityState = store
}

// SetUserDirectoryStore 設定使用者清單的本地查詢投影。
func (s *UserService) SetUserDirectoryStore(store UserDirectoryStore) {
	s.directoryMu.Lock()
	defer s.directoryMu.Unlock()
	s.directory = store
	s.directoryReady = false
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
func (s *UserService) List(ctx context.Context, keyword, roleKey string, page, pageSize int) ([]AuthUser, int, error) {
	if err := s.requireConfigured(); err != nil {
		return nil, 0, err
	}
	if page < 1 || pageSize < 1 {
		return nil, 0, fmt.Errorf("page and pageSize must be positive")
	}
	if s.directory != nil {
		if err := s.ensureDirectoryReady(ctx); err != nil {
			return nil, 0, err
		}
		return s.directory.ListDirectoryUsers(ctx, UserDirectoryFilter{
			Keyword:  strings.TrimSpace(keyword),
			RoleKey:  strings.TrimSpace(roleKey),
			Page:     page,
			PageSize: pageSize,
		})
	}
	users, err := s.admin.ListUsers(ctx)
	if err != nil {
		return nil, 0, err
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
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].Email == filtered[j].Email {
			return filtered[i].ID.String() < filtered[j].ID.String()
		}
		return filtered[i].Email < filtered[j].Email
	})
	total := len(filtered)
	start := (page - 1) * pageSize
	if start >= total {
		return []AuthUser{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return filtered[start:end], total, nil
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
	if in.Status == "" {
		in.Status = "active"
	}
	if err := validateUserStatus(in.Status); err != nil {
		return nil, err
	}
	if err := validateModuleKeys(in.CustomPermissions); err != nil {
		return nil, err
	}
	if err := s.ensureAuditConfigured(); err != nil {
		return nil, err
	}

	u, err := s.admin.CreateUser(ctx, in)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, fmt.Errorf("identity provider returned an empty user")
	}
	// Supabase 的 create response 可能尚未反映 ban_duration 或 metadata 的
	// 正規化結果；API 回應與本次已接受的 request 仍須保持一致。
	u.Status = in.Status
	u.RoleKey = in.RoleKey
	u.CustomPermissions = in.CustomPermissions
	state := UserSecurityStateFromAuthUser(*u)
	state.Status = in.Status
	state.RoleKey = in.RoleKey
	state.CustomPermissions = in.CustomPermissions
	s.syncSecurityState(ctx, state)
	s.syncDirectoryUser(ctx, *u)
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
			s.logAuditFailure("create", u.ID, err)
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
	if in.Status != nil {
		if err := validateUserStatus(*in.Status); err != nil {
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
	if u == nil {
		return nil, fmt.Errorf("identity provider returned an empty user")
	}
	// UpdateUser 的外部 response 可能省略未變更的 metadata；未指定的欄位以 mutation
	// 前快照為準，避免 response 的預設值把停用帳號或個人覆寫誤判成其他狀態。
	if in.RoleKey != nil {
		u.RoleKey = *in.RoleKey
	} else {
		u.RoleKey = before.RoleKey
	}
	if in.Status != nil {
		u.Status = *in.Status
	} else {
		u.Status = before.Status
	}
	u.CustomPermissions = cloneModulePermissions(before.CustomPermissions)
	state := UserSecurityStateFromAuthUser(*u)
	if state.RoleKey == "" {
		state.RoleKey = before.RoleKey
	}
	if state.Status == "" {
		state.Status = before.Status
	}
	if state.CustomPermissions == nil {
		state.CustomPermissions = before.CustomPermissions
	}
	s.syncSecurityState(ctx, state)
	s.syncDirectoryUser(ctx, *u)

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
			s.logAuditFailure("update", id, err)
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
	before, err := s.admin.GetUser(ctx, id)
	if err != nil {
		return err
	}
	if before == nil {
		return ErrUserNotFound
	}
	if err := s.admin.SetCustomPermissions(ctx, id, perms); err != nil {
		return err
	}
	if s.securityState != nil {
		s.syncCustomPermissionsState(ctx, id, perms)
	}
	updated := *before
	updated.CustomPermissions = cloneModulePermissions(perms)
	s.syncDirectoryUser(ctx, updated)

	if s.auditRepo != nil {
		entityIDStr := id.String()
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "update_permissions",
			EntityType: "users",
			EntityID:   &entityIDStr,
			BeforeData: newUserPermissionsAuditSnapshot(id, before.CustomPermissions),
			AfterData:  newUserPermissionsAuditSnapshot(id, perms),
		}); err != nil {
			s.logAuditFailure("update_permissions", id, err)
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
	if s.securityState != nil {
		if err := s.securityState.DeleteSecurityState(ctx, id); err != nil {
			slog.Error("identity_security_state_delete_failed", slog.String("entity_id", id.String()), slog.Any("error", err))
		}
	}
	s.deleteDirectoryUser(ctx, id)

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
			s.logAuditFailure("delete", id, err)
		}
	}
	if s.permissionCache != nil {
		s.permissionCache.InvalidateUser(id)
	}
	return nil
}

// ChangeSelfPassword 讓已登入使用者變更自己的密碼；先以舊密碼向 Supabase 驗證身分，
// 通過後才呼叫 Admin API 設定新密碼，未通過驗證不會呼叫 Admin API。
func (s *UserService) ChangeSelfPassword(ctx context.Context, actorID uuid.UUID, email, oldPassword, newPassword string, actorRoles ...string) error {
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
		if len(actorRoles) > 0 {
			actorRole = actorRoles[0]
		}
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "change_password",
			EntityType: "users",
			EntityID:   &entityIDStr,
		}); err != nil {
			s.logAuditFailure("change_password", actorID, err)
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
			s.logAuditFailure("reset_password", id, err)
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

func validateUserStatus(status string) error {
	if status != "active" && status != "inactive" {
		return ErrInvalidUserStatus
	}
	return nil
}

func (s *UserService) logAuditFailure(action string, entityID uuid.UUID, err error) {
	slog.Error("identity_audit_write_failed", slog.String("action", action), slog.String("entity_id", entityID.String()), slog.Any("error", err))
}

// ensureDirectoryReady 只在本地投影尚未初始化時完整同步一次外部帳號，之後的清單查詢
// 一律由 PostgreSQL 以條件、排序與分頁完成。初始化失敗不標記 ready，下一次請求可重試。
func (s *UserService) ensureDirectoryReady(ctx context.Context) error {
	s.directoryMu.Lock()
	defer s.directoryMu.Unlock()
	if s.directory == nil || s.directoryReady {
		return nil
	}
	users, err := s.admin.ListUsers(ctx)
	if err != nil {
		return err
	}
	for _, user := range users {
		if err := s.directory.UpsertDirectoryUser(ctx, user); err != nil {
			return err
		}
	}
	s.directoryReady = true
	return nil
}

// syncDirectoryUser 將外部 mutation 成功後的結果寫入清單投影；投影失敗只記錄，
// 不讓已完成的外部異動被 client 重試。
func (s *UserService) syncDirectoryUser(ctx context.Context, user AuthUser) {
	if s.directory == nil {
		return
	}
	if err := s.directory.UpsertDirectoryUser(ctx, user); err != nil {
		slog.Error("identity_user_directory_write_failed", slog.String("action", "upsert"), slog.String("entity_id", user.ID.String()), slog.Any("error", err))
	}
}

func (s *UserService) deleteDirectoryUser(ctx context.Context, id uuid.UUID) {
	if s.directory == nil {
		return
	}
	if err := s.directory.DeleteDirectoryUser(ctx, id); err != nil {
		slog.Error("identity_user_directory_write_failed", slog.String("action", "delete"), slog.String("entity_id", id.String()), slog.Any("error", err))
	}
}

// syncSecurityState 將外部帳號異動同步到共享投影；外部操作已成功時，投影故障不得讓 client 重試同一 mutation。
func (s *UserService) syncSecurityState(ctx context.Context, state UserSecurityState) {
	if s.securityState == nil {
		return
	}
	if state.Status == "" {
		state.Status = "active"
	}
	if err := s.securityState.UpsertSecurityState(ctx, state); err != nil {
		slog.Error("identity_security_state_write_failed", slog.String("entity_id", state.UserID.String()), slog.Any("error", err))
	}
}

// syncCustomPermissionsState 更新既有投影；若是尚未建立的舊帳號，先從外部身分
// 讀取完整安全狀態再建立，避免 repository 的 upsert 預設值把停用帳號變成啟用。
func (s *UserService) syncCustomPermissionsState(ctx context.Context, id uuid.UUID, perms map[string]ModulePermission) {
	state, err := s.securityState.GetSecurityState(ctx, id)
	if err != nil {
		slog.Error("identity_security_state_read_failed", slog.String("action", "update_permissions"), slog.String("entity_id", id.String()), slog.Any("error", err))
		return
	}
	if state != nil {
		if err := s.securityState.UpdateCustomPermissions(ctx, id, perms); err != nil {
			slog.Error("identity_security_state_write_failed", slog.String("action", "update_permissions"), slog.String("entity_id", id.String()), slog.Any("error", err))
		}
		return
	}

	user, err := s.admin.GetUser(ctx, id)
	if err != nil {
		slog.Error("identity_security_state_source_read_failed", slog.String("action", "update_permissions"), slog.String("entity_id", id.String()), slog.Any("error", err))
		return
	}
	if user == nil {
		slog.Error("identity_security_state_source_missing", slog.String("action", "update_permissions"), slog.String("entity_id", id.String()))
		return
	}
	newState := UserSecurityStateFromAuthUser(*user)
	newState.CustomPermissions = perms
	s.syncSecurityState(ctx, newState)
}

// UserSecurityStateFromAuthUser 只把授權需要的欄位投影到本地狀態表。
func UserSecurityStateFromAuthUser(user AuthUser) UserSecurityState {
	return UserSecurityState{
		UserID:            user.ID,
		Status:            user.Status,
		RoleKey:           user.RoleKey,
		CustomPermissions: user.CustomPermissions,
		UpdatedAt:         user.UpdatedAt,
	}
}

func cloneModulePermissions(perms map[string]ModulePermission) map[string]ModulePermission {
	if perms == nil {
		return nil
	}
	cloned := make(map[string]ModulePermission, len(perms))
	for key, permission := range perms {
		cloned[key] = permission
	}
	return cloned
}
