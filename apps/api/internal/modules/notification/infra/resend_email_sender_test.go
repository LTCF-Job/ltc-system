package infra

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResendEmailSender_SendEmail(t *testing.T) {
	var gotRequest struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		Text    string   `json:"text"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("expected bearer auth, got %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := &ResendEmailSender{
		apiKey:   "test-key",
		from:     "noreply@example.com",
		endpoint: server.URL,
		client:   server.Client(),
	}

	err := sender.SendEmail(context.Background(), "recipient@example.com", "主旨", "內容")

	require.NoError(t, err)
	require.Equal(t, "noreply@example.com", gotRequest.From)
	require.Equal(t, []string{"recipient@example.com"}, gotRequest.To)
	require.Equal(t, "主旨", gotRequest.Subject)
	require.Equal(t, "內容", gotRequest.Text)
}

func TestResendEmailSender_ReturnsErrorForHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	sender := &ResendEmailSender{apiKey: "test-key", from: "noreply@example.com", endpoint: server.URL, client: server.Client()}

	err := sender.SendEmail(context.Background(), "recipient@example.com", "主旨", "內容")

	require.Error(t, err)
}

func TestResendEmailSender_RequiresConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		sender ResendEmailSender
	}{
		{name: "missing api key", sender: ResendEmailSender{from: "noreply@example.com"}},
		{name: "missing from address", sender: ResendEmailSender{apiKey: "test-key"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sender.SendEmail(context.Background(), "recipient@example.com", "主旨", "內容")
			require.Error(t, err)
		})
	}
}
