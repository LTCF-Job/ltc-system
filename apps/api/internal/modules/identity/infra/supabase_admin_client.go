package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"ltc-system/apps/api/internal/modules/identity/app"
)

// SupabaseAdminClient 是 Supabase Auth Admin API 的 HTTP 用戶端實作。
type SupabaseAdminClient struct {
	baseURL    string
	serviceKey string
	httpClient *http.Client
}

// NewSupabaseAdminClient 建立 SupabaseAdminClient 實例；baseURL 或 serviceKey 為空時 Configured() 回 false。
func NewSupabaseAdminClient(baseURL, serviceKey string, httpClient *http.Client) *SupabaseAdminClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &SupabaseAdminClient{baseURL: strings.TrimRight(baseURL, "/"), serviceKey: serviceKey, httpClient: httpClient}
}

// Configured 回報是否已具備呼叫 Admin API 的必要設定。
func (c *SupabaseAdminClient) Configured() bool {
	return c.baseURL != "" && c.serviceKey != ""
}

func (c *SupabaseAdminClient) do(ctx context.Context, method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("apikey", c.serviceKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("supabase admin request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("supabase admin request returned %d: %s", resp.StatusCode, string(respBody))
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}
	return nil
}

// supabaseUserResponse 反映 Supabase Auth Admin API 的使用者 JSON 形狀。
type supabaseUserResponse struct {
	ID           string         `json:"id"`
	Email        string         `json:"email"`
	Phone        string         `json:"phone"`
	CreatedAt    time.Time      `json:"created_at"`
	LastSignInAt *time.Time     `json:"last_sign_in_at"`
	BannedUntil  string         `json:"banned_until"`
	AppMetadata  map[string]any `json:"app_metadata"`
	UserMetadata map[string]any `json:"user_metadata"`
}

type supabaseListUsersResponse struct {
	Users []supabaseUserResponse `json:"users"`
}

func toAuthUser(u supabaseUserResponse) app.AuthUser {
	id, _ := uuid.Parse(u.ID)
	displayName, _ := u.UserMetadata["display_name"].(string)
	role, _ := u.AppMetadata["role"].(string)
	roleKey, _ := u.AppMetadata["role_key"].(string)
	status := "active"
	if u.BannedUntil != "" {
		status = "inactive"
	}

	var perms map[string]app.ModulePermission
	if raw, ok := u.AppMetadata["custom_permissions"]; ok {
		if b, err := json.Marshal(raw); err == nil {
			_ = json.Unmarshal(b, &perms)
		}
	}

	return app.AuthUser{
		ID:                id,
		Email:             u.Email,
		DisplayName:       displayName,
		Phone:             u.Phone,
		Role:              role,
		RoleKey:           roleKey,
		CustomPermissions: perms,
		Status:            status,
		CreatedAt:         u.CreatedAt,
		LastSignInAt:      u.LastSignInAt,
	}
}

// ListUsers 取得所有使用者帳號。
func (c *SupabaseAdminClient) ListUsers(ctx context.Context) ([]app.AuthUser, error) {
	var resp supabaseListUsersResponse
	if err := c.do(ctx, http.MethodGet, "/auth/v1/admin/users", nil, &resp); err != nil {
		return nil, err
	}
	out := make([]app.AuthUser, 0, len(resp.Users))
	for _, u := range resp.Users {
		out = append(out, toAuthUser(u))
	}
	return out, nil
}

// GetUser 取得單一使用者帳號。
func (c *SupabaseAdminClient) GetUser(ctx context.Context, id uuid.UUID) (*app.AuthUser, error) {
	var resp supabaseUserResponse
	if err := c.do(ctx, http.MethodGet, "/auth/v1/admin/users/"+id.String(), nil, &resp); err != nil {
		return nil, err
	}
	u := toAuthUser(resp)
	return &u, nil
}

// CreateUser 建立新使用者；角色一律寫入 app_metadata，user_metadata 只放非授權的顯示資訊。
func (c *SupabaseAdminClient) CreateUser(ctx context.Context, in app.CreateAuthUserInput) (*app.AuthUser, error) {
	baseRole := in.RoleKey
	body := map[string]any{
		"email":         in.Email,
		"password":      in.Password,
		"email_confirm": true,
		"app_metadata": map[string]any{
			"role":     baseRole,
			"role_key": in.RoleKey,
		},
		"user_metadata": map[string]any{
			"display_name": in.DisplayName,
			"phone":        in.Phone,
		},
	}
	var resp supabaseUserResponse
	if err := c.do(ctx, http.MethodPost, "/auth/v1/admin/users", body, &resp); err != nil {
		return nil, err
	}
	u := toAuthUser(resp)
	return &u, nil
}

// UpdateUser 更新使用者基本資料與角色。
func (c *SupabaseAdminClient) UpdateUser(ctx context.Context, id uuid.UUID, in app.UpdateAuthUserInput) (*app.AuthUser, error) {
	body := map[string]any{}
	userMetadata := map[string]any{}
	if in.DisplayName != nil {
		userMetadata["display_name"] = *in.DisplayName
	}
	if in.Phone != nil {
		userMetadata["phone"] = *in.Phone
	}
	if len(userMetadata) > 0 {
		body["user_metadata"] = userMetadata
	}
	if in.RoleKey != nil {
		body["app_metadata"] = map[string]any{"role": *in.RoleKey, "role_key": *in.RoleKey}
	}
	if in.Status != nil {
		if *in.Status == "inactive" {
			body["ban_duration"] = "876000h"
		} else {
			body["ban_duration"] = "none"
		}
	}

	var resp supabaseUserResponse
	if err := c.do(ctx, http.MethodPut, "/auth/v1/admin/users/"+id.String(), body, &resp); err != nil {
		return nil, err
	}
	u := toAuthUser(resp)
	return &u, nil
}

// DeleteUser 刪除使用者帳號。
func (c *SupabaseAdminClient) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return c.do(ctx, http.MethodDelete, "/auth/v1/admin/users/"+id.String(), nil, nil)
}

// SetCustomPermissions 覆寫使用者個人自訂權限，存於 app_metadata.custom_permissions。
func (c *SupabaseAdminClient) SetCustomPermissions(ctx context.Context, id uuid.UUID, perms map[string]app.ModulePermission) error {
	body := map[string]any{
		"app_metadata": map[string]any{"custom_permissions": perms},
	}
	return c.do(ctx, http.MethodPut, "/auth/v1/admin/users/"+id.String(), body, nil)
}

// VerifyPassword 以帳密向 Supabase 的密碼授權端點驗證憑證是否正確。
func (c *SupabaseAdminClient) VerifyPassword(ctx context.Context, email, password string) error {
	body := map[string]any{"email": email, "password": password}
	return c.do(ctx, http.MethodPost, "/auth/v1/token?grant_type=password", body, nil)
}

// SetPassword 直接設定使用者的新密碼，供驗證舊密碼通過後呼叫。
func (c *SupabaseAdminClient) SetPassword(ctx context.Context, id uuid.UUID, newPassword string) error {
	body := map[string]any{"password": newPassword}
	return c.do(ctx, http.MethodPut, "/auth/v1/admin/users/"+id.String(), body, nil)
}

// CountUsersByRoleKey 統計採用該角色的使用者數。
func (c *SupabaseAdminClient) CountUsersByRoleKey(ctx context.Context, key string) (int, error) {
	if !c.Configured() {
		return 0, nil
	}
	users, err := c.ListUsers(ctx)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, u := range users {
		if u.RoleKey == key {
			count++
		}
	}
	return count, nil
}
