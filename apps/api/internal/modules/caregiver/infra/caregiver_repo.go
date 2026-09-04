package infra

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/caregiver/app"
)

const caregiverColumns = `c.id, c.site_id, COALESCE(s.name, ''), COALESCE(c.site_name_raw, ''), c.name, c.type, COALESCE(c.contact, ''), COALESCE(c.notes, ''), c.status, c.created_at, c.updated_at`

// CaregiverRepository 提供 caregivers 資料表之存取操作。
type CaregiverRepository struct {
	db *pgxpool.Pool
}

// NewCaregiverRepository 建立 CaregiverRepository 實例。
func NewCaregiverRepository(db *pgxpool.Pool) *CaregiverRepository {
	return &CaregiverRepository{db: db}
}

// List 取得照護人員清單，支援關鍵字、狀態、單位待關聯與資料待補齊篩選。excludePending 為 true
// 時排除單位待關聯（site_name_raw 未關聯既有據點）的資料列，供主列表與「待維護」分頁互斥呈現。
func (r *CaregiverRepository) List(ctx context.Context, q, status string, unresolvedLink, incomplete, excludePending bool, page, pageSize int) ([]app.Caregiver, int64, error) {
	offset := (page - 1) * pageSize
	query := `
		SELECT ` + caregiverColumns + `
		FROM caregivers c
		LEFT JOIN sites s ON s.id = c.site_id
		WHERE ($1 = '' OR c.name ILIKE '%' || $1 || '%')
		  AND ($2 = false OR (c.site_id IS NULL AND c.site_name_raw IS NOT NULL AND c.site_name_raw <> ''))
		  AND ($3 = false OR c.contact IS NULL OR c.contact = '' OR c.notes IS NULL OR c.notes = '')
		  AND ($6 = false OR NOT (c.site_id IS NULL AND c.site_name_raw IS NOT NULL AND c.site_name_raw <> ''))
		  AND ($7 = '' OR c.status = $7)
		ORDER BY c.name ASC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.db.Query(ctx, query, q, unresolvedLink, incomplete, pageSize, offset, excludePending, status)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query caregivers: %w", err)
	}
	defer rows.Close()

	var list []app.Caregiver
	for rows.Next() {
		var row caregiverRow
		if err := rows.Scan(&row.ID, &row.SiteID, &row.SiteName, &row.SiteNameRaw, &row.Name, &row.Type, &row.Contact, &row.Notes, &row.Status, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, row.toApp())
	}

	var total int64
	countQuery := `
		SELECT COUNT(*) FROM caregivers c
		WHERE ($1 = '' OR c.name ILIKE '%' || $1 || '%')
		  AND ($2 = false OR (c.site_id IS NULL AND c.site_name_raw IS NOT NULL AND c.site_name_raw <> ''))
		  AND ($3 = false OR c.contact IS NULL OR c.contact = '' OR c.notes IS NULL OR c.notes = '')
		  AND ($4 = false OR NOT (c.site_id IS NULL AND c.site_name_raw IS NOT NULL AND c.site_name_raw <> ''))
		  AND ($5 = '' OR c.status = $5)
	`
	_ = r.db.QueryRow(ctx, countQuery, q, unresolvedLink, incomplete, excludePending, status).Scan(&total)

	return list, total, nil
}

// GetByID 依 UUID 取得照護人員。
func (r *CaregiverRepository) GetByID(ctx context.Context, id uuid.UUID) (*app.Caregiver, error) {
	var row caregiverRow
	query := `SELECT ` + caregiverColumns + ` FROM caregivers c LEFT JOIN sites s ON s.id = c.site_id WHERE c.id = $1`
	err := r.db.QueryRow(ctx, query, id).
		Scan(&row.ID, &row.SiteID, &row.SiteName, &row.SiteNameRaw, &row.Name, &row.Type, &row.Contact, &row.Notes, &row.Status, &row.CreatedAt, &row.UpdatedAt)
	if err != nil {
		return nil, err
	}
	c := row.toApp()
	return &c, nil
}

// Create 新增照護人員。
func (r *CaregiverRepository) Create(ctx context.Context, c *app.Caregiver) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	query := `
		INSERT INTO caregivers (id, site_id, site_name_raw, name, type, contact, notes, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRow(ctx, query, c.ID, c.SiteID, nullableString(c.SiteNameRaw), c.Name, c.Type, nullableString(c.Contact), nullableString(c.Notes), c.Status).
		Scan(&c.CreatedAt, &c.UpdatedAt)
}

// Update 修改照護人員。
func (r *CaregiverRepository) Update(ctx context.Context, c *app.Caregiver) error {
	query := `
		UPDATE caregivers
		SET site_id = $2, site_name_raw = $3, name = $4, type = $5, contact = $6, notes = $7, status = $8, updated_at = now()
		WHERE id = $1
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, query, c.ID, c.SiteID, nullableString(c.SiteNameRaw), c.Name, c.Type, nullableString(c.Contact), nullableString(c.Notes), c.Status).
		Scan(&c.UpdatedAt)
}

// Delete 刪除照護人員。
func (r *CaregiverRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM caregivers WHERE id = $1`, id)
	return err
}

// nullableString 將空字串轉為 nil，避免選填欄位寫入空字串取代真正的 NULL。
func nullableString(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}
