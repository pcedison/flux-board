package main

import (
	"net/http"
	"time"
)

func setSessionCookie(w http.ResponseWriter, token string, expiry time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Expires:  expiry,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		Path:     "/",
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		Path:     "/",
	})
}
