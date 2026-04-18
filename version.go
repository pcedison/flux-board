package main

import (
	"os"
	"strings"
)

// buildVersion is injected at build time for release artifacts and container images.
var buildVersion = "dev"

func appVersion() string {
	if version := strings.TrimSpace(os.Getenv("APP_VERSION")); version != "" {
		return version
	}
	if version := strings.TrimSpace(buildVersion); version != "" {
		return version
	}
	return "dev"
}
