package transporthttp

import (
	stdhttp "net/http"
	"time"
)

const CookieName = "flux_session"

func SetSessionCookie(w stdhttp.ResponseWriter, token string, expiry time.Time, secure bool) {
	stdhttp.SetCookie(w, &stdhttp.Cookie{
		Name:     CookieName,
		Value:    token,
		Expires:  expiry,
		HttpOnly: true,
		SameSite: stdhttp.SameSiteStrictMode,
		Secure:   secure,
		Path:     "/",
	})
}

func ClearSessionCookie(w stdhttp.ResponseWriter, secure bool) {
	stdhttp.SetCookie(w, &stdhttp.Cookie{
		Name:     CookieName,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: stdhttp.SameSiteStrictMode,
		Secure:   secure,
		Path:     "/",
	})
}
