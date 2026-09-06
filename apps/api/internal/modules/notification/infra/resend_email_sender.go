package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const resendEmailEndpoint = "https://api.resend.com/emails"

// ResendEmailSender 將通知寄送至 Resend Email API。
type ResendEmailSender struct {
	apiKey   string
	from     string
	endpoint string
	client   *http.Client
}

// NewResendEmailSender 建立正式環境使用的 Resend 寄信 adapter。
func NewResendEmailSender(apiKey, from string, client *http.Client) *ResendEmailSender {
	if client == nil {
		client = &http.Client{}
	}
	return &ResendEmailSender{
		apiKey:   apiKey,
		from:     from,
		endpoint: resendEmailEndpoint,
		client:   client,
	}
}

func (s *ResendEmailSender) SendEmail(ctx context.Context, to, subject, body string) error {
	_, err := s.SendEmailWithMessageID(ctx, to, subject, body)
	return err
}

// SendEmailWithMessageID 將通知寄送至 Resend，並回傳 provider 的追蹤編號。
func (s *ResendEmailSender) SendEmailWithMessageID(ctx context.Context, to, subject, body string) (string, error) {
	if s.apiKey == "" {
		return "", errors.New("resend API key is not configured")
	}
	if s.from == "" {
		return "", errors.New("notification sender address is not configured")
	}

	payload, err := json.Marshal(struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		Text    string   `json:"text"`
	}{
		From:    s.from,
		To:      []string{to},
		Subject: subject,
		Text:    body,
	})
	if err != nil {
		return "", fmt.Errorf("encode Resend email request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create Resend email request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send email through Resend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("Resend returned HTTP %s", resp.Status)
	}
	var response struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		if errors.Is(err, io.EOF) {
			// Provider 呼叫已成功；若回應沒有 body，仍可保留空的追蹤編號。
			return "", nil
		}
		return "", fmt.Errorf("decode Resend email response: %w", err)
	}
	return response.ID, nil
}
