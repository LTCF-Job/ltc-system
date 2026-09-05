package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/masterdata/app"
)

// regionRow 是 regions 資料表的一列。
type regionRow struct {
	ID          uuid.UUID
	Name        string
	Description string
	Status      string
	SortOrder   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (r regionRow) toApp() app.Region {
	return app.Region{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Status:      r.Status,
		SortOrder:   r.SortOrder,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func (r *regionRow) scanTargets() []interface{} {
	return []interface{}{&r.ID, &r.Name, &r.Description, &r.Status, &r.SortOrder, &r.CreatedAt, &r.UpdatedAt}
}

const regionColumns = `id, name, description, status, sort_order, created_at, updated_at`

// RegionRepository 提供 regions 資料表之存取操作。
type RegionRepository struct {
	db *pgxpool.Pool
}

// NewRegionRepository 建立 RegionRepository 實例。
func NewRegionRepository(db *pgxpool.Pool) *RegionRepository {
	return &RegionRepository{db: db}
}

// List 取得區域分頁清單，支援關鍵字與狀態篩選。
func (r *RegionRepository) List(ctx context.Context, q, status string, page, pageSize int) ([]app.Region, int64, error) {
	if r.db == nil {
		return []app.Region{}, 0, nil
	}
	offset := (page - 1) * pageSize
	query := `
		SELECT ` + regionColumns + `
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

	var regions []app.Region
	for rows.Next() {
		var reg regionRow
		if err := rows.Scan(reg.scanTargets()...); err != nil {
			return nil, 0, err
		}
		regions = append(regions, reg.toApp())
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate regions: %w", err)
	}

	var total int64
	countQuery := `
		SELECT COUNT(*) FROM regions
		WHERE ($1 = '' OR name ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR status = $2)
	`
	if err := r.db.QueryRow(ctx, countQuery, q, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count regions: %w", err)
	}

	return regions, total, nil
}

// ListAll 取得所有區域清單（供下拉選單使用）。
func (r *RegionRepository) ListAll(ctx context.Context) ([]app.Region, error) {
	if r.db == nil {
		return []app.Region{}, nil
	}
	rows, err := r.db.Query(ctx, `SELECT `+regionColumns+` FROM regions WHERE status = 'active' ORDER BY sort_order ASC, name ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query all regions: %w", err)
	}
	defer rows.Close()

	var regions []app.Region
	for rows.Next() {
		var reg regionRow
		if err := rows.Scan(reg.scanTargets()...); err != nil {
			return nil, err
		}
		regions = append(regions, reg.toApp())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate all regions: %w", err)
	}
	return regions, nil
}

// GetByID 依 UUID 取得區域。
func (r *RegionRepository) GetByID(ctx context.Context, id uuid.UUID) (*app.Region, error) {
	return r.getOne(ctx, `SELECT `+regionColumns+` FROM regions WHERE id = $1`, id)
}

// GetByName 依區域名稱取得區域。
func (r *RegionRepository) GetByName(ctx context.Context, name string) (*app.Region, error) {
	return r.getOne(ctx, `SELECT `+regionColumns+` FROM regions WHERE name = $1 LIMIT 1`, name)
}

func (r *RegionRepository) getOne(ctx context.Context, query string, arg interface{}) (*app.Region, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	var reg regionRow
	if err := r.db.QueryRow(ctx, query, arg).Scan(reg.scanTargets()...); err != nil {
		return nil, err
	}
	region := reg.toApp()
	return &region, nil
}

// Create 新增區域。
func (r *RegionRepository) Create(ctx context.Context, reg *app.Region) error {
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
func (r *RegionRepository) Update(ctx context.Context, reg *app.Region) error {
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
	_, err := r.db.Exec(ctx, `DELETE FROM regions WHERE id = $1`, id)
	return err
}
