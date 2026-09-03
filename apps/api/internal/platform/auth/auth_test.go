package auth

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
	"ltc-system/apps/api/internal/platform/config"
)

// testIssuer 對應 config.SupabaseJWTIssuer，與測試 token 的 iss claim 一致。
const testIssuer = "https://test-project.supabase.co/auth/v1"

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
	w, _ := performAuthRequest(t, Middleware(cfg), "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_MalformedHeader(t *testing.T) {
	cfg := &config.Config{AppEnv: "production"}
	w, _ := performAuthRequest(t, Middleware(cfg), "Token abc")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// X-Mock-Role 曾是本機用的身分後門，已移除；帶著它但沒有 Authorization 仍必須被拒絕。
func TestAuthMiddleware_LocalMockHeaderNoLongerBypassesJWT(t *testing.T) {
	cfg := &config.Config{AppEnv: "local"}
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil)
	req.Header.Set("X-Mock-Role", "admin")
	c.Request = req

	Middleware(cfg)(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, c.GetString(ContextKeyActorRole))
}

// 本機無 JWKS 時的 ParseUnverified 降級不得在解析失敗時放行成 admin。
func TestAuthMiddleware_LocalUnparsableTokenRejected(t *testing.T) {
	cfg := &config.Config{AppEnv: "local"}
	w, c := performAuthRequest(t, Middleware(cfg), "Bearer not-a-jwt")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, c.GetString(ContextKeyActorRole))
}

func TestAuthMiddleware_LocalMockJWTPrefix(t *testing.T) {
	cfg := &config.Config{AppEnv: "local"}
	w, c := performAuthRequest(t, Middleware(cfg), "Bearer mock_jwt_admin_token")
	assert.False(t, w.Code == http.StatusUnauthorized)
	assert.Equal(t, "admin", c.GetString(ContextKeyActorRole))
}

func TestAuthMiddleware_ProductionRejectsMockJWTPrefix(t *testing.T) {
	srv, _ := newTestJWKSServer(t)
	defer srv.Close()

	cfg := &config.Config{AppEnv: "production", SupabaseJWKSURL: srv.URL}
	w, _ := performAuthRequest(t, Middleware(cfg), "Bearer mock_jwt_admin_token")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_ProductionIgnoresMockHeader(t *testing.T) {
	cfg := &config.Config{AppEnv: "production"}
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil)
	req.Header.Set("X-Mock-Role", "admin")
	c.Request = req

	Middleware(cfg)(c)

	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, c.GetString(ContextKeyActorRole))
}

