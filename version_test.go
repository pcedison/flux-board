package main

import "testing"

func TestAppVersionPrefersEnvironmentOverride(t *testing.T) {
	previousBuildVersion := buildVersion
	buildVersion = "0.1.4"
	t.Cleanup(func() {
		buildVersion = previousBuildVersion
	})
	t.Setenv("APP_VERSION", "9.9.9")

	if got := appVersion(); got != "9.9.9" {
		t.Fatalf("expected APP_VERSION override, got %q", got)
	}
}

func TestAppVersionFallsBackToBuildVersion(t *testing.T) {
	previousBuildVersion := buildVersion
	buildVersion = "0.1.4"
	t.Cleanup(func() {
		buildVersion = previousBuildVersion
	})
	t.Setenv("APP_VERSION", " ")

	if got := appVersion(); got != "0.1.4" {
		t.Fatalf("expected buildVersion fallback, got %q", got)
	}
}

func TestAppVersionFallsBackToDevWhenUnset(t *testing.T) {
	previousBuildVersion := buildVersion
	buildVersion = " "
	t.Cleanup(func() {
		buildVersion = previousBuildVersion
	})
	t.Setenv("APP_VERSION", "")

	if got := appVersion(); got != "dev" {
		t.Fatalf("expected dev fallback, got %q", got)
	}
}
