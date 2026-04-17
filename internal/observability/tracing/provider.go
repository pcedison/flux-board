package tracing

import (
	"context"
	"log/slog"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const defaultServiceName = "flux-board"

type ShutdownFunc func(context.Context) error

type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string
	Logger         *slog.Logger
}

func Configure(ctx context.Context, cfg Config) (ShutdownFunc, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		otel.SetTracerProvider(noop.NewTracerProvider())
		otel.SetTextMapPropagator(propagation.TraceContext{})
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(serviceAttributes(cfg)...))
	if err != nil {
		return nil, err
	}

	exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint))
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	logger.Info(
		"tracing enabled",
		slog.String("endpoint", endpoint),
		slog.String("service", defaultString(cfg.ServiceName, defaultServiceName)),
	)
	return provider.Shutdown, nil
}

func Tracer(scope string) trace.Tracer {
	scope = strings.Trim(strings.TrimSpace(scope), "/")
	if scope == "" {
		scope = "app"
	}
	return otel.Tracer(defaultServiceName + "/" + scope)
}

func StartInternalSpan(ctx context.Context, scope, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return Tracer(scope).Start(ctx, name, trace.WithSpanKind(trace.SpanKindInternal), trace.WithAttributes(attrs...))
}

func StartClientSpan(ctx context.Context, scope, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return Tracer(scope).Start(ctx, name, trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(attrs...))
}

func RecordError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func serviceAttributes(cfg Config) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("service.name", defaultString(cfg.ServiceName, defaultServiceName)),
	}
	if version := strings.TrimSpace(cfg.ServiceVersion); version != "" {
		attrs = append(attrs, attribute.String("service.version", version))
	}
	if env := strings.TrimSpace(cfg.Environment); env != "" {
		attrs = append(attrs, attribute.String("deployment.environment.name", env))
	}
	return attrs
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}
