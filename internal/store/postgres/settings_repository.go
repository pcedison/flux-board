package postgres

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"flux-board/internal/domain"
	"flux-board/internal/observability/tracing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
)

const (
	settingsRepositoryTracerScope = "store/postgres/settings"
	archiveRetentionSettingKey    = "archive_retention_days"
)

type SettingsRepository struct {
	db *pgxpool.Pool
}

func NewSettingsRepository(db *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{db: db}
}

func (r *SettingsRepository) BootstrapAdminExists(ctx context.Context, username string) (bool, error) {
	return NewAuthRepository(r.db).BootstrapAdminExists(ctx, username)
}

func (r *SettingsRepository) UpdatePasswordHash(ctx context.Context, username, passwordHash string, updatedAt int64) error {
	return NewAuthRepository(r.db).UpdatePasswordHash(ctx, username, passwordHash, updatedAt)
}

func (r *SettingsRepository) ListSessions(ctx context.Context, username string) ([]domain.SessionInfo, error) {
	return NewAuthRepository(r.db).ListSessions(ctx, username)
}

func (r *SettingsRepository) DeleteSessionsExcept(ctx context.Context, username string, keepTokens []string) error {
	return NewAuthRepository(r.db).DeleteSessionsExcept(ctx, username, keepTokens)
}

func (r *SettingsRepository) GetArchiveRetentionDays(ctx context.Context) (*int, error) {
	ctx, span := tracing.StartClientSpan(ctx, settingsRepositoryTracerScope, "postgres.settings.get_archive_retention_days",
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.collection", "app_settings"),
	)
	defer span.End()

	var value string
	err := r.db.QueryRow(ctx, `
		SELECT value
		FROM app_settings
		WHERE key=$1
	`, archiveRetentionSettingKey).Scan(&value)
	if err == nil {
		parsed, parseErr := strconv.Atoi(strings.TrimSpace(value))
		if parseErr != nil {
			tracing.RecordError(span, parseErr)
			return nil, parseErr
		}
		return &parsed, nil
	}
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	tracing.RecordError(span, err)
	return nil, err
}

func (r *SettingsRepository) SetArchiveRetentionDays(ctx context.Context, days *int, updatedAt int64) error {
	ctx, span := tracing.StartClientSpan(ctx, settingsRepositoryTracerScope, "postgres.settings.set_archive_retention_days",
		attribute.String("db.system", "postgresql"),
		attribute.String("db.collection", "app_settings"),
	)
	defer span.End()

	if days == nil {
		_, err := r.db.Exec(ctx, `DELETE FROM app_settings WHERE key=$1`, archiveRetentionSettingKey)
		if err != nil {
			tracing.RecordError(span, err)
		}
		return err
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO app_settings (key, value, updated_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE
		SET value=EXCLUDED.value,
		    updated_at=EXCLUDED.updated_at
	`, archiveRetentionSettingKey, strconv.Itoa(*days), updatedAt)
	if err != nil {
		tracing.RecordError(span, err)
	}
	return err
}

func (r *SettingsRepository) ReplaceBoardSnapshot(ctx context.Context, tasks []domain.Task, archived []domain.ArchivedTask) error {
	ctx, span := tracing.StartClientSpan(ctx, settingsRepositoryTracerScope, "postgres.settings.replace_board_snapshot",
		attribute.String("db.system", "postgresql"),
		attribute.String("db.collection", "tasks"),
		attribute.Int("flux.tasks.count", len(tasks)),
		attribute.Int("flux.archived_tasks.count", len(archived)),
	)
	defer span.End()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		tracing.RecordError(span, err)
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := deferTaskOrderConstraint(ctx, tx); err != nil {
		tracing.RecordError(span, err)
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM tasks`); err != nil {
		tracing.RecordError(span, err)
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM archived_tasks`); err != nil {
		tracing.RecordError(span, err)
		return err
	}

	for _, task := range normalizeTasksForImport(tasks) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO tasks (id, title, note, due, priority, status, sort_order, last_updated)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, task.ID, task.Title, task.Note, task.Due, task.Priority, task.Status, task.SortOrder, task.LastUpdated); err != nil {
			tracing.RecordError(span, err)
			return err
		}
	}

	for _, task := range archived {
		if _, err := tx.Exec(ctx, `
			INSERT INTO archived_tasks (id, title, note, due, priority, status, sort_order, archived_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, task.ID, task.Title, task.Note, task.Due, task.Priority, task.Status, task.SortOrder, task.ArchivedAt); err != nil {
			tracing.RecordError(span, err)
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		tracing.RecordError(span, err)
		return err
	}
	return nil
}

func normalizeTasksForImport(tasks []domain.Task) []domain.Task {
	grouped := make(map[string][]domain.Task, 3)
	statuses := make([]string, 0, 3)
	for _, task := range tasks {
		if _, ok := grouped[task.Status]; !ok {
			statuses = append(statuses, task.Status)
		}
		grouped[task.Status] = append(grouped[task.Status], task)
	}
	sort.Strings(statuses)

	normalized := make([]domain.Task, 0, len(tasks))
	for _, status := range statuses {
		lane := grouped[status]
		sort.SliceStable(lane, func(i, j int) bool {
			if lane[i].SortOrder != lane[j].SortOrder {
				return lane[i].SortOrder < lane[j].SortOrder
			}
			if lane[i].LastUpdated != lane[j].LastUpdated {
				return lane[i].LastUpdated > lane[j].LastUpdated
			}
			return lane[i].ID < lane[j].ID
		})
		for idx := range lane {
			lane[idx].SortOrder = idx
			normalized = append(normalized, lane[idx])
		}
	}
	return normalized
}

func lookupArchiveRetentionDays(ctx context.Context, db *pgxpool.Pool) (*int, error) {
	var raw string
	err := db.QueryRow(ctx, `
		SELECT value
		FROM app_settings
		WHERE key=$1
	`, archiveRetentionSettingKey).Scan(&raw)
	if err == nil {
		value, parseErr := strconv.Atoi(strings.TrimSpace(raw))
		if parseErr != nil {
			return nil, fmt.Errorf("parse archive retention days: %w", parseErr)
		}
		return &value, nil
	}
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return nil, err
}
