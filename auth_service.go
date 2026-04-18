package main

import authservice "flux-board/internal/service/auth"

var (
	errAuthBlocked        = authservice.ErrBlocked
	errAuthInvalidSession  = authservice.ErrInvalidSession
)

type AuthService = authservice.Service

func (a *App) authService() AuthService {
	if a.authSvc != nil {
		return a.authSvc
	}
	if a.authTracker == nil {
		a.authTracker = authservice.NewLoginTracker()
	}
	return authservice.New(a.authRepository(), authservice.Options{
		PasswordVerifier:     a.passwordVerifier,
		SessionGetter:        a.sessionGetter,
		SessionCreator:       a.sessionCreator,
		SessionDeleter:       a.sessionDeleter,
		AuditRecorder:        a.auditRecorder,
		RequestIDFromContext: requestIDFromContext,
		Tracker:              a.authTracker,
	})
}
