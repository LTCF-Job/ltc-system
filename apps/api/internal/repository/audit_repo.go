package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// AuditFilter 定義稽核日誌查詢過濾條件。
type AuditFilter struct {
	ActorID    *uuid.UUID
	Action     string
	EntityType string
	EntityID   string
	StartDate  *time.Time
	EndDate    *time.Time
	Q          string
	Page       int
	PageSize   int
}

// AuditRepository 提供 audit_log 資料表之存取操作。
type AuditRepository struct {
	db *pgxpool.Pool
}

// NewAuditRepository 建立 AuditRepository 實例。
func NewAuditRepository(db *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{db: db}
}

// Insert 寫入一筆不可變之稽核日誌。
func (r *AuditRepository) Insert(ctx context.Context, item *AuditLogEntity) error {
	if r.db == nil {
		return nil
	}

	query := `
		INSERT INTO audit_log (actor_id, actor_role, action, entity_type, entity_id, before_data, after_data, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`
	db := pgxdb.FromContext(ctx, r.db)
	return db.QueryRow(ctx, query, item.ActorID, item.ActorRole, item.Action, item.EntityType, item.EntityID, item.BeforeData, item.AfterData, item.IPAddress, item.UserAgent).
		Scan(&item.ID, &item.CreatedAt)
}

// List 依據多條件篩選並分頁查詢稽核紀錄。
func (r *AuditRepository) List(ctx context.Context, f AuditFilter) ([]AuditLogEntity, int64, error) {
	if r.db == nil {
		return []AuditLogEntity{}, 0, nil
	}

	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
	offset := (f.Page - 1) * f.PageSize

	query := `
		SELECT id, actor_id, actor_role, action, entity_type, entity_id, before_data, after_data, ip_address, user_agent, created_at
		FROM audit_log
		WHERE ($1::uuid IS NULL OR actor_id = $1)
		  AND ($2 = '' OR action = $2)
		  AND ($3 = '' OR entity_type = $3)
		  AND ($4 = '' OR entity_id = $4)
		  AND ($5::timestamptz IS NULL OR created_at >= $5)
		  AND ($6::timestamptz IS NULL OR created_at <= $6)
		  AND ($7 = '' OR action ILIKE '%' || $7 || '%' OR entity_type ILIKE '%' || $7 || '%' OR entity_id ILIKE '%' || $7 || '%')
		ORDER BY created_at DESC, id DESC
		LIMIT $8 OFFSET $9
	`

	rows, err := r.db.Query(ctx, query, f.ActorID, f.Action, f.EntityType, f.EntityID, f.StartDate, f.EndDate, f.Q, f.PageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []AuditLogEntity
	for rows.Next() {
		var l AuditLogEntity
		if err := rows.Scan(&l.ID, &l.ActorID, &l.ActorRole, &l.Action, &l.EntityType, &l.EntityID, &l.BeforeData, &l.AfterData, &l.IPAddress, &l.UserAgent, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}

	var total int64
	countQuery := `
		SELECT COUNT(*) FROM audit_log
		WHERE ($1::uuid IS NULL OR actor_id = $1)
		  AND ($2 = '' OR action = $2)
		  AND ($3 = '' OR entity_type = $3)
		  AND ($4 = '' OR entity_id = $4)
		  AND ($5::timestamptz IS NULL OR created_at >= $5)
		  AND ($6::timestamptz IS NULL OR created_at <= $6)
		  AND ($7 = '' OR action ILIKE '%' || $7 || '%' OR entity_type ILIKE '%' || $7 || '%' OR entity_id ILIKE '%' || $7 || '%')
	`
	_ = r.db.QueryRow(ctx, countQuery, f.ActorID, f.Action, f.EntityType, f.EntityID, f.StartDate, f.EndDate, f.Q).Scan(&total)

	return logs, total, nil
}
