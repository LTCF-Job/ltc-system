package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config 定義系統執行時之全部環境變數設定。
type Config struct {
	Port                        string        `envconfig:"PORT" default:"8080"`
	AppEnv                      string        `envconfig:"APP_ENV" required:"true"`
	DataPlane                   string        `envconfig:"DATA_PLANE" default:"production"`
	DatabaseURL                 string        `envconfig:"DATABASE_URL" default:"postgres://postgres:postgres@localhost:5432/ltc_system?sslmode=disable"`
	DBMaxOpenConns              int           `envconfig:"DB_MAX_OPEN_CONNS" default:"5"`
	DBMaxIdleConns              int           `envconfig:"DB_MAX_IDLE_CONNS" default:"2"`
	EncryptionKeyB64            string        `envconfig:"ENCRYPTION_KEY" default:"MDEwMjAzMDQwNTA2MDcwODAxMDIwMzA0MDUwNjA3MDg="` // 32 bytes base64 for dev
	HMACKeyB64                  string        `envconfig:"HMAC_KEY" default:"MDkwODAwMDcwNjA1MDQwMzA5MDgwMDA3MDYwNTA0MDM="`       // 32 bytes base64 for dev
	SupabaseJWKSURL             string        `envconfig:"SUPABASE_JWKS_URL"`
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

	// AuthMiddleware（internal/platform/auth）以此值比對 JWT 的 data_plane claim，拼字錯誤會讓所有合法憑證被判定跨環境而全部拒絕，需在啟動時就攔下。
	if cfg.DataPlane != "production" && cfg.DataPlane != "demo" {
		return nil, fmt.Errorf("DATA_PLANE must be \"production\" or \"demo\", got %q", cfg.DataPlane)
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

	if cfg.SupabaseURL == "" && cfg.SupabaseProjectRef != "" {
		cfg.SupabaseURL = fmt.Sprintf("https://%s.supabase.co", cfg.SupabaseProjectRef)
	}
	// 金鑰留空時 identity 模組的端點會誠實回 503，不強制擋住啟動——這裡只提醒維運人員。
	if cfg.AppEnv == "production" && cfg.SupabaseServiceRoleKey == "" {
		slog.Warn("SUPABASE_SERVICE_ROLE_KEY is not set; user/role management endpoints will return 503")
	}

	return &cfg, nil
}
