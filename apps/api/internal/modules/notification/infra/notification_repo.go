package infra

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/notification/app"
	"ltc-system/apps/api/internal/platform/clock"
	"ltc-system/apps/api/internal/platform/pgxdb"
)

// NotificationRepository 提供 notification_recipients 與 notification_log 資料表之存取操作。
type NotificationRepository struct {
	db *pgxpool.Pool
}

// NewNotificationRepository 建立 NotificationRepository 實例。
func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// ListRecipients 依通知主題取得收件人清單。
func (r *NotificationRepository) ListRecipients(ctx context.Context, topic string, activeOnly bool) ([]app.Recipient, error) {
	if r.db == nil {
		return nil, fmt.Errorf("notification database is not configured")
	}
	db := pgxdb.FromContext(ctx, r.db)

	query := `
		SELECT id, topic, recipient_type, target_role, user_id, COALESCE(email, ''), display_name, active, created_by, created_at
		FROM notification_recipients
		WHERE ($1 = '' OR topic = $1)
		  AND (NOT $2 OR active = true)
		ORDER BY topic ASC, id ASC
	`
	rows, err := db.Query(ctx, query, topic, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to query recipients: %w", err)
	}
	defer rows.Close()

	var recipients []app.Recipient
	for rows.Next() {
		item, err := scanRecipient(rows)
		if err != nil {
			return nil, err
		}
		recipients = append(recipients, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate recipients: %w", err)
	}
	return recipients, nil
}

// recipientScanner 是 pgx.Row 與 pgx.Rows 共同擁有的最小 Scan 介面。
type recipientScanner interface {
	Scan(dest ...interface{}) error
}

func scanRecipient(row recipientScanner) (app.Recipient, error) {
	var item app.Recipient
	err := row.Scan(&item.ID, &item.Topic, &item.RecipientType, &item.TargetRole, &item.UserID, &item.Email, &item.DisplayName, &item.Active, &item.CreatedBy, &item.CreatedAt)
	return item, err
}

// GetRecipientByID 依 ID 取得單一收件人。
func (r *NotificationRepository) GetRecipientByID(ctx context.Context, id int64) (*app.Recipient, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	db := pgxdb.FromContext(ctx, r.db)

	query := `
		SELECT id, topic, recipient_type, target_role, user_id, COALESCE(email, ''), display_name, active, created_by, created_at
		FROM notification_recipients WHERE id = $1
	`
	item, err := scanRecipient(db.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, app.ErrRecipientNotFound
		}
		return nil, err
	}
	return &item, nil
}

// CreateRecipient 新增通知收件人。
func (r *NotificationRepository) CreateRecipient(ctx context.Context, item *app.Recipient) error {
	if r.db == nil {
		return fmt.Errorf("notification database is not configured")
	}
	db := pgxdb.FromContext(ctx, r.db)

	recipientType := item.RecipientType
	if recipientType == "" {
		recipientType = "email"
	}
	var email *string
	if recipientType == "email" {
		email = &item.Email
	}

	query := `
		INSERT INTO notification_recipients (topic, recipient_type, target_role, user_id, email, display_name, active, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`
	return db.QueryRow(ctx, query, item.Topic, recipientType, item.TargetRole, item.UserID, email, item.DisplayName, item.Active, item.CreatedBy).
		Scan(&item.ID, &item.CreatedAt)
}

