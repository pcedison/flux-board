package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	transporthttp "flux-board/internal/transport/http"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewHTTPServerAddsRequestIDHeaderToObservedRoutes(t *testing.T) {
	observability := transporthttp.NewObservability(transporthttp.ObservabilityOptions{
		Registry: prometheus.NewRegistry(),
	})
	server := newHTTPServer("8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := requestIDFromContext(r.Context()); got == "" {
			t.Fatal("expected request id in context")
		}
		w.WriteHeader(http.StatusNoContent)
	}), observability)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
	if got := rec.Header().Get(requestIDHeader); got == "" {
		t.Fatal("expected request id response header")
	}
}

func TestNewHTTPServerLogsObservedRequestsAsStructuredJSONOnly(t *testing.T) {
	var logBuffer bytes.Buffer
	observability := transporthttp.NewObservability(transporthttp.ObservabilityOptions{
		Logger:   transporthttp.NewLogger("production", &logBuffer),
		Registry: prometheus.NewRegistry(),
	})
	server := newHTTPServer("8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldObserveRequest(r.URL.Path) && requestIDFromContext(r.Context()) == "" {
			t.Fatal("expected request id in context")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}), observability)

	apiReq := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	apiReq.RemoteAddr = "127.0.0.1:1234"
	apiRec := httptest.NewRecorder()
	server.Handler.ServeHTTP(apiRec, apiReq)

	lines := strings.Split(strings.TrimSpace(logBuffer.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected one structured log line, got %d (%q)", len(lines), logBuffer.String())
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("decode structured log: %v", err)
	}
	if entry["msg"] != "http access" {
		t.Fatalf("expected access log message, got %+v", entry)
	}
	if entry["path"] != "/readyz" {
		t.Fatalf("expected path /readyz, got %+v", entry)
	}
	if entry["route"] != "/readyz" {
		t.Fatalf("expected normalized route /readyz, got %+v", entry)
	}
	if entry["status"] != float64(http.StatusOK) {
		t.Fatalf("expected status 200, got %+v", entry)
	}
	if entry["request_id"] != apiRec.Header().Get(requestIDHeader) {
		t.Fatalf("expected request id %q, got %+v", apiRec.Header().Get(requestIDHeader), entry)
	}

	logBuffer.Reset()

	staticReq := httptest.NewRequest(http.MethodGet, "/legacy/", nil)
	staticRec := httptest.NewRecorder()
	server.Handler.ServeHTTP(staticRec, staticReq)

	if logBuffer.Len() != 0 {
		t.Fatalf("expected no access log for legacy static route, got %q", logBuffer.String())
	}
	if staticRec.Header().Get(requestIDHeader) != "" {
		t.Fatalf("expected no request id header for legacy static route, got %q", staticRec.Header().Get(requestIDHeader))
	}
}

func TestRequestIDFromContextReturnsEmptyWhenMissing(t *testing.T) {
	if got := requestIDFromContext(context.Background()); got != "" {
		t.Fatalf("expected empty request id, got %q", got)
	}
}
