package main

import (
	"context"
	"net/http"

	transporthttp "flux-board/internal/transport/http"
)

func sessionFromContext(ctx context.Context) (sessionState, bool) {
	return transporthttp.SessionFromContext(ctx)
}

func clientIDFromRequest(r *http.Request) string {
	return transporthttp.ClientIDFromRequest(r)
}
