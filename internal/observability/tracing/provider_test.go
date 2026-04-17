package tracing

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestConfigureNoopWhenEndpointUnset(t *testing.T) {
	t.Cleanup(func() {
		otel.SetTracerProvider(noop.NewTracerProvider())
	})

	shutdown, err := Configure(context.Background(), Config{
		ServiceName: "flux-board-test",
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("Configure returned error: %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown returned error: %v", err)
	}

	_, span := Tracer("tests").Start(context.Background(), "noop-span")
	span.End()
}

func TestConfigureExportsToOTLPEndpoint(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/traces" {
			t.Fatalf("expected /v1/traces, got %s", r.URL.Path)
		}
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			t.Fatalf("read body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Cleanup(func() {
		otel.SetTracerProvider(noop.NewTracerProvider())
	})

	shutdown, err := Configure(context.Background(), Config{
		ServiceName:    "flux-board-test",
		ServiceVersion: "test",
		Environment:    "test",
		Endpoint:       server.URL,
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("Configure returned error: %v", err)
	}

	ctx, span := StartInternalSpan(context.Background(), "tests", "exported-span")
	span.AddEvent("test-event")
	span.End()

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown returned error: %v", err)
	}

	if requests.Load() == 0 {
		t.Fatal("expected OTLP exporter to send at least one request")
	}
}
