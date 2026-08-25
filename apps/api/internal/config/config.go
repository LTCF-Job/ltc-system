package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config 定義系統執行時之全部環境變數設定。
type Config struct {
	Port                string        `envconfig:"PORT" default:"8080"`
	AppEnv              string        `envconfig:"APP_ENV" default:"local"`
	DatabaseURL         string        `envconfig:"DATABASE_URL" default:"postgres://postgres:postgres@localhost:5432/ltc_system?sslmode=disable"`
	DBMaxOpenConns      int           `envconfig:"DB_MAX_OPEN_CONNS" default:"5"`
	DBMaxIdleConns      int           `envconfig:"DB_MAX_IDLE_CONNS" default:"2"`
	EncryptionKeyB64    string        `envconfig:"ENCRYPTION_KEY" default:"MDEwMjAzMDQwNTA2MDcwODAxMDIwMzA0MDUwNjA3MDg="` // 32 bytes base64 for dev
	HMACKeyB64          string        `envconfig:"HMAC_KEY" default:"MDkwODAwMDcwNjA1MDQwMzA5MDgwMDA3MDYwNTA0MDM="`       // 32 bytes base64 for dev
	SupabaseJWKSURL     string        `envconfig:"SUPABASE_JWKS_URL"`
	SupabaseProjectRef  string        `envconfig:"SUPABASE_PROJECT_REF"`
	StorageBucket       string        `envconfig:"STORAGE_BUCKET" default:"ltc-exports"`
	StorageSignedURLTTL time.Duration `envconfig:"STORAGE_SIGNED_URL_TTL" default:"24h"`
	GoogleSAJSON        string        `envconfig:"GOOGLE_SA_JSON"`
	ResendAPIKey        string        `envconfig:"RESEND_API_KEY"`
	NotifyFrom          string        `envconfig:"NOTIFY_FROM" default:"noreply@ltc.example.com"`
	SentryDSN           string        `envconfig:"SENTRY_DSN"`
	LogLevel            string        `envconfig:"LOG_LEVEL" default:"info"`

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

	return &cfg, nil
}
