package main

import (
	"context"
	"errors"
	"net/http"
	"time"
)

const readinessTimeout = 2 * time.Second

func (a *App) handleHealthz(w http.ResponseWriter, r *http.Request) {
	a.transportHandler().HandleHealthz(w, r)
}

func (a *App) handleReadyz(w http.ResponseWriter, r *http.Request) {
	a.transportHandler().HandleReadyz(w, r)
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
