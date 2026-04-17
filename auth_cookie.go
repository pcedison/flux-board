package main

import (
	"net/http"
	"time"

	transporthttp "flux-board/internal/transport/http"
)

func setSessionCookie(w http.ResponseWriter, token string, expiry time.Time, secure bool) {
	transporthttp.SetSessionCookie(w, token, expiry, secure)
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	transporthttp.ClearSessionCookie(w, secure)
}
