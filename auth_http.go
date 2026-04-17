package main

import "net/http"

func (a *App) auth(next http.HandlerFunc) http.HandlerFunc {
	return a.transportHandler().Auth(next)
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	a.transportHandler().HandleLogin(w, r)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	a.transportHandler().HandleLogout(w, r)
}

func (a *App) handleGetSession(w http.ResponseWriter, r *http.Request) {
	a.transportHandler().HandleGetSession(w, r)
}
