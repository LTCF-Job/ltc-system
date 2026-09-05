package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingEmailSender struct{}

func (failingEmailSender) SendEmail(context.Context, string, string, string) error {
	return errors.New("provider unavailable")
}

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
	repo := emptyStore{}
	sender := &MockEmailSender{}
	svc := NewNotificationService(repo, nil, sender)

	ctx := context.Background()
	err := svc.SendNotification(ctx, "missing_report", "測試告警", "未回報筆數: 3")
	assert.NoError(t, err)
	assert.Empty(t, sender.SentList, "無收件人時不應呼叫 SendEmail")
}

// emptyStore 沒有任何收件人，讓測試能驗證「查無收件人」的分支而不需資料庫。
type emptyStore struct{}

func (emptyStore) ListRecipients(context.Context, string, bool) ([]Recipient, error) { return nil, nil }
func (emptyStore) GetRecipientByID(context.Context, int64) (*Recipient, error)       { return nil, nil }
func (emptyStore) CreateRecipient(context.Context, *Recipient) error                 { return nil }
func (emptyStore) UpdateRecipient(context.Context, int64, string, *string, bool) (*Recipient, error) {
	return nil, nil
}
func (emptyStore) DeleteRecipient(context.Context, int64) error { return nil }
func (emptyStore) InsertLog(context.Context, *Log) error        { return nil }
func (emptyStore) ListLogs(context.Context, string, int, int) ([]Log, int64, error) {
	return nil, 0, nil
}
func (emptyStore) BatchCreateRecipients(context.Context, []Recipient) ([]Recipient, error) {
	return nil, nil
}
func (emptyStore) BatchDeleteRecipients(context.Context, []int64) (int64, error) { return 0, nil }

// fakeRecipientStore 記錄批次寫入呼叫，供斷言白名單守門與 audit rollback 語意。
type fakeRecipientStore struct {
	emptyStore
	batchCreateCalled bool
	created           []Recipient
	createErr         error
	batchDeleteCalled bool
	deleteCount       int64
	deleteErr         error
	listRecipients    []Recipient
}

func (f *fakeRecipientStore) ListRecipients(context.Context, string, bool) ([]Recipient, error) {
	return f.listRecipients, nil
}

func (f *fakeRecipientStore) BatchCreateRecipients(_ context.Context, items []Recipient) ([]Recipient, error) {
	f.batchCreateCalled = true
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.created = items
	return items, nil
}

func (f *fakeRecipientStore) BatchDeleteRecipients(_ context.Context, ids []int64) (int64, error) {
	f.batchDeleteCalled = true
	if f.deleteErr != nil {
		return 0, f.deleteErr
	}
	return f.deleteCount, nil
}

type fakeNotificationAuditWriter struct {
	entries []AuditEntry
}

func (f *fakeNotificationAuditWriter) Write(_ context.Context, e AuditEntry) error {
	f.entries = append(f.entries, e)
	return nil
}

func TestNotificationService_BatchCreateRecipients_RejectsUnknownTopic(t *testing.T) {
	store := &fakeRecipientStore{}
	svc := NewNotificationService(store, nil, &MockEmailSender{})

	_, err := svc.BatchCreateRecipients(context.Background(), []BatchRecipientInput{
		{Topic: "not_a_real_topic", Email: "a@example.com"},
	}, uuid.New(), "admin")

	assert.Error(t, err)
	assert.False(t, store.batchCreateCalled, "白名單守門未通過時不應呼叫 store")
}

func TestNotificationService_BatchCreateRecipients_Success(t *testing.T) {
	store := &fakeRecipientStore{}
	audit := &fakeNotificationAuditWriter{}
	svc := NewNotificationService(store, audit, &MockEmailSender{})

	created, err := svc.BatchCreateRecipients(context.Background(), []BatchRecipientInput{
		{Topic: "missing_report", Email: "a@example.com"},
		{Topic: "month_end", Email: "b@example.com"},
	}, uuid.New(), "admin")

	assert.NoError(t, err)
	assert.Len(t, created, 2)
	if assert.Len(t, audit.entries, 1) {
		assert.Equal(t, "batch_create_recipients", audit.entries[0].Action)
	}
}

func TestNotificationService_BatchDeleteRecipients_ReturnsCount(t *testing.T) {
	store := &fakeRecipientStore{deleteCount: 3}
	audit := &fakeNotificationAuditWriter{}
	svc := NewNotificationService(store, audit, &MockEmailSender{})

	count, err := svc.BatchDeleteRecipients(context.Background(), []int64{1, 2, 3}, uuid.New(), "admin")

	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
	if assert.Len(t, audit.entries, 1) {
		assert.Equal(t, "batch_delete_recipients", audit.entries[0].Action)
	}
}

func TestNotificationService_SendNotification_FailsRecipientsWithoutResolvedEmail(t *testing.T) {
	store := &fakeRecipientStore{
		listRecipients: []Recipient{
			{ID: 1, RecipientType: "email", Email: "a@example.com"},
			{ID: 2, RecipientType: "role", Email: ""}, // role 型別尚未接上身分模組展開，Email 為空
		},
	}
	sender := &MockEmailSender{}
	svc := NewNotificationService(store, nil, sender)

	err := svc.SendNotification(context.Background(), "missing_report", "測試", "內容")

	assert.Error(t, err)
	require.Len(t, sender.SentList, 1, "只有已解析出 email 的收件人應該被寄送")
	assert.Equal(t, "a@example.com", sender.SentList[0].To)
}

func TestNotificationService_SendNotification_ReturnsErrorWhenProviderFails(t *testing.T) {
	store := &fakeRecipientStore{listRecipients: []Recipient{{ID: 1, Email: "a@example.com"}}}
	svc := NewNotificationService(store, nil, failingEmailSender{})

	err := svc.SendNotification(context.Background(), "missing_report", "測試", "內容")

	assert.Error(t, err)
}
