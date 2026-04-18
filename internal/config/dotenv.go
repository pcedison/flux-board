package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	if err := loadDotEnvFile(".env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		panic(fmt.Errorf("load .env: %w", err))
	}
}

func loadDotEnvFile(path string) error {
	return godotenv.Load(path)
}
