package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Recipient 代表一位通知收件人設定。
type Recipient struct {
	ID            int64
	Topic         string
	RecipientType string
	Email         string
	TargetRole    *string
	UserID        *uuid.UUID
	DisplayName   *string
	Active        bool
	CreatedBy     uuid.UUID
	CreatedAt     time.Time
}

// RecipientAuditSnapshot 是通知收件人設定的明確稽核快照，不直接序列化 Recipient
// 或保存未遮罩的電子郵件。
type RecipientAuditSnapshot struct {
	ID            int64      `json:"id"`
	Topic         string     `json:"topic"`
	RecipientType string     `json:"recipientType"`
	Email         string     `json:"email"`
	TargetRole    *string    `json:"targetRole,omitempty"`
	UserID        *uuid.UUID `json:"userId,omitempty"`
	Active        bool       `json:"active"`
}

// RecipientBatchAuditSnapshot 是批次收件人異動的明確稽核 DTO。
type RecipientBatchAuditSnapshot struct {
	Recipients []RecipientAuditSnapshot `json:"recipients"`
}

// RecipientBatchAuditSummary 是批次刪除完成後的結果摘要。
type RecipientBatchAuditSummary struct {
	Count int64 `json:"count"`
}

// AuditSnapshot 產生通知收件人設定的明確稽核快照。
func (r Recipient) AuditSnapshot() RecipientAuditSnapshot {
	return RecipientAuditSnapshot{
		ID:            r.ID,
		Topic:         r.Topic,
		RecipientType: r.RecipientType,
		Email:         maskEmail(r.Email),
		TargetRole:    r.TargetRole,
		UserID:        r.UserID,
		Active:        r.Active,
	}
}

// Log 代表一筆通知發送留痕。
type Log struct {
	ID              int64
	Topic           string
	Channel         string
	RecipientEmails []string
	Subject         string
	ContentSummary  *string
	Status          string
	ErrorMessage    *string
	TriggeredBy     *uuid.UUID
	TriggeredByName *string
	SentAt          time.Time
}

// AuditEntry 是本模組寫入稽核日誌的內容。
type AuditEntry struct {
	ActorID    *uuid.UUID
	ActorRole  *string
	Action     string
	EntityType string
	EntityID   *string
	BeforeData interface{}
	AfterData  interface{}
}

// AuditWriter 定義收件人異動留痕的寫入邊界。
type AuditWriter interface {
	Write(ctx context.Context, e AuditEntry) error
}

// Store 定義通知收件人與寄送紀錄的讀寫邊界。
type Store interface {
	ListRecipients(ctx context.Context, topic string, activeOnly bool) ([]Recipient, error)
	GetRecipientByID(ctx context.Context, id int64) (*Recipient, error)
	CreateRecipient(ctx context.Context, item *Recipient) error
	UpdateRecipient(ctx context.Context, id int64, email string, displayName *string, active bool) (*Recipient, error)
	DeleteRecipient(ctx context.Context, id int64) error
	InsertLog(ctx context.Context, log *Log) error
	ListLogs(ctx context.Context, topic string, page, pageSize int) ([]Log, int64, error)
	BatchCreateRecipients(ctx context.Context, items []Recipient) ([]Recipient, error)
	BatchDeleteRecipients(ctx context.Context, ids []int64) (int64, error)
}
