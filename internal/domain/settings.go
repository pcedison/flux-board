package domain

import "context"

type SessionInfo struct {
	Token      string `json:"token"`
	CreatedAt  int64  `json:"createdAt"`
	ExpiresAt  int64  `json:"expiresAt"`
	LastSeenAt int64  `json:"lastSeenAt"`
	ClientIP   string `json:"clientIP"`
	Current    bool   `json:"current"`
}

type AppSettings struct {
	ArchiveRetentionDays *int `json:"archiveRetentionDays"`
}

type ExportBundle struct {
	Version    string         `json:"version"`
	ExportedAt int64          `json:"exportedAt"`
	Settings   AppSettings    `json:"settings"`
	Tasks      []Task         `json:"tasks"`
	Archived   []ArchivedTask `json:"archived"`
}

type SettingsRepository interface {
	BootstrapAdminExists(context.Context, string) (bool, error)
	UpdatePasswordHash(context.Context, string, string, int64) error
	ListSessions(context.Context, string) ([]SessionInfo, error)
	DeleteSessionsExcept(context.Context, string, []string) error
	GetArchiveRetentionDays(context.Context) (*int, error)
	SetArchiveRetentionDays(context.Context, *int, int64) error
	ReplaceBoardSnapshot(context.Context, []Task, []ArchivedTask) error
}
