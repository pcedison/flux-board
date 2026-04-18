package postgres

import (
	"context"
	"errors"
	"time"

	"flux-board/internal/domain"
	"flux-board/internal/observability/tracing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/crypto/bcrypt"
)

const authRepositoryTracerScope = "store/postgres/auth"

type AuthRepository struct {
	db *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) BootstrapPasswordHash(ctx context.Context, username string) (string, error) {
	ctx, span := tracing.StartClientSpan(ctx, authRepositoryTracerScope, "postgres.auth.bootstrap_password_hash",
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.collection", "users"),
		attribute.String("auth.username", username),
	)
	defer span.End()

	var hash string
	if err := r.db.QueryRow(ctx,
		`SELECT password_hash FROM users WHERE username=$1`, username).Scan(&hash); err != nil {
		tracing.RecordError(span, err)
		return "", err
	}
	return hash, nil
}

func (r *AuthRepository) EnsureBootstrapAdmin(ctx context.Context, username, password string) error {
	ctx, span := tracing.StartClientSpan(ctx, authRepositoryTracerScope, "postgres.auth.ensure_bootstrap_admin",
		attribute.String("db.system", "postgresql"),
		attribute.String("db.collection", "users"),
		attribute.String("auth.username", username),
	)
	defer span.End()

	var existingHash string
	err := r.db.QueryRow(ctx,
		`SELECT password_hash FROM users WHERE username=$1`, username).Scan(&existingHash)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		if password == "" {
			return nil
		}
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if hashErr != nil {
			tracing.RecordError(span, hashErr)
			return hashErr
		}

		now := time.Now().UnixMilli()
		tx, txErr := r.db.Begin(ctx)
		if txErr != nil {
			tracing.RecordError(span, txErr)
			return txErr
		}
		defer func() {
			_ = tx.Rollback(ctx)
		}()

		if _, txErr = tx.Exec(ctx, `
			INSERT INTO users (username, password_hash, role, created_at, updated_at)
			VALUES ($1, $2, 'admin', $3, $3)
		`, username, string(hash), now); txErr != nil {
			tracing.RecordError(span, txErr)
			return txErr
		}

		if err := tx.Commit(ctx); err != nil {
			tracing.RecordError(span, err)
			return err
		}
		return nil
	default:
		tracing.RecordError(span, err)
		return err
	}
}

