package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/repository"
)

// EmailSender 定義電子郵件發送介面。
type EmailSender interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

// LogEmailSender 供本機或測試環境使用之日誌模擬發送器。
type LogEmailSender struct{}

func (s *LogEmailSender) SendEmail(ctx context.Context, to, subject, body string) error {
	slog.Info("Simulated email sent", slog.String("to", to), slog.String("subject", subject))
	return nil
}

// NotificationService 負責系統告警與通知派送及收件人管理。
type NotificationService struct {
	repo      *repository.NotificationRepository
	auditRepo *repository.AuditRepository
	sender    EmailSender
}

// NewNotificationService 建立 NotificationService 實例。
func NewNotificationService(repo *repository.NotificationRepository, auditRepo *repository.AuditRepository, sender EmailSender) *NotificationService {
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
	recipients, err := s.repo.ListRecipients(ctx, topic, true)
	if err != nil {
		return fmt.Errorf("failed to load notification recipients: %w", err)
	}

	// 規格書 §8.4：收件人為空時不寄送，改寫入失敗留痕
	if len(recipients) == 0 {
		errMsg := "無設定收件人"
		logItem := &repository.NotificationLogEntity{
			Channel: "email",
			Target:  "-",
			Topic:   topic,
			Body:    fmt.Sprintf("[%s] %s", subject, body),
			SentAt:  time.Now().UTC(),
			Success: false,
			Error:   &errMsg,
		}
		_ = s.repo.InsertLog(ctx, logItem)
		slog.Warn("Notification not sent because recipient list is empty", slog.String("topic", topic))
		return nil
	}

	for _, r := range recipients {
		sendErr := s.sender.SendEmail(ctx, r.Email, subject, body)
		logItem := &repository.NotificationLogEntity{
			Channel: "email",
			Target:  r.Email,
			Topic:   topic,
			Body:    fmt.Sprintf("[%s] %s", subject, body),
			SentAt:  time.Now().UTC(),
			Success: sendErr == nil,
		}
		if sendErr != nil {
			errStr := sendErr.Error()
			logItem.Error = &errStr
		}
		_ = s.repo.InsertLog(ctx, logItem)
	}

	return nil
}

// ListRecipients 取得收件人清單。
func (s *NotificationService) ListRecipients(ctx context.Context, topic string) ([]repository.NotificationRecipientEntity, error) {
	return s.repo.ListRecipients(ctx, topic, false)
}

// CreateRecipient 新增收件人並留存稽核紀錄。
func (s *NotificationService) CreateRecipient(ctx context.Context, topic, email string, displayName *string, actorID uuid.UUID, actorRole string) (*repository.NotificationRecipientEntity, error) {
	item := &repository.NotificationRecipientEntity{
		Topic:       topic,
		Email:       email,
		DisplayName: displayName,
		Active:      true,
		CreatedBy:   actorID,
	}

	if err := s.repo.CreateRecipient(ctx, item); err != nil {
		return nil, err
	}

	// 留存 setting_change 稽核紀錄
	if s.auditRepo != nil {
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
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
func (s *NotificationService) UpdateRecipient(ctx context.Context, id int64, email string, displayName *string, active bool, actorID uuid.UUID, actorRole string) (*repository.NotificationRecipientEntity, error) {
	before, _ := s.repo.GetRecipientByID(ctx, id)

	item, err := s.repo.UpdateRecipient(ctx, id, email, displayName, active)
	if err != nil {
		return nil, err
	}


	if s.auditRepo != nil {
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
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
		_ = s.auditRepo.Insert(ctx, &repository.AuditLogEntity{
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
func (s *NotificationService) ListLogs(ctx context.Context, topic string, page, pageSize int) ([]repository.NotificationLogEntity, int64, error) {
	return s.repo.ListLogs(ctx, topic, page, pageSize)
}
