package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegionRepository 提供 regions 資料表之存取操作。
type RegionRepository struct {
	db *pgxpool.Pool
}

// NewRegionRepository 建立 RegionRepository 實例。
func NewRegionRepository(db *pgxpool.Pool) *RegionRepository {
	return &RegionRepository{db: db}
}

// List 取得區域分頁清單，支援關鍵字與狀態篩選。
func (r *RegionRepository) List(ctx context.Context, q, status string, page, pageSize int) ([]RegionEntity, int64, error) {
	if r.db == nil {
		return []RegionEntity{}, 0, nil
	}
	offset := (page - 1) * pageSize
	query := `
		SELECT id, name, description, status, sort_order, created_at, updated_at
		FROM regions
		WHERE ($1 = '' OR name ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR status = $2)
		ORDER BY sort_order ASC, name ASC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Query(ctx, query, q, status, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query regions: %w", err)
	}
	defer rows.Close()

	var regions []RegionEntity
	for rows.Next() {
		var reg RegionEntity
		if err := rows.Scan(&reg.ID, &reg.Name, &reg.Description, &reg.Status, &reg.SortOrder, &reg.CreatedAt, &reg.UpdatedAt); err != nil {
			return nil, 0, err
		}
		regions = append(regions, reg)
	}

	var total int64
	countQuery := `
		SELECT COUNT(*) FROM regions
		WHERE ($1 = '' OR name ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR status = $2)
	`
	_ = r.db.QueryRow(ctx, countQuery, q, status).Scan(&total)

	return regions, total, nil
}

// ListAll 取得所有區域清單（供下拉選單使用）。
func (r *RegionRepository) ListAll(ctx context.Context) ([]RegionEntity, error) {
	if r.db == nil {
		return []RegionEntity{}, nil
	}
	query := `
		SELECT id, name, description, status, sort_order, created_at, updated_at
		FROM regions
		ORDER BY sort_order ASC, name ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all regions: %w", err)
	}
	defer rows.Close()

	var regions []RegionEntity
	for rows.Next() {
		var reg RegionEntity
		if err := rows.Scan(&reg.ID, &reg.Name, &reg.Description, &reg.Status, &reg.SortOrder, &reg.CreatedAt, &reg.UpdatedAt); err != nil {
			return nil, err
		}
		regions = append(regions, reg)
	}
	return regions, nil
}

// GetByID 依 UUID 取得區域。
func (r *RegionRepository) GetByID(ctx context.Context, id uuid.UUID) (*RegionEntity, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	query := `
		SELECT id, name, description, status, sort_order, created_at, updated_at
		FROM regions WHERE id = $1
	`
	var reg RegionEntity
	err := r.db.QueryRow(ctx, query, id).Scan(&reg.ID, &reg.Name, &reg.Description, &reg.Status, &reg.SortOrder, &reg.CreatedAt, &reg.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &reg, nil
}

// GetByName 依區域名稱取得區域。
func (r *RegionRepository) GetByName(ctx context.Context, name string) (*RegionEntity, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	query := `
		SELECT id, name, description, status, sort_order, created_at, updated_at
		FROM regions WHERE name = $1 LIMIT 1
	`
	var reg RegionEntity
	err := r.db.QueryRow(ctx, query, name).Scan(&reg.ID, &reg.Name, &reg.Description, &reg.Status, &reg.SortOrder, &reg.CreatedAt, &reg.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &reg, nil
}

// Create 新增區域。
func (r *RegionRepository) Create(ctx context.Context, reg *RegionEntity) error {
	if r.db == nil {
		return fmt.Errorf("database not connected")
	}
	query := `
		INSERT INTO regions (id, name, description, status, sort_order)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`
	if reg.ID == uuid.Nil {
		reg.ID = uuid.New()
	}
	return r.db.QueryRow(ctx, query, reg.ID, reg.Name, reg.Description, reg.Status, reg.SortOrder).
		Scan(&reg.CreatedAt, &reg.UpdatedAt)
}

// Update 修改區域。
func (r *RegionRepository) Update(ctx context.Context, reg *RegionEntity) error {
	if r.db == nil {
		return fmt.Errorf("database not connected")
	}
	query := `
		UPDATE regions
		SET name = $2, description = $3, status = $4, sort_order = $5, updated_at = now()
		WHERE id = $1
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, query, reg.ID, reg.Name, reg.Description, reg.Status, reg.SortOrder).
		Scan(&reg.UpdatedAt)
}

// Delete 刪除區域。
func (r *RegionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if r.db == nil {
		return fmt.Errorf("database not connected")
	}
	query := `DELETE FROM regions WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
