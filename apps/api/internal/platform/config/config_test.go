package config

import (
	"reflect"
	"testing"
)

const (
	testEncryptionKeyB64 = "YWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWFhYWE="
	testHMACKeyB64       = "YmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmJiYmI="
)

func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "")
	t.Setenv("SUPABASE_JWKS_URL", "")
	t.Setenv("SUPABASE_JWT_ISSUER", "")
	t.Setenv("SUPABASE_PROJECT_REF", "")
	t.Setenv("SUPABASE_SERVICE_ROLE_KEY", "")
	t.Setenv("ENCRYPTION_KEY", devDefaultEncryptionKeyB64)
	t.Setenv("HMAC_KEY", devDefaultHMACKeyB64)
	t.Setenv("ALLOWED_ORIGINS", "")
}

// setProductionEnv 設定一組通過所有 production 驗證的環境變數，供各測試單獨拿掉其中一項。
func setProductionEnv(t *testing.T) {
	t.Helper()
	clearEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("SUPABASE_JWKS_URL", "https://example.supabase.co/auth/v1/.well-known/jwks.json")
	t.Setenv("SUPABASE_JWT_ISSUER", "https://example.supabase.co/auth/v1")
	t.Setenv("SUPABASE_SERVICE_ROLE_KEY", "service-role-key")
	t.Setenv("ENCRYPTION_KEY", testEncryptionKeyB64)
	t.Setenv("HMAC_KEY", testHMACKeyB64)
	t.Setenv("ALLOWED_ORIGINS", "https://example.vercel.app")
}

func TestLoadFromEnv_RequiresAppEnv(t *testing.T) {
	clearEnv(t)
	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected error when APP_ENV is unset, got nil")
	}
}

func TestLoadFromEnv_RejectsUnknownAppEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "staging")
	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected error for unrecognized APP_ENV value, got nil")
	}
}

func TestLoadFromEnv_ProductionRequiresJWKS(t *testing.T) {
	setProductionEnv(t)
	t.Setenv("SUPABASE_JWKS_URL", "")
	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected error when APP_ENV=production without SUPABASE_JWKS_URL, got nil")
	}
}

func TestLoadFromEnv_ProductionRequiresAllowedOrigins(t *testing.T) {
	setProductionEnv(t)
	t.Setenv("ALLOWED_ORIGINS", "")
	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected error when APP_ENV=production without ALLOWED_ORIGINS, got nil")
	}
}

func TestLoadFromEnv_ProductionRequiresServiceRoleKey(t *testing.T) {
	setProductionEnv(t)
	t.Setenv("SUPABASE_SERVICE_ROLE_KEY", "")
	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected error when APP_ENV=production without SUPABASE_SERVICE_ROLE_KEY, got nil")
	}
}

func TestLoadFromEnv_ProductionRejectsDevDefaultEncryptionKey(t *testing.T) {
	setProductionEnv(t)
	t.Setenv("ENCRYPTION_KEY", devDefaultEncryptionKeyB64)
	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected error when APP_ENV=production reuses the development default ENCRYPTION_KEY, got nil")
	}
}

func TestLoadFromEnv_ProductionRejectsDevDefaultHMACKey(t *testing.T) {
	setProductionEnv(t)
	t.Setenv("HMAC_KEY", devDefaultHMACKeyB64)
	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected error when APP_ENV=production reuses the development default HMAC_KEY, got nil")
	}
}

func TestLoadFromEnv_ProductionRequiresJWTIssuer(t *testing.T) {
	setProductionEnv(t)
	t.Setenv("SUPABASE_JWT_ISSUER", "")
	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected error when APP_ENV=production without SUPABASE_JWT_ISSUER and SUPABASE_PROJECT_REF, got nil")
	}
}

func TestLoadFromEnv_DerivesJWTIssuerFromProjectRef(t *testing.T) {
	setProductionEnv(t)
	t.Setenv("SUPABASE_JWT_ISSUER", "")
	t.Setenv("SUPABASE_PROJECT_REF", "abcdefg")
	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.SupabaseJWTIssuer != "https://abcdefg.supabase.co/auth/v1" {
		t.Fatalf("unexpected derived issuer: %q", cfg.SupabaseJWTIssuer)
	}
}

func TestLoadFromEnv_ExplicitJWTIssuerWins(t *testing.T) {
	setProductionEnv(t)
	t.Setenv("SUPABASE_PROJECT_REF", "abcdefg")
	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.SupabaseJWTIssuer != "https://example.supabase.co/auth/v1" {
		t.Fatalf("expected explicit issuer to be kept, got %q", cfg.SupabaseJWTIssuer)
	}
}

func TestLoadFromEnv_ProductionWithJWKSSucceeds(t *testing.T) {
	setProductionEnv(t)
	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.AppEnv != "production" {
		t.Fatalf("expected AppEnv=production, got %q", cfg.AppEnv)
	}
}

func TestLoadFromEnv_LocalDoesNotRequireJWKS(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "local")
	if _, err := LoadFromEnv(); err != nil {
		t.Fatalf("expected no error for local without JWKS, got %v", err)
	}
}

// local 環境沿用原始碼內建的 dev default 金鑰、且未設定 issuer 與 service role key 時仍應正常啟動。
func TestLoadFromEnv_LocalAllowsDevDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "local")
	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.EncryptionKeyB64 != devDefaultEncryptionKeyB64 || cfg.HMACKeyB64 != devDefaultHMACKeyB64 {
		t.Fatal("expected local to fall back to the development default keys")
	}
	if cfg.SupabaseJWTIssuer != "" || cfg.SupabaseServiceRoleKey != "" {
		t.Fatal("expected issuer and service role key to stay empty for local")
	}
}

// struct tag 的 default 只能寫字面量，與常數脫鉤後 production 的沿用檢查會靜默失效。
func TestDevDefaultConstantsMatchStructTags(t *testing.T) {
	typ := reflect.TypeOf(Config{})
	for field, want := range map[string]string{
		"EncryptionKeyB64": devDefaultEncryptionKeyB64,
		"HMACKeyB64":       devDefaultHMACKeyB64,
	} {
		f, ok := typ.FieldByName(field)
		if !ok {
			t.Fatalf("field %s not found", field)
		}
		if got := f.Tag.Get("default"); got != want {
			t.Fatalf("%s default tag %q does not match constant %q", field, got, want)
		}
	}
}
