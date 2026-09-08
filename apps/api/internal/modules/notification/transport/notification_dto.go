package transport

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/notification/app"
)

// RecipientResponse 是通知收件人對外的 API 契約，欄位與
// apps/web/src/types/api.d.ts 的 NotificationRecipientDTO 對齊。
type RecipientResponse struct {
	// 資料庫是 bigint 自增，前端契約統一以字串承載識別碼。
	ID            string     `json:"id"`
	Topic         string     `json:"topic"`
	RecipientType string     `json:"recipientType"`
	Email         string     `json:"email"`
	TargetRole    *string    `json:"targetRole"`
	UserID        *uuid.UUID `json:"userId"`
	DisplayName   *string    `json:"displayName"`
	Active        bool       `json:"active"`
	CreatedBy     uuid.UUID  `json:"createdBy"`
	CreatedAt     time.Time  `json:"createdAt"`
}

func newRecipientResponse(r app.Recipient) RecipientResponse {
	return RecipientResponse{
		ID:            strconv.FormatInt(r.ID, 10),
		Topic:         r.Topic,
		RecipientType: r.RecipientType,
		Email:         r.Email,
		TargetRole:    r.TargetRole,
		UserID:        r.UserID,
		DisplayName:   r.DisplayName,
		Active:        r.Active,
		CreatedBy:     r.CreatedBy,
		CreatedAt:     r.CreatedAt,
	}
}

func newRecipientResponses(list []app.Recipient) []RecipientResponse {
	out := make([]RecipientResponse, 0, len(list))
	for _, r := range list {
		out = append(out, newRecipientResponse(r))
	}
	return out
}

// LogResponse 是通知發送紀錄對外的 API 契約，欄位與
// apps/web/src/types/api.d.ts 的 NotificationLogDTO 對齊。
type LogResponse struct {
	ID              string     `json:"id"`
	Topic           string     `json:"topic"`
	Channel         string     `json:"channel"`
	RecipientEmails []string   `json:"recipientEmails"`
	Subject         string     `json:"subject"`
	ContentSummary  *string    `json:"contentSummary"`
	Status          string     `json:"status"`
	ErrorMessage    *string    `json:"errorMessage"`
	TriggeredBy     *uuid.UUID `json:"triggeredBy"`
	TriggeredByName *string    `json:"triggeredByName"`
	SentAt          time.Time  `json:"sentAt"`
}

func newLogResponses(list []app.Log) []LogResponse {
	out := make([]LogResponse, 0, len(list))
	for _, l := range list {
		emails := l.RecipientEmails
		if emails == nil {
			emails = []string{}
		}
		out = append(out, LogResponse{
			ID:              strconv.FormatInt(l.ID, 10),
			Topic:           l.Topic,
			Channel:         l.Channel,
			RecipientEmails: emails,
			Subject:         l.Subject,
			ContentSummary:  l.ContentSummary,
			Status:          l.Status,
			ErrorMessage:    l.ErrorMessage,
			TriggeredBy:     l.TriggeredBy,
			TriggeredByName: l.TriggeredByName,
			SentAt:          l.SentAt,
		})
	}
	return out
}
