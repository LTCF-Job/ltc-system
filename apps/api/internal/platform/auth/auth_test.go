package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ltc-system/apps/api/internal/config"
)

func performAuthRequest(t *testing.T, h gin.HandlerFunc, authHeader string) (*httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	c.Request = req
	h(c)
	return w, c
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	cfg := &config.Config{AppEnv: "production"}
	w, _ := performAuthRequest(t, AuthMiddleware(cfg), "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_MalformedHeader(t *testing.T) {
	cfg := &config.Config{AppEnv: "production"}
	w, _ := performAuthRequest(t, AuthMiddleware(cfg), "Token abc")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_LocalMockHeaderBypassesJWT(t *testing.T) {
	cfg := &config.Config{AppEnv: "local"}
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil)
	req.Header.Set("X-Mock-Role", "admin")
	c.Request = req

	AuthMiddleware(cfg)(c)

	assert.False(t, c.IsAborted())
	assert.Equal(t, "admin", c.GetString(ContextKeyActorRole))
}

func TestAuthMiddleware_LocalMockJWTPrefix(t *testing.T) {
	cfg := &config.Config{AppEnv: "local"}
	w, c := performAuthRequest(t, AuthMiddleware(cfg), "Bearer mock_jwt_admin_token")
	assert.False(t, w.Code == http.StatusUnauthorized)
	assert.Equal(t, "admin", c.GetString(ContextKeyActorRole))
}

// 未設定 SUPABASE_JWKS_URL 時，正式環境必須拒絕請求而不是信任未驗證的 JWT。
func TestAuthMiddleware_ProductionWithoutJWKS_Rejects(t *testing.T) {
	cfg := &config.Config{AppEnv: "production", SupabaseJWKSURL: ""}
	forged := forgeUnsignedToken(t, "admin")
	w, _ := performAuthRequest(t, AuthMiddleware(cfg), "Bearer "+forged)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// 偽造（未簽章驗證通過）的 token 在設定 JWKS 後必須被拒絕，這是本次修正要堵住的漏洞。
func TestAuthMiddleware_RejectsForgedTokenWhenJWKSConfigured(t *testing.T) {
	srv, _ := newTestJWKSServer(t)
	defer srv.Close()

	cfg := &config.Config{AppEnv: "production", SupabaseJWKSURL: srv.URL}
	forged := forgeUnsignedToken(t, "admin")
	w, _ := performAuthRequest(t, AuthMiddleware(cfg), "Bearer "+forged)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_AcceptsValidSignedToken(t *testing.T) {
	srv, key := newTestJWKSServer(t)
	defer srv.Close()

	cfg := &config.Config{AppEnv: "production", SupabaseJWKSURL: srv.URL}
	actorID := uuid.New()
	signed := signTestToken(t, key, "test-kid", actorID.String(), "staff")

	w, c := performAuthRequest(t, AuthMiddleware(cfg), "Bearer "+signed)
	require.Equal(t, http.StatusOK, w.Code)
	assert.False(t, c.IsAborted())
	assert.Equal(t, "staff", c.GetString(ContextKeyActorRole))
	assert.Equal(t, actorID, GetActorID(c))
}

// forgeUnsignedToken 產生「alg 有效但簽章任意偽造」的 token，模擬攻擊者在無簽章驗證時可捏造的憑證。
func forgeUnsignedToken(t *testing.T, role string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub":  uuid.NewString(),
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	forgedKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	signed, err := token.SignedString(forgedKey)
	require.NoError(t, err)
	return signed
}

func signTestToken(t *testing.T, key *rsa.PrivateKey, kid, sub, role string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub":  sub,
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	token.Header["kid"] = kid
	signed, err := token.SignedString(key)
	require.NoError(t, err)
	return signed
}

// newTestJWKSServer 啟動一個回傳單一 RSA 公鑰 JWKS 文件的測試伺服器。
func newTestJWKSServer(t *testing.T) (*httptest.Server, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": "test-kid",
				"n":   base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(bigIntToBytes(key.PublicKey.E)),
			},
		},
	}
	body, err := json.Marshal(jwks)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	return srv, key
}

func bigIntToBytes(e int) []byte {
	// RSA 公開指數常為 65537 (0x010001)，以最小位元組表示編碼進 JWK。
	if e == 65537 {
		return []byte{0x01, 0x00, 0x01}
	}
	b := make([]byte, 0, 4)
	for e > 0 {
		b = append([]byte{byte(e & 0xff)}, b...)
		e >>= 8
	}
	return b
}
