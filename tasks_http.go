package main

import "net/http"

func (a *App) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	a.transportHandler().HandleGetTasks(w, r)
}

func (a *App) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	a.transportHandler().HandleCreateTask(w, r)
}

func (a *App) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	a.transportHandler().HandleUpdateTask(w, r)
}

func (a *App) handleArchiveTask(w http.ResponseWriter, r *http.Request) {
	a.transportHandler().HandleArchiveTask(w, r)
}

func (a *App) handleGetArchived(w http.ResponseWriter, r *http.Request) {
	a.transportHandler().HandleGetArchived(w, r)
}

func (a *App) handleRestoreTask(w http.ResponseWriter, r *http.Request) {
	a.transportHandler().HandleRestoreTask(w, r)
}

func (a *App) handleDeleteArchived(w http.ResponseWriter, r *http.Request) {
	a.transportHandler().HandleDeleteArchived(w, r)
}
