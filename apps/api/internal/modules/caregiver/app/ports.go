package app

import (
	"context"

	"github.com/google/uuid"
)

// ActorContext 代表照護人員主檔異動的操作者與來源資訊。
type ActorContext struct {
	ActorID   uuid.UUID
	ActorRole string
	IPAddress string
	UserAgent string
}

// AuditEntry 是照護人員模組交給共用 audit service 的資料。
type AuditEntry struct {
	ActorID    *uuid.UUID
	ActorRole  *string
	Action     string
	EntityType string
	EntityID   *string
	BeforeData interface{}
	AfterData  interface{}
	IPAddress  *string
	UserAgent  *string
}

// AuditWriter 定義照護人員 mutation 的稽核寫入邊界。
type AuditWriter interface {
	Write(ctx context.Context, e AuditEntry) error
}

// CaregiverStore 定義照護人員主檔的讀寫邊界。
type CaregiverStore interface {
	List(ctx context.Context, q, status string, pending, excludePending bool, page, pageSize int) ([]Caregiver, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Caregiver, error)
	Create(ctx context.Context, c *Caregiver) error
	Update(ctx context.Context, c *Caregiver) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SpreadsheetReader 將上傳的 Excel 位元組解碼為逐工作表的儲存格文字。
type SpreadsheetReader interface {
	ReadTables(data []byte) (tables [][][]string, sheetNames []string, err error)
}

// TemplateRenderer 產生批次匯入的 Excel 範本位元組。
type TemplateRenderer interface {
	RenderCaregiverImportTemplate() ([]byte, error)
}
