package infra

import (
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/caregiver/app"
)

// caregiverRow 是 caregivers 資料表（含 sites 單位名稱）的一列，只在本套件內存在。
// site_name_raw／contact／notes 皆以 SQL COALESCE 轉為空字串，維持 app.Caregiver 不需
// 處理 NULL 的簡單型別。
type caregiverRow struct {
	ID          uuid.UUID
	SiteID      *uuid.UUID
	SiteName    string
	SiteNameRaw string
	Name        string
	Type        string
	Contact     string
	Notes       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (r caregiverRow) toApp() app.Caregiver {
	return app.Caregiver{
		ID:          r.ID,
		SiteID:      r.SiteID,
		SiteName:    r.SiteName,
		SiteNameRaw: r.SiteNameRaw,
		Name:        r.Name,
		Type:        r.Type,
		Contact:     r.Contact,
		Notes:       r.Notes,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}
