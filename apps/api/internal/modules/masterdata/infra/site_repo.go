package infra

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	OpenDays  []int16
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (r siteRow) toApp() app.Site {
	return app.Site{
		ID:        r.ID,
		Name:      r.Name,
		Address:   r.Address,
		Region:    r.Region,
		OpenDays:  r.OpenDays,
		Status:    r.Status,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

const siteColumns = `id, name, address, region, open_days, status, created_at, updated_at`

// SiteRepository 提供 sites 資料表之存取操作。
type SiteRepository struct {
	db *pgxpool.Pool
}

// NewSiteRepository 建立 SiteRepository 實例。
func NewSiteRepository(db *pgxpool.Pool) *SiteRepository {
	return &SiteRepository{db: db}
}

// List 取得單位清單並支援區域與關鍵字篩選。
func (r *SiteRepository) List(ctx context.Context, region, q string, page, pageSize int) ([]app.Site, int64, error) {
	offset := (page - 1) * pageSize
	query := `
		SELECT ` + siteColumns + `
		FROM sites
		WHERE ($1 = '' OR region = $1)
		  AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR address ILIKE '%' || $2 || '%')
		ORDER BY name ASC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Query(ctx, query, region, q, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query sites: %w", err)
	}
	defer rows.Close()

	var sites []app.Site
	for rows.Next() {
		var s siteRow
		if err := rows.Scan(&s.ID, &s.Name, &s.Address, &s.Region, &s.OpenDays, &s.Status, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, err
		}
		sites = append(sites, s.toApp())
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

// GetByID 依 UUID 取得單位。
func (r *SiteRepository) GetByID(ctx context.Context, id uuid.UUID) (*app.Site, error) {
	return r.getOne(ctx, `SELECT `+siteColumns+` FROM sites WHERE id = $1`, id)
}

// GetByName 依單位名稱尋找（支援匯入比對）。
func (r *SiteRepository) GetByName(ctx context.Context, name string) (*app.Site, error) {
	return r.getOne(ctx, `SELECT `+siteColumns+` FROM sites WHERE name = $1 LIMIT 1`, name)
}

func (r *SiteRepository) getOne(ctx context.Context, query string, arg interface{}) (*app.Site, error) {
	var s siteRow
	err := r.db.QueryRow(ctx, query, arg).
		Scan(&s.ID, &s.Name, &s.Address, &s.Region, &s.OpenDays, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	site := s.toApp()
	return &site, nil
}

// Create 新增單位。
func (r *SiteRepository) Create(ctx context.Context, s *app.Site) error {
	query := `
		INSERT INTO sites (id, name, address, region, open_days, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at
	`
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	err := r.db.QueryRow(ctx, query, s.ID, s.Name, s.Address, s.Region, s.OpenDays, s.Status).
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

// Update 修改單位。
func (r *SiteRepository) Update(ctx context.Context, s *app.Site) error {
	query := `
		UPDATE sites
		SET name = $2, address = $3, region = $4, open_days = $5, status = $6, updated_at = now()
		WHERE id = $1
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query, s.ID, s.Name, s.Address, s.Region, s.OpenDays, s.Status).
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

// Delete 刪除單位。若該單位仍被個案排班參照，資料庫外鍵限制會回傳錯誤。
func (r *SiteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if r.db == nil {
		return fmt.Errorf("database not connected")
	}
	_, err := r.db.Exec(ctx, `DELETE FROM sites WHERE id = $1`, id)
	return err
}
