package main

import (
	"os"
	"strings"
)

func appVersion() string {
	if version := strings.TrimSpace(os.Getenv("APP_VERSION")); version != "" {
		return version
	}
	return "dev"
}
