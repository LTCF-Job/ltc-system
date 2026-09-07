package infra

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/modules/identity/app"
)

func TestSupabaseAdminClient_Configured(t *testing.T) {
	assert.False(t, NewSupabaseAdminClient("", "", nil).Configured())
	assert.False(t, NewSupabaseAdminClient("https://x.supabase.co", "", nil).Configured())
	assert.True(t, NewSupabaseAdminClient("https://x.supabase.co", "key", nil).Configured())
}

func TestSupabaseAdminClient_CreateUser_SendsRoleInAppMetadataOnly(t *testing.T) {
	var capturedAuth, capturedAPIKey, capturedPath string
	var capturedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedAPIKey = r.Header.Get("apikey")
		capturedPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "00000000-0000-0000-0000-000000000001",
			"email": "new@example.com",
		})
	}))
	defer srv.Close()

	client := NewSupabaseAdminClient(srv.URL, "service-role-key", nil)

	_, err := client.CreateUser(t.Context(), app.CreateAuthUserInput{
		Email:       "new@example.com",
		Password:    "test12345",
		DisplayName: "測試使用者",
		RoleKey:     "dispatcher",
	})
	require.NoError(t, err)

	assert.Equal(t, "Bearer service-role-key", capturedAuth)
	assert.Equal(t, "service-role-key", capturedAPIKey)
	assert.Equal(t, "/auth/v1/admin/users", capturedPath)

	appMetadata, ok := capturedBody["app_metadata"].(map[string]any)
	require.True(t, ok, "app_metadata 必須存在")
	assert.Equal(t, "dispatcher", appMetadata["role"])
	assert.Equal(t, "dispatcher", appMetadata["role_key"])

	userMetadata, ok := capturedBody["user_metadata"].(map[string]any)
	require.True(t, ok, "user_metadata 必須存在")
	_, hasRole := userMetadata["role"]
	assert.False(t, hasRole, "user_metadata 不應包含 role 欄位——角色只能寫入 app_metadata")
	assert.Equal(t, "測試使用者", userMetadata["display_name"])
}

func TestSupabaseAdminClient_NonSuccessStatusReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"invalid api key"}`))
	}))
	defer srv.Close()

	client := NewSupabaseAdminClient(srv.URL, "bad-key", nil)
	_, err := client.ListUsers(t.Context())
	assert.Error(t, err)
	assert.Equal(t, "supabase admin request returned HTTP 401", err.Error())
	assert.NotContains(t, err.Error(), "invalid api key")
}

func TestSupabaseAdminClient_VerifyPassword_UsesGrantTypePasswordEndpoint(t *testing.T) {
	var capturedPath, capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := NewSupabaseAdminClient(srv.URL, "key", nil)
	err := client.VerifyPassword(t.Context(), "a@example.com", "password123")
	require.NoError(t, err)
	assert.Equal(t, "/auth/v1/token", capturedPath)
	assert.Equal(t, "grant_type=password", capturedQuery)
}
