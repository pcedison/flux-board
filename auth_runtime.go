package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"time"
)

func (a *App) allowLoginAttempt(clientID string) bool {
	now := time.Now()
	a.loginMu.Lock()
	defer a.loginMu.Unlock()

	state := a.loginAttempts[clientID]
	return !now.Before(state.BlockedUntil)
}

func (a *App) recordFailedLogin(clientID string) {
	now := time.Now()
	a.loginMu.Lock()
	defer a.loginMu.Unlock()

	state := a.loginAttempts[clientID]
	if state.WindowStart.IsZero() || now.Sub(state.WindowStart) > loginWindow {
		state = loginAttemptState{WindowStart: now}
	}

	state.Failures++
	if state.Failures >= maxLoginFailures {
		state.BlockedUntil = now.Add(loginBlockDuration)
		state.Failures = 0
		state.WindowStart = now
	}

	a.loginAttempts[clientID] = state
}

func (a *App) clearLoginAttempts(clientID string) {
	a.loginMu.Lock()
	defer a.loginMu.Unlock()
	delete(a.loginAttempts, clientID)
}

func newToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("generate token: %v", err)
	}
	return hex.EncodeToString(bytes)
}
