package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	storepostgres "flux-board/internal/store/postgres"
	transporthttp "flux-board/internal/transport/http"
)

func (a *App) startBackgroundLoops(ctx context.Context) {
	go a.runCleanupLoop(ctx, time.Hour, "archive cleanup", a.cleanupArchivedTasks)
	go a.runCleanupLoop(ctx, sessionCleanupTicker, "session cleanup", a.cleanupExpiredSessions)
}

func (a *App) runCleanupLoop(ctx context.Context, interval time.Duration, label string, cleanup func(context.Context) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := cleanup(cleanupCtx)
			cancel()
			if err != nil && ctx.Err() == nil && !errors.Is(err, context.Canceled) {
				slog.Default().Error("background cleanup failed", slog.String("cleanup", label), slog.Any("err", err))
			}
		}
	}
}

func (a *App) cleanupArchivedTasks(ctx context.Context) error {
	return storepostgres.CleanupArchivedTasks(ctx, a.db)
}

func (a *App) cleanupExpiredSessions(ctx context.Context) error {
	return storepostgres.CleanupExpiredSessions(ctx, a.db, time.Now())
}

func (a *App) securityHeaders(next http.Handler) http.Handler {
	return transporthttp.SecurityHeaders(next)
}
