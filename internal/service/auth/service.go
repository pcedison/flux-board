package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"flux-board/internal/domain"
	"flux-board/internal/observability/tracing"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/crypto/bcrypt"
)

const (
	BootstrapAdmin     = "admin"
	SessionDuration    = 14 * 24 * time.Hour
	LoginWindow        = 15 * time.Minute
	MaxLoginFailures   = 10
	LoginBlockDuration = 15 * time.Minute
)

var (
	ErrBlocked         = errors.New("auth blocked")
	ErrInvalidPassword = errors.New("invalid password")
	ErrInvalidSession  = errors.New("invalid session")
	ErrSetupRequired   = errors.New("setup required")
)

const tracerScope = "service/auth"

type Service interface {
	Authenticate(context.Context, string, string) (LoginResult, error)
	Logout(context.Context, string, string) error
	SessionFromToken(context.Context, string, string) (domain.Session, error)
}

type LoginResult struct {
	Token     string
	Username  string
	ExpiresAt time.Time
}

type Options struct {
	Clock                func() time.Time
	TokenGenerator       func() string
	PasswordVerifier     func(context.Context, string) (bool, error)
	SessionGetter        func(context.Context, string) (domain.Session, error)
	SessionCreator       func(context.Context, string, string, string, time.Time) error
	SessionDeleter       func(context.Context, string) error
	AuditRecorder        func(context.Context, domain.AuthAuditEvent) error
	RequestIDFromContext func(context.Context) string
	Tracker              *LoginTracker
}

type loginAttemptState struct {
	WindowStart  time.Time
	Failures     int
	BlockedUntil time.Time
}

type LoginTracker struct {
	mu       sync.Mutex
	attempts map[string]loginAttemptState
}

type service struct {
	repo    domain.AuthRepository
	options Options
	tracker *LoginTracker
}

func New(repo domain.AuthRepository, opts Options) Service {
	tracker := opts.Tracker
	if tracker == nil {
		tracker = NewLoginTracker()
	}
	return service{
		repo:    repo,
		options: opts,
		tracker: tracker,
	}
}

func NewLoginTracker() *LoginTracker {
	return &LoginTracker{
		attempts: make(map[string]loginAttemptState),
	}
}

func (t *LoginTracker) Allow(clientID string, now time.Time) bool {
	if t == nil {
		return true
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	state := t.attempts[clientID]
	return !now.Before(state.BlockedUntil)
}

func (t *LoginTracker) RecordFailure(clientID string, now time.Time) {
	if t == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	state := t.attempts[clientID]
	if state.WindowStart.IsZero() || now.Sub(state.WindowStart) > LoginWindow {
		state = loginAttemptState{WindowStart: now}
	}

	state.Failures++
	if state.Failures >= MaxLoginFailures {
		state.BlockedUntil = now.Add(LoginBlockDuration)
		state.Failures = 0
		state.WindowStart = now
	}

	t.attempts[clientID] = state
}

func (t *LoginTracker) Clear(clientID string) {
	if t == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.attempts, clientID)
}

func (s service) Authenticate(ctx context.Context, password, clientIP string) (LoginResult, error) {
	ctx, span := tracing.StartInternalSpan(ctx, tracerScope, "auth.authenticate",
		attribute.String("auth.username", BootstrapAdmin),
		attribute.Bool("auth.client_present", clientIP != ""),
	)
	defer span.End()

	now := s.now()
	if !s.tracker.Allow(clientIP, now) {
		span.SetAttributes(attribute.String("auth.outcome", "blocked"))
		s.recordAuthEvent(ctx, domain.AuthAuditEvent{
			EventType: "login",
			Outcome:   "blocked",
			ClientIP:  clientIP,
			Details:   "too many failed attempts",
		})
		return LoginResult{}, ErrBlocked
	}

	match, err := s.verifyLoginPassword(ctx, password)
	if err != nil {
		span.SetAttributes(attribute.String("auth.outcome", "error"))
		tracing.RecordError(span, err)
		s.recordAuthEvent(ctx, domain.AuthAuditEvent{
			Username:  BootstrapAdmin,
			EventType: "login",
			Outcome:   "error",
			ClientIP:  clientIP,
			Details:   "password verification failed",
		})
		return LoginResult{}, err
	}
	if !match {
		span.SetAttributes(attribute.String("auth.outcome", "failed"))
		s.tracker.RecordFailure(clientIP, now)
		s.recordAuthEvent(ctx, domain.AuthAuditEvent{
			Username:  BootstrapAdmin,
			EventType: "login",
			Outcome:   "failed",
			ClientIP:  clientIP,
		})
		return LoginResult{}, ErrInvalidPassword
	}

	s.tracker.Clear(clientIP)

	token, err := s.newToken()
	if err != nil {
		span.SetAttributes(attribute.String("auth.outcome", "error"))
		tracing.RecordError(span, err)
		s.recordAuthEvent(ctx, domain.AuthAuditEvent{
			Username:  BootstrapAdmin,
			EventType: "login",
			Outcome:   "error",
			ClientIP:  clientIP,
			Details:   "token generation failed",
		})
		return LoginResult{}, err
	}
	expiresAt := now.Add(SessionDuration)
	if err := s.createSession(ctx, token, BootstrapAdmin, clientIP, expiresAt); err != nil {
		span.SetAttributes(attribute.String("auth.outcome", "error"))
		tracing.RecordError(span, err)
		s.recordAuthEvent(ctx, domain.AuthAuditEvent{
			Username:  BootstrapAdmin,
			EventType: "login",
			Outcome:   "error",
			ClientIP:  clientIP,
			Details:   "session create failed",
		})
		return LoginResult{}, err
	}

	s.recordAuthEvent(ctx, domain.AuthAuditEvent{
		Username:  BootstrapAdmin,
		EventType: "login",
		Outcome:   "success",
		ClientIP:  clientIP,
	})
	span.SetAttributes(
		attribute.String("auth.outcome", "success"),
		attribute.String("auth.session.username", BootstrapAdmin),
	)

	return LoginResult{
		Token:     token,
		Username:  BootstrapAdmin,
		ExpiresAt: expiresAt,
	}, nil
}

