package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL  string
	AppPassword  string
	Port         string
	AppEnv       string
	CookieSecure bool
}

func Load() (Config, error) {
	databaseURL, err := require("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}
	appPassword, err := require("APP_PASSWORD")
	if err != nil {
		return Config{}, err
	}

	appEnv := get("APP_ENV", "production")
	return Config{
		DatabaseURL:  databaseURL,
		AppPassword:  appPassword,
		Port:         get("PORT", "8080"),
		AppEnv:       appEnv,
		CookieSecure: appEnv != "development",
	}, nil
}

func require(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("required env var %s is not set", key)
	}
	return value, nil
}

func get(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
