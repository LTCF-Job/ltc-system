package transport

import (
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/caregiver/app"
)

// CaregiverResponse 代表回傳給前端的照護人員資料。SiteName 是自由輸入的據點文字，
// 非必填，不關聯據點主檔。
type CaregiverResponse struct {
	ID        uuid.UUID `json:"id"`
	SiteName  string    `json:"siteName,omitempty"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Contact   string    `json:"contact,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newCaregiverResponse(c app.Caregiver) CaregiverResponse {
	return CaregiverResponse{
		ID:        c.ID,
		SiteName:  c.SiteName,
		Name:      c.Name,
		Type:      c.Type,
		Contact:   c.Contact,
		Notes:     c.Notes,
		Status:    c.Status,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
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
// specialist（照專）。
type CreateCaregiverRequest struct {
	SiteName string `json:"siteName"`
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required,oneof=case_manager specialist"`
	Contact  string `json:"contact"`
	Notes    string `json:"notes"`
	Status   string `json:"status"`
}

// UpdateCaregiverRequest 代表更新照護人員請求，欄位為 nil 表示不變更。
type UpdateCaregiverRequest struct {
	SiteName *string `json:"siteName"`
	Name     *string `json:"name"`
	Type     *string `json:"type" binding:"omitempty,oneof=case_manager specialist"`
	Contact  *string `json:"contact"`
	Notes    *string `json:"notes"`
	Status   *string `json:"status"`
}
