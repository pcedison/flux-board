package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
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
				log.Printf("%s error: %v", label, err)
			}
		}
	}
}

func (a *App) cleanupArchivedTasks(ctx context.Context) error {
	cutoff := time.Now().Add(-archiveRetention).UnixMilli()
	_, err := a.db.Exec(ctx, `DELETE FROM archived_tasks WHERE archived_at < $1`, cutoff)
	return err
}

func (a *App) cleanupExpiredSessions(ctx context.Context) error {
	_, err := a.db.Exec(ctx,
		`DELETE FROM sessions WHERE expires_at < $1 OR revoked_at IS NOT NULL`, time.Now().UnixMilli())
	return err
}

func (a *App) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Pragma", "no-cache")
		}
		next.ServeHTTP(w, r)
	})
}
