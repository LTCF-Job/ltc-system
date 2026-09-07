package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/audit/app"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// AuditRepository 提供 audit_log 資料表之存取操作。
type AuditRepository struct {
	db *pgxpool.Pool
}

var errAuditDatabaseNotConfigured = errors.New("audit database is not configured")

// NewAuditRepository 建立 AuditRepository 實例。
func NewAuditRepository(db *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{db: db}
}

// Insert 寫入一筆不可變之稽核日誌。
func (r *AuditRepository) Insert(ctx context.Context, e app.Entry) error {
	if r.db == nil {
		return errAuditDatabaseNotConfigured
	}

	query := `
		INSERT INTO audit_log (actor_id, actor_role, action, entity_type, entity_id, before_data, after_data, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`
	beforeData, err := marshalAuditPayload(e.BeforeData)
	if err != nil {
		return fmt.Errorf("failed to marshal audit before_data: %w", err)
	}
	afterData, err := marshalAuditPayload(e.AfterData)
	if err != nil {
		return fmt.Errorf("failed to marshal audit after_data: %w", err)
	}

	var id int64
	var createdAt time.Time
	db := pgxdb.FromContext(ctx, r.db)
	return db.QueryRow(ctx, query, e.ActorID, e.ActorRole, e.Action, e.EntityType, e.EntityID, beforeData, afterData, e.IPAddress, e.UserAgent).
		Scan(&id, &createdAt)
}

// marshalAuditPayload 把快照序列化成 JSON 字串再交給 pgx。before_data／after_data 是 jsonb，
// 而連線走 simple protocol（見 cmd/server/main.go），pgx 對任意 struct 或 map 找不到 encode plan，
// 直接傳值會讓每一筆帶快照的稽核都寫入失敗。回傳 *string 讓 nil 快照仍寫入 SQL NULL。
func marshalAuditPayload(v interface{}) (*string, error) {
	if v == nil {
		return nil, nil
	}
	// 已經是 JSON 字串或位元組的快照原樣沿用，不再包一層。
	switch payload := v.(type) {
	case string:
		return &payload, nil
	case []byte:
		s := string(payload)
		return &s, nil
	}
	encoded, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	s := string(encoded)
	return &s, nil
}

// List 依據多條件篩選並分頁查詢稽核紀錄。
func (r *AuditRepository) List(ctx context.Context, f app.Filter) ([]app.Record, int64, error) {
	if r.db == nil {
		return nil, 0, errAuditDatabaseNotConfigured
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

	var logs []app.Record
	for rows.Next() {
		var l app.Record
		if err := rows.Scan(&l.ID, &l.ActorID, &l.ActorRole, &l.Action, &l.EntityType, &l.EntityID, &l.BeforeData, &l.AfterData, &l.IPAddress, &l.UserAgent, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate audit logs: %w", err)
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
	if err := r.db.QueryRow(ctx, countQuery, f.ActorID, f.Action, f.EntityType, f.EntityID, f.StartDate, f.EndDate, f.Q).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	return logs, total, nil
}
