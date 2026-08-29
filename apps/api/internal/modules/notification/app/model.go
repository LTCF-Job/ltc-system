package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Recipient 代表一位通知收件人設定。
type Recipient struct {
	ID          int64
	Topic       string
	Email       string
	DisplayName *string
	Active      bool
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
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
}
