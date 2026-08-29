package config

import "testing"

func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "")
	t.Setenv("SUPABASE_JWKS_URL", "")
	t.Setenv("ALLOWED_ORIGINS", "")
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
	clearEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("ALLOWED_ORIGINS", "https://example.vercel.app")
	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected error when APP_ENV=production without SUPABASE_JWKS_URL, got nil")
	}
}

func TestLoadFromEnv_ProductionRequiresAllowedOrigins(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("SUPABASE_JWKS_URL", "https://example.supabase.co/auth/v1/.well-known/jwks.json")
	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected error when APP_ENV=production without ALLOWED_ORIGINS, got nil")
	}
}

func TestLoadFromEnv_ProductionWithJWKSSucceeds(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("SUPABASE_JWKS_URL", "https://example.supabase.co/auth/v1/.well-known/jwks.json")
	t.Setenv("ALLOWED_ORIGINS", "https://example.vercel.app")
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
