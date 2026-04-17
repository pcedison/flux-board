package postgres

import (
	"context"
	"errors"
	"time"

	"flux-board/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthRepository struct {
	db *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) BootstrapPasswordHash(ctx context.Context, username string) (string, error) {
	var hash string
	if err := r.db.QueryRow(ctx,
		`SELECT password_hash FROM users WHERE username=$1`, username).Scan(&hash); err != nil {
		return "", err
	}
	return hash, nil
}

func (r *AuthRepository) EnsureBootstrapAdmin(ctx context.Context, username, password string) error {
	var existingHash string
	err := r.db.QueryRow(ctx,
		`SELECT password_hash FROM users WHERE username=$1`, username).Scan(&existingHash)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if hashErr != nil {
			return hashErr
		}

		now := time.Now().UnixMilli()
		tx, txErr := r.db.Begin(ctx)
		if txErr != nil {
			return txErr
		}
		defer tx.Rollback(ctx)

		if _, txErr = tx.Exec(ctx, `
			INSERT INTO users (username, password_hash, role, created_at, updated_at)
			VALUES ($1, $2, 'admin', $3, $3)
		`, username, string(hash), now); txErr != nil {
			return txErr
		}

		return tx.Commit(ctx)
	default:
		return err
	}
}

func (r *AuthRepository) GetActiveSession(ctx context.Context, token string) (domain.Session, error) {
	var session domain.Session
	var expiresAt int64
	if err := r.db.QueryRow(ctx, `
		SELECT username, expires_at
		FROM sessions
		WHERE token=$1 AND revoked_at IS NULL AND expires_at > $2
	`, token, time.Now().UnixMilli()).Scan(&session.Username, &expiresAt); err != nil {
		return domain.Session{}, err
	}

	session.Token = token
	session.ExpiresAt = time.UnixMilli(expiresAt)
	return session, nil
}

func (r *AuthRepository) CreateSession(ctx context.Context, token, username, clientIP string, expiresAt time.Time) error {
	now := time.Now().UnixMilli()
	_, err := r.db.Exec(ctx, `
		INSERT INTO sessions (token, username, created_at, expires_at, revoked_at, last_seen_at, client_ip)
		VALUES ($1, $2, $3, $4, NULL, $3, $5)
	`, token, username, now, expiresAt.UnixMilli(), clientIP)
	return err
}

func (r *AuthRepository) DeleteSession(ctx context.Context, token string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM sessions WHERE token=$1`, token)
	return err
}

func (r *AuthRepository) RecordAuthEvent(ctx context.Context, event domain.AuthAuditEvent) error {
	if event.CreatedAt == 0 {
		event.CreatedAt = time.Now().UnixMilli()
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO auth_audit_logs (username, event_type, outcome, client_ip, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, event.Username, event.EventType, event.Outcome, event.ClientIP, event.Details, event.CreatedAt)
	return err
}
