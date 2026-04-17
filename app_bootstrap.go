package main

import (
	"context"

	"flux-board/internal/config"
	authservice "flux-board/internal/service/auth"
	storepostgres "flux-board/internal/store/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

func connectDatabase(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	return storepostgres.Connect(ctx, databaseURL)
}

func newApp(cfg config.Config, pool *pgxpool.Pool) *App {
	return &App{
		db:                pool,
		authTracker:       authservice.NewLoginTracker(),
		bootstrapPassword: cfg.AppPassword,
		cookieSecure:      cfg.CookieSecure,
	}
}

func (a *App) bootstrap() error {
	return a.initSchema()
}

func (a *App) ensureBootstrapAdmin(ctx context.Context) error {
	return a.authRepository().EnsureBootstrapAdmin(ctx, bootstrapAdmin, a.bootstrapPassword)
}
