package main

import (
	"net/http"

	transporthttp "flux-board/internal/transport/http"
)

func clientIDFromRequest(r *http.Request) string {
	return transporthttp.ClientIDFromRequest(r)
}