// 未設定 SUPABASE_JWKS_URL 時，正式環境必須拒絕請求而不是信任未驗證的 JWT。
func TestAuthMiddleware_ProductionWithoutJWKS_Rejects(t *testing.T) {
	cfg := &config.Config{AppEnv: "production", SupabaseJWKSURL: ""}
	forged := forgeUnsignedToken(t, "admin")
	w, _ := performAuthRequest(t, Middleware(cfg), "Bearer "+forged)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// 偽造（未簽章驗證通過）的 token 在設定 JWKS 後必須被拒絕，這是本次修正要堵住的漏洞。
func TestAuthMiddleware_RejectsForgedTokenWhenJWKSConfigured(t *testing.T) {
	srv, _ := newTestJWKSServer(t)
	defer srv.Close()

	cfg := &config.Config{AppEnv: "production", SupabaseJWKSURL: srv.URL, SupabaseJWTIssuer: testIssuer}
	forged := forgeUnsignedToken(t, "admin")
	w, _ := performAuthRequest(t, Middleware(cfg), "Bearer "+forged)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_AcceptsValidSignedToken(t *testing.T) {
	srv, key := newTestJWKSServer(t)
	defer srv.Close()

	cfg := &config.Config{AppEnv: "production", SupabaseJWKSURL: srv.URL, SupabaseJWTIssuer: testIssuer}
	actorID := uuid.New()
	signed := signTestToken(t, key, "test-kid", actorID.String(), "staff", "")

	w, c := performAuthRequest(t, Middleware(cfg), "Bearer "+signed)
	require.Equal(t, http.StatusOK, w.Code)
	assert.False(t, c.IsAborted())
	assert.Equal(t, "staff", c.GetString(ContextKeyActorRole))
	assert.Equal(t, actorID, GetActorID(c))
}

// TestAuthMiddleware_ProductionRejectsDemoToken 驗證正式 API 拒絕帶 app_metadata.data_plane = "demo" 的 JWT。
func TestAuthMiddleware_ProductionRejectsDemoToken(t *testing.T) {
	srv, key := newTestJWKSServer(t)
	defer srv.Close()

	cfg := &config.Config{AppEnv: "production", SupabaseJWKSURL: srv.URL, SupabaseJWTIssuer: testIssuer, DataPlane: DataPlaneProduction}
	signed := signTestToken(t, key, "test-kid", uuid.NewString(), "staff", DataPlaneDemo)

	w, c := performAuthRequest(t, Middleware(cfg), "Bearer "+signed)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.True(t, c.IsAborted())
}

// TestAuthMiddleware_DemoRejectsProductionToken 驗證 Demo API 拒絕沒有 data_plane 或 data_plane = "production" 的 JWT。
func TestAuthMiddleware_DemoRejectsProductionToken(t *testing.T) {
	srv, key := newTestJWKSServer(t)
	defer srv.Close()

	cfg := &config.Config{AppEnv: "production", SupabaseJWKSURL: srv.URL, SupabaseJWTIssuer: testIssuer, DataPlane: DataPlaneDemo}
	signed := signTestToken(t, key, "test-kid", uuid.NewString(), "staff", "")

	w, c := performAuthRequest(t, Middleware(cfg), "Bearer "+signed)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.True(t, c.IsAborted())
}

// TestAuthMiddleware_DemoAcceptsDemoToken 驗證 Demo API 放行帶有 app_metadata.data_plane = "demo" 的 JWT。
func TestAuthMiddleware_DemoAcceptsDemoToken(t *testing.T) {
	srv, key := newTestJWKSServer(t)
	defer srv.Close()

	cfg := &config.Config{AppEnv: "production", SupabaseJWKSURL: srv.URL, SupabaseJWTIssuer: testIssuer, DataPlane: DataPlaneDemo}
	signed := signTestToken(t, key, "test-kid", uuid.NewString(), "staff", DataPlaneDemo)

	w, c := performAuthRequest(t, Middleware(cfg), "Bearer "+signed)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, DataPlaneDemo, GetActorDataPlane(c))
}

// TestAuthMiddleware_IgnoresUserMetadataRole 是本次提權修補的核心迴歸測試：使用者可自行寫入的
// user_metadata.role 不得影響授權角色。
func TestAuthMiddleware_IgnoresUserMetadataRole(t *testing.T) {
	srv, key := newTestJWKSServer(t)
	defer srv.Close()

	cfg := &config.Config{AppEnv: "production", SupabaseJWKSURL: srv.URL, SupabaseJWTIssuer: testIssuer}
	signed := signClaims(t, key, "test-kid", jwt.MapClaims{
		"sub":           uuid.NewString(),
		"iss":           testIssuer,
		"aud":           "authenticated",
		"exp":           time.Now().Add(time.Hour).Unix(),
		"user_metadata": map[string]interface{}{"role": "admin"},
	})

	w, c := performAuthRequest(t, Middleware(cfg), "Bearer "+signed)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "viewer", c.GetString(ContextKeyActorRole))
}

// TestAuthMiddleware_IgnoresTopLevelRoleClaim 驗證 Supabase 頂層 role claim（Postgres role）不會被當成業務角色。
func TestAuthMiddleware_IgnoresTopLevelRoleClaim(t *testing.T) {
	srv, key := newTestJWKSServer(t)
	defer srv.Close()

	cfg := &config.Config{AppEnv: "production", SupabaseJWKSURL: srv.URL, SupabaseJWTIssuer: testIssuer}
	signed := signClaims(t, key, "test-kid", jwt.MapClaims{
		"sub":  uuid.NewString(),
		"iss":  testIssuer,
		"aud":  "authenticated",
		"role": "service_role",
		"exp":  time.Now().Add(time.Hour).Unix(),
	})

	w, c := performAuthRequest(t, Middleware(cfg), "Bearer "+signed)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "viewer", c.GetString(ContextKeyActorRole))
}

