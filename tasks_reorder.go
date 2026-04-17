package main

import "net/http"

func (a *App) handleReorderTask(w http.ResponseWriter, r *http.Request) {
	a.transportHandler().HandleReorderTask(w, r)
}
