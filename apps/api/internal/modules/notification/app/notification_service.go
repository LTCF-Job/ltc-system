package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// EmailSender 定義電子郵件發送介面。
type EmailSender interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

// SendResult 記錄一次通知事件實際成功與失敗的寄送數量。
type SendResult struct {
	Sent   int
	Failed int
}

// LogEmailSender 供本機或測試環境使用之日誌模擬發送器。
type LogEmailSender struct{}

func (s *LogEmailSender) SendEmail(ctx context.Context, to, subject, body string) error {
	slog.Info("Simulated email sent", slog.String("to", to), slog.String("subject", subject))
	return nil
}

// NotificationService 負責系統告警與通知派送及收件人管理。
type NotificationService struct {
	repo      Store
	auditRepo AuditWriter
	sender    EmailSender
}

// NewNotificationService 建立 NotificationService 實例。
func NewNotificationService(repo Store, auditRepo AuditWriter, sender EmailSender) *NotificationService {
	if sender == nil {
		sender = &LogEmailSender{}
	}
	return &NotificationService{
		repo:      repo,
		auditRepo: auditRepo,
		sender:    sender,
	}
}

// SendNotification 依主題將通知寄送給所有啟用的收件人，並記錄發送留痕。
func (s *NotificationService) SendNotification(ctx context.Context, topic, subject, body string) error {
	result, err := s.SendNotificationWithResult(ctx, topic, subject, body)
	if err != nil {
		return err
	}
	if result.Failed > 0 {
		return fmt.Errorf("notification delivery failed for %d recipient(s)", result.Failed)
	}
	return nil
}

// SendNotificationWithResult 派送通知並回傳實際結果；部分收件人失敗時不再
// 靜默回傳成功，呼叫端可依 Failed 觸發 retry 或告警。
func (s *NotificationService) SendNotificationWithResult(ctx context.Context, topic, subject, body string) (SendResult, error) {
	recipients, err := s.repo.ListRecipients(ctx, topic, true)
	if err != nil {
		return SendResult{}, fmt.Errorf("failed to load notification recipients: %w", err)
	}

	// 規格書 §8.4：收件人為空時不寄送，改寫入失敗留痕
	if len(recipients) == 0 {
		errMsg := "無設定收件人"
		logItem := &Log{
			Topic:           topic,
			Channel:         "email",
			RecipientEmails: []string{},
			Subject:         subject,
			ContentSummary:  &body,
			Status:          "failed",
			ErrorMessage:    &errMsg,
			SentAt:          time.Now().UTC(),
		}
		_ = s.repo.InsertLog(ctx, logItem)
		slog.Warn("Notification not sent because recipient list is empty", slog.String("topic", topic))
		return SendResult{}, nil
	}

	result := SendResult{}
	for _, r := range recipients {
		if r.Email == "" {
			slog.Warn("Skipping notification recipient without a resolved email",
				slog.String("topic", topic), slog.String("recipientType", r.RecipientType), slog.Int64("recipientId", r.ID))
			continue
		}

		var sendErr error
		if s.sender != nil {
			sendErr = s.sender.SendEmail(ctx, r.Email, subject, body)
		}
		status := "sent"
		var errStr *string
		if sendErr != nil {
			status = "failed"
			msg := sendErr.Error()
			errStr = &msg
			result.Failed++
		} else {
			result.Sent++
		}
		logItem := &Log{
			Topic:           topic,
			Channel:         "email",
			RecipientEmails: []string{r.Email},
			Subject:         subject,
			ContentSummary:  &body,
			Status:          status,
			ErrorMessage:    errStr,
			SentAt:          time.Now().UTC(),
		}
		if err := s.repo.InsertLog(ctx, logItem); err != nil {
			return result, fmt.Errorf("failed to record notification log: %w", err)
		}
	}

	return result, nil
}

// ListRecipients 取得收件人清單。
func (s *NotificationService) ListRecipients(ctx context.Context, topic string) ([]Recipient, error) {
	return s.repo.ListRecipients(ctx, topic, false)
}

