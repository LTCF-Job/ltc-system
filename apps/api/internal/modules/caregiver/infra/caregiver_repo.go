package infra

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/caregiver/app"
)

const caregiverColumns = `c.id, COALESCE(c.site_name, ''), c.name, c.type, COALESCE(c.contact, ''), COALESCE(c.notes, ''), c.status, c.created_at, c.updated_at`

// CaregiverRepository 提供 caregivers 資料表之存取操作。
type CaregiverRepository struct {
	db *pgxpool.Pool
}

// NewCaregiverRepository 建立 CaregiverRepository 實例。
func NewCaregiverRepository(db *pgxpool.Pool) *CaregiverRepository {
	return &CaregiverRepository{db: db}
}

// List 取得照護人員清單，支援關鍵字、狀態與待維護篩選。待維護的判定是 caregivers.is_pending
// 這個 generated column（姓名或類型未填寫），pending 只取待維護資料列，excludePending 反之排除，
// 供主列表與「待維護」分頁互斥呈現。
func (r *CaregiverRepository) List(ctx context.Context, q, status string, pending, excludePending bool, page, pageSize int) ([]app.Caregiver, int64, error) {
	offset := (page - 1) * pageSize
	query := `
		SELECT ` + caregiverColumns + `
		FROM caregivers c
		WHERE ($1 = '' OR c.name ILIKE '%' || $1 || '%')
		  AND ($2 = false OR c.is_pending)
		  AND ($3 = false OR NOT c.is_pending)
		  AND ($6 = '' OR c.status = $6)
		ORDER BY c.name ASC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.db.Query(ctx, query, q, pending, excludePending, pageSize, offset, status)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query caregivers: %w", err)
	}
	defer rows.Close()

	var list []app.Caregiver
	for rows.Next() {
		var row caregiverRow
		if err := rows.Scan(&row.ID, &row.SiteName, &row.Name, &row.Type, &row.Contact, &row.Notes, &row.Status, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, row.toApp())
	}

	var total int64
	countQuery := `
		SELECT COUNT(*) FROM caregivers c
		WHERE ($1 = '' OR c.name ILIKE '%' || $1 || '%')
		  AND ($2 = false OR c.is_pending)
		  AND ($3 = false OR NOT c.is_pending)
		  AND ($4 = '' OR c.status = $4)
	`
	_ = r.db.QueryRow(ctx, countQuery, q, pending, excludePending, status).Scan(&total)

	return list, total, nil
}

// GetByID 依 UUID 取得照護人員。
func (r *CaregiverRepository) GetByID(ctx context.Context, id uuid.UUID) (*app.Caregiver, error) {
	var row caregiverRow
	query := `SELECT ` + caregiverColumns + ` FROM caregivers c WHERE c.id = $1`
	err := r.db.QueryRow(ctx, query, id).
		Scan(&row.ID, &row.SiteName, &row.Name, &row.Type, &row.Contact, &row.Notes, &row.Status, &row.CreatedAt, &row.UpdatedAt)
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
		INSERT INTO caregivers (id, site_name, name, type, contact, notes, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRow(ctx, query, c.ID, nullableString(c.SiteName), c.Name, c.Type, nullableString(c.Contact), nullableString(c.Notes), c.Status).
		Scan(&c.CreatedAt, &c.UpdatedAt)
}

// Update 修改照護人員。
func (r *CaregiverRepository) Update(ctx context.Context, c *app.Caregiver) error {
	query := `
		UPDATE caregivers
		SET site_name = $2, name = $3, type = $4, contact = $5, notes = $6, status = $7, updated_at = now()
		WHERE id = $1
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, query, c.ID, nullableString(c.SiteName), c.Name, c.Type, nullableString(c.Contact), nullableString(c.Notes), c.Status).
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
