package main

import (
	"flux-board/internal/domain"
	storepostgres "flux-board/internal/store/postgres"
)

func (a *App) settingsRepository() domain.SettingsRepository {
	if a.settingsRepo != nil {
		return a.settingsRepo
	}
	if a.db == nil {
		return nil
	}
	return storepostgres.NewSettingsRepository(a.db)
}