// CreateRecipient 新增收件人並留存稽核紀錄；目前唯一的建立入口只支援 email 型別。
func (s *NotificationService) CreateRecipient(ctx context.Context, topic, email string, displayName *string, actorID uuid.UUID, actorRole string) (*Recipient, error) {
	item := &Recipient{
		Topic:         topic,
		RecipientType: "email",
		Email:         email,
		DisplayName:   displayName,
		Active:        true,
		CreatedBy:     actorID,
	}

	if err := s.repo.CreateRecipient(ctx, item); err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "setting_change",
			EntityType: "notification_recipient",
			EntityID:   &item.Email,
			AfterData:  item,
		})
	}

	return item, nil
}

// UpdateRecipient 修改收件人設定。
func (s *NotificationService) UpdateRecipient(ctx context.Context, id int64, email string, displayName *string, active bool, actorID uuid.UUID, actorRole string) (*Recipient, error) {
	before, _ := s.repo.GetRecipientByID(ctx, id)

	item, err := s.repo.UpdateRecipient(ctx, id, email, displayName, active)
	if err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "setting_change",
			EntityType: "notification_recipient",
			EntityID:   &item.Email,
			BeforeData: before,
			AfterData:  item,
		})
	}

	return item, nil
}

// DeleteRecipient 刪除收件人。
func (s *NotificationService) DeleteRecipient(ctx context.Context, id int64, actorID uuid.UUID, actorRole string) error {
	before, _ := s.repo.GetRecipientByID(ctx, id)

	if err := s.repo.DeleteRecipient(ctx, id); err != nil {
		return err
	}

	if s.auditRepo != nil {
		var entityID string
		if before != nil {
			entityID = before.Email
		}
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "setting_change",
			EntityType: "notification_recipient",
			EntityID:   &entityID,
			BeforeData: before,
		})
	}

	return nil
}

// ListLogs 取得通知發送歷史。
func (s *NotificationService) ListLogs(ctx context.Context, topic string, page, pageSize int) ([]Log, int64, error) {
	return s.repo.ListLogs(ctx, topic, page, pageSize)
}

// notificationTopics 是允許的通知主題白名單，與 notification_recipients 資料表的 CHECK 約束一致。
var notificationTopics = map[string]bool{
	"missing_report": true,
	"driver_leave":   true,
	"month_end":      true,
	"export_failed":  true,
}

// BatchRecipientInput 代表批次新增收件人的單筆輸入。
type BatchRecipientInput struct {
	Topic       string
	Email       string
	DisplayName *string
}

// BatchCreateRecipients 批次新增收件人並留存單筆彙整稽核紀錄。
func (s *NotificationService) BatchCreateRecipients(ctx context.Context, items []BatchRecipientInput, actorID uuid.UUID, actorRole string) ([]Recipient, error) {
	entries := make([]Recipient, 0, len(items))
	for _, in := range items {
		if !notificationTopics[in.Topic] {
			return nil, fmt.Errorf("unsupported notification topic: %s", in.Topic)
		}
		entries = append(entries, Recipient{
			Topic:       in.Topic,
			Email:       in.Email,
			DisplayName: in.DisplayName,
			CreatedBy:   actorID,
		})
	}

	created, err := s.repo.BatchCreateRecipients(ctx, entries)
	if err != nil {
		return nil, err
	}

	if s.auditRepo != nil && len(created) > 0 {
		emails := make([]string, 0, len(created))
		for _, r := range created {
			emails = append(emails, r.Email)
		}
		entityID := fmt.Sprintf("batch:%d", len(created))
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "batch_create_recipients",
			EntityType: "notification_recipient",
			EntityID:   &entityID,
			AfterData:  emails,
		})
	}

	return created, nil
}

// BatchDeleteRecipients 批次刪除收件人並留存單筆彙整稽核紀錄。
func (s *NotificationService) BatchDeleteRecipients(ctx context.Context, ids []int64, actorID uuid.UUID, actorRole string) (int64, error) {
	count, err := s.repo.BatchDeleteRecipients(ctx, ids)
	if err != nil {
		return 0, err
	}

	if s.auditRepo != nil && count > 0 {
		entityID := fmt.Sprintf("batch:%d", count)
		_ = s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "batch_delete_recipients",
			EntityType: "notification_recipient",
			EntityID:   &entityID,
			AfterData:  map[string]int64{"count": count},
		})
	}

	return count, nil
}
