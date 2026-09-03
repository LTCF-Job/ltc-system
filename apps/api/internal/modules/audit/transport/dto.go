package transport

import (
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/audit/app"
)

// RecordResponse 是稽核紀錄的 API 形狀，json tag 與搬遷前逐欄一致。
type RecordResponse struct {
	ID         int64       `json:"id"`
	ActorID    *uuid.UUID  `json:"actorId,omitempty"`
	ActorRole  *string     `json:"actorRole,omitempty"`
	Action     string      `json:"action"`
	EntityType string      `json:"entityType"`
	EntityID   *string     `json:"entityId,omitempty"`
	BeforeData interface{} `json:"beforeData,omitempty"`
	AfterData  interface{} `json:"afterData,omitempty"`
	IPAddress  *string     `json:"ipAddress,omitempty"`
	UserAgent  *string     `json:"userAgent,omitempty"`
	CreatedAt  time.Time   `json:"createdAt"`
}

func newRecordResponses(list []app.Record) []RecordResponse {
	if list == nil {
		return nil
	}
	out := make([]RecordResponse, 0, len(list))
	for _, r := range list {
		out = append(out, RecordResponse{
			ID: r.ID, ActorID: r.ActorID, ActorRole: r.ActorRole, Action: r.Action,
			EntityType: r.EntityType, EntityID: r.EntityID, BeforeData: r.BeforeData,
			AfterData: r.AfterData, IPAddress: r.IPAddress, UserAgent: r.UserAgent, CreatedAt: r.CreatedAt,
		})
	}
	return out
}
