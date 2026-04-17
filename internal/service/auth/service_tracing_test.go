package auth

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestAuthenticateEmitsTracingSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider()
	provider.RegisterSpanProcessor(recorder)
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(noop.NewTracerProvider())
		_ = provider.Shutdown(context.Background())
	})

	service := New(nil, Options{
		PasswordVerifier: func(context.Context, string) (bool, error) {
			return true, nil
		},
		TokenGenerator: func() string {
			return "generated-token"
		},
		SessionCreator: func(context.Context, string, string, string, time.Time) error {
			return nil
		},
	})

	if _, err := service.Authenticate(context.Background(), "secret", "127.0.0.1"); err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}

	ended := recorder.Ended()
	if len(ended) == 0 {
		t.Fatal("expected at least one finished span")
	}

	var authenticateSpan sdktrace.ReadOnlySpan
	for _, span := range ended {
		if span.Name() == "auth.authenticate" {
			authenticateSpan = span
			break
		}
	}
	if authenticateSpan == nil {
		t.Fatalf("expected auth.authenticate span, got %d spans", len(ended))
	}

	attrs := authenticateSpan.Attributes()
	if !hasTraceAttribute(attrs, "auth.outcome", "success") {
		t.Fatalf("expected success outcome attribute, got %+v", attrs)
	}
}

func hasTraceAttribute(attrs []attribute.KeyValue, key, value string) bool {
	for _, attr := range attrs {
		if string(attr.Key) == key && attr.Value.AsString() == value {
			return true
		}
	}
	return false
}
