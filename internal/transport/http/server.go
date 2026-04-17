package transporthttp

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	stdhttp "net/http"
	"strings"
	"time"
)

type MuxOptions struct {
	LegacyFS fs.FS
	WebFS    fs.FS
}

type ServerOptions struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func NewMux(handler *Handler, options MuxOptions) (*stdhttp.ServeMux, error) {
	if options.LegacyFS == nil {
		return nil, errors.New("legacy fs is required")
	}

	mux := stdhttp.NewServeMux()
	mux.HandleFunc("GET /healthz", handler.HandleHealthz)
	mux.HandleFunc("GET /readyz", handler.HandleReadyz)
	mux.HandleFunc("POST /api/auth/login", handler.HandleLogin)
	mux.HandleFunc("POST /api/auth/logout", handler.HandleLogout)
	mux.HandleFunc("GET /api/auth/me", handler.Auth(handler.HandleGetSession))

	mux.HandleFunc("GET /api/tasks", handler.Auth(handler.HandleGetTasks))
	mux.HandleFunc("POST /api/tasks", handler.Auth(handler.HandleCreateTask))
	mux.HandleFunc("PUT /api/tasks/{id}", handler.Auth(handler.HandleUpdateTask))
	mux.HandleFunc("POST /api/tasks/{id}/reorder", handler.Auth(handler.HandleReorderTask))
	mux.HandleFunc("DELETE /api/tasks/{id}", handler.Auth(handler.HandleArchiveTask))

	mux.HandleFunc("GET /api/archived", handler.Auth(handler.HandleGetArchived))
	mux.HandleFunc("POST /api/archived/{id}/restore", handler.Auth(handler.HandleRestoreTask))
	mux.HandleFunc("DELETE /api/archived/{id}", handler.Auth(handler.HandleDeleteArchived))

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
	return &stdhttp.Server{
		Addr:              ":" + port,
		Handler:           ObservabilityMiddleware(handler),
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
			log.Printf("server shutdown error: %v", err)
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
