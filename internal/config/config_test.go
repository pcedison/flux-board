package config

import (
	"os"
	"path/filepath"
	"testing"
)

func unsetEnvForTest(t *testing.T, keys ...string) {
	t.Helper()

	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("Unsetenv(%q) returned error: %v", key, err)
		}

		if ok {
			t.Cleanup(func() {
				_ = os.Setenv(key, value)
			})
			continue
		}

		t.Cleanup(func() {
			_ = os.Unsetenv(key)
		})
	}
}

func TestLoadDotEnvFileLoadsValuesWithoutOverridingExistingEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	if err := os.WriteFile(path, []byte("DATABASE_URL=postgres://from-file\nAPP_PASSWORD=from-file\nAPP_ENV=development\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	unsetEnvForTest(t, "DATABASE_URL", "APP_ENV")
	t.Setenv("APP_PASSWORD", "existing")

	if err := loadDotEnvFile(path); err != nil {
		t.Fatalf("loadDotEnvFile returned error: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.DatabaseURL != "postgres://from-file" {
		t.Fatalf("expected DATABASE_URL from .env, got %q", cfg.DatabaseURL)
	}
	if cfg.AppPassword != "existing" {
		t.Fatalf("expected existing APP_PASSWORD to be preserved, got %q", cfg.AppPassword)
	}
	if cfg.AppEnv != "development" {
		t.Fatalf("expected APP_ENV from .env, got %q", cfg.AppEnv)
	}
}

func TestLoadDefaultsProductionCookieSecure(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("APP_PASSWORD", "secret")
	t.Setenv("APP_ENV", "")
	t.Setenv("PORT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %q", cfg.Port)
	}
	if !cfg.CookieSecure {
		t.Fatal("expected cookieSecure to default to true outside development")
	}
}

func TestLoadDevelopmentDisablesSecureCookie(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("APP_PASSWORD", "secret")
	t.Setenv("APP_ENV", "development")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.CookieSecure {
		t.Fatal("expected cookieSecure to be false in development")
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("APP_PASSWORD", "secret")
	t.Setenv("APP_ENV", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected missing DATABASE_URL to fail")
	}
}
