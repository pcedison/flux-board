package main

import (
	"flux-board/internal/domain"
	storepostgres "flux-board/internal/store/postgres"
)

func (a *App) authRepository() domain.AuthRepository {
	if a.authRepo != nil {
		return a.authRepo
	}
	if a.db == nil {
		return nil
	}
	return storepostgres.NewAuthRepository(a.db)
}
