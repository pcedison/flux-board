package postgres

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"sort"
	"strings"
	"time"

	"flux-board/internal/observability/tracing"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
)

const migrationLockID int64 = 4_247_155_010_001
const postgresTracerScope = "store/postgres"

var RequiredSchemaObjects = []string{
	"schema_migrations",
	"tasks",
	"archived_tasks",
	"users",
	"sessions",
	"auth_audit_logs",
	"idx_tasks_status_sort_order",
	"idx_archived_tasks_archived_at",
	"idx_sessions_expires_at",
	"idx_sessions_username",
	"idx_auth_audit_logs_created_at",
}

var RequiredSchemaConstraints = []string{
	"tasks_status_allowed",
	"tasks_priority_allowed",
	"tasks_due_format",
	"tasks_sort_order_nonnegative",
	"tasks_status_sort_order_unique",
	"archived_tasks_status_allowed",
	"archived_tasks_priority_allowed",
	"archived_tasks_due_format",
	"archived_tasks_sort_order_nonnegative",
}

func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	ctx, span := tracing.StartClientSpan(ctx, postgresTracerScope, "postgres.connect",
		attribute.String("db.system", "postgresql"),
	)
	defer span.End()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		tracing.RecordError(span, err)
		return nil, err
	}
	return pool, nil
}

func InitializeSchema(ctx context.Context, db *pgxpool.Pool, migrationFS fs.FS, migrationPattern, bootstrapUsername, bootstrapPassword string) error {
	ctx, span := tracing.StartClientSpan(ctx, postgresTracerScope, "postgres.initialize_schema",
		attribute.String("db.system", "postgresql"),
		attribute.String("migration.pattern", migrationPattern),
	)
	defer span.End()

	if err := RunMigrations(ctx, db, migrationFS, migrationPattern); err != nil {
		tracing.RecordError(span, err)
		return err
	}
	if err := NewAuthRepository(db).EnsureBootstrapAdmin(ctx, bootstrapUsername, bootstrapPassword); err != nil {
		tracing.RecordError(span, err)
		return err
	}
	return nil
}

func RunMigrations(ctx context.Context, db *pgxpool.Pool, migrationFS fs.FS, migrationPattern string) error {
	ctx, span := tracing.StartClientSpan(ctx, postgresTracerScope, "postgres.run_migrations",
		attribute.String("db.system", "postgresql"),
		attribute.String("migration.pattern", migrationPattern),
	)
	defer span.End()

	conn, err := db.Acquire(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, migrationLockID); err != nil {
		tracing.RecordError(span, err)
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() {
		if _, unlockErr := conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, migrationLockID); unlockErr != nil {
			slog.Default().Error("migration unlock error", slog.Any("err", unlockErr))
		}
	}()

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			checksum   TEXT NOT NULL DEFAULT '',
			applied_at BIGINT NOT NULL
		)
	`); err != nil {
		tracing.RecordError(span, err)
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	if _, err := conn.Exec(ctx, `
		ALTER TABLE schema_migrations
		ADD COLUMN IF NOT EXISTS checksum TEXT NOT NULL DEFAULT ''
	`); err != nil {
		tracing.RecordError(span, err)
		return fmt.Errorf("ensure schema_migrations checksum: %w", err)
	}

	entries, err := fs.Glob(migrationFS, migrationPattern)
	if err != nil {
		tracing.RecordError(span, err)
		return fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(entries)
	span.SetAttributes(attribute.Int("migration.count", len(entries)))

	for _, entry := range entries {
		version := migrationVersion(entry)
		sqlBytes, err := fs.ReadFile(migrationFS, entry)
		if err != nil {
			tracing.RecordError(span, err)
			return fmt.Errorf("read migration %s: %w", entry, err)
		}
		checksum := migrationChecksum(sqlBytes)

		applied, recordedChecksum, err := migrationState(ctx, conn, version)
		if err != nil {
			tracing.RecordError(span, err)
			return err
		}
		if applied {
			if recordedChecksum == "" {
				if _, err := conn.Exec(ctx,
					`UPDATE schema_migrations SET checksum=$1 WHERE version=$2`,
					checksum,
					version,
				); err != nil {
					tracing.RecordError(span, err)
					return fmt.Errorf("backfill migration checksum %s: %w", entry, err)
				}
				continue
			}
			if recordedChecksum != checksum {
				err := fmt.Errorf("migration %s checksum mismatch: recorded=%s current=%s", version, recordedChecksum, checksum)
				tracing.RecordError(span, err)
				return err
			}
			continue
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			tracing.RecordError(span, err)
			return fmt.Errorf("begin migration %s: %w", entry, err)
		}
		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			tx.Rollback(ctx)
			tracing.RecordError(span, err)
			return fmt.Errorf("apply migration %s: %w", entry, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO schema_migrations (version, checksum, applied_at) VALUES ($1, $2, $3)`,
			version,
			checksum,
			time.Now().UnixMilli(),
		); err != nil {
			tx.Rollback(ctx)
			tracing.RecordError(span, err)
			return fmt.Errorf("record migration %s: %w", entry, err)
		}
		if err := tx.Commit(ctx); err != nil {
			tracing.RecordError(span, err)
			return fmt.Errorf("commit migration %s: %w", entry, err)
		}
	}

	if err := validateSchemaBaseline(ctx, conn); err != nil {
		tracing.RecordError(span, err)
		return err
	}
	return nil
}

