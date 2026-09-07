package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/platform/clock"
)

// EmailSender 定義電子郵件發送介面。
type EmailSender interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

// EmailSenderWithMessageID 是可回傳 provider 追蹤編號的寄信 adapter；舊 adapter
// 仍可只實作 EmailSender，delivery store 會保留空的 provider_message_id。
type EmailSenderWithMessageID interface {
	SendEmailWithMessageID(ctx context.Context, to, subject, body string) (string, error)
}

// SendResult 記錄一次通知事件實際成功與失敗的寄送數量。
type SendResult struct {
	Sent   int
	Failed int
}

// LogEmailSender 只寫日誌、不對外送信的模擬發送器；未設定寄信 provider 時（含 production）由它承接。
type LogEmailSender struct{}

func (s *LogEmailSender) SendEmail(ctx context.Context, to, subject, body string) error {
	slog.Info("Simulated email sent")
	return nil
}

type dedupStore interface {
	ClaimNotificationEvent(ctx context.Context, dedupKey, topic string) (bool, error)
	CompleteNotificationEvent(ctx context.Context, dedupKey string) error
	ReleaseNotificationEvent(ctx context.Context, dedupKey string) error
}

// deliveryStore 以「通知事件＋收件人」為粒度保存派送狀態，讓部分成功重試時
// 已成功的收件人不會再次呼叫外部 provider。
type deliveryStore interface {
	ClaimNotificationDelivery(ctx context.Context, eventID, recipientKey string) (bool, error)
	CompleteNotificationDelivery(ctx context.Context, eventID, recipientKey, providerMessageID string) error
	FailNotificationDelivery(ctx context.Context, eventID, recipientKey, lastError string) error
}

type unavailableEmailSender struct{}

func (s *unavailableEmailSender) SendEmail(ctx context.Context, to, subject, body string) error {
	return fmt.Errorf("email sender is not configured")
}

// NotificationService 負責系統告警與通知派送及收件人管理。
type NotificationService struct {
	repo          Store
	auditRepo     AuditWriter
	sender        EmailSender
	businessClock clock.Clock
}

// NotificationOption 調整通知服務的業務時間來源。
type NotificationOption func(*NotificationService)

// WithNotificationClock 注入臺灣業務時間，讓通知留痕可穩定測試。
func WithNotificationClock(c clock.Clock) NotificationOption {
	return func(s *NotificationService) { s.businessClock = c }
}

