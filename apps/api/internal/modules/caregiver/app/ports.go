package app

import (
	"context"

	"github.com/google/uuid"
)

// CaregiverStore 定義照護人員主檔的讀寫邊界。
type CaregiverStore interface {
	List(ctx context.Context, q string, unresolvedLink, incomplete bool, page, pageSize int) ([]Caregiver, int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Caregiver, error)
	Create(ctx context.Context, c *Caregiver) error
	Update(ctx context.Context, c *Caregiver) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SiteRef 是匯入或關聯比對據點時需要的最小資訊。
type SiteRef struct {
	ID   uuid.UUID
	Name string
}

// SiteLookup 提供以名稱比對據點的查詢，供批次匯入比對「單位」欄位使用。
type SiteLookup interface {
	GetByName(ctx context.Context, name string) (*SiteRef, error)
}

// SpreadsheetReader 將上傳的 Excel 位元組解碼為逐工作表的儲存格文字。
type SpreadsheetReader interface {
	ReadTables(data []byte) (tables [][][]string, sheetNames []string, err error)
}

// TemplateRenderer 產生批次匯入的 Excel 範本位元組。
type TemplateRenderer interface {
	RenderCaregiverImportTemplate() ([]byte, error)
}
