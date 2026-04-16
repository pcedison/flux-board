package main

import (
	"context"
	"net"
	"net/http"
	"strings"
)

func sessionFromContext(ctx context.Context) (sessionState, bool) {
	session, ok := ctx.Value(sessionContextKey).(sessionState)
	return session, ok
}

func clientIDFromRequest(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