// TestAuthMiddleware_PreservesAppMetadataRoles 驗證 app_metadata.role 原樣帶入 Context，
// 包含管理員自建的角色 key——這裡不可加角色白名單，否則自訂角色使用者會被靜默降級為 viewer。
func TestAuthMiddleware_PreservesAppMetadataRoles(t *testing.T) {
	srv, key := newTestJWKSServer(t)
	defer srv.Close()

	cfg := &config.Config{AppEnv: "production", SupabaseJWKSURL: srv.URL, SupabaseJWTIssuer: testIssuer}
	for _, role := range []string{"admin", "dispatcher", "staff", "driver", "viewer", "dispatcher_1"} {
		signed := signTestToken(t, key, "test-kid", uuid.NewString(), role, "")
		w, c := performAuthRequest(t, Middleware(cfg), "Bearer "+signed)
		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, role, c.GetString(ContextKeyActorRole))
	}
}

// TestAuthMiddleware_MissingAppMetadataRoleFallsBackToViewer 驗證 app_metadata 未帶 role 時採最小權限預設。
func TestAuthMiddleware_MissingAppMetadataRoleFallsBackToViewer(t *testing.T) {
	srv, key := newTestJWKSServer(t)
	defer srv.Close()

	cfg := &config.Config{AppEnv: "production", SupabaseJWKSURL: srv.URL, SupabaseJWTIssuer: testIssuer}
	signed := signTestToken(t, key, "test-kid", uuid.NewString(), "", "")

	w, c := performAuthRequest(t, Middleware(cfg), "Bearer "+signed)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "viewer", c.GetString(ContextKeyActorRole))
}

// TestAuthMiddleware_RejectsWrongIssuer 驗證他人 Supabase 專案簽出的 token 會被拒絕。
func TestAuthMiddleware_RejectsWrongIssuer(t *testing.T) {
	srv, key := newTestJWKSServer(t)
	defer srv.Close()

	cfg := &config.Config{AppEnv: "production", SupabaseJWKSURL: srv.URL, SupabaseJWTIssuer: testIssuer}
	signed := signClaims(t, key, "test-kid", jwt.MapClaims{
		"sub": uuid.NewString(),
		"iss": "https://attacker-project.supabase.co/auth/v1",
		"aud": "authenticated",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	w, _ := performAuthRequest(t, Middleware(cfg), "Bearer "+signed)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAuthMiddleware_RejectsWrongAudience 驗證非 authenticated 受眾（如 service_role 憑證）會被拒絕。
func TestAuthMiddleware_RejectsWrongAudience(t *testing.T) {
	srv, key := newTestJWKSServer(t)
	defer srv.Close()

	cfg := &config.Config{AppEnv: "production", SupabaseJWKSURL: srv.URL, SupabaseJWTIssuer: testIssuer}
	signed := signClaims(t, key, "test-kid", jwt.MapClaims{
		"sub": uuid.NewString(),
		"iss": testIssuer,
		"aud": "service_role",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	w, _ := performAuthRequest(t, Middleware(cfg), "Bearer "+signed)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAuthMiddleware_RejectsNonUUIDSubject 驗證 sub 無法解析為 UUID 時直接拒絕，避免以 uuid.Nil 身分放行。
func TestAuthMiddleware_RejectsNonUUIDSubject(t *testing.T) {
	srv, key := newTestJWKSServer(t)
	defer srv.Close()

	cfg := &config.Config{AppEnv: "production", SupabaseJWKSURL: srv.URL, SupabaseJWTIssuer: testIssuer}
	signed := signTestToken(t, key, "test-kid", "not-a-uuid", "staff", "")

	w, c := performAuthRequest(t, Middleware(cfg), "Bearer "+signed)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.True(t, c.IsAborted())
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

// signTestToken 產生一張合法的 Supabase 使用者 token，角色寫在 app_metadata。
func signTestToken(t *testing.T, key *rsa.PrivateKey, kid, sub, role, dataPlane string) string {
	t.Helper()
	appMetadata := map[string]interface{}{}
	if role != "" {
		appMetadata["role"] = role
	}
	if dataPlane != "" {
		appMetadata["data_plane"] = dataPlane
	}
	return signClaims(t, key, kid, jwt.MapClaims{
		"sub":          sub,
		"iss":          testIssuer,
		"aud":          "authenticated",
		"exp":          time.Now().Add(time.Hour).Unix(),
		"app_metadata": appMetadata,
	})
}

// signClaims 以測試金鑰簽出指定 claims，供需要偽造個別欄位的案例使用。
func signClaims(t *testing.T, key *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
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