func (r *AuthRepository) BootstrapAdminExists(ctx context.Context, username string) (bool, error) {
	ctx, span := tracing.StartClientSpan(ctx, authRepositoryTracerScope, "postgres.auth.bootstrap_admin_exists",
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.collection", "users"),
		attribute.String("auth.username", username),
	)
	defer span.End()

	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)`, username).Scan(&exists); err != nil {
		tracing.RecordError(span, err)
		return false, err
	}
	return exists, nil
}

func (r *AuthRepository) UpdatePasswordHash(ctx context.Context, username, passwordHash string, updatedAt int64) error {
	ctx, span := tracing.StartClientSpan(ctx, authRepositoryTracerScope, "postgres.auth.update_password_hash",
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "UPDATE"),
		attribute.String("db.collection", "users"),
		attribute.String("auth.username", username),
	)
	defer span.End()

	tag, err := r.db.Exec(ctx, `
		UPDATE users
		SET password_hash=$1, updated_at=$2
		WHERE username=$3
	`, passwordHash, updatedAt, username)
	if err != nil {
		tracing.RecordError(span, err)
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *AuthRepository) GetActiveSession(ctx context.Context, token string) (domain.Session, error) {
	ctx, span := tracing.StartClientSpan(ctx, authRepositoryTracerScope, "postgres.auth.get_active_session",
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.collection", "sessions"),
		attribute.Bool("auth.token_present", token != ""),
	)
	defer span.End()

	var session domain.Session
	var expiresAt int64
	now := time.Now().UnixMilli()
	if err := r.db.QueryRow(ctx, `
		UPDATE sessions
		SET last_seen_at=$2
		WHERE token=$1 AND revoked_at IS NULL AND expires_at > $2
		RETURNING username, expires_at
	`, token, now).Scan(&session.Username, &expiresAt); err != nil {
		tracing.RecordError(span, err)
		return domain.Session{}, err
	}

	session.Token = token
	session.ExpiresAt = time.UnixMilli(expiresAt)
	return session, nil
}

func (r *AuthRepository) CreateSession(ctx context.Context, token, username, clientIP string, expiresAt time.Time) error {
	ctx, span := tracing.StartClientSpan(ctx, authRepositoryTracerScope, "postgres.auth.create_session",
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.collection", "sessions"),
		attribute.String("auth.username", username),
		attribute.Bool("auth.token_present", token != ""),
	)
	defer span.End()

	now := time.Now().UnixMilli()
	_, err := r.db.Exec(ctx, `
		INSERT INTO sessions (token, username, created_at, expires_at, revoked_at, last_seen_at, client_ip)
		VALUES ($1, $2, $3, $4, NULL, $3, $5)
	`, token, username, now, expiresAt.UnixMilli(), clientIP)
	if err != nil {
		tracing.RecordError(span, err)
	}
	return err
}

func (r *AuthRepository) DeleteSession(ctx context.Context, token string) error {
	ctx, span := tracing.StartClientSpan(ctx, authRepositoryTracerScope, "postgres.auth.delete_session",
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "DELETE"),
		attribute.String("db.collection", "sessions"),
		attribute.Bool("auth.token_present", token != ""),
	)
	defer span.End()

	_, err := r.db.Exec(ctx, `DELETE FROM sessions WHERE token=$1`, token)
	if err != nil {
		tracing.RecordError(span, err)
	}
	return err
}

func (r *AuthRepository) ListSessions(ctx context.Context, username string) ([]domain.SessionInfo, error) {
	ctx, span := tracing.StartClientSpan(ctx, authRepositoryTracerScope, "postgres.auth.list_sessions",
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.collection", "sessions"),
		attribute.String("auth.username", username),
	)
	defer span.End()

	rows, err := r.db.Query(ctx, `
		SELECT token, created_at, expires_at, last_seen_at, client_ip
		FROM sessions
		WHERE username=$1
		ORDER BY last_seen_at DESC, created_at DESC, token
	`, username)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}
	defer rows.Close()

	sessions := make([]domain.SessionInfo, 0)
	for rows.Next() {
		var session domain.SessionInfo
		if err := rows.Scan(
			&session.Token,
			&session.CreatedAt,
			&session.ExpiresAt,
			&session.LastSeenAt,
			&session.ClientIP,
		); err != nil {
			tracing.RecordError(span, err)
			return nil, err
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}
	return sessions, nil
}

func (r *AuthRepository) DeleteSessionsExcept(ctx context.Context, username string, keepTokens []string) error {
	ctx, span := tracing.StartClientSpan(ctx, authRepositoryTracerScope, "postgres.auth.delete_sessions_except",
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "DELETE"),
		attribute.String("db.collection", "sessions"),
		attribute.String("auth.username", username),
	)
	defer span.End()

	var err error
	if len(keepTokens) == 0 {
		_, err = r.db.Exec(ctx, `DELETE FROM sessions WHERE username=$1`, username)
	} else {
		_, err = r.db.Exec(ctx, `
			DELETE FROM sessions
			WHERE username=$1 AND token <> ALL($2::text[])
		`, username, keepTokens)
	}
	if err != nil {
		tracing.RecordError(span, err)
	}
	return err
}

func (r *AuthRepository) RecordAuthEvent(ctx context.Context, event domain.AuthAuditEvent) error {
	ctx, span := tracing.StartClientSpan(ctx, authRepositoryTracerScope, "postgres.auth.record_auth_event",
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.collection", "auth_audit_logs"),
		attribute.String("auth.event_type", event.EventType),
		attribute.String("auth.outcome", event.Outcome),
	)
	defer span.End()

	if event.CreatedAt == 0 {
		event.CreatedAt = time.Now().UnixMilli()
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO auth_audit_logs (username, event_type, outcome, client_ip, details, request_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, event.Username, event.EventType, event.Outcome, event.ClientIP, event.Details, event.RequestID, event.CreatedAt)
	if err != nil {
		tracing.RecordError(span, err)
	}
	return err
}
