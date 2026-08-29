package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"ltc-system/apps/api/internal/modules/notification/app"
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
		return []app.Recipient{}, nil
	}

	query := `
		SELECT id, topic, email, display_name, active, created_by, created_at
		FROM notification_recipients
		WHERE ($1 = '' OR topic = $1)
		  AND (NOT $2 OR active = true)
		ORDER BY topic ASC, id ASC
	`
	rows, err := r.db.Query(ctx, query, topic, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to query recipients: %w", err)
	}
	defer rows.Close()

	var recipients []app.Recipient
	for rows.Next() {
		var item app.Recipient
		if err := rows.Scan(&item.ID, &item.Topic, &item.Email, &item.DisplayName, &item.Active, &item.CreatedBy, &item.CreatedAt); err != nil {
			return nil, err
		}
		recipients = append(recipients, item)
	}
	return recipients, nil
}

// GetRecipientByID 依 ID 取得單一收件人。
func (r *NotificationRepository) GetRecipientByID(ctx context.Context, id int64) (*app.Recipient, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database unavailable")
	}

	query := `
		SELECT id, topic, email, display_name, active, created_by, created_at
		FROM notification_recipients WHERE id = $1
	`
	var item app.Recipient
	err := r.db.QueryRow(ctx, query, id).Scan(&item.ID, &item.Topic, &item.Email, &item.DisplayName, &item.Active, &item.CreatedBy, &item.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// CreateRecipient 新增通知收件人。
func (r *NotificationRepository) CreateRecipient(ctx context.Context, item *app.Recipient) error {
	if r.db == nil {
		return nil
	}

	query := `
		INSERT INTO notification_recipients (topic, email, display_name, active, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	return r.db.QueryRow(ctx, query, item.Topic, item.Email, item.DisplayName, item.Active, item.CreatedBy).
		Scan(&item.ID, &item.CreatedAt)
}

// UpdateRecipient 修改通知收件人設定。
func (r *NotificationRepository) UpdateRecipient(ctx context.Context, id int64, email string, displayName *string, active bool) (*app.Recipient, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database unavailable")
	}

	query := `
		UPDATE notification_recipients
		SET email = $2, display_name = $3, active = $4
		WHERE id = $1
		RETURNING id, topic, email, display_name, active, created_by, created_at
	`
	var item app.Recipient
	err := r.db.QueryRow(ctx, query, id, email, displayName, active).
		Scan(&item.ID, &item.Topic, &item.Email, &item.DisplayName, &item.Active, &item.CreatedBy, &item.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// DeleteRecipient 刪除通知收件人。
func (r *NotificationRepository) DeleteRecipient(ctx context.Context, id int64) error {
	if r.db == nil {
		return nil
	}

	query := `DELETE FROM notification_recipients WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// InsertLog 寫入通知發送日誌留痕。
func (r *NotificationRepository) InsertLog(ctx context.Context, log *app.Log) error {
	if r.db == nil {
		return nil
	}

	query := `
		INSERT INTO notification_log (topic, channel, recipient_emails, subject, content_summary, status, error_message, triggered_by, triggered_by_name, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	if log.SentAt.IsZero() {
		log.SentAt = time.Now().UTC()
	}
	return r.db.QueryRow(ctx, query,
		log.Topic, log.Channel, log.RecipientEmails, log.Subject, log.ContentSummary,
		log.Status, log.ErrorMessage, log.TriggeredBy, log.TriggeredByName, log.SentAt,
	).Scan(&log.ID)
}

// ListLogs 取得通知日誌清單（支援主題篩選與分頁）。
func (r *NotificationRepository) ListLogs(ctx context.Context, topic string, page, pageSize int) ([]app.Log, int64, error) {
	if r.db == nil {
		return []app.Log{}, 0, nil
	}

	offset := (page - 1) * pageSize
	query := `
		SELECT id, topic, channel, recipient_emails, subject, content_summary, status, error_message, triggered_by, triggered_by_name, sent_at
		FROM notification_log
		WHERE ($1 = '' OR topic = $1)
		ORDER BY sent_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, topic, pageSize, offset)
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

	var total int64
	countQuery := `SELECT COUNT(*) FROM notification_log WHERE ($1 = '' OR topic = $1)`
	_ = r.db.QueryRow(ctx, countQuery, topic).Scan(&total)

	return logs, total, nil
}
