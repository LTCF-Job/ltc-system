package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/modules/identity/app"
	"ltc-system/apps/api/internal/platform/auth"
)

// stubAdminProvider 是 transport 層測試用的最小 app.AdminIdentityProvider 實作。
type stubAdminProvider struct {
	verifyErr         error
	setPasswordCalled bool
}

func (s *stubAdminProvider) Configured() bool { return true }
func (s *stubAdminProvider) ListUsers(ctx context.Context) ([]app.AuthUser, error) {
	return nil, nil
}
func (s *stubAdminProvider) GetUser(ctx context.Context, id uuid.UUID) (*app.AuthUser, error) {
	return nil, nil
}
func (s *stubAdminProvider) CreateUser(ctx context.Context, in app.CreateAuthUserInput) (*app.AuthUser, error) {
	return nil, nil
}
func (s *stubAdminProvider) UpdateUser(ctx context.Context, id uuid.UUID, in app.UpdateAuthUserInput) (*app.AuthUser, error) {
	return nil, nil
}
func (s *stubAdminProvider) DeleteUser(ctx context.Context, id uuid.UUID) error { return nil }
func (s *stubAdminProvider) SetCustomPermissions(ctx context.Context, id uuid.UUID, perms map[string]app.ModulePermission) error {
	return nil
}
func (s *stubAdminProvider) VerifyPassword(ctx context.Context, email, password string) error {
	return s.verifyErr
}
func (s *stubAdminProvider) SetPassword(ctx context.Context, id uuid.UUID, newPassword string) error {
	s.setPasswordCalled = true
	return nil
}

type stubAuditWriter struct{}

func (stubAuditWriter) Write(ctx context.Context, e app.AuditEntry) error { return nil }

func newUserTestRouter(svc *app.UserService, actorID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(auth.ContextKeyActorID, actorID)
		c.Set(auth.ContextKeyActorRole, "admin")
		c.Set(auth.ContextKeyUserEmail, "actor@example.com")
		c.Next()
	})
	h := NewIdentityHandler(svc)
	r.POST("/auth/change-password", h.ChangeSelfPassword)
	r.DELETE("/users/:id", h.DeleteUser)
	return r
}

// 改密碼打錯舊密碼時，畫面必須顯示「目前密碼不正確」，且不得回 401（會觸發前端全域登出）。
func TestChangeSelfPassword_WrongOldPassword_Returns422NotUnauthenticated(t *testing.T) {
	admin := &stubAdminProvider{verifyErr: assert.AnError}
	svc := app.NewUserService(admin, nil, nil)
	r := newUserTestRouter(svc, uuid.New())

	body := `{"oldPassword":"wrong-old-password","newPassword":"newpass1"}`
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/auth/change-password", strings.NewReader(body)))

	assert.NotEqual(t, http.StatusUnauthorized, w.Code, "舊密碼錯誤不得回 401，否則前端會觸發全域登出")

	var resp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "目前密碼不正確", resp.Error.Message)
	assert.False(t, admin.setPasswordCalled)
}

// 刪除自己的帳號時，訊息必須說明具體原因，而不是共用的「權限不足，無法執行此操作」。
func TestDeleteUser_CannotDeleteSelf_ReturnsSpecificReason(t *testing.T) {
	admin := &stubAdminProvider{}
	svc := app.NewUserService(admin, nil, stubAuditWriter{})
	actorID := uuid.New()
	r := newUserTestRouter(svc, actorID)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/users/"+actorID.String(), nil))

	assert.Equal(t, http.StatusForbidden, w.Code)
	var resp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "不可刪除自己的帳號", resp.Error.Message)
	assert.NotEqual(t, "權限不足，無法執行此操作", resp.Error.Message)
}

func TestChangeSelfPassword_CorrectOldPassword_Succeeds(t *testing.T) {
	admin := &stubAdminProvider{}
	svc := app.NewUserService(admin, nil, stubAuditWriter{})
	r := newUserTestRouter(svc, uuid.New())

	body := `{"oldPassword":"old-password","newPassword":"newpass1"}`
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/auth/change-password", strings.NewReader(body)))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, admin.setPasswordCalled)
}
