package main

import (
	"context"

	transporthttp "flux-board/internal/transport/http"
)

const requestIDHeader = transporthttp.RequestIDHeader

func requestIDFromContext(ctx context.Context) string {
	return transporthttp.RequestIDFromContext(ctx)
}

func shouldObserveRequest(path string) bool {
	return transporthttp.ShouldObserveRequest(path)
}
