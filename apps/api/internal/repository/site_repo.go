package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SiteRepository 提供 sites 資料表之存取操作。
type SiteRepository struct {
	db *pgxpool.Pool
}

// NewSiteRepository 建立 SiteRepository 實例。
func NewSiteRepository(db *pgxpool.Pool) *SiteRepository {
	return &SiteRepository{db: db}
}

// List 取得據點清單並支援區域與關鍵字篩選。
func (r *SiteRepository) List(ctx context.Context, region, q string, page, pageSize int) ([]SiteEntity, int64, error) {
	offset := (page - 1) * pageSize
	query := `
		SELECT id, code, name, address, region, open_days, status, created_at, updated_at
		FROM sites
		WHERE ($1 = '' OR region = $1)
		  AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR address ILIKE '%' || $2 || '%')
		ORDER BY code ASC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Query(ctx, query, region, q, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query sites: %w", err)
	}
	defer rows.Close()

	var sites []SiteEntity
	for rows.Next() {
		var s SiteEntity
		if err := rows.Scan(&s.ID, &s.Code, &s.Name, &s.Address, &s.Region, &s.OpenDays, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, err
		}
		sites = append(sites, s)
	}

	var total int64
	countQuery := `
		SELECT COUNT(*) FROM sites
		WHERE ($1 = '' OR region = $1)
		  AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR address ILIKE '%' || $2 || '%')
	`
	_ = r.db.QueryRow(ctx, countQuery, region, q).Scan(&total)

	return sites, total, nil
}

// GetByID 依 UUID 取得據點。
func (r *SiteRepository) GetByID(ctx context.Context, id uuid.UUID) (*SiteEntity, error) {
	query := `
		SELECT id, code, name, address, region, open_days, status, created_at, updated_at
		FROM sites WHERE id = $1
	`
	var s SiteEntity
	err := r.db.QueryRow(ctx, query, id).Scan(&s.ID, &s.Code, &s.Name, &s.Address, &s.Region, &s.OpenDays, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetByName 依據點名稱尋找（支援匯入比對）。
func (r *SiteRepository) GetByName(ctx context.Context, name string) (*SiteEntity, error) {
	query := `
		SELECT id, code, name, address, region, open_days, status, created_at, updated_at
		FROM sites WHERE name = $1 LIMIT 1
	`
	var s SiteEntity
	err := r.db.QueryRow(ctx, query, name).Scan(&s.ID, &s.Code, &s.Name, &s.Address, &s.Region, &s.OpenDays, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Create 新增據點。
func (r *SiteRepository) Create(ctx context.Context, s *SiteEntity) error {
	query := `
		INSERT INTO sites (id, code, name, address, region, open_days, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return r.db.QueryRow(ctx, query, s.ID, s.Code, s.Name, s.Address, s.Region, s.OpenDays, s.Status).
		Scan(&s.CreatedAt, &s.UpdatedAt)
}

// Update 修改據點。
func (r *SiteRepository) Update(ctx context.Context, s *SiteEntity) error {
	query := `
		UPDATE sites
		SET name = $2, address = $3, region = $4, open_days = $5, status = $6, updated_at = now()
		WHERE id = $1
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, query, s.ID, s.Name, s.Address, s.Region, s.OpenDays, s.Status).
		Scan(&s.UpdatedAt)
}