func (s service) Logout(ctx context.Context, token, clientIP string) error {
	ctx, span := tracing.StartInternalSpan(ctx, tracerScope, "auth.logout",
		attribute.String("auth.username", BootstrapAdmin),
		attribute.Bool("auth.token_present", token != ""),
	)
	defer span.End()

	if err := s.deleteSession(ctx, token); err != nil {
		span.SetAttributes(attribute.String("auth.outcome", "error"))
		tracing.RecordError(span, err)
		s.recordAuthEvent(ctx, domain.AuthAuditEvent{
			EventType: "logout",
			Outcome:   "error",
			ClientIP:  clientIP,
			Details:   "session delete failed",
		})
		return err
	}

	s.recordAuthEvent(ctx, domain.AuthAuditEvent{
		Username:  BootstrapAdmin,
		EventType: "logout",
		Outcome:   "success",
		ClientIP:  clientIP,
	})
	span.SetAttributes(attribute.String("auth.outcome", "success"))
	return nil
}

func (s service) SessionFromToken(ctx context.Context, token, clientIP string) (domain.Session, error) {
	ctx, span := tracing.StartInternalSpan(ctx, tracerScope, "auth.session_from_token",
		attribute.Bool("auth.token_present", token != ""),
	)
	defer span.End()

	session, err := s.getActiveSession(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			span.SetAttributes(attribute.String("auth.outcome", "invalid"))
			s.recordAuthEvent(ctx, domain.AuthAuditEvent{
				EventType: "session",
				Outcome:   "invalid",
				ClientIP:  clientIP,
				Details:   "session lookup failed",
			})
			return domain.Session{}, ErrInvalidSession
		}

		span.SetAttributes(attribute.String("auth.outcome", "error"))
		tracing.RecordError(span, err)
		s.recordAuthEvent(ctx, domain.AuthAuditEvent{
			EventType: "session",
			Outcome:   "error",
			ClientIP:  clientIP,
			Details:   "session lookup error",
		})
		return domain.Session{}, err
	}

	span.SetAttributes(
		attribute.String("auth.outcome", "success"),
		attribute.String("auth.session.username", session.Username),
	)
	return session, nil
}

func (s service) verifyLoginPassword(ctx context.Context, given string) (bool, error) {
	if s.options.PasswordVerifier != nil {
		return s.options.PasswordVerifier(ctx, given)
	}
	if s.repo == nil {
		return false, errors.New("auth repository not configured")
	}

	hash, err := s.repo.BootstrapPasswordHash(ctx, BootstrapAdmin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrSetupRequired
		}
		return false, err
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(given)) == nil, nil
}

func (s service) getActiveSession(ctx context.Context, token string) (domain.Session, error) {
	if s.options.SessionGetter != nil {
		return s.options.SessionGetter(ctx, token)
	}
	if s.repo == nil {
		return domain.Session{}, errors.New("auth repository not configured")
	}
	return s.repo.GetActiveSession(ctx, token)
}

func (s service) createSession(ctx context.Context, token, username, clientIP string, expiresAt time.Time) error {
	if s.options.SessionCreator != nil {
		return s.options.SessionCreator(ctx, token, username, clientIP, expiresAt)
	}
	if s.repo == nil {
		return errors.New("auth repository not configured")
	}
	return s.repo.CreateSession(ctx, token, username, clientIP, expiresAt)
}

func (s service) deleteSession(ctx context.Context, token string) error {
	if s.options.SessionDeleter != nil {
		return s.options.SessionDeleter(ctx, token)
	}
	if s.repo == nil {
		return errors.New("auth repository not configured")
	}
	return s.repo.DeleteSession(ctx, token)
}

func (s service) recordAuthEvent(ctx context.Context, event domain.AuthAuditEvent) {
	event.CreatedAt = s.now().UnixMilli()
	if event.RequestID == "" && s.options.RequestIDFromContext != nil {
		event.RequestID = s.options.RequestIDFromContext(ctx)
	}

	if s.options.AuditRecorder != nil {
		if err := s.options.AuditRecorder(ctx, event); err != nil {
			slog.Default().Error("auth audit recorder error", slog.String("request_id", event.RequestID), slog.Any("err", err))
		}
		return
	}
	if s.repo == nil {
		return
	}
	if err := s.repo.RecordAuthEvent(ctx, event); err != nil {
		slog.Default().Error("auth audit insert error", slog.String("request_id", event.RequestID), slog.Any("err", err))
	}
}

func (s service) now() time.Time {
	if s.options.Clock != nil {
		return s.options.Clock()
	}
	return time.Now()
}

func (s service) newToken() (string, error) {
	if s.options.TokenGenerator != nil {
		return s.options.TokenGenerator(), nil
	}

	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
