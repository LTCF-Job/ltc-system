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

// RoleRepository 提供 roles 資料表之存取操作。
type RoleRepository struct {
	db *pgxpool.Pool
}

// NewRoleRepository 建立 RoleRepository 實例。
func NewRoleRepository(db *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{db: db}
}

const roleColumns = `id, key, name, description, tag_type, is_system, base_role, permissions, created_at, updated_at`

func scanRole(row interface{ Scan(dest ...any) error }) (app.Role, error) {
	var r app.Role
	var permBytes []byte
	if err := row.Scan(&r.ID, &r.Key, &r.Name, &r.Description, &r.TagType, &r.IsSystem, &r.BaseRole, &permBytes, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return r, err
	}
	if len(permBytes) > 0 {
		if err := json.Unmarshal(permBytes, &r.Permissions); err != nil {
			return r, fmt.Errorf("failed to unmarshal role permissions: %w", err)
		}
	}
	return r, nil
}

// List 取得所有角色。
func (r *RoleRepository) List(ctx context.Context) ([]app.Role, error) {
	if r.db == nil {
		return []app.Role{}, nil
	}
	db := pgxdb.FromContext(ctx, r.db)
	rows, err := db.Query(ctx, `SELECT `+roleColumns+` FROM roles ORDER BY is_system DESC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []app.Role
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, role)
	}
	return list, rows.Err()
}

// GetByID 依 UUID 取得角色。
func (r *RoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*app.Role, error) {
	if r.db == nil {
		return nil, nil
	}
	db := pgxdb.FromContext(ctx, r.db)
	role, err := scanRole(db.QueryRow(ctx, `SELECT `+roleColumns+` FROM roles WHERE id = $1`, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// GetByKey 依 key 取得角色。
func (r *RoleRepository) GetByKey(ctx context.Context, key string) (*app.Role, error) {
	if r.db == nil {
		return nil, nil
	}
	db := pgxdb.FromContext(ctx, r.db)
	role, err := scanRole(db.QueryRow(ctx, `SELECT `+roleColumns+` FROM roles WHERE key = $1`, key))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// Create 新增角色。
func (r *RoleRepository) Create(ctx context.Context, role *app.Role) error {
	if r.db == nil {
		return nil
	}
	permBytes, err := json.Marshal(role.Permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal role permissions: %w", err)
	}
	db := pgxdb.FromContext(ctx, r.db)
	return db.QueryRow(ctx, `
		INSERT INTO roles (id, key, name, description, tag_type, is_system, base_role, permissions)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`, role.ID, role.Key, role.Name, role.Description, role.TagType, role.IsSystem, role.BaseRole, permBytes).
		Scan(&role.CreatedAt, &role.UpdatedAt)
}

// Update 更新角色。
func (r *RoleRepository) Update(ctx context.Context, role *app.Role) error {
	if r.db == nil {
		return nil
	}
	permBytes, err := json.Marshal(role.Permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal role permissions: %w", err)
	}
	db := pgxdb.FromContext(ctx, r.db)
	return db.QueryRow(ctx, `
		UPDATE roles
		SET name = $2, description = $3, tag_type = $4, base_role = $5, permissions = $6, updated_at = now()
		WHERE id = $1
		RETURNING updated_at
	`, role.ID, role.Name, role.Description, role.TagType, role.BaseRole, permBytes).Scan(&role.UpdatedAt)
}

// Delete 刪除角色。
func (r *RoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if r.db == nil {
		return nil
	}
	db := pgxdb.FromContext(ctx, r.db)
	_, err := db.Exec(ctx, `DELETE FROM roles WHERE id = $1`, id)
	return err
}