// UpdateRecipient 修改通知收件人設定。
func (r *NotificationRepository) UpdateRecipient(ctx context.Context, id int64, email string, displayName *string, active bool) (*app.Recipient, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	db := pgxdb.FromContext(ctx, r.db)

	query := `
		UPDATE notification_recipients
		SET email = $2, display_name = $3, active = $4
		WHERE id = $1
		RETURNING id, topic, recipient_type, target_role, user_id, COALESCE(email, ''), display_name, active, created_by, created_at
	`
	item, err := scanRecipient(db.QueryRow(ctx, query, id, email, displayName, active))
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// DeleteRecipient 刪除通知收件人。
func (r *NotificationRepository) DeleteRecipient(ctx context.Context, id int64) error {
	if r.db == nil {
		return fmt.Errorf("notification database is not configured")
	}
	db := pgxdb.FromContext(ctx, r.db)

	query := `DELETE FROM notification_recipients WHERE id = $1`
	_, err := db.Exec(ctx, query, id)
	return err
}

// InsertLog 寫入通知發送日誌留痕。
func (r *NotificationRepository) InsertLog(ctx context.Context, log *app.Log) error {
	if r.db == nil {
		return fmt.Errorf("notification database is not configured")
	}
	db := pgxdb.FromContext(ctx, r.db)

	query := `
		INSERT INTO notification_log (topic, channel, recipient_emails, subject, content_summary, status, error_message, triggered_by, triggered_by_name, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	if log.SentAt.IsZero() {
		log.SentAt = clock.Now()
	}
	return db.QueryRow(ctx, query,
		log.Topic, log.Channel, log.RecipientEmails, log.Subject, log.ContentSummary,
		log.Status, log.ErrorMessage, log.TriggeredBy, log.TriggeredByName, log.SentAt,
	).Scan(&log.ID)
}

func (r *NotificationRepository) ClaimNotificationEvent(ctx context.Context, dedupKey, topic string) (bool, error) {
	if r.db == nil {
		return false, fmt.Errorf("notification database is not configured")
	}
	db := pgxdb.FromContext(ctx, r.db)
	var claimed bool
	err := db.QueryRow(ctx, `
		INSERT INTO notification_event_dedup (dedup_key, topic, status, claimed_at)
		VALUES ($1, $2, 'processing', now())
		ON CONFLICT (dedup_key) DO NOTHING
		RETURNING true
	`, dedupKey, topic).Scan(&claimed)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to claim notification event: %w", err)
	}
	return claimed, nil
}

func (r *NotificationRepository) CompleteNotificationEvent(ctx context.Context, dedupKey string) error {
	if r.db == nil {
		return fmt.Errorf("notification database is not configured")
	}
	_, err := pgxdb.FromContext(ctx, r.db).Exec(ctx, `
		UPDATE notification_event_dedup SET status = 'sent', sent_at = now()
		WHERE dedup_key = $1
	`, dedupKey)
	return err
}

func (r *NotificationRepository) ReleaseNotificationEvent(ctx context.Context, dedupKey string) error {
	if r.db == nil {
		return fmt.Errorf("notification database is not configured")
	}
	_, err := pgxdb.FromContext(ctx, r.db).Exec(ctx, `DELETE FROM notification_event_dedup WHERE dedup_key = $1`, dedupKey)
	return err
}

// ListLogs 取得通知日誌清單（支援主題篩選與分頁）。
func (r *NotificationRepository) ListLogs(ctx context.Context, topic string, page, pageSize int) ([]app.Log, int64, error) {
	if r.db == nil {
		return nil, 0, fmt.Errorf("notification database is not configured")
	}
	db := pgxdb.FromContext(ctx, r.db)

	offset := (page - 1) * pageSize
	query := `
		SELECT id, topic, channel, recipient_emails, subject, content_summary, status, error_message, triggered_by, triggered_by_name, sent_at
		FROM notification_log
		WHERE ($1 = '' OR topic = $1)
		ORDER BY sent_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := db.Query(ctx, query, topic, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query notification logs: %w", err)
	}
	defer rows.Close()

	var logs []app.Log
	for rows.Next() {
		var l app.Log
		if err := rows.Scan(
			&l.ID, &l.Topic, &l.Channel, &l.RecipientEmails, &l.Subject, &l.ContentSummary,
			&l.Status, &l.ErrorMessage, &l.TriggeredBy, &l.TriggeredByName, &l.SentAt,
		); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate notification logs: %w", err)
	}

	var total int64
	countQuery := `SELECT COUNT(*) FROM notification_log WHERE ($1 = '' OR topic = $1)`
	if err := db.QueryRow(ctx, countQuery, topic).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count notification logs: %w", err)
	}

	return logs, total, nil
}

// BatchCreateRecipients 批次新增收件人；topic+email 重複者靜默略過，回傳只含實際新增的列。
func (r *NotificationRepository) BatchCreateRecipients(ctx context.Context, items []app.Recipient) ([]app.Recipient, error) {
	if r.db == nil {
		return nil, fmt.Errorf("notification database is not configured")
	}
	if len(items) == 0 {
		return []app.Recipient{}, nil
	}
	db := pgxdb.FromContext(ctx, r.db)

	topics := make([]string, len(items))
	emails := make([]string, len(items))
	displayNames := make([]*string, len(items))
	createdBys := make([]uuid.UUID, len(items))
	for i, item := range items {
		topics[i] = item.Topic
		emails[i] = item.Email
		displayNames[i] = item.DisplayName
		createdBys[i] = item.CreatedBy
	}

	query := `
		INSERT INTO notification_recipients (topic, recipient_type, email, display_name, active, created_by)
		SELECT t, 'email', e, d, true, c
		FROM unnest($1::text[], $2::text[], $3::text[], $4::uuid[]) AS u(t, e, d, c)
		ON CONFLICT (topic, email) WHERE recipient_type = 'email' DO NOTHING
		RETURNING id, topic, recipient_type, target_role, user_id, COALESCE(email, ''), display_name, active, created_by, created_at
	`
	rows, err := db.Query(ctx, query, topics, emails, displayNames, createdBys)
	if err != nil {
		return nil, fmt.Errorf("failed to batch create recipients: %w", err)
	}
	defer rows.Close()

	var created []app.Recipient
	for rows.Next() {
		item, err := scanRecipient(rows)
		if err != nil {
			return nil, err
		}
		created = append(created, item)
	}
	return created, rows.Err()
}

// BatchDeleteRecipients 批次刪除收件人，回傳實際刪除筆數。
func (r *NotificationRepository) BatchDeleteRecipients(ctx context.Context, ids []int64) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("notification database is not configured")
	}
	if len(ids) == 0 {
		return 0, nil
	}
	db := pgxdb.FromContext(ctx, r.db)

	tag, err := db.Exec(ctx, `DELETE FROM notification_recipients WHERE id = ANY($1::bigint[])`, ids)
	if err != nil {
		return 0, fmt.Errorf("failed to batch delete recipients: %w", err)
	}
	return tag.RowsAffected(), nil
}
