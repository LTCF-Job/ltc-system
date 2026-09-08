package infra

import (
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/caregiver/app"
)

// caregiverRow 是 caregivers 資料表的一列。site_name／contact／notes 皆以 SQL
// COALESCE 轉為空字串，維持 app.Caregiver 不需處理 NULL 的簡單型別。
type caregiverRow struct {
	ID        uuid.UUID
	SiteName  string
	Name      string
	Type      string
	Contact   string
	Notes     string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (r caregiverRow) toApp() app.Caregiver {
	return app.Caregiver{
		ID:        r.ID,
		SiteName:  r.SiteName,
		Name:      r.Name,
		Type:      r.Type,
		Contact:   r.Contact,
		Notes:     r.Notes,
		Status:    r.Status,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