func CleanupArchivedTasks(ctx context.Context, db *pgxpool.Pool, archiveRetention time.Duration) error {
	ctx, span := tracing.StartClientSpan(ctx, postgresTracerScope, "postgres.cleanup_archived_tasks",
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "DELETE"),
		attribute.String("db.collection", "archived_tasks"),
	)
	defer span.End()

	cutoff := time.Now().Add(-archiveRetention).UnixMilli()
	_, err := db.Exec(ctx, `DELETE FROM archived_tasks WHERE archived_at < $1`, cutoff)
	if err != nil {
		tracing.RecordError(span, err)
	}
	return err
}

func CleanupExpiredSessions(ctx context.Context, db *pgxpool.Pool, now time.Time) error {
	ctx, span := tracing.StartClientSpan(ctx, postgresTracerScope, "postgres.cleanup_expired_sessions",
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "DELETE"),
		attribute.String("db.collection", "sessions"),
	)
	defer span.End()

	_, err := db.Exec(ctx, `DELETE FROM sessions WHERE expires_at < $1 OR revoked_at IS NOT NULL`, now.UnixMilli())
	if err != nil {
		tracing.RecordError(span, err)
	}
	return err
}

func migrationState(ctx context.Context, conn *pgxpool.Conn, version string) (bool, string, error) {
	var exists bool
	var checksum string
	if err := conn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1), COALESCE((SELECT checksum FROM schema_migrations WHERE version=$1), '')`,
		version,
	).Scan(&exists, &checksum); err != nil {
		return false, "", fmt.Errorf("check migration %s: %w", version, err)
	}
	return exists, checksum, nil
}

func migrationChecksum(contents []byte) string {
	sum := sha256.Sum256(contents)
	return fmt.Sprintf("%x", sum[:])
}

func validateSchemaBaseline(ctx context.Context, conn *pgxpool.Conn) error {
	for _, objectName := range RequiredSchemaObjects {
		var resolved string
		if err := conn.QueryRow(ctx,
			`SELECT COALESCE(to_regclass($1)::text, '')`,
			objectName,
		).Scan(&resolved); err != nil {
			return fmt.Errorf("validate schema object %s: %w", objectName, err)
		}
		if resolved == "" {
			return fmt.Errorf("required schema object %s is missing after migrations", objectName)
		}
	}
	for _, constraintName := range RequiredSchemaConstraints {
		var exists bool
		if err := conn.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_constraint c
				JOIN pg_class r ON r.oid = c.conrelid
				JOIN pg_namespace n ON n.oid = r.relnamespace
				WHERE c.conname = $1
				  AND n.nspname = current_schema()
			)
		`, constraintName).Scan(&exists); err != nil {
			return fmt.Errorf("validate schema constraint %s: %w", constraintName, err)
		}
		if !exists {
			return fmt.Errorf("required schema constraint %s is missing after migrations", constraintName)
		}
	}
	return nil
}

func migrationVersion(entry string) string {
	base := path.Base(entry)
	return strings.TrimSuffix(base, path.Ext(base))
}
