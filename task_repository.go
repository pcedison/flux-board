package main

import (
	"flux-board/internal/domain"
	storepostgres "flux-board/internal/store/postgres"
)

type TaskRepository = domain.TaskRepository

func (a *App) taskRepository() TaskRepository {
	if a.taskRepo != nil {
		return a.taskRepo
	}
	if a.db == nil {
		return nil
	}
	return storepostgres.NewTaskRepository(a.db, archiveRetention)
}