// NewNotificationService 建立 NotificationService 實例。
func NewNotificationService(repo Store, auditRepo AuditWriter, sender EmailSender, options ...NotificationOption) *NotificationService {
	if sender == nil {
		// 缺少 sender 必須 fail closed；不可把沒有真正寄信的 log sender 當成成功。
		sender = &unavailableEmailSender{}
	}
	service := &NotificationService{
		repo:      repo,
		auditRepo: auditRepo,
		sender:    sender,
	}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *NotificationService) now() time.Time {
	if s.businessClock != nil {
		return s.businessClock.Now()
	}
	return clock.Now()
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

// SendNotificationDedup 以事件 key 保證 scheduler retry 不會重複寄送同一通知。
// 舊的測試／離線 store 若未實作 dedup port，仍維持原本的單次派送行為。
func (s *NotificationService) SendNotificationDedup(ctx context.Context, topic, subject, body, dedupKey string) error {
	store, ok := s.repo.(dedupStore)
	if !ok || dedupKey == "" {
		return s.SendNotification(ctx, topic, subject, body)
	}
	claimed, err := store.ClaimNotificationEvent(ctx, dedupKey, topic)
	if err != nil {
		return fmt.Errorf("claim notification event: %w", err)
	}
	if !claimed {
		return nil
	}
	result, err := s.sendNotificationWithResult(ctx, topic, subject, body, dedupKey)
	if err != nil {
		if releaseErr := store.ReleaseNotificationEvent(ctx, dedupKey); releaseErr != nil {
			return fmt.Errorf("%w; release notification event: %v", err, releaseErr)
		}
		return err
	}
	if result.Failed > 0 {
		if releaseErr := store.ReleaseNotificationEvent(ctx, dedupKey); releaseErr != nil {
			return fmt.Errorf("notification delivery failed for %d recipient(s); release notification event: %v", result.Failed, releaseErr)
		}
		return fmt.Errorf("notification delivery failed for %d recipient(s)", result.Failed)
	}
	if err := store.CompleteNotificationEvent(ctx, dedupKey); err != nil {
		return fmt.Errorf("complete notification event: %w", err)
	}
	return nil
}

// SendNotificationWithResult 派送通知並回傳實際結果；部分收件人失敗時不再
// 靜默回傳成功，呼叫端可依 Failed 觸發 retry 或告警。
func (s *NotificationService) SendNotificationWithResult(ctx context.Context, topic, subject, body string) (SendResult, error) {
	return s.sendNotificationWithResult(ctx, topic, subject, body, "")
}

func (s *NotificationService) sendNotificationWithResult(ctx context.Context, topic, subject, body, eventID string) (SendResult, error) {
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
			ContentSummary:  notificationContentSummary(topic),
			Status:          "failed",
			ErrorMessage:    &errMsg,
			SentAt:          s.now(),
		}
		if err := s.repo.InsertLog(ctx, logItem); err != nil {
			return SendResult{}, fmt.Errorf("failed to record notification failure: %w", err)
		}
		slog.Warn("Notification not sent because recipient list is empty", slog.String("topic", topic))
		return SendResult{}, fmt.Errorf("%w: %s", ErrNoNotificationRecipients, topic)
	}

	deliveries, _ := s.repo.(deliveryStore)
	result := SendResult{}
	for _, r := range recipients {
		recipientKey := notificationRecipientKey(r)
		deliveryClaimed := false
		if deliveries != nil && eventID != "" {
			claimed, err := deliveries.ClaimNotificationDelivery(ctx, eventID, recipientKey)
			if err != nil {
				return result, fmt.Errorf("claim notification delivery: %w", err)
			}
			if !claimed {
				// false 代表這位收件人的 delivery 已完成，或仍由另一個
				// worker 處理；同一事件的 global claim 已避免後者在正常路徑發生。
				result.Sent++
				continue
			}
			deliveryClaimed = true
		}

		if r.Email == "" {
			slog.Warn("Skipping notification recipient without a resolved email",
				slog.String("topic", topic), slog.String("recipientType", r.RecipientType), slog.Int64("recipientId", r.ID))
			result.Failed++
			errMsg := "收件人沒有可用的電子郵件地址"
			if deliveryClaimed {
				if err := deliveries.FailNotificationDelivery(ctx, eventID, recipientKey, errMsg); err != nil {
					return result, fmt.Errorf("record notification delivery failure: %w", err)
				}
			}
			if err := s.repo.InsertLog(ctx, &Log{
				Topic:           topic,
				Channel:         "email",
				RecipientEmails: []string{},
				Subject:         subject,
				ContentSummary:  notificationContentSummary(topic),
				Status:          "failed",
				ErrorMessage:    &errMsg,
				SentAt:          s.now(),
			}); err != nil {
				return result, fmt.Errorf("failed to record notification log: %w", err)
			}
			continue
		}

		providerMessageID, sendErr := s.sendEmail(ctx, r.Email, subject, body)
		status := "sent"
		var errStr *string
		if sendErr != nil {
			status = "failed"
			// Provider 的原始錯誤可能含收件人或第三方 request detail，不寫入長期通知紀錄。
			msg := "電子郵件服務商拒絕寄送"
			errStr = &msg
			result.Failed++
			if deliveryClaimed {
				if err := deliveries.FailNotificationDelivery(ctx, eventID, recipientKey, msg); err != nil {
					return result, fmt.Errorf("record notification delivery failure: %w", err)
				}
			}
		} else {
			result.Sent++
			if deliveryClaimed {
				if err := deliveries.CompleteNotificationDelivery(ctx, eventID, recipientKey, providerMessageID); err != nil {
					return result, fmt.Errorf("complete notification delivery: %w", err)
				}
			}
		}
		logItem := &Log{
			Topic:           topic,
			Channel:         "email",
			RecipientEmails: []string{maskEmail(r.Email)},
			Subject:         subject,
			ContentSummary:  notificationContentSummary(topic),
			Status:          status,
			ErrorMessage:    errStr,
			SentAt:          s.now(),
		}
		if err := s.repo.InsertLog(ctx, logItem); err != nil {
			return result, fmt.Errorf("failed to record notification log: %w", err)
		}
	}

	return result, nil
}

func (s *NotificationService) sendEmail(ctx context.Context, to, subject, body string) (string, error) {
	if sender, ok := s.sender.(EmailSenderWithMessageID); ok {
		return sender.SendEmailWithMessageID(ctx, to, subject, body)
	}
	return "", s.sender.SendEmail(ctx, to, subject, body)
}

func notificationRecipientKey(r Recipient) string {
	if r.ID != 0 {
		return fmt.Sprintf("%s:%d", r.RecipientType, r.ID)
	}
	return fmt.Sprintf("%s:%s", r.RecipientType, strings.ToLower(strings.TrimSpace(r.Email)))
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
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "setting_change",
			EntityType: "notification_recipient",
			EntityID:   notificationRecipientAuditID(item),
			AfterData:  itemAuditSnapshot(item),
		}); err != nil {
			slog.Error("notification recipient audit write failed", "action", "create", "error", err)
		}
	}

	return item, nil
}

