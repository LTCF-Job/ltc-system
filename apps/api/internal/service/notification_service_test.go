package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"ltc-system/apps/api/internal/repository"
)

// MockEmailSender 供測試之 mock 寄信元件。
type MockEmailSender struct {
	SentList []struct {
		To      string
		Subject string
		Body    string
	}
}

func (m *MockEmailSender) SendEmail(ctx context.Context, to, subject, body string) error {
	m.SentList = append(m.SentList, struct {
		To      string
		Subject string
		Body    string
	}{To: to, Subject: subject, Body: body})
	return nil
}

func TestNotificationService_EmptyRecipients_WritesFailureLog(t *testing.T) {
	// 驗證規格書 §8.4：收件人為空時不寄送，改留存 success=false 與 error='無設定收件人'
	repo := repository.NewNotificationRepository(nil)
	sender := &MockEmailSender{}
	svc := NewNotificationService(repo, nil, sender)

	ctx := context.Background()
	err := svc.SendNotification(ctx, "missing_report", "測試告警", "未回報筆數: 3")
	assert.NoError(t, err)
	assert.Empty(t, sender.SentList, "無收件人時不應呼叫 SendEmail")
}
