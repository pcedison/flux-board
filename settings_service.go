package main

import (
	settingsservice "flux-board/internal/service/settings"
)

type SettingsService = settingsservice.Service

func (a *App) settingsService() SettingsService {
	if a.settingsSvc != nil {
		return a.settingsSvc
	}
	return settingsservice.New(
		a.authRepository(),
		a.settingsRepository(),
		a.taskRepository(),
		a.authService(),
		appVersion(),
		settingsservice.Options{},
	)
}
