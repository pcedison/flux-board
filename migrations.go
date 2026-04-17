package main

import (
	"context"
	"embed"

	storepostgres "flux-board/internal/store/postgres"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

var requiredSchemaObjects = storepostgres.RequiredSchemaObjects
var requiredSchemaConstraints = storepostgres.RequiredSchemaConstraints

func (a *App) initSchema() error {
	return storepostgres.InitializeSchema(
		context.Background(),
		a.db,
		migrationFiles,
		"migrations/*.sql",
		bootstrapAdmin,
		a.bootstrapPassword,
	)
}
