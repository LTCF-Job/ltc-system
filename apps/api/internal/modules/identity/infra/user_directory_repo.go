package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/identity/app"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// UserDirectoryRepository 保存 Supabase Auth 使用者的可查詢本地投影。
// 它只服務後台清單查詢；登入與授權狀態仍由 Auth／security state projection 負責。
type UserDirectoryRepository struct {
	db *pgxpool.Pool
}

// NewUserDirectoryRepository 建立使用者目錄 repository。
func NewUserDirectoryRepository(db *pgxpool.Pool) *UserDirectoryRepository {
	return &UserDirectoryRepository{db: db}
}

// ListDirectoryUsers 由 PostgreSQL 執行關鍵字、角色、排序與分頁，避免每次請求
// 都完整掃描 Supabase Admin API 的所有使用者。
func (r *UserDirectoryRepository) ListDirectoryUsers(ctx context.Context, filter app.UserDirectoryFilter) ([]app.AuthUser, int, error) {
	if r.db == nil {
		return nil, 0, fmt.Errorf("user directory database is not configured")
	}
	keyword := strings.TrimSpace(filter.Keyword)
	roleKey := strings.TrimSpace(filter.RoleKey)
	offset := (filter.Page - 1) * filter.PageSize
	rows, err := pgxdb.FromContext(ctx, r.db).Query(ctx, `
		SELECT user_id, email, display_name, phone, role_key, status,
		       custom_permissions, created_at, updated_at, last_sign_in_at,
		       COUNT(*) OVER() AS total_count
		FROM auth_user_directory
		WHERE ($1 = '' OR email ILIKE '%' || $1 || '%' OR display_name ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR role_key = $2)
		ORDER BY email ASC, user_id ASC
		LIMIT $3 OFFSET $4
	`, keyword, roleKey, filter.PageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query user directory: %w", err)
	}
	defer rows.Close()

	users := make([]app.AuthUser, 0)
	total := 0
	for rows.Next() {
		var user app.AuthUser
		var permissionBytes []byte
		var rowTotal int64
		if err := rows.Scan(
			&user.ID, &user.Email, &user.DisplayName, &user.Phone, &user.RoleKey, &user.Status,
			&permissionBytes, &user.CreatedAt, &user.UpdatedAt, &user.LastSignInAt, &rowTotal,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user directory: %w", err)
		}
		if len(permissionBytes) > 0 {
			if err := json.Unmarshal(permissionBytes, &user.CustomPermissions); err != nil {
				return nil, 0, fmt.Errorf("failed to unmarshal user directory permissions: %w", err)
			}
		}
		user.Role = user.RoleKey
		users = append(users, user)
		total = int(rowTotal)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate user directory: %w", err)
	}
	return users, total, nil
}

// UpsertDirectoryUser 更新外部 Auth 使用者的清單投影。
func (r *UserDirectoryRepository) UpsertDirectoryUser(ctx context.Context, user app.AuthUser) error {
	if r.db == nil {
		return fmt.Errorf("user directory database is not configured")
	}
	if user.Status == "" {
		user.Status = "active"
	}
	if user.RoleKey == "" {
		user.RoleKey = user.Role
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = user.CreatedAt
	}
	permissionBytes, err := json.Marshal(user.CustomPermissions)
	if err != nil {
		return fmt.Errorf("failed to marshal user directory permissions: %w", err)
	}
	// 連線走 simple protocol（見 cmd/server/main.go），[]byte 會被編成 bytea 字面值而無法寫入
	// jsonb 欄位，必須以字串傳入並明確轉型。
	_, err = pgxdb.FromContext(ctx, r.db).Exec(ctx, `
		INSERT INTO auth_user_directory (
			user_id, email, display_name, phone, role_key, status,
			custom_permissions, created_at, updated_at, last_sign_in_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10)
		ON CONFLICT (user_id) DO UPDATE SET
			email = EXCLUDED.email,
			display_name = EXCLUDED.display_name,
			phone = EXCLUDED.phone,
			role_key = EXCLUDED.role_key,
			status = EXCLUDED.status,
			custom_permissions = EXCLUDED.custom_permissions,
			created_at = EXCLUDED.created_at,
			updated_at = EXCLUDED.updated_at,
			last_sign_in_at = EXCLUDED.last_sign_in_at
	`, user.ID, user.Email, user.DisplayName, user.Phone, user.RoleKey, user.Status,
		string(permissionBytes), user.CreatedAt, user.UpdatedAt, user.LastSignInAt)
	return err
}

// DeleteDirectoryUser 移除已從外部 Auth 刪除的使用者投影。
func (r *UserDirectoryRepository) DeleteDirectoryUser(ctx context.Context, id uuid.UUID) error {
	if r.db == nil {
		return fmt.Errorf("user directory database is not configured")
	}
	_, err := pgxdb.FromContext(ctx, r.db).Exec(ctx, `DELETE FROM auth_user_directory WHERE user_id = $1`, id)
	return err
}

var _ app.UserDirectoryStore = (*UserDirectoryRepository)(nil)
