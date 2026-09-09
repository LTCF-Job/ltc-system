package infra

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/masterdata/app"
)

// siteRow 是 sites 資料表的一列，只在本套件內存在。
type siteRow struct {
	ID        uuid.UUID
	Name      string
	Address   string
	Region    string
	Status    string
	Remarks   *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (r siteRow) toApp() app.Site {
	return app.Site{
		ID:        r.ID,
		Name:      r.Name,
		Address:   r.Address,
		Region:    r.Region,
		Status:    r.Status,
		Remarks:   derefString(r.Remarks),
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

const siteColumns = `id, name, COALESCE(address, ''), COALESCE(region, ''), status, remarks, created_at, updated_at`

// SiteRepository 提供 sites 資料表之存取操作。
type SiteRepository struct {
	db *pgxpool.Pool
}

// NewSiteRepository 建立 SiteRepository 實例。
func NewSiteRepository(db *pgxpool.Pool) *SiteRepository {
	return &SiteRepository{db: db}
}

// List 取得據點清單並支援區域、狀態與關鍵字篩選。
func (r *SiteRepository) List(ctx context.Context, region, q, status string, page, pageSize int) ([]app.Site, int64, error) {
	offset := (page - 1) * pageSize
	query := `
		SELECT ` + siteColumns + `
		FROM sites
		WHERE ($1 = '' OR COALESCE(region, '') ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR COALESCE(address, '') ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR status = $3)
		ORDER BY name ASC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.db.Query(ctx, query, region, q, status, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query sites: %w", err)
	}
	defer rows.Close()

	var sites []app.Site
	for rows.Next() {
		var s siteRow
		if err := rows.Scan(&s.ID, &s.Name, &s.Address, &s.Region, &s.Status, &s.Remarks, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, err
		}
		sites = append(sites, s.toApp())
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate sites: %w", err)
	}

	var total int64
	countQuery := `
		SELECT COUNT(*) FROM sites
		WHERE ($1 = '' OR COALESCE(region, '') ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR COALESCE(address, '') ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR status = $3)
	`
	if err := r.db.QueryRow(ctx, countQuery, region, q, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count sites: %w", err)
	}

	return sites, total, nil
}

// GetByID 依 UUID 取得據點。
func (r *SiteRepository) GetByID(ctx context.Context, id uuid.UUID) (*app.Site, error) {
	return r.getOne(ctx, `SELECT `+siteColumns+` FROM sites WHERE id = $1`, id)
}

// GetByName 依據點名稱尋找（支援匯入比對）。
func (r *SiteRepository) GetByName(ctx context.Context, name string) (*app.Site, error) {
	return r.getOne(ctx, `SELECT `+siteColumns+` FROM sites WHERE name = $1 LIMIT 1`, name)
}

func (r *SiteRepository) getOne(ctx context.Context, query string, arg interface{}) (*app.Site, error) {
	var s siteRow
	err := r.db.QueryRow(ctx, query, arg).
		Scan(&s.ID, &s.Name, &s.Address, &s.Region, &s.Status, &s.Remarks, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, app.ErrSiteNotFound
		}
		return nil, err
	}
	site := s.toApp()
	return &site, nil
}

// Create 新增據點。
func (r *SiteRepository) Create(ctx context.Context, s *app.Site) error {
	query := `
		INSERT INTO sites (id, name, address, region, status, remarks)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6)
		RETURNING created_at, updated_at
	`
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	err := r.db.QueryRow(ctx, query, s.ID, s.Name, s.Address, s.Region, s.Status, nullableText(s.Remarks)).
		Scan(&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return app.ErrDuplicateSiteName
		}
		return err
	}
	return nil
}

// Update 修改據點。
func (r *SiteRepository) Update(ctx context.Context, s *app.Site) error {
	query := `
		UPDATE sites
		SET name = $2, address = $3, region = NULLIF($4, ''), status = $5, remarks = $6, updated_at = now()
		WHERE id = $1
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query, s.ID, s.Name, s.Address, s.Region, s.Status, nullableText(s.Remarks)).
		Scan(&s.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return app.ErrDuplicateSiteName
		}
		return err
	}
	return nil
}

// Delete 刪除據點。若該據點仍被個案排班參照，資料庫外鍵限制會回傳錯誤。
func (r *SiteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if r.db == nil {
		return fmt.Errorf("database not connected")
	}
	_, err := r.db.Exec(ctx, `DELETE FROM sites WHERE id = $1`, id)
	return err
}
