package main

import (
	"context"
	"errors"
	"net/http"
	"time"
)

const readinessTimeout = 2 * time.Second

func (a *App) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	setProbeHeaders(w)
	jsonResp(w, map[string]string{"status": "ok"})
}

func (a *App) handleReadyz(w http.ResponseWriter, r *http.Request) {
	setProbeHeaders(w)
	if err := a.checkReadiness(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "not ready")
		return
	}
	jsonResp(w, map[string]string{"status": "ready"})
}

func setProbeHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}

func (a *App) checkReadiness(ctx context.Context) error {
	if a.readinessChecker != nil {
		return a.readinessChecker(ctx)
	}
	if a.db == nil {
		return errors.New("db not configured")
	}

	readinessCtx, cancel := context.WithTimeout(ctx, readinessTimeout)
	defer cancel()
	return a.db.Ping(readinessCtx)
}
