package infra

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/identity/app"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// UserSecurityStateRepository 是授權與停用檢查使用的共享本地投影。
type UserSecurityStateRepository struct {
	db *pgxpool.Pool
}

// NewUserSecurityStateRepository 建立使用者安全狀態投影 repository。
func NewUserSecurityStateRepository(db *pgxpool.Pool) *UserSecurityStateRepository {
	return &UserSecurityStateRepository{db: db}
}

func (r *UserSecurityStateRepository) GetSecurityState(ctx context.Context, id uuid.UUID) (*app.UserSecurityState, error) {
	if r.db == nil {
		return nil, fmt.Errorf("user security state database is not configured")
	}
	db := pgxdb.FromContext(ctx, r.db)
	var state app.UserSecurityState
	var permissionBytes []byte
	err := db.QueryRow(ctx, `
		SELECT user_id, status, role_key, custom_permissions, permission_version, updated_at
		FROM auth_user_security_states
		WHERE user_id = $1
	`, id).Scan(
		&state.UserID,
		&state.Status,
		&state.RoleKey,
		&permissionBytes,
		&state.PermissionVersion,
		&state.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if len(permissionBytes) > 0 {
		if err := json.Unmarshal(permissionBytes, &state.CustomPermissions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal user security permissions: %w", err)
		}
	}
	return &state, nil
}

// UpsertSecurityState 建立或遞增使用者的共享授權版本。
func (r *UserSecurityStateRepository) UpsertSecurityState(ctx context.Context, state app.UserSecurityState) error {
	if r.db == nil {
		return fmt.Errorf("user security state database is not configured")
	}
	if state.Status == "" {
		state.Status = "active"
	}
	permissionBytes, err := json.Marshal(state.CustomPermissions)
	if err != nil {
		return fmt.Errorf("failed to marshal user security permissions: %w", err)
	}
	db := pgxdb.FromContext(ctx, r.db)
	// 連線走 simple protocol（見 cmd/server/main.go），[]byte 會被編成 bytea 字面值而無法寫入
	// jsonb 欄位，必須以字串傳入並明確轉型。
	_, err = db.Exec(ctx, `
		INSERT INTO auth_user_security_states (user_id, status, role_key, custom_permissions)
		VALUES ($1, $2, $3, $4::jsonb)
		ON CONFLICT (user_id) DO UPDATE SET
			status = EXCLUDED.status,
			role_key = EXCLUDED.role_key,
			custom_permissions = EXCLUDED.custom_permissions,
			permission_version = auth_user_security_states.permission_version + 1,
			updated_at = now()
	`, state.UserID, state.Status, state.RoleKey, string(permissionBytes))
	return err
}

// UpdateCustomPermissions 只更新個人覆蓋權限，保留狀態與角色並遞增共享版本。
func (r *UserSecurityStateRepository) UpdateCustomPermissions(ctx context.Context, id uuid.UUID, perms map[string]app.ModulePermission) error {
	if r.db == nil {
		return fmt.Errorf("user security state database is not configured")
	}
	permissionBytes, err := json.Marshal(perms)
	if err != nil {
		return fmt.Errorf("failed to marshal user security permissions: %w", err)
	}
	db := pgxdb.FromContext(ctx, r.db)
	_, err = db.Exec(ctx, `
		INSERT INTO auth_user_security_states (user_id, status, role_key, custom_permissions)
		VALUES ($1, 'active', '', $2::jsonb)
		ON CONFLICT (user_id) DO UPDATE SET
			custom_permissions = EXCLUDED.custom_permissions,
			permission_version = auth_user_security_states.permission_version + 1,
			updated_at = now()
	`, id, string(permissionBytes))
	return err
}

// DeleteSecurityState 刪除已不存在帳號的本地授權投影。
func (r *UserSecurityStateRepository) DeleteSecurityState(ctx context.Context, id uuid.UUID) error {
	if r.db == nil {
		return fmt.Errorf("user security state database is not configured")
	}
	db := pgxdb.FromContext(ctx, r.db)
	_, err := db.Exec(ctx, `DELETE FROM auth_user_security_states WHERE user_id = $1`, id)
	return err
}
