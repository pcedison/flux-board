package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"

	transporthttp "flux-board/internal/transport/http"
)

func (a *App) transportHandler() *transporthttp.Handler {
	return transporthttp.NewHandler(
		a.taskService(),
		a.authService(),
		transporthttp.HandlerOptions{
			CookieSecure:     a.cookieSecure,
			AuthBodyLimit:    authBodyLimit,
			TaskBodyLimit:    taskBodyLimit,
			ReadinessChecker: a.checkReadiness,
		},
	)
}

func newMux(app *App) (*http.ServeMux, error) {
	stripped, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, fmt.Errorf("static fs: %w", err)
	}
	return transporthttp.NewMux(app.transportHandler(), transporthttp.MuxOptions{
		LegacyFS: stripped,
		WebFS:    app.webRuntimeFS,
	})
}

func newHTTPServer(port string, handler http.Handler) *http.Server {
	return transporthttp.NewServer(port, handler, transporthttp.ServerOptions{
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	})
}

func installGracefulShutdown(server *http.Server, shutdownSignals context.Context) {
	transporthttp.InstallGracefulShutdown(server, shutdownSignals, shutdownTimeout)
}
