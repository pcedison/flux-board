package config

import "testing"

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
