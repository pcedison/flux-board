package main

import (
	"context"
	"net/http"

	transporthttp "flux-board/internal/transport/http"
)

const requestIDHeader = transporthttp.RequestIDHeader

func observabilityMiddleware(next http.Handler) http.Handler {
	return transporthttp.ObservabilityMiddleware(next)
}

func requestIDFromContext(ctx context.Context) string {
	return transporthttp.RequestIDFromContext(ctx)
}

func shouldObserveRequest(path string) bool {
	return transporthttp.ShouldObserveRequest(path)
}
