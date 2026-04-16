package main

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const migrationLockID int64 = 4_247_155_010_001

var requiredSchemaObjects = []string{
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

var requiredSchemaConstraints = []string{
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

func (a *App) initSchema() error {
	if err := runMigrations(context.Background(), a.db); err != nil {
		return err
	}
	return a.ensureBootstrapAdmin(context.Background())
}

func runMigrations(ctx context.Context, db *pgxpool.Pool) error {
	conn, err := db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, migrationLockID); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() {
		if _, unlockErr := conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, migrationLockID); unlockErr != nil {
			log.Printf("migration unlock error: %v", unlockErr)
		}
	}()

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			checksum   TEXT NOT NULL DEFAULT '',
			applied_at BIGINT NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	if _, err := conn.Exec(ctx, `
		ALTER TABLE schema_migrations
		ADD COLUMN IF NOT EXISTS checksum TEXT NOT NULL DEFAULT ''
	`); err != nil {
		return fmt.Errorf("ensure schema_migrations checksum: %w", err)
	}

	entries, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(entries)

	for _, entry := range entries {
		version := migrationVersion(entry)
		sqlBytes, err := migrationFiles.ReadFile(entry)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry, err)
		}
		checksum := migrationChecksum(sqlBytes)

		applied, recordedChecksum, err := migrationState(ctx, conn, version)
		if err != nil {
			return err
		}
		if applied {
			if recordedChecksum == "" {
				if _, err := conn.Exec(ctx,
					`UPDATE schema_migrations SET checksum=$1 WHERE version=$2`,
					checksum,
					version,
				); err != nil {
					return fmt.Errorf("backfill migration checksum %s: %w", entry, err)
				}
				continue
			}
			if recordedChecksum != checksum {
				return fmt.Errorf("migration %s checksum mismatch: recorded=%s current=%s", version, recordedChecksum, checksum)
			}
			continue
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", entry, err)
		}
		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s: %w", entry, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO schema_migrations (version, checksum, applied_at) VALUES ($1, $2, $3)`,
			version,
			checksum,
			time.Now().UnixMilli(),
		); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", entry, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry, err)
		}
	}

	return validateSchemaBaseline(ctx, conn)
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
	for _, objectName := range requiredSchemaObjects {
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
	for _, constraintName := range requiredSchemaConstraints {
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
