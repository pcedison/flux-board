package main

import (
	"time"

	authservice "flux-board/internal/service/auth"
)

func (a *App) allowLoginAttempt(clientID string) bool {
	return a.authTrackerOrInit().Allow(clientID, time.Now())
}

func (a *App) recordFailedLogin(clientID string) {
	a.authTrackerOrInit().RecordFailure(clientID, time.Now())
}

func (a *App) clearLoginAttempts(clientID string) {
	a.authTrackerOrInit().Clear(clientID)
}

func (a *App) authTrackerOrInit() *authservice.LoginTracker {
	if a.authTracker == nil {
		a.authTracker = authservice.NewLoginTracker()
	}
	return a.authTracker
}
