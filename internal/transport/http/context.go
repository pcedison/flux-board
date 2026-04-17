package transporthttp

import (
	"context"
	"net"
	stdhttp "net/http"
	"strings"

	"flux-board/internal/domain"
)

type sessionContextKey struct{}

func SessionFromContext(ctx context.Context) (domain.Session, bool) {
	session, ok := ctx.Value(sessionContextKey{}).(domain.Session)
	return session, ok
}

func withSession(ctx context.Context, session domain.Session) context.Context {
	return context.WithValue(ctx, sessionContextKey{}, session)
}

func ClientIDFromRequest(r *stdhttp.Request) string {
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
