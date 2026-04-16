package main

import (
	"context"
	"errors"
	"time"

	"flux-board/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func connectDatabase(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, databaseURL)
}

func newApp(cfg config.Config, pool *pgxpool.Pool) *App {
	return &App{
		db:                pool,
		bootstrapPassword: cfg.AppPassword,
		cookieSecure:      cfg.CookieSecure,
		loginAttempts:     make(map[string]loginAttemptState),
	}
}

func (a *App) bootstrap() error {
	return a.initSchema()
}

func (a *App) ensureBootstrapAdmin(ctx context.Context) error {
	var existingHash string
	err := a.db.QueryRow(ctx,
		`SELECT password_hash FROM users WHERE username=$1`, bootstrapAdmin).Scan(&existingHash)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(a.bootstrapPassword), bcrypt.DefaultCost)
		if hashErr != nil {
			return hashErr
		}

		now := time.Now().UnixMilli()
		tx, txErr := a.db.Begin(ctx)
		if txErr != nil {
			return txErr
		}
		defer tx.Rollback(ctx)

		if _, txErr = tx.Exec(ctx, `
			INSERT INTO users (username, password_hash, role, created_at, updated_at)
			VALUES ($1, $2, 'admin', $3, $3)
		`, bootstrapAdmin, string(hash), now); txErr != nil {
			return txErr
		}

		return tx.Commit(ctx)
	default:
		return err
	}
}
