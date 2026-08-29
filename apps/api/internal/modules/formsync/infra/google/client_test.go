package google_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/modules/formsync/infra/google"
)

func TestNewClient_NoServiceAccountConfigured(t *testing.T) {
	ctx := context.Background()
	client, err := google.NewClient(ctx, "")
	require.NoError(t, err)
	// 未設定 Service Account 憑證時必須回傳 nil，讓呼叫端視為「Google 功能不可用」
	// 並回傳明確錯誤；不得建立一個看似可用、實際回傳假資料的 client。
	assert.Nil(t, client)
}
