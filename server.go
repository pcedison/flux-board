package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func newMux(app *App) (*http.ServeMux, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", app.handleHealthz)
	mux.HandleFunc("GET /readyz", app.handleReadyz)
	mux.HandleFunc("POST /api/auth/login", app.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", app.handleLogout)
	mux.HandleFunc("GET /api/auth/me", app.auth(app.handleGetSession))

	mux.HandleFunc("GET /api/tasks", app.auth(app.handleGetTasks))
	mux.HandleFunc("POST /api/tasks", app.auth(app.handleCreateTask))
	mux.HandleFunc("PUT /api/tasks/{id}", app.auth(app.handleUpdateTask))
	mux.HandleFunc("POST /api/tasks/{id}/reorder", app.auth(app.handleReorderTask))
	mux.HandleFunc("DELETE /api/tasks/{id}", app.auth(app.handleArchiveTask))

	mux.HandleFunc("GET /api/archived", app.auth(app.handleGetArchived))
	mux.HandleFunc("POST /api/archived/{id}/restore", app.auth(app.handleRestoreTask))
	mux.HandleFunc("DELETE /api/archived/{id}", app.auth(app.handleDeleteArchived))

	stripped, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, fmt.Errorf("static fs: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(stripped)))
	return mux, nil
}

func newHTTPServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

func installGracefulShutdown(server *http.Server) {
	shutdownSignals, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		defer stop()
		<-shutdownSignals.Done()
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("server shutdown error: %v", err)
		}
	}()
}
