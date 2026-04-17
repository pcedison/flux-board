package transporthttp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	stdhttp "net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const RequestIDHeader = "X-Request-Id"

type requestIDContextKey struct{}
type loggerContextKey struct{}

type ObservabilityOptions struct {
	Logger   *slog.Logger
	Registry *prometheus.Registry
}

type Observability struct {
	logger          *slog.Logger
	registry        *prometheus.Registry
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	metricsHandler  stdhttp.Handler
}

func NewLogger(appEnv string, writer io.Writer) *slog.Logger {
	if writer == nil {
		writer = os.Stdout
	}

	switch strings.ToLower(strings.TrimSpace(appEnv)) {
	case "", "production":
		return slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{}))
	default:
		return slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{}))
	}
}

func NewObservability(options ObservabilityOptions) *Observability {
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}

	registry := options.Registry
	if registry == nil {
		registry = prometheus.NewRegistry()
	}

	obs := &Observability{
		logger:   logger,
		registry: registry,
		requestsTotal: registerCounterVec(registry, prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "flux_board_http_requests_total",
				Help: "Total observed HTTP requests handled by Flux Board.",
			},
			[]string{"method", "route", "status"},
		)),
		requestDuration: registerHistogramVec(registry, prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "flux_board_http_request_duration_seconds",
				Help:    "Observed HTTP request latency in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "route", "status"},
		)),
	}

	registerCollector(registry, collectors.NewGoCollector())
	registerCollector(registry, collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	obs.metricsHandler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	return obs
}

func ObservabilityMiddleware(next stdhttp.Handler) stdhttp.Handler {
	return NewObservability(ObservabilityOptions{}).Middleware(next)
}

func (o *Observability) Middleware(next stdhttp.Handler) stdhttp.Handler {
	obs := ensureObservability(o)
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if !ShouldObserveRequest(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		requestID := newRequestID()
		requestLogger := obs.logger.With(slog.String("request_id", requestID))
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		ctx = context.WithValue(ctx, loggerContextKey{}, requestLogger)

		rec := &statusRecorder{ResponseWriter: w}
		start := time.Now()
		observedRequest := r.WithContext(ctx)

		rec.Header().Set(RequestIDHeader, requestID)
		next.ServeHTTP(rec, observedRequest)

		route := rec.RoutePattern()
		if route == "" {
			route = normalizeObservedRoute(observedRequest)
		}
		status := strconv.Itoa(rec.StatusCode())
		duration := time.Since(start)

		obs.requestsTotal.WithLabelValues(observedRequest.Method, route, status).Inc()
		obs.requestDuration.WithLabelValues(observedRequest.Method, route, status).Observe(duration.Seconds())
		requestLogger.Info(
			"http access",
			slog.String("client", ClientIDFromRequest(observedRequest)),
			slog.String("method", observedRequest.Method),
			slog.String("route", route),
			slog.String("path", observedRequest.URL.Path),
			slog.Int("status", rec.StatusCode()),
			slog.Int("bytes", rec.BytesWritten()),
			slog.Int64("duration_ms", duration.Milliseconds()),
		)
	})
}

func (o *Observability) MetricsHandler() stdhttp.Handler {
	return ensureObservability(o).metricsHandler
}

func (o *Observability) Logger() *slog.Logger {
	return ensureObservability(o).logger
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func LoggerFromContext(ctx context.Context) *slog.Logger {
	logger, _ := ctx.Value(loggerContextKey{}).(*slog.Logger)
	if logger != nil {
		return logger
	}
	return slog.Default()
}

func ShouldObserveRequest(path string) bool {
	return strings.HasPrefix(path, "/api/") || path == "/healthz" || path == "/readyz" || path == "/metrics"
}

func normalizeObservedRoute(r *stdhttp.Request) string {
	return r.URL.Path
}

func WithObservedRoute(pattern string, next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if recorder, ok := w.(interface{ SetRoutePattern(string) }); ok {
			recorder.SetRoutePattern(pattern)
		}
		next.ServeHTTP(w, r)
	})
}

func newRequestID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err == nil {
		return hex.EncodeToString(buf[:])
	}
	return hex.EncodeToString([]byte(time.Now().Format("150405.000000000")))
}

func ensureObservability(obs *Observability) *Observability {
	if obs != nil {
		return obs
	}
	return NewObservability(ObservabilityOptions{})
}

func registerCollector(registry *prometheus.Registry, collector prometheus.Collector) {
	if err := registry.Register(collector); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return
		}
		panic(err)
	}
}

func registerCounterVec(registry *prometheus.Registry, collector *prometheus.CounterVec) *prometheus.CounterVec {
	if err := registry.Register(collector); err != nil {
		if existing, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return existing.ExistingCollector.(*prometheus.CounterVec)
		}
		panic(err)
	}
	return collector
}

func registerHistogramVec(registry *prometheus.Registry, collector *prometheus.HistogramVec) *prometheus.HistogramVec {
	if err := registry.Register(collector); err != nil {
		if existing, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return existing.ExistingCollector.(*prometheus.HistogramVec)
		}
		panic(err)
	}
	return collector
}

type statusRecorder struct {
	stdhttp.ResponseWriter
	status       int
	bytes        int
	routePattern string
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = stdhttp.StatusOK
	}
	n, err := r.ResponseWriter.Write(p)
	r.bytes += n
	return n, err
}

func (r *statusRecorder) StatusCode() int {
	if r.status == 0 {
		return stdhttp.StatusOK
	}
	return r.status
}

func (r *statusRecorder) BytesWritten() int {
	return r.bytes
}

func (r *statusRecorder) SetRoutePattern(pattern string) {
	r.routePattern = strings.TrimSpace(pattern)
}

func (r *statusRecorder) RoutePattern() string {
	pattern := strings.TrimSpace(r.routePattern)
	if pattern == "" {
		return ""
	}
	if idx := strings.Index(pattern, " "); idx >= 0 {
		pattern = strings.TrimSpace(pattern[idx+1:])
	}
	return pattern
}
