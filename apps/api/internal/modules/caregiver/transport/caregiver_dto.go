package transport

import (
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/caregiver/app"
)

// CaregiverResponse 代表回傳給前端的照護人員資料。SiteID 為 nil 且 SiteNameRaw 有值
// 時，代表匯入時的單位名稱尚未關聯既有單位。
type CaregiverResponse struct {
	ID          uuid.UUID  `json:"id"`
	SiteID      *uuid.UUID `json:"siteId,omitempty"`
	SiteName    string     `json:"siteName,omitempty"`
	SiteNameRaw string     `json:"siteNameRaw,omitempty"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Contact     string     `json:"contact,omitempty"`
	Notes       string     `json:"notes,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

func newCaregiverResponse(c app.Caregiver) CaregiverResponse {
	return CaregiverResponse{
		ID:          c.ID,
		SiteID:      c.SiteID,
		SiteName:    c.SiteName,
		SiteNameRaw: c.SiteNameRaw,
		Name:        c.Name,
		Type:        c.Type,
		Contact:     c.Contact,
		Notes:       c.Notes,
		Status:      c.Status,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func newCaregiverResponses(list []app.Caregiver) []CaregiverResponse {
	if list == nil {
		return nil
	}
	out := make([]CaregiverResponse, 0, len(list))
	for _, c := range list {
		out = append(out, newCaregiverResponse(c))
	}
	return out
}

// CreateCaregiverRequest 代表新增照護人員請求。Type 僅接受 case_manager（個管）或
// specialist（專護）。
type CreateCaregiverRequest struct {
	SiteID  *uuid.UUID `json:"siteId"`
	Name    string     `json:"name" binding:"required"`
	Type    string     `json:"type" binding:"required,oneof=case_manager specialist"`
	Contact string     `json:"contact"`
	Notes   string     `json:"notes"`
	Status  string     `json:"status"`
}

// UpdateCaregiverRequest 代表更新照護人員請求，欄位為 nil 表示不變更。
type UpdateCaregiverRequest struct {
	SiteID  *uuid.UUID `json:"siteId"`
	Name    *string    `json:"name"`
	Type    *string    `json:"type" binding:"omitempty,oneof=case_manager specialist"`
	Contact *string    `json:"contact"`
	Notes   *string    `json:"notes"`
	Status  *string    `json:"status"`
}

// LinkCaregiverSiteRequest 代表將照護人員關聯至既有單位的請求。
type LinkCaregiverSiteRequest struct {
	SiteID uuid.UUID `json:"siteId" binding:"required"`
}
