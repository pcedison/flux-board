package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	transporthttp "flux-board/internal/transport/http"

	"github.com/prometheus/client_golang/prometheus"
)

func (a *App) transportHandler() *transporthttp.Handler {
	return transporthttp.NewHandlerWithSettings(
		a.taskService(),
		a.authService(),
		a.settingsService(),
		transporthttp.HandlerOptions{
			CookieSecure:         a.cookieSecure,
			AuthBodyLimit:        authBodyLimit,
			SettingsBodyLimit:    2 << 20,
			TaskBodyLimit:        taskBodyLimit,
			ReadinessChecker:     a.checkReadiness,
			AppEnvironment:       a.appEnv,
			AppVersion:           a.version,
			ArchiveCleanupEvery:  time.Hour,
			SessionCleanupEvery:  sessionCleanupTicker,
			RuntimeArtifact:      "self-contained-root-runtime",
			RuntimeOwnershipPath: "/",
			LegacyRollbackPath:   "/legacy/",
		},
	)
}

func newMux(app *App) (*http.ServeMux, error) {
	stripped, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, fmt.Errorf("static fs: %w", err)
	}
	return transporthttp.NewMux(app.transportHandler(), transporthttp.MuxOptions{
		LegacyFS:      stripped,
		WebFS:         app.webRuntimeFS,
		Observability: app.observabilityRuntime(),
	})
}

func newHTTPServer(port string, handler http.Handler, observability ...*transporthttp.Observability) *http.Server {
	var obs *transporthttp.Observability
	if len(observability) > 0 {
		obs = observability[0]
	}
	return transporthttp.NewServer(port, handler, transporthttp.ServerOptions{
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		Observability:     obs,
	})
}

func installGracefulShutdown(server *http.Server, shutdownSignals context.Context) {
	transporthttp.InstallGracefulShutdown(server, shutdownSignals, shutdownTimeout)
}

func (a *App) observabilityRuntime() *transporthttp.Observability {
	if a != nil && a.observability != nil {
		return a.observability
	}

	var registry = (*prometheus.Registry)(nil)
	if a != nil {
		registry = a.metricsRegistry
	}
	observability := transporthttp.NewObservability(transporthttp.ObservabilityOptions{
		Registry: registry,
	})
	if a != nil {
		a.observability = observability
	}
	return observability
}
