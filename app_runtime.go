package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

func (a *App) startBackgroundLoops() {
	go a.archiveCleanupLoop()
	go a.sessionCleanupLoop()
}

// archiveCleanupLoop deletes archived tasks older than archiveRetention every hour.
func (a *App) archiveCleanupLoop() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-archiveRetention).UnixMilli()
		if _, err := a.db.Exec(context.Background(),
			`DELETE FROM archived_tasks WHERE archived_at < $1`, cutoff); err != nil {
			log.Printf("archive cleanup error: %v", err)
		}
	}
}

func (a *App) sessionCleanupLoop() {
	ticker := time.NewTicker(sessionCleanupTicker)
	defer ticker.Stop()

	for range ticker.C {
		if _, err := a.db.Exec(context.Background(),
			`DELETE FROM sessions WHERE expires_at < $1 OR revoked_at IS NOT NULL`, time.Now().UnixMilli()); err != nil {
			log.Printf("session cleanup error: %v", err)
		}
	}
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
