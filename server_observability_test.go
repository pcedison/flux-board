package main

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewHTTPServerAddsRequestIDHeaderToObservedRoutes(t *testing.T) {
	server := newHTTPServer("8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := requestIDFromContext(r.Context()); got == "" {
			t.Fatal("expected request id in context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

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

func TestNewHTTPServerLogsObservedRequestsOnly(t *testing.T) {
	var logBuffer bytes.Buffer
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	log.SetOutput(&logBuffer)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
	})

	server := newHTTPServer("8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldObserveRequest(r.URL.Path) && requestIDFromContext(r.Context()) == "" {
			t.Fatal("expected request id in context")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	apiReq := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	apiReq.RemoteAddr = "127.0.0.1:1234"
	apiRec := httptest.NewRecorder()
	server.Handler.ServeHTTP(apiRec, apiReq)

	if !strings.Contains(logBuffer.String(), "path=/readyz") {
		t.Fatalf("expected access log for observed route, got %q", logBuffer.String())
	}
	if !strings.Contains(logBuffer.String(), "status=200") {
		t.Fatalf("expected status in access log, got %q", logBuffer.String())
	}
	if !strings.Contains(logBuffer.String(), "request_id="+apiRec.Header().Get(requestIDHeader)) {
		t.Fatalf("expected request id in access log, got %q", logBuffer.String())
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
