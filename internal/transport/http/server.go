package transporthttp

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	stdhttp "net/http"
	"strings"
	"time"
)

type MuxOptions struct {
	LegacyFS      fs.FS
	WebFS         fs.FS
	Observability *Observability
}

type ServerOptions struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	Observability     *Observability
}

func NewMux(handler *Handler, options MuxOptions) (*stdhttp.ServeMux, error) {
	if options.LegacyFS == nil {
		return nil, errors.New("legacy fs is required")
	}

	mux := stdhttp.NewServeMux()
	observability := ensureObservability(options.Observability)
	mux.Handle("GET /healthz", WithObservedRoute("GET /healthz", stdhttp.HandlerFunc(handler.HandleHealthz)))
	mux.Handle("GET /readyz", WithObservedRoute("GET /readyz", stdhttp.HandlerFunc(handler.HandleReadyz)))
	mux.Handle("GET /metrics", WithObservedRoute("GET /metrics", observability.MetricsHandler()))
	mux.Handle("GET /api/status", WithObservedRoute("GET /api/status", stdhttp.HandlerFunc(handler.HandleStatus)))
	mux.Handle("GET /api/bootstrap/status", WithObservedRoute("GET /api/bootstrap/status", stdhttp.HandlerFunc(handler.HandleBootstrapStatus)))
	mux.Handle("POST /api/bootstrap/setup", WithObservedRoute("POST /api/bootstrap/setup", stdhttp.HandlerFunc(handler.HandleBootstrapSetup)))
	mux.Handle("POST /api/auth/login", WithObservedRoute("POST /api/auth/login", stdhttp.HandlerFunc(handler.HandleLogin)))
	mux.Handle("POST /api/auth/logout", WithObservedRoute("POST /api/auth/logout", stdhttp.HandlerFunc(handler.HandleLogout)))
	mux.Handle("GET /api/auth/me", WithObservedRoute("GET /api/auth/me", stdhttp.HandlerFunc(handler.Auth(handler.HandleGetSession))))

	mux.Handle("GET /api/tasks", WithObservedRoute("GET /api/tasks", stdhttp.HandlerFunc(handler.Auth(handler.HandleGetTasks))))
	mux.Handle("POST /api/tasks", WithObservedRoute("POST /api/tasks", stdhttp.HandlerFunc(handler.Auth(handler.HandleCreateTask))))
	mux.Handle("PUT /api/tasks/{id}", WithObservedRoute("PUT /api/tasks/{id}", stdhttp.HandlerFunc(handler.Auth(handler.HandleUpdateTask))))
	mux.Handle("POST /api/tasks/{id}/reorder", WithObservedRoute("POST /api/tasks/{id}/reorder", stdhttp.HandlerFunc(handler.Auth(handler.HandleReorderTask))))
	mux.Handle("DELETE /api/tasks/{id}", WithObservedRoute("DELETE /api/tasks/{id}", stdhttp.HandlerFunc(handler.Auth(handler.HandleArchiveTask))))

	mux.Handle("GET /api/archived", WithObservedRoute("GET /api/archived", stdhttp.HandlerFunc(handler.Auth(handler.HandleGetArchived))))
	mux.Handle("POST /api/archived/{id}/restore", WithObservedRoute("POST /api/archived/{id}/restore", stdhttp.HandlerFunc(handler.Auth(handler.HandleRestoreTask))))
	mux.Handle("DELETE /api/archived/{id}", WithObservedRoute("DELETE /api/archived/{id}", stdhttp.HandlerFunc(handler.Auth(handler.HandleDeleteArchived))))
	mux.Handle("GET /api/settings", WithObservedRoute("GET /api/settings", stdhttp.HandlerFunc(handler.Auth(handler.HandleGetSettings))))
	mux.Handle("PATCH /api/settings", WithObservedRoute("PATCH /api/settings", stdhttp.HandlerFunc(handler.Auth(handler.HandleUpdateSettings))))
	mux.Handle("POST /api/settings/password", WithObservedRoute("POST /api/settings/password", stdhttp.HandlerFunc(handler.Auth(handler.HandleChangePassword))))
	mux.Handle("GET /api/settings/sessions", WithObservedRoute("GET /api/settings/sessions", stdhttp.HandlerFunc(handler.Auth(handler.HandleGetSessions))))
	mux.Handle("DELETE /api/settings/sessions/{token}", WithObservedRoute("DELETE /api/settings/sessions/{token}", stdhttp.HandlerFunc(handler.Auth(handler.HandleDeleteSession))))
	mux.Handle("GET /api/export", WithObservedRoute("GET /api/export", stdhttp.HandlerFunc(handler.Auth(handler.HandleExport))))
	mux.Handle("POST /api/import", WithObservedRoute("POST /api/import", stdhttp.HandlerFunc(handler.Auth(handler.HandleImport))))

	rootRuntime, err := NewRootWebRuntimeHandler(options.WebFS)
	if err != nil {
		return nil, fmt.Errorf("root runtime: %w", err)
	}
	mux.Handle("GET /legacy", stdhttp.RedirectHandler("/legacy/", stdhttp.StatusPermanentRedirect))
	mux.Handle("/legacy/", stdhttp.StripPrefix("/legacy/", stdhttp.FileServer(stdhttp.FS(options.LegacyFS))))
	mux.Handle("GET /next", nextAliasRedirect("/"))
	mux.Handle("/next/", nextAliasRedirect(""))
	mux.Handle("/", rootRuntime)
	return mux, nil
}

func SecurityHeaders(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Pragma", "no-cache")
		}
		next.ServeHTTP(w, r)
	})
}

func NewServer(port string, handler stdhttp.Handler, options ServerOptions) *stdhttp.Server {
	observability := ensureObservability(options.Observability)
	return &stdhttp.Server{
		Addr:              ":" + port,
		Handler:           observability.Middleware(handler),
		ReadHeaderTimeout: options.ReadHeaderTimeout,
		ReadTimeout:       options.ReadTimeout,
		WriteTimeout:      options.WriteTimeout,
		IdleTimeout:       options.IdleTimeout,
	}
}

func InstallGracefulShutdown(server *stdhttp.Server, shutdownSignals context.Context, timeout time.Duration) {
	go func() {
		<-shutdownSignals.Done()
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			slog.Default().Error("server shutdown error", slog.Any("err", err))
		}
	}()
}

func nextAliasRedirect(defaultPath string) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		target := strings.TrimPrefix(r.URL.Path, "/next")
		if target == "" {
			target = defaultPath
		}
		if target == "" {
			target = "/"
		}
		if !strings.HasPrefix(target, "/") {
			target = "/" + target
		}
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		stdhttp.Redirect(w, r, target, stdhttp.StatusPermanentRedirect)
	})
}