// UpdateRecipient 修改收件人設定。
func (s *NotificationService) UpdateRecipient(ctx context.Context, id int64, email string, displayName *string, active bool, actorID uuid.UUID, actorRole string) (*Recipient, error) {
	before, err := s.repo.GetRecipientByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load notification recipient: %w", err)
	}

	item, err := s.repo.UpdateRecipient(ctx, id, email, displayName, active)
	if err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "setting_change",
			EntityType: "notification_recipient",
			EntityID:   notificationRecipientAuditID(item),
			BeforeData: itemAuditSnapshot(before),
			AfterData:  itemAuditSnapshot(item),
		}); err != nil {
			slog.Error("notification recipient audit write failed", "action", "update", "recipient_id", id, "error", err)
		}
	}

	return item, nil
}

// DeleteRecipient 刪除收件人。
func (s *NotificationService) DeleteRecipient(ctx context.Context, id int64, actorID uuid.UUID, actorRole string) error {
	before, err := s.repo.GetRecipientByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to load notification recipient: %w", err)
	}

	if err := s.repo.DeleteRecipient(ctx, id); err != nil {
		return err
	}

	if s.auditRepo != nil {
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "setting_change",
			EntityType: "notification_recipient",
			EntityID:   notificationRecipientAuditID(before),
			BeforeData: itemAuditSnapshot(before),
		}); err != nil {
			slog.Error("notification recipient audit write failed", "action", "delete", "recipient_id", id, "error", err)
		}
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
		entityID := fmt.Sprintf("batch:%d", len(created))
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "batch_create_recipients",
			EntityType: "notification_recipient",
			EntityID:   &entityID,
			AfterData:  recipientBatchAuditSnapshot(created),
		}); err != nil {
			slog.Error("notification recipient batch audit write failed", "action", "batch_create_recipients", "error", err)
		}
	}

	return created, nil
}

// BatchDeleteRecipients 批次刪除收件人並留存單筆彙整稽核紀錄。
func (s *NotificationService) BatchDeleteRecipients(ctx context.Context, ids []int64, actorID uuid.UUID, actorRole string) (int64, error) {
	var before []Recipient
	if s.auditRepo != nil {
		before = make([]Recipient, 0, len(ids))
		for _, id := range ids {
			item, err := s.repo.GetRecipientByID(ctx, id)
			if err != nil {
				return 0, fmt.Errorf("failed to load notification recipient %d: %w", id, err)
			}
			if item != nil {
				before = append(before, *item)
			}
		}
	}
	count, err := s.repo.BatchDeleteRecipients(ctx, ids)
	if err != nil {
		return 0, err
	}

	if s.auditRepo != nil && count > 0 {
		entityID := fmt.Sprintf("batch:%d", count)
		if err := s.auditRepo.Write(ctx, AuditEntry{
			ActorID:    &actorID,
			ActorRole:  &actorRole,
			Action:     "batch_delete_recipients",
			EntityType: "notification_recipient",
			EntityID:   &entityID,
			BeforeData: recipientBatchAuditSnapshot(before),
			AfterData:  RecipientBatchAuditSummary{Count: count},
		}); err != nil {
			slog.Error("notification recipient batch audit write failed", "action", "batch_delete_recipients", "error", err)
		}
	}

	return count, nil
}

func notificationContentSummary(topic string) *string {
	summary := fmt.Sprintf("notification topic: %s", topic)
	return &summary
}

func maskEmail(email string) string {
	email = strings.TrimSpace(email)
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return "[REDACTED]"
	}
	local := []rune(email[:at])
	if len(local) == 1 {
		return string(local) + "***" + email[at:]
	}
	return string(local[:1]) + "***" + email[at:]
}

func itemAuditSnapshot(item *Recipient) interface{} {
	if item == nil {
		return nil
	}
	snapshot := item.AuditSnapshot()
	return snapshot
}

func recipientBatchAuditSnapshot(items []Recipient) RecipientBatchAuditSnapshot {
	snapshots := make([]RecipientAuditSnapshot, 0, len(items))
	for i := range items {
		snapshots = append(snapshots, items[i].AuditSnapshot())
	}
	return RecipientBatchAuditSnapshot{Recipients: snapshots}
}

func notificationRecipientAuditID(item *Recipient) *string {
	if item == nil {
		return nil
	}
	value := fmt.Sprintf("%d", item.ID)
	if item.ID == 0 {
		value = maskEmail(item.Email)
	}
	return &value
}
