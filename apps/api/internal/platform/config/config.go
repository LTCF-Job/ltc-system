package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// 這兩把金鑰以字面值寫在原始碼與 struct tag 的 default 中，僅供本機開發與測試使用；
// struct tag 只接受字面量，故 default 保留字面值，常數用於正式環境的沿用比對。
const (
	devDefaultEncryptionKeyB64 = "MDEwMjAzMDQwNTA2MDcwODAxMDIwMzA0MDUwNjA3MDg="
	devDefaultHMACKeyB64       = "MDkwODAwMDcwNjA1MDQwMzA5MDgwMDA3MDYwNTA0MDM="
)

// Config 定義系統執行時之全部環境變數設定。
type Config struct {
	Port                        string        `envconfig:"PORT" default:"8080"`
	AppEnv                      string        `envconfig:"APP_ENV" required:"true"`
	DatabaseURL                 string        `envconfig:"DATABASE_URL" default:"postgres://postgres:postgres@localhost:5432/ltc_system?sslmode=disable"`
	DBMaxOpenConns              int           `envconfig:"DB_MAX_OPEN_CONNS" default:"5"`
	DBMaxIdleConns              int           `envconfig:"DB_MAX_IDLE_CONNS" default:"2"`
	EncryptionKeyB64            string        `envconfig:"ENCRYPTION_KEY" default:"MDEwMjAzMDQwNTA2MDcwODAxMDIwMzA0MDUwNjA3MDg="` // 32 bytes base64 for dev
	HMACKeyB64                  string        `envconfig:"HMAC_KEY" default:"MDkwODAwMDcwNjA1MDQwMzA5MDgwMDA3MDYwNTA0MDM="`       // 32 bytes base64 for dev
	SupabaseJWKSURL             string        `envconfig:"SUPABASE_JWKS_URL"`
	SupabaseJWTIssuer           string        `envconfig:"SUPABASE_JWT_ISSUER"`
	SupabaseProjectRef          string        `envconfig:"SUPABASE_PROJECT_REF"`
	AllowedOrigins              string        `envconfig:"ALLOWED_ORIGINS"`
	StorageBucket               string        `envconfig:"STORAGE_BUCKET" default:"ltc-exports"`
	StorageSignedURLTTL         time.Duration `envconfig:"STORAGE_SIGNED_URL_TTL" default:"24h"`
	ResendAPIKey                string        `envconfig:"RESEND_API_KEY"`
	NotifyFrom                  string        `envconfig:"NOTIFY_FROM" default:"noreply@ltc.example.com"`
	SentryDSN                   string        `envconfig:"SENTRY_DSN"`
	LogLevel                    string        `envconfig:"LOG_LEVEL" default:"info"`
	GovernmentHolidayAPITimeout time.Duration `envconfig:"GOVERNMENT_HOLIDAY_API_TIMEOUT" default:"10s"`
	SupabaseURL                 string        `envconfig:"SUPABASE_URL"`
	SupabaseServiceRoleKey      string        `envconfig:"SUPABASE_SERVICE_ROLE_KEY"`
	SupabaseAdminTimeout        time.Duration `envconfig:"SUPABASE_ADMIN_API_TIMEOUT" default:"10s"`
	DefaultAdminEmail           string        `envconfig:"DEFAULT_ADMIN_EMAIL"`
	DefaultAdminPassword        string        `envconfig:"DEFAULT_ADMIN_PASSWORD"`

	// 解析後的金鑰 bytes
	EncryptionKey []byte `ignored:"true"`
	HMACKey       []byte `ignored:"true"`
}

// LoadFromEnv 從系統環境變數載入並驗證設定值。
func LoadFromEnv() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process env config: %w", err)
	}

	// APP_ENV 決定 AuthMiddleware 是否放行 mock 憑證，禁止靜默預設值以避免正式環境誤留開發後門
	if cfg.AppEnv != "local" && cfg.AppEnv != "production" {
		return nil, fmt.Errorf("APP_ENV must be explicitly set to \"local\" or \"production\", got %q", cfg.AppEnv)
	}

	// 正式環境缺少 JWKS 時 AuthMiddleware 會對每個請求回應 500，改為啟動時直接拒絕啟動
	if cfg.AppEnv == "production" && cfg.SupabaseJWKSURL == "" {
		return nil, errors.New("SUPABASE_JWKS_URL is required when APP_ENV=production")
	}

	// 正式環境未設定白名單時 CORS 會退回全放行，改為啟動時直接拒絕啟動
	if cfg.AppEnv == "production" && cfg.AllowedOrigins == "" {
		return nil, errors.New("ALLOWED_ORIGINS is required when APP_ENV=production")
	}

	encKey, err := base64.StdEncoding.DecodeString(cfg.EncryptionKeyB64)
	if err != nil || len(encKey) != 32 {
		return nil, errors.New("ENCRYPTION_KEY must be a valid 32-byte base64 string")
	}
	cfg.EncryptionKey = encKey

	hmacKey, err := base64.StdEncoding.DecodeString(cfg.HMACKeyB64)
	if err != nil || len(hmacKey) != 32 {
		return nil, errors.New("HMAC_KEY must be a valid 32-byte base64 string")
	}
	cfg.HMACKey = hmacKey

	// 確保加密金鑰與 HMAC 金鑰不同
	if cfg.EncryptionKeyB64 == cfg.HMACKeyB64 {
		return nil, errors.New("ENCRYPTION_KEY and HMAC_KEY must not be identical")
	}

	// dev default 金鑰公開在原始碼中，正式環境沿用等同任何人都能解開身分證等欄位的密文、並離線反查 HMAC 索引
	if cfg.AppEnv == "production" && cfg.EncryptionKeyB64 == devDefaultEncryptionKeyB64 {
		return nil, errors.New("ENCRYPTION_KEY must not use the development default when APP_ENV=production; the default is published in source code, so encrypted national IDs could be decrypted by anyone")
	}
	if cfg.AppEnv == "production" && cfg.HMACKeyB64 == devDefaultHMACKeyB64 {
		return nil, errors.New("HMAC_KEY must not use the development default when APP_ENV=production; the default is published in source code, so blind-index values could be reversed offline")
	}

	if cfg.SupabaseURL == "" && cfg.SupabaseProjectRef != "" {
		cfg.SupabaseURL = fmt.Sprintf("https://%s.supabase.co", cfg.SupabaseProjectRef)
	}

	if cfg.SupabaseJWTIssuer == "" && cfg.SupabaseProjectRef != "" {
		cfg.SupabaseJWTIssuer = fmt.Sprintf("https://%s.supabase.co/auth/v1", cfg.SupabaseProjectRef)
	}

	// issuer 是 AuthMiddleware 驗證 JWT iss claim 的唯一依據，正式環境缺值等於不驗發行者
	if cfg.AppEnv == "production" && cfg.SupabaseJWTIssuer == "" {
		return nil, errors.New("SUPABASE_JWT_ISSUER (or SUPABASE_PROJECT_REF to derive it) is required when APP_ENV=production")
	}

	// 缺金鑰時 userCustomPermissionResolver 會 fail-open，使用者個人層級的權限覆蓋靜默失效，被降權者回復為角色矩陣的完整權限
	if cfg.AppEnv == "production" && cfg.SupabaseServiceRoleKey == "" {
		return nil, errors.New("SUPABASE_SERVICE_ROLE_KEY is required when APP_ENV=production; without it user-level custom permissions silently stop applying and down-scoped users fall back to full role-matrix permissions")
	}

	return &cfg, nil
}
